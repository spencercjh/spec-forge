// Package grpcprotoc provides gRPC-protoc framework extraction functionality.
package grpcprotoc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spencercjh/spec-forge/internal/executor"
	"github.com/spencercjh/spec-forge/internal/extractor"
)

const (
	protocCommand         = "protoc"
	defaultOutputFileName = "openapi"
	defaultFormat         = "json"
)

// ErrNoProtoFiles indicates no proto files were found in the project.
var ErrNoProtoFiles = errors.New("no proto files found in project")

// ErrOutputFileNotFound indicates the generated OpenAPI file was not found after protoc execution.
var ErrOutputFileNotFound = errors.New("output file not found after protoc execution")

// Generator generates OpenAPI specs from gRPC-protoc projects.
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

// Generate generates OpenAPI spec by invoking protoc with protoc-gen-connect-openapi plugin.
func (g *Generator) Generate(ctx context.Context, projectPath string, projectInfo *extractor.ProjectInfo, opts *extractor.GenerateOptions) (*extractor.GenerateResult, error) {
	slog.Info("starting OpenAPI spec generation for gRPC-protoc project", "project", projectPath)

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

	// Get gRPC-protoc info from project info
	info, ok := projectInfo.FrameworkData.(*Info)
	if !ok || info == nil {
		slog.Debug("framework data not set, detecting project")
		// If FrameworkData is not set, detect it
		detectedInfo, detectErr := g.detector.Detect(absPath)
		if detectErr != nil {
			slog.Error("failed to detect gRPC-protoc project", "path", absPath, "error", detectErr)
			return nil, fmt.Errorf("failed to detect gRPC-protoc project: %w", detectErr)
		}
		var typeOk bool
		info, typeOk = detectedInfo.FrameworkData.(*Info)
		if !typeOk {
			slog.Error("failed to get gRPC-protoc info from detected project")
			return nil, errors.New("failed to get gRPC-protoc info from detected project")
		}
	}

	// Check if there are any proto files
	if len(info.ProtoFiles) == 0 {
		slog.Error("no proto files found in project", "path", absPath)
		return nil, ErrNoProtoFiles
	}
	slog.Debug("proto files found", "count", len(info.ProtoFiles), "files", info.ProtoFiles)

	// Determine output directory
	outputDir := opts.OutputDir
	if outputDir == "" {
		outputDir = absPath
	}

	// Ensure output directory exists
	if mkdirErr := os.MkdirAll(outputDir, 0o755); mkdirErr != nil {
		slog.Error("failed to create output directory", "dir", outputDir, "error", mkdirErr)
		return nil, fmt.Errorf("failed to create output directory %s: %w", outputDir, mkdirErr)
	}

	// Build protoc command arguments
	args := g.buildProtocArgs(info, outputDir, opts)

	slog.Debug("executing protoc command",
		"command", protocCommand,
		"args", args,
		"workingDir", absPath,
		"timeout", opts.Timeout)

	result, execErr := g.executor.Execute(ctx, &executor.ExecuteOptions{
		Command:    protocCommand,
		Args:       args,
		WorkingDir: absPath,
		Timeout:    opts.Timeout,
	})
	if execErr != nil {
		slog.Error("protoc command failed", "error", execErr, "command", protocCommand, "args", args)
		return nil, fmt.Errorf("protoc command failed: %w", execErr)
	}

	if result.ExitCode != 0 {
		out := combineOutput(result.Stdout, result.Stderr)
		slog.Error("protoc command failed", "exitCode", result.ExitCode, "output", out)
		return nil, fmt.Errorf("protoc command failed with exit code %d:\n%s", result.ExitCode, out)
	}

	slog.Debug("protoc command succeeded")

	// Find the generated output file
	outputPath, findErr := g.findOutputFile(info, outputDir, opts.Format)
	if findErr != nil {
		slog.Error("failed to find generated OpenAPI file", "outputDir", outputDir, "error", findErr)
		return nil, fmt.Errorf("%w: %s", ErrOutputFileNotFound, outputDir)
	}

	generateResult := &extractor.GenerateResult{
		SpecFilePath: outputPath,
		Format:       opts.Format,
	}
	slog.Info("OpenAPI spec generation completed", "output", outputPath, "format", opts.Format)

	return generateResult, nil
}

// buildProtocArgs constructs the protoc command arguments.
func (g *Generator) buildProtocArgs(info *Info, outputDir string, opts *extractor.GenerateOptions) []string {
	var args []string

	// Add import paths (-I flags)
	seenPaths := make(map[string]bool)

	// Add import paths detected from project
	for _, path := range info.ImportPaths {
		if !seenPaths[path] {
			seenPaths[path] = true
			args = append(args, "-I"+path)
		}
	}

	// Add extra import paths from CLI flags (--proto-import-path)
	for _, path := range opts.ProtoImportPaths {
		if !seenPaths[path] {
			seenPaths[path] = true
			args = append(args, "-I"+path)
		}
	}

	// Add connect-openapi output and options
	args = append(args,
		"--connect-openapi_out="+outputDir,
		"--connect-openapi_opt=features=google.api.http",
	)

	// Add format option for YAML
	if opts.Format == "yaml" || opts.Format == "yml" {
		args = append(args, "--connect-openapi_opt=format=yaml")
	}

	// Add proto files
	args = append(args, info.ProtoFiles...)

	return args
}

// findOutputFile locates the generated OpenAPI file.
func (g *Generator) findOutputFile(_ *Info, outputDir, format string) (string, error) {
	// protoc-gen-connect-openapi generates files like: <proto_filename>.openapi.json or .yaml
	entries, readErr := os.ReadDir(outputDir)
	if readErr != nil {
		return "", fmt.Errorf("failed to read output directory: %w", readErr)
	}

	// Determine expected extension
	expectedExt := ".openapi.json"
	if format == "yaml" || format == "yml" {
		expectedExt = ".openapi.yaml"
	}

	// Look for the generated file
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasSuffix(name, expectedExt) {
			return filepath.Join(outputDir, name), nil
		}
	}

	// Also check for any .openapi.json or .openapi.yaml files
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasSuffix(name, ".openapi.json") || strings.HasSuffix(name, ".openapi.yaml") {
			return filepath.Join(outputDir, name), nil
		}
	}

	return "", ErrOutputFileNotFound
}

// combineOutput combines stdout and stderr for error messages.
func combineOutput(stdout, stderr string) string {
	stdout = strings.TrimSpace(stdout)
	stderr = strings.TrimSpace(stderr)
	if stdout != "" && stderr != "" {
		return stdout + "\n" + stderr
	}
	if stdout != "" {
		return stdout
	}
	return stderr
}
