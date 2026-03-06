// Package gozero provides go-zero framework specific extraction functionality.
package gozero

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi2conv"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/spencercjh/spec-forge/internal/executor"
	"github.com/spencercjh/spec-forge/internal/extractor"
	"gopkg.in/yaml.v3"
)

const (
	defaultOutputFileName = "openapi"
	defaultFormat         = "json"
	goctlCmd              = "goctl"
)

// Generator generates OpenAPI specs from go-zero projects.
type Generator struct {
	detector *Detector
	executor executor.Interface
}

// NewGenerator creates a new Generator instance.
func NewGenerator() *Generator {
	return &Generator{
		detector: NewDetector(),
		executor: executor.NewExecutor(),
	}
}

// NewGeneratorWithExecutor creates a new Generator with a custom executor (for testing).
func NewGeneratorWithExecutor(exec executor.Interface) *Generator {
	return &Generator{
		detector: NewDetector(),
		executor: exec,
	}
}

// Generate generates OpenAPI spec by invoking goctl command and converting Swagger 2.0 to OpenAPI 3.0.
func (g *Generator) Generate(ctx context.Context, projectPath string, opts *extractor.GenerateOptions) (*extractor.GenerateResult, error) {
	if opts == nil {
		opts = &extractor.GenerateOptions{}
	}

	// Apply defaults
	if opts.Timeout <= 0 {
		opts.Timeout = 5 * time.Minute
	}
	if opts.Format == "" {
		opts.Format = defaultFormat
	}
	if opts.OutputFile == "" {
		opts.OutputFile = defaultOutputFileName
	}

	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve project path: %w", err)
	}

	// Detect project info
	info, err := g.detector.Detect(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to detect go-zero project: %w", err)
	}

	// Check if goctl is available
	if !info.HasGoctl {
		return nil, fmt.Errorf("goctl command not found in PATH. Please install goctl: go install github.com/zeromicro/go-zero/tools/goctl@latest")
	}

	// Generate Swagger 2.0 spec using goctl
	swaggerPath, err := g.generateSwagger(ctx, absPath, info, opts)
	if err != nil {
		return nil, err
	}

	// Convert Swagger 2.0 to OpenAPI 3.0
	result, err := g.convertSwaggerToOpenAPI(swaggerPath, opts)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// generateSwagger generates Swagger 2.0 spec using goctl command.
func (g *Generator) generateSwagger(ctx context.Context, workDir string, info *ProjectInfo, opts *extractor.GenerateOptions) (string, error) {
	// Build goctl command arguments
	// goctl api plugin -plugin goctl-swagger="swagger -filename openapi.json" -api <api_file> -dir <output_dir>
	// For simplicity, we use: goctl api swagger -filename openapi.json -api <api_file>
	args := []string{
		"api",
		"swagger",
	}

	// Use a temporary filename for the swagger file to avoid conflicts with the output file
	// goctl always generates .json files (Swagger 2.0), regardless of the requested format
	swaggerFilename := opts.OutputFile + ".swagger.json"
	args = append(args, "-filename", swaggerFilename)

	// Find the main API file (usually in the api/ directory or the first one found)
	apiFile := g.findMainAPIFile(workDir, info)
	if apiFile == "" {
		return "", fmt.Errorf("no .api files found in project")
	}
	args = append(args, "-api", apiFile)

	// Execute goctl command
	result, err := g.executor.Execute(ctx, &executor.ExecuteOptions{
		Command:    goctlCmd,
		Args:       args,
		WorkingDir: workDir,
		Timeout:    opts.Timeout,
	})
	if err != nil {
		return "", fmt.Errorf("goctl swagger generation failed: %w", err)
	}

	if result.ExitCode != 0 {
		out := combineOutput(result.Stdout, result.Stderr)
		if out == "" {
			return "", fmt.Errorf("goctl swagger generation failed with exit code %d (no output)", result.ExitCode)
		}
		return "", fmt.Errorf("goctl swagger generation failed with exit code %d:\n%s", result.ExitCode, out)
	}

	// Return the path to the generated swagger file
	swaggerPath := filepath.Join(workDir, swaggerFilename)

	return swaggerPath, nil
}

// findMainAPIFile finds the main API file to use for swagger generation.
func (g *Generator) findMainAPIFile(workDir string, info *ProjectInfo) string {
	if len(info.APIFiles) == 0 {
		return ""
	}

	// Prefer API files in the api/ directory
	for _, apiFile := range info.APIFiles {
		relPath, _ := filepath.Rel(workDir, apiFile)
		if strings.HasPrefix(relPath, "api") || strings.HasPrefix(relPath, "api/") {
			return apiFile
		}
	}

	// Fallback to the first API file found
	return info.APIFiles[0]
}

// convertSwaggerToOpenAPI converts Swagger 2.0 spec to OpenAPI 3.0 spec.
func (g *Generator) convertSwaggerToOpenAPI(swaggerPath string, opts *extractor.GenerateOptions) (*extractor.GenerateResult, error) {
	// Load Swagger 2.0 document
	data, err := os.ReadFile(swaggerPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Swagger 2.0 spec from %s: %w", swaggerPath, err)
	}

	swagger2Doc := &openapi2.T{}
	if err := swagger2Doc.UnmarshalJSON(data); err != nil {
		return nil, fmt.Errorf("failed to parse Swagger 2.0 spec from %s: %w", swaggerPath, err)
	}

	// Apply patches for known goctl swagger bugs (#5426-5428)
	PatchSwagger(swagger2Doc)

	// Convert to OpenAPI 3.0
	openAPIDoc, err := openapi2conv.ToV3(swagger2Doc)
	if err != nil {
		return nil, fmt.Errorf("failed to convert Swagger 2.0 to OpenAPI 3.0: %w", err)
	}

	// Determine output path
	workDir := filepath.Dir(swaggerPath)
	outputFile := opts.OutputFile
	if outputFile == "" {
		outputFile = defaultOutputFileName
	}

	var outputPath string
	var outputData []byte

	// Marshal based on format
	if opts.Format == "yaml" {
		outputPath = filepath.Join(workDir, outputFile+".yaml")
		outputData, err = marshalYAML(openAPIDoc)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal OpenAPI 3.0 to YAML: %w", err)
		}
	} else {
		outputPath = filepath.Join(workDir, outputFile+".json")
		outputData, err = openAPIDoc.MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal OpenAPI 3.0 to JSON: %w", err)
		}
	}

	// Write the converted spec to file
	if err := os.WriteFile(outputPath, outputData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write OpenAPI 3.0 spec to %s: %w", outputPath, err)
	}

	// Clean up the temporary Swagger 2.0 file (only if it's different from output)
	if swaggerPath != outputPath {
		_ = os.Remove(swaggerPath)
	}

	return &extractor.GenerateResult{
		SpecFilePath: outputPath,
		Format:       opts.Format,
	}, nil
}

// marshalYAML marshals an OpenAPI 3.0 document to YAML format.
func marshalYAML(doc *openapi3.T) ([]byte, error) {
	// Convert to map for YAML marshaling
	data, err := doc.MarshalJSON()
	if err != nil {
		return nil, err
	}

	// Parse JSON into interface{}
	var jsonData interface{}
	if err := yaml.Unmarshal(data, &jsonData); err != nil {
		return nil, err
	}

	// Marshal to YAML
	return yaml.Marshal(jsonData)
}

// combineOutput combines stdout and stderr for error messages.
func combineOutput(stdout, stderr string) string {
	stdout = strings.TrimSpace(stdout)
	stderr = strings.TrimSpace(stderr)

	if stdout == "" {
		return stderr
	}
	if stderr == "" {
		return stdout
	}
	return stdout + "\n" + stderr
}
