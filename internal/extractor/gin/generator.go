package gin

import (
	"context"
	"fmt"
	"go/ast"
	"os"
	"path/filepath"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/spencercjh/spec-forge/internal/extractor"
	"gopkg.in/yaml.v3"
)

// Generator generates OpenAPI specs from Gin projects using AST parsing.
type Generator struct{}

// NewGenerator creates a new Generator instance.
func NewGenerator() *Generator {
	return &Generator{}
}

// Generate generates OpenAPI spec from Gin project.
func (g *Generator) Generate(ctx context.Context, projectPath string, info *extractor.ProjectInfo, opts *extractor.GenerateOptions) (*extractor.GenerateResult, error) {
	ginInfo, ok := info.FrameworkData.(*Info)
	if !ok {
		return nil, fmt.Errorf("invalid framework data type")
	}

	// Step 1: Parse AST files
	parser := NewASTParser(projectPath)
	if err := parser.ParseFiles(); err != nil {
		return nil, fmt.Errorf("failed to parse files: %w", err)
	}

	// Step 2: Extract routes
	routes, err := parser.ExtractRoutes()
	if err != nil {
		return nil, fmt.Errorf("failed to extract routes: %w", err)
	}

	// Step 3: Analyze handlers
	analyzer := NewHandlerAnalyzer(parser.fset, parser.files)
	handlerInfos := make(map[string]*HandlerInfo)
	for _, route := range routes {
		// Find handler function
		handlerDecl := g.findHandlerDecl(route.HandlerName, parser.files)
		if handlerDecl != nil {
			handlerInfo, err := analyzer.AnalyzeHandler(handlerDecl)
			if err != nil {
				continue // Log but continue
			}
			handlerInfos[route.HandlerName] = handlerInfo
		}
	}

	// Step 4: Extract schemas
	schemaExtractor := NewSchemaExtractor(parser.files)
	schemas := make(openapi3.Schemas)
	for _, handlerInfo := range handlerInfos {
		if handlerInfo.BodyType != "" && handlerInfo.BodyType != "map[string]any" {
			if schema, err := schemaExtractor.ExtractSchema(handlerInfo.BodyType); err == nil {
				schemas[handlerInfo.BodyType] = schema
			}
		}
		for _, resp := range handlerInfo.Responses {
			if resp.GoType != "" && resp.GoType != "map[string]any" {
				if schema, err := schemaExtractor.ExtractSchema(resp.GoType); err == nil {
					schemas[resp.GoType] = schema
				}
			}
		}
	}

	// Step 5: Build OpenAPI document
	doc := g.buildOpenAPIDoc(ginInfo, routes, handlerInfos, schemas)

	// Step 6: Write output
	outputPath, err := g.writeOutput(doc, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to write output: %w", err)
	}

	return &extractor.GenerateResult{
		SpecFilePath: outputPath,
		Format:       opts.Format,
	}, nil
}

// findHandlerDecl finds a handler function declaration by name.
func (g *Generator) findHandlerDecl(name string, files map[string]*ast.File) *ast.FuncDecl {
	if name == "" {
		return nil
	}
	for _, file := range files {
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == name {
				return fn
			}
		}
	}
	return nil
}

// buildOpenAPIDoc builds the OpenAPI document.
func (g *Generator) buildOpenAPIDoc(info *Info, routes []Route, handlerInfos map[string]*HandlerInfo, schemas openapi3.Schemas) *openapi3.T {
	title := "Gin API"
	if info.ModuleName != "" {
		title = info.ModuleName
	}

	doc := &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:   title,
			Version: "1.0.0",
		},
		Paths: openapi3.NewPaths(),
	}

	if len(schemas) > 0 {
		doc.Components = &openapi3.Components{
			Schemas: schemas,
		}
	}

	// Build paths
	for _, route := range routes {
		pathItem := doc.Paths.Find(route.FullPath)
		if pathItem == nil {
			pathItem = &openapi3.PathItem{}
			doc.Paths.Set(route.FullPath, pathItem)
		}

		operation := g.buildOperation(route, handlerInfos[route.HandlerName], schemas)
		setOperationForMethod(pathItem, route.Method, operation)
	}

	return doc
}

// buildOperation builds an OpenAPI operation from a route.
func (g *Generator) buildOperation(route Route, handlerInfo *HandlerInfo, schemas openapi3.Schemas) *openapi3.Operation {
	operation := &openapi3.Operation{
		OperationID: route.HandlerName,
		Summary:     route.HandlerName,
	}

	if handlerInfo == nil {
		return operation
	}

	// Add parameters
	operation.Parameters = make(openapi3.Parameters, 0)

	// Path parameters
	for _, param := range handlerInfo.PathParams {
		operation.Parameters = append(operation.Parameters, &openapi3.ParameterRef{
			Value: &openapi3.Parameter{
				Name:        param.Name,
				In:          "path",
				Required:    param.Required,
				Description: "Path parameter",
				Schema:      &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"string"}}},
			},
		})
	}

	// Query parameters
	for _, param := range handlerInfo.QueryParams {
		operation.Parameters = append(operation.Parameters, &openapi3.ParameterRef{
			Value: &openapi3.Parameter{
				Name:        param.Name,
				In:          "query",
				Required:    param.Required,
				Description: "Query parameter",
				Schema:      &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"string"}}},
			},
		})
	}

	// Request body
	if handlerInfo.BodyType != "" && handlerInfo.BodyType != "map[string]any" {
		var schemaRef *openapi3.SchemaRef
		if _, exists := schemas[handlerInfo.BodyType]; exists {
			schemaRef = &openapi3.SchemaRef{Ref: "#/components/schemas/" + handlerInfo.BodyType}
		} else {
			// Fallback to generic object if schema not found
			schemaRef = &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"object"}}}
		}
		operation.RequestBody = &openapi3.RequestBodyRef{
			Value: &openapi3.RequestBody{
				Content: openapi3.Content{
					"application/json": {
						Schema: schemaRef,
					},
				},
			},
		}
	}

	// Responses
	operation.Responses = openapi3.NewResponses()
	for _, resp := range handlerInfo.Responses {
		desc := fmt.Sprintf("HTTP %d response", resp.StatusCode)
		response := &openapi3.Response{
			Description: &desc,
			Content:     openapi3.Content{},
		}

		if resp.GoType != "" {
			if resp.GoType == "map[string]any" {
				response.Content["application/json"] = &openapi3.MediaType{
					Schema: &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"object"}}},
				}
			} else if _, exists := schemas[resp.GoType]; exists {
				response.Content["application/json"] = &openapi3.MediaType{
					Schema: &openapi3.SchemaRef{
						Ref: "#/components/schemas/" + resp.GoType,
					},
				}
			} else {
				// Fallback to generic object if schema not found
				response.Content["application/json"] = &openapi3.MediaType{
					Schema: &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"object"}}},
				}
			}
		}

		statusCode := fmt.Sprintf("%d", resp.StatusCode)
		operation.Responses.Set(statusCode, &openapi3.ResponseRef{Value: response})
	}

	// Default response if none specified
	if len(handlerInfo.Responses) == 0 {
		desc := "Success"
		operation.Responses.Set("200", &openapi3.ResponseRef{
			Value: &openapi3.Response{
				Description: &desc,
			},
		})
	}

	return operation
}

// setOperationForMethod sets the operation for the given HTTP method.
func setOperationForMethod(pathItem *openapi3.PathItem, method string, operation *openapi3.Operation) {
	switch method {
	case "GET":
		pathItem.Get = operation
	case "POST":
		pathItem.Post = operation
	case "PUT":
		pathItem.Put = operation
	case "DELETE":
		pathItem.Delete = operation
	case "PATCH":
		pathItem.Patch = operation
	case "HEAD":
		pathItem.Head = operation
	case "OPTIONS":
		pathItem.Options = operation
	}
}

// writeOutput writes the OpenAPI document to file.
func (g *Generator) writeOutput(doc *openapi3.T, opts *extractor.GenerateOptions) (string, error) {
	outputDir := opts.OutputDir
	if outputDir == "" {
		outputDir = "."
	}

	outputFile := opts.OutputFile
	if outputFile == "" {
		outputFile = "openapi"
	}

	format := opts.Format
	if format == "" {
		format = "json"
	}

	var data []byte
	var err error
	var ext string

	switch format {
	case "yaml", "yml":
		data, err = yaml.Marshal(doc)
		ext = ".yaml"
	default: // json
		data, err = doc.MarshalJSON()
		ext = ".json"
	}

	if err != nil {
		return "", err
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	outputPath := filepath.Join(outputDir, outputFile+ext)
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return "", err
	}

	return outputPath, nil
}
