package gin

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"gopkg.in/yaml.v3"

	"github.com/spencercjh/spec-forge/internal/extractor"
)

// Generator generates OpenAPI specs from Gin projects using AST parsing.
type Generator struct{}

// ginHType represents the gin.H map type for OpenAPI spec generation.
const ginHType = "map[string]any"

// NewGenerator creates a new Generator instance.
func NewGenerator() *Generator {
	return &Generator{}
}

// Generate generates OpenAPI spec from Gin project.
func (g *Generator) Generate(ctx context.Context, projectPath string, info *extractor.ProjectInfo, opts *extractor.GenerateOptions) (*extractor.GenerateResult, error) {
	slog.InfoContext(ctx, "Generating OpenAPI spec from Gin project", "path", projectPath)

	ginInfo, ok := info.FrameworkData.(*Info)
	if !ok {
		slog.ErrorContext(ctx, "Invalid framework data type")
		return nil, errors.New("invalid framework data type")
	}

	// Step 1: Parse AST files
	slog.DebugContext(ctx, "Parsing AST files", "path", projectPath)
	parser := NewASTParser(projectPath)
	if err := parser.ParseFiles(); err != nil {
		slog.ErrorContext(ctx, "Failed to parse files", "error", err)
		return nil, fmt.Errorf("failed to parse files: %w", err)
	}
	slog.DebugContext(ctx, "Parsed AST files", "count", len(parser.files))

	// Step 2: Extract routes
	routes, err := parser.ExtractRoutes(opts.ExcludeRoutes, opts.ExcludeRoutePrefixes)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to extract routes", "error", err)
		return nil, fmt.Errorf("failed to extract routes: %w", err)
	}
	slog.InfoContext(ctx, "Extracted routes", "count", len(routes))

	// Step 3: Analyze handlers
	slog.DebugContext(ctx, "Analyzing handlers")
	analyzer := NewHandlerAnalyzer(parser.fset, parser.files)
	handlerInfos := make(map[string]*HandlerInfo)
	for _, route := range routes {
		// Find handler function
		handlerDecl := g.findHandlerDecl(route.HandlerName, parser.files, route.HandlerFile)
		if handlerDecl != nil {
			handlerInfo, analyzeErr := analyzer.AnalyzeHandler(handlerDecl)
			if analyzeErr != nil {
				slog.WarnContext(ctx, "Failed to analyze handler", "handler", route.HandlerName, "error", analyzeErr)
				continue
			}
			handlerInfos[route.HandlerName] = handlerInfo
			slog.DebugContext(ctx, "Analyzed handler", "handler", route.HandlerName, "bodyType", handlerInfo.BodyType, "responses", len(handlerInfo.Responses))
		} else {
			slog.WarnContext(ctx, "Handler function not found", "handler", route.HandlerName)
		}
	}

	// Step 4: Extract schemas
	slog.DebugContext(ctx, "Extracting schemas")
	schemaExtractor := NewSchemaExtractor(parser.files)
	schemas := make(openapi3.Schemas)

	// KNOWN LIMITATION: Schema name collision
	// We use short names (e.g., "User" instead of "models.User") for schema keys.
	// If multiple packages define types with the same name (e.g., models.User and dto.User),
	// the last one extracted will overwrite previous ones, producing incorrect $refs.
	//
	// This is a trade-off for readability - OpenAPI specs with short names are cleaner.
	// In practice, this rarely causes issues as most projects use distinct type names.
	//
	// TODO: If this becomes a real issue, implement collision detection and use
	// prefixed names (e.g., "models_User") only when conflicts are detected.
	for _, handlerInfo := range handlerInfos {
		if handlerInfo.BodyType != "" && handlerInfo.BodyType != ginHType {
			extractedSchemas, extractErr := schemaExtractor.ExtractAllSchemas(handlerInfo.BodyType)
			if extractErr == nil {
				for name, schema := range extractedSchemas {
					// Use short name (without package prefix) for schema key
					// NOTE: See KNOWN LIMITATION above about potential name collisions
					shortName := name
					if idx := strings.LastIndex(name, "."); idx != -1 {
						shortName = name[idx+1:]
					}
					schemas[shortName] = schema
				}
				slog.DebugContext(ctx, "Extracted request body schemas", "type", handlerInfo.BodyType, "count", len(extractedSchemas))
			} else {
				slog.WarnContext(ctx, "Failed to extract schemas", "type", handlerInfo.BodyType, "error", extractErr)
			}
		}
		for _, resp := range handlerInfo.Responses {
			if resp.GoType != "" && resp.GoType != ginHType {
				extractedSchemas, extractErr := schemaExtractor.ExtractAllSchemas(resp.GoType)
				if extractErr == nil {
					for name, schema := range extractedSchemas {
						// Use short name (without package prefix) for schema key
						// NOTE: See KNOWN LIMITATION above about potential name collisions
						shortName := name
						if idx := strings.LastIndex(name, "."); idx != -1 {
							shortName = name[idx+1:]
						}
						schemas[shortName] = schema
					}
					slog.DebugContext(ctx, "Extracted response schemas", "type", resp.GoType, "count", len(extractedSchemas))
				} else {
					slog.WarnContext(ctx, "Failed to extract schemas", "type", resp.GoType, "error", extractErr)
				}
			}
		}
	}
	slog.InfoContext(ctx, "Extracted schemas", "count", len(schemas))

	// Step 5: Build OpenAPI document
	slog.DebugContext(ctx, "Building OpenAPI document")
	doc := g.buildOpenAPIDoc(ginInfo, routes, handlerInfos, schemas)

	// Step 6: Write output
	outputPath, err := g.writeOutput(doc, opts)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to write output", "error", err)
		return nil, fmt.Errorf("failed to write output: %w", err)
	}

	slog.InfoContext(ctx, "Generated OpenAPI spec", "path", outputPath, "format", opts.Format)

	return &extractor.GenerateResult{
		SpecFilePath: outputPath,
		Format:       opts.Format,
	}, nil
}

// findHandlerDecl finds a handler function declaration by name.
// Supports both local handlers ("getUser") and cross-package handlers ("handlers.GetUser").
//
// For cross-package references, the package prefix is resolved by checking imports
// in the route registration file. This handles both direct package names and import
// aliases (e.g., "customapis.CreateProject" resolves correctly when import uses an alias).
// Only package-level functions (not methods) are considered, since Gin handlers
// are always standalone functions accepting *gin.Context.
func (g *Generator) findHandlerDecl(name string, files map[string]*ast.File, handlerFile string) *ast.FuncDecl {
	if name == "" {
		return nil
	}

	parts := strings.Split(name, ".")
	if len(parts) == 2 {
		pkgRef := parts[0]
		funcName := parts[1]

		// Resolve the package reference (alias or direct name) to actual package names
		pkgNames := g.resolvePkgRef(pkgRef, files, handlerFile)

		// Primary: match by resolved package names + function name (no receiver)
		for _, pkgName := range pkgNames {
			for _, file := range files {
				if file.Name.Name != pkgName {
					continue
				}
				for _, decl := range file.Decls {
					if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == funcName && fn.Recv == nil {
						return fn
					}
				}
			}
		}

		// Fallback: search all files for package-level functions
		for _, file := range files {
			for _, decl := range file.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == funcName && fn.Recv == nil {
					return fn
				}
			}
		}
	} else {
		// Local handler: just search by name
		for _, file := range files {
			for _, decl := range file.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == name {
					return fn
				}
			}
		}
	}
	return nil
}

// resolvePkgRef resolves a package reference (e.g., "apis" or "customapis") to actual
// package names by examining imports in the route registration file.
// Returns the raw ref as a fallback if no imports match.
func (g *Generator) resolvePkgRef(ref string, files map[string]*ast.File, handlerFile string) []string {
	if handlerFile == "" {
		return []string{ref}
	}

	// Find the route registration file's AST
	hf, ok := files[handlerFile]
	if !ok {
		return []string{ref}
	}

	var resolved []string
	for _, imp := range hf.Imports {
		importPath := strings.Trim(imp.Path.Value, "\"")

		if imp.Name != nil {
			// Named import: import alias "gin-crosspkg/apis"
			if imp.Name.Name == ref {
				// Found alias match — resolve to actual package name from parsed files
				if pkgName := pkgNameForImport(importPath, files); pkgName != "" {
					resolved = append(resolved, pkgName)
				}
			}
		} else {
			// Unnamed import: last path segment is the default package name
			lastSlash := strings.LastIndex(importPath, "/")
			pkgName := importPath[lastSlash+1:]
			if pkgName == ref {
				resolved = append(resolved, pkgName)
			}
		}
	}

	if len(resolved) > 0 {
		return resolved
	}
	return []string{ref}
}

// pkgNameForImport resolves an import path to its actual package name by finding
// parsed files whose directory matches the import path suffix.
func pkgNameForImport(importPath string, files map[string]*ast.File) string {
	// Extract the directory suffix from the import path
	// e.g., "gin-crosspkg/apis" → "apis"
	lastSlash := strings.LastIndex(importPath, "/")
	if lastSlash == -1 {
		return ""
	}
	dirSuffix := "/" + importPath[lastSlash+1:] + "/"

	for path, file := range files {
		// Check if the file is in a directory matching the import path suffix
		// e.g., "/home/user/project/apis/handlers.go" contains "/apis/"
		lastDirSlash := strings.LastIndex(path, "/")
		if lastDirSlash == -1 {
			continue
		}
		dir := path[:lastDirSlash]
		// Check if the directory ends with the expected suffix
		if strings.HasSuffix(dir, dirSuffix[:len(dirSuffix)-1]) {
			return file.Name.Name
		}
	}
	return ""
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

		operation := g.buildOperation(&route, handlerInfos[route.HandlerName], schemas)
		setOperationForMethod(pathItem, route.Method, operation)
	}

	return doc
}

// buildOperation builds an OpenAPI operation from a route.
func (g *Generator) buildOperation(route *Route, handlerInfo *HandlerInfo, schemas openapi3.Schemas) *openapi3.Operation {
	// Generate deterministic OperationID/Summary for anonymous handlers
	operationID := route.HandlerName
	summary := route.HandlerName
	if operationID == "" {
		// Fallback: use METHOD + path pattern (e.g., "GET_users_id")
		operationID = route.Method + "_" + strings.ReplaceAll(strings.Trim(route.FullPath, "/"), "/", "_")
		summary = route.Method + " " + route.FullPath
	}

	// Clean operationId: strip package prefix (e.g., "apis.CreateProject" → "CreateProject")
	if idx := strings.LastIndex(operationID, "."); idx != -1 {
		operationID = operationID[idx+1:]
	}

	operation := &openapi3.Operation{
		OperationID: operationID,
		Summary:     summary,
	}

	// Infer tag from route path
	if tag := inferTag(route.FullPath); tag != "" {
		operation.Tags = []string{tag}
	}

	if handlerInfo == nil {
		// Add default response for handlers we couldn't analyze
		operation.Responses = openapi3.NewResponses()
		desc := "Success"
		operation.Responses.Set("200", &openapi3.ResponseRef{
			Value: &openapi3.Response{
				Description: &desc,
			},
		})

		// Extract path parameters from the route path
		operation.Parameters = g.extractPathParamsFromRoute(route.FullPath)

		return operation
	}

	// Add parameters
	operation.Parameters = make(openapi3.Parameters, 0)

	// Path parameters: extract from the route path (source of truth)
	// rather than relying solely on c.Param() calls in the handler,
	// since handlers may not explicitly call c.Param() for every path segment.
	operation.Parameters = append(operation.Parameters, g.extractPathParamsFromRoute(route.FullPath)...)

	// Query parameters
	for _, param := range handlerInfo.QueryParams {
		operation.Parameters = append(operation.Parameters, &openapi3.ParameterRef{
			Value: &openapi3.Parameter{
				Name:        param.Name,
				In:          "query",
				Required:    param.Required,
				Description: "Query parameter",
				Schema:      &openapi3.SchemaRef{Value: GoTypeToSchema(param.GoType)},
			},
		})
	}

	// Header parameters
	for _, param := range handlerInfo.HeaderParams {
		operation.Parameters = append(operation.Parameters, &openapi3.ParameterRef{
			Value: &openapi3.Parameter{
				Name:        param.Name,
				In:          "header",
				Required:    param.Required,
				Description: "Header parameter",
				Schema:      &openapi3.SchemaRef{Value: GoTypeToSchema(param.GoType)},
			},
		})
	}

	// Handle request body / form parameters based on binding source
	g.buildRequestBody(operation, handlerInfo, schemas)

	// Responses
	operation.Responses = openapi3.NewResponses()
	for _, resp := range handlerInfo.Responses {
		desc := fmt.Sprintf("HTTP %d response", resp.StatusCode)
		response := &openapi3.Response{
			Description: &desc,
			Content:     openapi3.Content{},
		}

		if resp.GoType != "" {
			if resp.GoType == ginHType {
				response.Content["application/json"] = &openapi3.MediaType{
					Schema: &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"object"}}},
				}
			} else {
				// Use short name (without package prefix) for schema lookup and $ref
				shortName := resp.GoType
				if idx := strings.LastIndex(resp.GoType, "."); idx != -1 {
					shortName = resp.GoType[idx+1:]
				}
				if _, exists := schemas[shortName]; exists {
					response.Content["application/json"] = &openapi3.MediaType{
						Schema: &openapi3.SchemaRef{
							Ref: "#/components/schemas/" + shortName,
						},
					}
				} else {
					// Fallback to generic object if schema not found
					response.Content["application/json"] = &openapi3.MediaType{
						Schema: &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"object"}}},
					}
				}
			}
		}

		statusCode := strconv.Itoa(resp.StatusCode)
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

// buildRequestBody builds the request body or form parameters based on binding source.
func (g *Generator) buildRequestBody(operation *openapi3.Operation, handlerInfo *HandlerInfo, schemas openapi3.Schemas) {
	// Handle file upload parameters (multipart/form-data)
	if len(handlerInfo.FileParams) > 0 {
		content := make(openapi3.Content)
		schema := &openapi3.Schema{
			Type:       &openapi3.Types{"object"},
			Properties: make(openapi3.Schemas),
		}
		// Add file parameters
		for _, param := range handlerInfo.FileParams {
			schema.Properties[param.Name] = &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type:   &openapi3.Types{"string"},
					Format: "binary",
				},
			}
			if param.Required {
				schema.Required = append(schema.Required, param.Name)
			}
		}
		// Also include form parameters if present (for mixed form-data)
		for _, param := range handlerInfo.FormParams {
			// Only add if not already added as file param
			if _, exists := schema.Properties[param.Name]; !exists {
				schema.Properties[param.Name] = &openapi3.SchemaRef{
					Value: GoTypeToSchema(param.GoType),
				}
				if param.Required {
					schema.Required = append(schema.Required, param.Name)
				}
			}
		}
		content["multipart/form-data"] = &openapi3.MediaType{Schema: &openapi3.SchemaRef{Value: schema}}
		operation.RequestBody = &openapi3.RequestBodyRef{
			Value: &openapi3.RequestBody{
				Required: true,
				Content:  content,
			},
		}
		return
	}

	// Handle body binding based on source
	if handlerInfo.BodyType == "" || handlerInfo.BodyType == ginHType {
		return
	}

	// Use short name (without package prefix) for schema lookup and $ref
	shortBodyType := handlerInfo.BodyType
	if idx := strings.LastIndex(handlerInfo.BodyType, "."); idx != -1 {
		shortBodyType = handlerInfo.BodyType[idx+1:]
	}

	var schemaRef *openapi3.SchemaRef
	if _, exists := schemas[shortBodyType]; exists {
		schemaRef = &openapi3.SchemaRef{Ref: "#/components/schemas/" + shortBodyType}
	} else {
		// Fallback to generic object if schema not found
		schemaRef = &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"object"}}}
	}

	// Determine content type based on binding source
	contentType := "application/json"
	switch handlerInfo.BindingSrc {
	case BindingSourceXML:
		contentType = "application/xml"
	case BindingSourceYAML:
		contentType = "application/x-yaml"
	case BindingSourceQuery:
		// Query binding - don't create request body, parameters are in query
		return
	case BindingSourceForm:
		contentType = "application/x-www-form-urlencoded"
		// For form binding, build schema from form params to use correct field names
		if len(handlerInfo.FormParams) > 0 {
			formSchema := &openapi3.Schema{
				Type:       &openapi3.Types{"object"},
				Properties: make(openapi3.Schemas),
			}
			for _, param := range handlerInfo.FormParams {
				formSchema.Properties[param.Name] = &openapi3.SchemaRef{
					Value: GoTypeToSchema(param.GoType),
				}
				if param.Required {
					formSchema.Required = append(formSchema.Required, param.Name)
				}
			}
			schemaRef = &openapi3.SchemaRef{Value: formSchema}
		}
	}

	operation.RequestBody = &openapi3.RequestBodyRef{
		Value: &openapi3.RequestBody{
			Content: openapi3.Content{
				contentType: {
					Schema: schemaRef,
				},
			},
		},
	}
}

// setOperationForMethod sets the operation for the given HTTP method.
func setOperationForMethod(pathItem *openapi3.PathItem, method string, operation *openapi3.Operation) {
	switch method {
	case http.MethodGet:
		pathItem.Get = operation
	case http.MethodPost:
		pathItem.Post = operation
	case http.MethodPut:
		pathItem.Put = operation
	case http.MethodDelete:
		pathItem.Delete = operation
	case http.MethodPatch:
		pathItem.Patch = operation
	case http.MethodHead:
		pathItem.Head = operation
	case http.MethodOptions:
		pathItem.Options = operation
	}
}

// inferTag infers an OpenAPI tag from a route path.
// It extracts the first meaningful path segment after skipping API version prefixes.
func inferTag(path string) string {
	parts := strings.SplitSeq(strings.Trim(path, "/"), "/")
	for p := range parts {
		if p == "" || strings.HasPrefix(p, "{") {
			continue
		}
		// Skip common API prefixes
		lower := strings.ToLower(p)
		if lower == "api" || strings.HasPrefix(lower, "v") && len(lower) > 1 {
			// Check if it's a version like v1, v2, v3
			if lower == "api" || (len(lower) == 2 && lower[1] >= '0' && lower[1] <= '9') {
				continue
			}
		}
		// Capitalize first letter
		if p[0] >= 'a' && p[0] <= 'z' {
			return string(p[0]-'a'+'A') + p[1:]
		}
		return p
	}
	return ""
}

// extractPathParamsFromRoute extracts path parameters from a route path.
// For example, "/users/{id}" returns a parameter for "id".
func (g *Generator) extractPathParamsFromRoute(path string) openapi3.Parameters {
	var params openapi3.Parameters

	// Find all {param} patterns in the path
	for i := 0; i < len(path); i++ {
		if path[i] != '{' {
			continue
		}
		// Find the closing brace
		end := strings.Index(path[i:], "}")
		if end == -1 {
			continue
		}
		end += i

		// Extract parameter name
		paramName := path[i+1 : end]

		// Add the path parameter
		params = append(params, &openapi3.ParameterRef{
			Value: &openapi3.Parameter{
				Name:        paramName,
				In:          "path",
				Required:    true,
				Description: "Path parameter",
				Schema:      &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"string"}}},
			},
		})

		i = end
	}

	return params
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
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	outputPath := filepath.Join(outputDir, outputFile+ext)
	if err := os.WriteFile(outputPath, data, 0o600); err != nil {
		return "", err
	}

	return outputPath, nil
}
