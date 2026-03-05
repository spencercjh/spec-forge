package spring

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spencercjh/spec-forge/internal/executor"
	"github.com/spencercjh/spec-forge/internal/extractor"
)

// mockExecutor implements executor.Interface for testing.
type mockExecutor struct {
	executeFunc func(ctx context.Context, opts *executor.ExecuteOptions) (*executor.ExecuteResult, error)
}

func (m *mockExecutor) Execute(ctx context.Context, opts *executor.ExecuteOptions) (*executor.ExecuteResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, opts)
	}
	return &executor.ExecuteResult{ExitCode: 0}, nil
}

// Ensure mockExecutor implements Interface
var _ executor.Interface = (*mockExecutor)(nil)

func TestGenerator_ResolveMavenCommand(t *testing.T) {
	tmpDir := t.TempDir()

	// Test 1: No wrapper, should fallback to mvn
	gen := NewGenerator()
	cmd := gen.resolveMavenCommand(tmpDir)
	if cmd != "mvn" {
		t.Errorf("expected 'mvn', got '%s'", cmd)
	}

	// Test 2: Wrapper in current directory
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mvnwPath := filepath.Join(subDir, "mvnw")
	if err := os.WriteFile(mvnwPath, []byte("#!/bin/bash"), 0o755); err != nil {
		t.Fatal(err)
	}

	cmd = gen.resolveMavenCommand(subDir)
	if cmd != "./mvnw" {
		t.Errorf("expected './mvnw', got '%s'", cmd)
	}

	// Test 3: Wrapper in parent directory (multi-module scenario)
	childDir := filepath.Join(subDir, "child")
	if err := os.Mkdir(childDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Create pom.xml in parent to simulate multi-module
	pomPath := filepath.Join(subDir, "pom.xml")
	if err := os.WriteFile(pomPath, []byte("<project></project>"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd = gen.resolveMavenCommand(childDir)
	// Should find wrapper in parent
	if cmd == "mvn" {
		t.Error("expected to find wrapper in parent directory")
	}
}

func TestGenerator_ResolveGradleCommand(t *testing.T) {
	tmpDir := t.TempDir()

	// Test 1: No wrapper, should fallback to gradle
	gen := NewGenerator()
	cmd := gen.resolveGradleCommand(tmpDir)
	if cmd != "gradle" {
		t.Errorf("expected 'gradle', got '%s'", cmd)
	}

	// Test 2: Wrapper in current directory
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	gradlewPath := filepath.Join(subDir, "gradlew")
	if err := os.WriteFile(gradlewPath, []byte("#!/bin/bash"), 0o755); err != nil {
		t.Fatal(err)
	}

	cmd = gen.resolveGradleCommand(subDir)
	if cmd != "./gradlew" {
		t.Errorf("expected './gradlew', got '%s'", cmd)
	}

	// Test 3: Wrapper in parent directory (multi-module scenario)
	childDir := filepath.Join(subDir, "child")
	if err := os.Mkdir(childDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Create build.gradle in parent to simulate multi-module
	buildPath := filepath.Join(subDir, "build.gradle")
	if err := os.WriteFile(buildPath, []byte("plugins {}"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd = gen.resolveGradleCommand(childDir)
	// Should find wrapper in parent
	if cmd == "gradle" {
		t.Error("expected to find wrapper in parent directory")
	}
}

func TestGenerator_Generate_Maven(t *testing.T) {
	// Create temp directory with a mock target directory and spec file
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "target")
	if err := os.Mkdir(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a mock openapi.json
	specFile := filepath.Join(targetDir, "openapi.json")
	specContent := `{"openapi": "3.0.0", "info": {"title": "Test API", "version": "1.0.0"}}`
	if err := os.WriteFile(specFile, []byte(specContent), 0o644); err != nil {
		t.Fatal(err)
	}

	mockExec := &mockExecutor{
		executeFunc: func(ctx context.Context, opts *executor.ExecuteOptions) (*executor.ExecuteResult, error) {
			// Verify Maven command structure
			if opts.Command != "mvn" {
				t.Errorf("expected mvn command, got %s", opts.Command)
			}
			// Verify Maven uses "verify" phase per springdoc documentation
			if len(opts.Args) == 0 || opts.Args[0] != "verify" {
				t.Errorf("expected 'verify' as first arg, got %v", opts.Args)
			}
			return &executor.ExecuteResult{ExitCode: 0}, nil
		},
	}

	gen := NewGeneratorWithExecutor(mockExec)
	info := &extractor.ProjectInfo{
		BuildTool:     BuildToolMaven,
		BuildFilePath: filepath.Join(tmpDir, "pom.xml"),
	}

	result, err := gen.Generate(context.Background(), tmpDir, info, &extractor.GenerateOptions{
		Format:    "json",
		SkipTests: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Format != "json" {
		t.Errorf("expected format json, got %s", result.Format)
	}

	if result.SpecFilePath == "" {
		t.Error("expected spec file path to be set")
	}
}

func TestGenerator_Generate_Gradle(t *testing.T) {
	// Create temp directory with a mock build directory and spec file
	tmpDir := t.TempDir()
	buildDir := filepath.Join(tmpDir, "build")
	if err := os.Mkdir(buildDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a mock openapi.json
	specFile := filepath.Join(buildDir, "openapi.json")
	specContent := `{"openapi": "3.0.0", "info": {"title": "Test API", "version": "1.0.0"}}`
	if err := os.WriteFile(specFile, []byte(specContent), 0o644); err != nil {
		t.Fatal(err)
	}

	mockExec := &mockExecutor{
		executeFunc: func(ctx context.Context, opts *executor.ExecuteOptions) (*executor.ExecuteResult, error) {
			// Verify Gradle command structure
			if opts.Command != "gradle" {
				t.Errorf("expected gradle command, got %s", opts.Command)
			}
			// Verify Gradle uses "generateOpenApiDocs" task per springdoc documentation
			if len(opts.Args) == 0 || opts.Args[0] != "generateOpenApiDocs" {
				t.Errorf("expected 'generateOpenApiDocs' as first arg, got %v", opts.Args)
			}
			return &executor.ExecuteResult{ExitCode: 0}, nil
		},
	}

	gen := NewGeneratorWithExecutor(mockExec)
	info := &extractor.ProjectInfo{
		BuildTool:     BuildToolGradle,
		BuildFilePath: filepath.Join(tmpDir, "build.gradle"),
	}

	result, err := gen.Generate(context.Background(), tmpDir, info, &extractor.GenerateOptions{
		Format:    "json",
		SkipTests: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Format != "json" {
		t.Errorf("expected format json, got %s", result.Format)
	}
}

func TestGenerator_Generate_DefaultOptions(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "target")
	if err := os.Mkdir(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}

	specFile := filepath.Join(targetDir, "openapi.json")
	if err := os.WriteFile(specFile, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}

	var capturedTimeout time.Duration
	mockExec := &mockExecutor{
		executeFunc: func(ctx context.Context, opts *executor.ExecuteOptions) (*executor.ExecuteResult, error) {
			capturedTimeout = opts.Timeout
			return &executor.ExecuteResult{ExitCode: 0}, nil
		},
	}

	gen := NewGeneratorWithExecutor(mockExec)
	info := &extractor.ProjectInfo{
		BuildTool:     BuildToolMaven,
		BuildFilePath: filepath.Join(tmpDir, "pom.xml"),
	}

	// Pass nil options - should use defaults
	result, err := gen.Generate(context.Background(), tmpDir, info, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check default timeout was applied
	if capturedTimeout != 5*time.Minute {
		t.Errorf("expected default timeout of 5 minutes, got %v", capturedTimeout)
	}

	if result.Format != "json" {
		t.Errorf("expected default format json, got %s", result.Format)
	}
}

func TestGenerator_Generate_UnsupportedBuildTool(t *testing.T) {
	gen := NewGenerator()
	info := &extractor.ProjectInfo{
		BuildTool: extractor.BuildTool("unknown"),
	}

	_, err := gen.Generate(context.Background(), ".", info, nil)
	if err == nil {
		t.Fatal("expected error for unsupported build tool")
	}
}

func TestGenerator_Generate_MavenFailure(t *testing.T) {
	mockExec := &mockExecutor{
		executeFunc: func(ctx context.Context, opts *executor.ExecuteOptions) (*executor.ExecuteResult, error) {
			return &executor.ExecuteResult{
				ExitCode: 1,
				Stderr:   "Build failed",
			}, nil
		},
	}

	gen := NewGeneratorWithExecutor(mockExec)
	info := &extractor.ProjectInfo{
		BuildTool:     BuildToolMaven,
		BuildFilePath: "pom.xml",
	}

	_, err := gen.Generate(context.Background(), t.TempDir(), info, &extractor.GenerateOptions{})
	if err == nil {
		t.Fatal("expected error for failed maven build")
	}
}

func TestGenerator_findGeneratedSpec(t *testing.T) {
	tmpDir := t.TempDir()

	// Create target directory with spec file
	targetDir := filepath.Join(tmpDir, "target")
	if err := os.Mkdir(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}

	specFile := filepath.Join(targetDir, "openapi.json")
	if err := os.WriteFile(specFile, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}

	gen := NewGenerator()
	info := &extractor.ProjectInfo{BuildTool: BuildToolMaven}
	opts := &extractor.GenerateOptions{Format: "json", OutputFile: "openapi"}

	path, err := gen.findGeneratedSpec(tmpDir, info, "target", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if path == "" {
		t.Error("expected spec file path")
	}

	// Verify the path is absolute
	if !filepath.IsAbs(path) {
		t.Error("expected absolute path")
	}
}

func TestGenerator_findGeneratedSpec_Yaml(t *testing.T) {
	tmpDir := t.TempDir()

	// Create target directory with yaml spec file
	targetDir := filepath.Join(tmpDir, "target")
	if err := os.Mkdir(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}

	specFile := filepath.Join(targetDir, "openapi.yaml")
	if err := os.WriteFile(specFile, []byte(`openapi: 3.0.0`), 0o644); err != nil {
		t.Fatal(err)
	}

	gen := NewGenerator()
	info := &extractor.ProjectInfo{BuildTool: BuildToolMaven}
	opts := &extractor.GenerateOptions{Format: "yaml", OutputFile: "openapi"}

	path, err := gen.findGeneratedSpec(tmpDir, info, "target", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !filepath.IsAbs(path) {
		t.Error("expected absolute path")
	}

	if filepath.Ext(path) != ".yaml" {
		t.Errorf("expected .yaml extension, got %s", filepath.Ext(path))
	}
}

func TestGenerator_findGeneratedSpec_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	// Create target directory without spec file
	targetDir := filepath.Join(tmpDir, "target")
	if err := os.Mkdir(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}

	gen := NewGenerator()
	info := &extractor.ProjectInfo{BuildTool: BuildToolMaven}
	opts := &extractor.GenerateOptions{Format: "json", OutputFile: "openapi"}

	_, err := gen.findGeneratedSpec(tmpDir, info, "target", opts)
	if err == nil {
		t.Fatal("expected error when spec file not found")
	}
}
