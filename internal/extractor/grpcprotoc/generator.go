// Package grpcprotoc provides gRPC-protoc framework extraction functionality.
package grpcprotoc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
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

// ErrNoServiceProtoFiles indicates no proto files with service definitions were found.
var ErrNoServiceProtoFiles = errors.New("no proto files with service definitions found in project")

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

	// Check if there are service proto files
	if len(info.ServiceProtoFiles) == 0 {
		slog.Error("no service proto files found in project", "path", absPath)
		return nil, ErrNoServiceProtoFiles
	}
	slog.Debug("service proto files found", "count", len(info.ServiceProtoFiles), "files", info.ServiceProtoFiles)

	// Determine output directory
	outputDir := opts.OutputDir
	if outputDir == "" {
		outputDir = absPath
	}

	// Convert outputDir to absolute path if it's not already
	if !filepath.IsAbs(outputDir) {
		var absErr error
		outputDir, absErr = filepath.Abs(filepath.Join(absPath, outputDir))
		if absErr != nil {
			slog.Error("failed to resolve output directory", "dir", outputDir, "error", absErr)
			return nil, fmt.Errorf("failed to resolve output directory: %w", absErr)
		}
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

	// Add import paths detected from project (convert to relative paths if needed)
	for _, path := range info.ImportPaths {
		relPath := g.toRelativePath(path, info.ProtoRoot)
		if !seenPaths[relPath] {
			seenPaths[relPath] = true
			args = append(args, "-I"+relPath)
		}
	}

	// Add extra import paths from CLI flags (--proto-import-path)
	for _, path := range opts.ProtoImportPaths {
		if !seenPaths[path] {
			seenPaths[path] = true
			args = append(args, "-I"+path)
		}
	}

	// Add connect-openapi output
	args = append(args, "--connect-openapi_out="+outputDir)

	// Add format option — protoc-gen-connect-openapi defaults to YAML,
	// so we must pass format=json explicitly for JSON output.
	switch opts.Format {
	case "json":
		args = append(args, "--connect-openapi_opt=format=json")
	case "yaml", "yml":
		args = append(args, "--connect-openapi_opt=format=yaml")
	}

	// Add output name option if specified (controls the base filename)
	if opts.OutputFile != "" && opts.OutputFile != defaultOutputFileName {
		args = append(args, "--connect-openapi_opt=output_name="+opts.OutputFile)
	}

	// Enable google.api.http annotations support if detected
	if info.HasGoogleAPI {
		args = append(args, "--connect-openapi_opt=features=google.api.http")
	}

	// Add only service proto files (those with service definitions)
	// to avoid duplicate definition errors from importing common proto files
	for _, protoFile := range info.ServiceProtoFiles {
		relPath := g.toRelativePath(protoFile, info.ProtoRoot)
		args = append(args, relPath)
	}

	return args
}

// toRelativePath converts an absolute path to a relative path from the base directory.
// If the path is not under the base directory, it returns the original path.
func (g *Generator) toRelativePath(path, base string) string {
	if filepath.IsAbs(path) && filepath.IsAbs(base) {
		rel, err := filepath.Rel(base, path)
		if err == nil {
			return rel
		}
	}
	return path
}

// findOutputFile locates the generated OpenAPI file.
// protoc-gen-connect-openapi generates files mirroring the proto source directory structure
// inside the output directory (e.g., outputDir/proto/user.openapi.json).
func (g *Generator) findOutputFile(info *Info, outputDir, format string) (string, error) {
	// Determine expected extension
	expectedExt := ".openapi.json"
	if format == "yaml" || format == "yml" {
		expectedExt = ".openapi.yaml"
	}

	// Optimization: when there's exactly one service proto file, compute the expected output path directly
	if len(info.ServiceProtoFiles) == 1 {
		serviceFile := info.ServiceProtoFiles[0]
		relDir := g.toRelativePath(filepath.Dir(serviceFile), info.ProtoRoot)
		expectedPath := filepath.Join(outputDir, relDir, strings.TrimSuffix(filepath.Base(serviceFile), ".proto")+expectedExt)
		if _, err := os.Stat(expectedPath); err == nil {
			slog.Debug("found expected output file for single service proto", "file", expectedPath)
			return expectedPath, nil
		}
	}

	// Collect all candidate files from the output directory tree
	var candidates []string
	walkErr := filepath.Walk(outputDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !fi.IsDir() && strings.HasSuffix(fi.Name(), expectedExt) {
			candidates = append(candidates, path)
		}
		return nil
	})
	if walkErr != nil {
		slog.Debug("error walking output directory", "dir", outputDir, "error", walkErr)
	}

	// If no candidates found, return error
	if len(candidates) == 0 {
		return "", ErrOutputFileNotFound
	}

	// If exactly one candidate, return it
	if len(candidates) == 1 {
		return candidates[0], nil
	}

	// Multiple candidates found - return an explicit error
	slices.Sort(candidates)
	return "", fmt.Errorf("multiple OpenAPI output files found (%d): %v. "+
		"This indicates multiple service proto files generated separate specs. "+
		"To resolve: either (1) run spec-forge separately per service proto directory, or "+
		"(2) configure protoc-gen-connect-openapi to generate a single combined spec. "+
		"See: https://github.com/sudorandom/protoc-gen-connect-openapi for configuration options",
		len(candidates), candidates)
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
