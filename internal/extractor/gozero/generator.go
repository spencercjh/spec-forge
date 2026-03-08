// Package gozero provides go-zero framework specific extraction functionality.
package gozero

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi2conv"
	"github.com/getkin/kin-openapi/openapi3"
	"gopkg.in/yaml.v3"

	"github.com/spencercjh/spec-forge/internal/executor"
	"github.com/spencercjh/spec-forge/internal/extractor"
)

const (
	defaultOutputFileName = "openapi"
	defaultFormat         = "json"
	goctlCmd              = "goctl"
)

// ExtractorName is the name of this extractor.
const ExtractorName = "gozero"

// Extractor implements the extractor.Extractor interface for go-zero projects.
type Extractor struct {
	detector  *Detector
	patcher   *Patcher
	generator *Generator
}

func NewExtractor() *Extractor {
	return &Extractor{
		detector:  NewDetector(),
		patcher:   NewPatcher(),
		generator: NewGenerator(),
	}
}

// Name returns the extractor name.
func (e *Extractor) Name() string {
	return ExtractorName
}

// Detect analyzes a project and returns its information if it's a go-zero project.
func (e *Extractor) Detect(projectPath string) (*extractor.ProjectInfo, error) {
	return e.detector.Detect(projectPath)
}

// Patch checks if goctl is available for the go-zero project.
func (e *Extractor) Patch(_ string, _ *extractor.PatchOptions) (*extractor.PatchResult, error) {
	_, err := e.patcher.Patch("")
	if err != nil {
		return nil, err
	}
	// go-zero doesn't modify project files, so return empty result.
	return &extractor.PatchResult{}, nil
}

// Generate produces the OpenAPI spec from the go-zero project.
func (e *Extractor) Generate(ctx context.Context, projectPath string, info *extractor.ProjectInfo, opts *extractor.GenerateOptions) (*extractor.GenerateResult, error) {
	return e.generator.Generate(ctx, projectPath, info, opts)
}

// Restore is a no-op for go-zero projects as we don't modify files.
func (e *Extractor) Restore(_, _ string) error {
	return nil
}

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
func (g *Generator) Generate(ctx context.Context, projectPath string, projectInfo *extractor.ProjectInfo, opts *extractor.GenerateOptions) (*extractor.GenerateResult, error) {
	slog.Info("starting OpenAPI spec generation", "project", projectPath)

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
	slog.Debug("generation options",
		"timeout", opts.Timeout,
		"format", opts.Format,
		"outputFile", opts.OutputFile,
		"outputDir", opts.OutputDir)

	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		slog.Error("failed to resolve project path", "path", projectPath, "error", err)
		return nil, fmt.Errorf("failed to resolve project path: %w", err)
	}

	// Get go-zero info from project info
	info, ok := projectInfo.FrameworkData.(*Info)
	if !ok || info == nil {
		slog.Debug("framework data not set, detecting project")
		// If FrameworkData is not set, detect it
		detectedInfo, detectErr := g.detector.Detect(absPath)
		if detectErr != nil {
			slog.Error("failed to detect go-zero project", "path", absPath, "error", detectErr)
			return nil, fmt.Errorf("failed to detect go-zero project: %w", detectErr)
		}
		var typeOk bool
		info, typeOk = detectedInfo.FrameworkData.(*Info)
		if !typeOk {
			slog.Error("failed to get go-zero info from detected project")
			return nil, errors.New("failed to get go-zero info from detected project")
		}
	}

	// Check if goctl is available
	if !info.HasGoctl {
		slog.Error("goctl not found in PATH")
		return nil, errors.New("goctl command not found in PATH. Please install goctl: go install github.com/zeromicro/go-zero/tools/goctl@latest")
	}

	// Generate Swagger 2.0 spec using goctl
	slog.Info("generating Swagger 2.0 spec using goctl", "apiFiles", len(info.APIFiles))
	swaggerPath, err := g.generateSwagger(ctx, absPath, info, opts)
	if err != nil {
		return nil, err
	}
	slog.Debug("generated Swagger 2.0 spec", "path", swaggerPath)

	// Convert Swagger 2.0 to OpenAPI 3.0
	slog.Info("converting Swagger 2.0 to OpenAPI 3.0")
	result, err := g.convertSwaggerToOpenAPI(swaggerPath, opts)
	if err != nil {
		return nil, err
	}

	slog.Info("OpenAPI spec generation completed", "output", result.SpecFilePath, "format", result.Format)
	return result, nil
}

// generateSwagger generates Swagger 2.0 spec using goctl command.
func (g *Generator) generateSwagger(ctx context.Context, workDir string, info *Info, opts *extractor.GenerateOptions) (string, error) {
	slog.Debug("patching .api files to work around goctl bugs")
	// Patch .api files to work around goctl parser bugs (#5425)
	apiPatcher := NewAPIFilePatcher()
	defer apiPatcher.Cleanup()

	patchedFiles, err := apiPatcher.PatchAPIFiles(info.APIFiles)
	if err != nil {
		slog.Error("failed to patch API files", "error", err)
		return "", fmt.Errorf("failed to patch API files: %w", err)
	}
	// Only log if files were actually modified (not just checked)
	if apiPatcher.HasPatchedFiles() {
		slog.Info("patched .api files for goctl compatibility", "count", len(patchedFiles))
	}

	// Build goctl command arguments
	// goctl api swagger -filename openapi.json -api <api_file> -dir <output_dir>
	args := []string{
		"api",
		"swagger",
	}

	// Use a temporary filename for the swagger file to avoid conflicts with the output file
	// goctl always generates .json files (Swagger 2.0), regardless of the requested format
	// Note: goctl automatically appends .json to the filename, so we use .swagger without extension
	swaggerFilename := opts.OutputFile + ".swagger"
	args = append(args, "-filename", swaggerFilename)

	// Find the main API file (usually in the api/ directory or the first one found)
	apiFile := g.findMainAPIFile(workDir, info, patchedFiles)
	if apiFile == "" {
		slog.Error("no .api files found in project")
		return "", errors.New("no .api files found in project")
	}
	slog.Debug("selected main API file", "file", apiFile)

	// Add api file and output directory (required by goctl)
	args = append(args, "-api", apiFile, "-dir", workDir)

	// Execute goctl command
	slog.Debug("executing goctl command", "command", goctlCmd, "args", args, "workDir", workDir)
	result, err := g.executor.Execute(ctx, &executor.ExecuteOptions{
		Command:    goctlCmd,
		Args:       args,
		WorkingDir: workDir,
		Timeout:    opts.Timeout,
	})
	if err != nil {
		slog.Error("goctl swagger generation failed", "error", err)
		return "", fmt.Errorf("goctl swagger generation failed: %w", err)
	}

	if result.ExitCode != 0 {
		out := combineOutput(result.Stdout, result.Stderr)
		slog.Error("goctl swagger generation failed", "exitCode", result.ExitCode, "output", out)
		if out == "" {
			return "", fmt.Errorf("goctl swagger generation failed with exit code %d (no output)", result.ExitCode)
		}
		return "", fmt.Errorf("goctl swagger generation failed with exit code %d:\n%s", result.ExitCode, out)
	}

	slog.Debug("goctl swagger generation succeeded")

	// Return the path to the generated swagger file
	// goctl automatically appends .json to the filename
	swaggerPath := filepath.Join(workDir, swaggerFilename+".json")

	return swaggerPath, nil
}

// findMainAPIFile finds the main API file to use for swagger generation.
// Returns the patched file path if available, otherwise returns the original.
func (g *Generator) findMainAPIFile(workDir string, info *Info, patchedFiles map[string]string) string {
	if len(info.APIFiles) == 0 {
		return ""
	}

	var selectedFile string

	// Prefer API files in the api/ directory
	for _, apiFile := range info.APIFiles {
		relPath, err := filepath.Rel(workDir, apiFile)
		if err != nil {
			continue
		}
		// Use filepath.ToSlash for cross-platform consistency, then check for "api/" prefix
		normalizedPath := filepath.ToSlash(relPath)
		if strings.HasPrefix(normalizedPath, "api/") {
			selectedFile = apiFile
			break
		}
	}

	// Fallback to the first API file found
	if selectedFile == "" {
		selectedFile = info.APIFiles[0]
	}

	// Return patched path if available
	if patchedPath, ok := patchedFiles[selectedFile]; ok {
		return patchedPath
	}
	return selectedFile
}

// convertSwaggerToOpenAPI converts Swagger 2.0 spec to OpenAPI 3.0 spec.
func (g *Generator) convertSwaggerToOpenAPI(swaggerPath string, opts *extractor.GenerateOptions) (*extractor.GenerateResult, error) {
	slog.Debug("reading Swagger 2.0 spec", "path", swaggerPath)

	// Load Swagger 2.0 document
	data, err := os.ReadFile(swaggerPath)
	if err != nil {
		slog.Error("failed to read Swagger 2.0 spec", "path", swaggerPath, "error", err)
		return nil, fmt.Errorf("failed to read Swagger 2.0 spec from %s: %w", swaggerPath, err)
	}

	swagger2Doc := &openapi2.T{}
	if unmarshalErr := swagger2Doc.UnmarshalJSON(data); unmarshalErr != nil {
		slog.Error("failed to parse Swagger 2.0 spec", "path", swaggerPath, "error", unmarshalErr)
		return nil, fmt.Errorf("failed to parse Swagger 2.0 spec from %s: %w", swaggerPath, unmarshalErr)
	}
	slog.Debug("parsed Swagger 2.0 spec", "title", swagger2Doc.Info.Title, "version", swagger2Doc.Info.Version)

	// Apply patches for known goctl swagger bugs (#5426-5428)
	slog.Debug("applying patches for known goctl swagger bugs")
	PatchSwagger(swagger2Doc)

	// Convert to OpenAPI 3.0
	slog.Debug("converting to OpenAPI 3.0")
	openAPIDoc, err := openapi2conv.ToV3(swagger2Doc)
	if err != nil {
		slog.Error("failed to convert Swagger 2.0 to OpenAPI 3.0", "error", err)
		return nil, fmt.Errorf("failed to convert Swagger 2.0 to OpenAPI 3.0: %w", err)
	}
	slog.Debug("converted to OpenAPI 3.0", "title", openAPIDoc.Info.Title, "version", openAPIDoc.Info.Version)

	// Determine output directory (respect opts.OutputDir if provided)
	var outputDir string
	if opts.OutputDir != "" {
		outputDir = opts.OutputDir
		slog.Debug("using specified output directory", "dir", outputDir)
		// Ensure output directory exists
		if mkdirErr := os.MkdirAll(outputDir, 0o755); mkdirErr != nil {
			slog.Error("failed to create output directory", "dir", outputDir, "error", mkdirErr)
			return nil, fmt.Errorf("failed to create output directory %s: %w", outputDir, mkdirErr)
		}
	} else {
		outputDir = filepath.Dir(swaggerPath)
		slog.Debug("using default output directory", "dir", outputDir)
	}

	outputFile := opts.OutputFile
	if outputFile == "" {
		outputFile = defaultOutputFileName
	}

	var outputPath string
	var outputData []byte

	// Marshal based on format
	if opts.Format == "yaml" {
		outputPath = filepath.Join(outputDir, outputFile+".yaml")
		outputData, err = marshalYAML(openAPIDoc)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal OpenAPI 3.0 to YAML: %w", err)
		}
	} else {
		outputPath = filepath.Join(outputDir, outputFile+".json")
		outputData, err = openAPIDoc.MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal OpenAPI 3.0 to JSON: %w", err)
		}
	}

	// Write the converted spec to file
	slog.Debug("writing OpenAPI 3.0 spec", "path", outputPath, "format", opts.Format)
	if err := os.WriteFile(outputPath, outputData, 0o600); err != nil {
		slog.Error("failed to write OpenAPI 3.0 spec", "path", outputPath, "error", err)
		return nil, fmt.Errorf("failed to write OpenAPI 3.0 spec to %s: %w", outputPath, err)
	}
	slog.Info("written OpenAPI 3.0 spec", "path", outputPath, "size", len(outputData))

	// Clean up the temporary Swagger 2.0 file (only if it's different from output)
	if swaggerPath != outputPath {
		slog.Debug("cleaning up temporary swagger file", "path", swaggerPath)
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

	// Parse JSON into any
	var jsonData any
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
