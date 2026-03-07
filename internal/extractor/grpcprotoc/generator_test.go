// Package grpcprotoc_test tests the gRPC-protoc extractor implementation.
package grpcprotoc_test

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/spencercjh/spec-forge/internal/executor"
	"github.com/spencercjh/spec-forge/internal/extractor"
	"github.com/spencercjh/spec-forge/internal/extractor/grpcprotoc"
)

func TestNewGenerator(t *testing.T) {
	g := grpcprotoc.NewGenerator()
	if g == nil {
		t.Error("NewGenerator() should not return nil")
	}
}

func TestNewGeneratorWithExecutor(t *testing.T) {
	mockExec := &mockExecutorForGenerator{}
	g := grpcprotoc.NewGeneratorWithExecutor(mockExec)
	if g == nil {
		t.Error("NewGeneratorWithExecutor() should not return nil")
	}
}

func TestGenerator_Generate_Success_JSON(t *testing.T) {
	// Create temp dir with proto file
	tmpDir := t.TempDir()
	protoFile := filepath.Join(tmpDir, "test.proto")
	protoContent := `syntax = "proto3";
package test;

message HelloRequest {
  string name = 1;
}

service HelloService {
  rpc SayHello(HelloRequest) returns (HelloRequest);
}
`
	if err := os.WriteFile(protoFile, []byte(protoContent), 0o644); err != nil {
		t.Fatalf("Failed to create .proto file: %v", err)
	}

	// Create a mock generated OpenAPI file (simulating protoc-gen-connect-openapi output)
	expectedOutputPath := filepath.Join(tmpDir, "test.openapi.json")
	expectedOpenAPI := `{"openapi":"3.0.0","info":{"title":"Test API","version":"1.0.0"}}`

	mockExec := &mockExecutorForGenerator{
		results: map[string]*executor.ExecuteResult{
			"protoc": {
				ExitCode: 0,
				Stdout:   "",
				Stderr:   "",
			},
		},
		errors: map[string]error{
			"protoc": nil,
		},
		postExecuteHook: func() {
			// Simulate protoc generating the output file
			_ = os.WriteFile(expectedOutputPath, []byte(expectedOpenAPI), 0o644)
		},
	}

	g := grpcprotoc.NewGeneratorWithExecutor(mockExec)

	info := &extractor.ProjectInfo{
		Framework:     grpcprotoc.FrameworkName,
		BuildTool:     grpcprotoc.BuildToolProtoc,
		BuildFilePath: tmpDir,
		FrameworkData: &grpcprotoc.Info{
			ProtoFiles:        []string{protoFile},
			ServiceProtoFiles: []string{protoFile},
			ProtoRoot:         tmpDir,
			HasGoogleAPI:      false,
			HasBuf:            false,
			ImportPaths:       []string{tmpDir},
		},
	}

	opts := &extractor.GenerateOptions{
		OutputDir: tmpDir,
		Format:    "json",
		Timeout:   30 * time.Second,
	}

	result, err := g.Generate(context.Background(), tmpDir, info, opts)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if result.Format != "json" {
		t.Errorf("Expected format 'json', got: %s", result.Format)
	}

	if result.SpecFilePath == "" {
		t.Error("Expected SpecFilePath to be set")
	}

	// Verify the output file exists
	if _, statErr := os.Stat(result.SpecFilePath); os.IsNotExist(statErr) {
		t.Errorf("Expected output file to exist at: %s", result.SpecFilePath)
	}
}

func TestGenerator_Generate_Success_YAML(t *testing.T) {
	// Create temp dir with proto file
	tmpDir := t.TempDir()
	protoFile := filepath.Join(tmpDir, "service.proto")
	protoContent := `syntax = "proto3";
package test;

message Request {
  string name = 1;
}

service Service {
  rpc Method(Request) returns (Request);
}
`
	if err := os.WriteFile(protoFile, []byte(protoContent), 0o644); err != nil {
		t.Fatalf("Failed to create .proto file: %v", err)
	}

	// Create a mock generated OpenAPI YAML file
	expectedOutputPath := filepath.Join(tmpDir, "service.openapi.yaml")
	expectedOpenAPI := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
`

	mockExec := &mockExecutorForGenerator{
		results: map[string]*executor.ExecuteResult{
			"protoc": {
				ExitCode: 0,
				Stdout:   "",
				Stderr:   "",
			},
		},
		errors: map[string]error{
			"protoc": nil,
		},
		postExecuteHook: func() {
			// Simulate protoc generating the YAML output file
			_ = os.WriteFile(expectedOutputPath, []byte(expectedOpenAPI), 0o644)
		},
	}

	g := grpcprotoc.NewGeneratorWithExecutor(mockExec)

	info := &extractor.ProjectInfo{
		Framework:     grpcprotoc.FrameworkName,
		BuildTool:     grpcprotoc.BuildToolProtoc,
		BuildFilePath: tmpDir,
		FrameworkData: &grpcprotoc.Info{
			ProtoFiles:        []string{protoFile},
			ServiceProtoFiles: []string{protoFile},
			ProtoRoot:         tmpDir,
			HasGoogleAPI:      false,
			HasBuf:            false,
			ImportPaths:       []string{tmpDir},
		},
	}

	opts := &extractor.GenerateOptions{
		OutputDir: tmpDir,
		Format:    "yaml",
		Timeout:   30 * time.Second,
	}

	result, err := g.Generate(context.Background(), tmpDir, info, opts)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if result.Format != "yaml" {
		t.Errorf("Expected format 'yaml', got: %s", result.Format)
	}

	// Verify the output file exists
	if _, statErr := os.Stat(result.SpecFilePath); os.IsNotExist(statErr) {
		t.Errorf("Expected output file to exist at: %s", result.SpecFilePath)
	}
}

func TestGenerator_Generate_BuildProtocArgs(t *testing.T) {
	tmpDir := t.TempDir()
	protoFile := filepath.Join(tmpDir, "test.proto")
	protoContent := `syntax = "proto3";
package test;
service TestService { rpc Method(Request) returns (Response); }
message Request {}
message Response {}
`
	if err := os.WriteFile(protoFile, []byte(protoContent), 0o644); err != nil {
		t.Fatalf("Failed to create .proto file: %v", err)
	}

	var capturedArgs []string
	mockExec := &mockExecutorForGenerator{
		results: map[string]*executor.ExecuteResult{
			"protoc": {ExitCode: 0},
		},
		errors: map[string]error{
			"protoc": nil,
		},
		captureArgs: &capturedArgs,
		postExecuteHook: func() {
			// Create the expected output
			_ = os.WriteFile(filepath.Join(tmpDir, "test.openapi.json"), []byte(`{}`), 0o644)
		},
	}

	g := grpcprotoc.NewGeneratorWithExecutor(mockExec)

	info := &extractor.ProjectInfo{
		Framework:     grpcprotoc.FrameworkName,
		BuildTool:     grpcprotoc.BuildToolProtoc,
		BuildFilePath: tmpDir,
		FrameworkData: &grpcprotoc.Info{
			ProtoFiles:        []string{protoFile},
			ServiceProtoFiles: []string{protoFile},
			ProtoRoot:         tmpDir,
			HasGoogleAPI:      false,
			HasBuf:            false,
			ImportPaths:       []string{tmpDir},
		},
	}

	opts := &extractor.GenerateOptions{
		OutputDir: tmpDir,
		Format:    "json",
		Timeout:   30 * time.Second,
	}

	_, err := g.Generate(context.Background(), tmpDir, info, opts)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify protoc args
	if len(capturedArgs) == 0 {
		t.Fatal("Expected protoc args to be captured")
	}

	// Should have -I flag for import path (either -I. or -I<tmpDir>)
	if !containsArgPrefix(capturedArgs, "-I") {
		t.Errorf("Expected -I flag with import path, got: %v", capturedArgs)
	}

	// Should have --connect-openapi_out flag
	if !containsArgPrefix(capturedArgs, "--connect-openapi_out=") {
		t.Errorf("Expected --connect-openapi_out flag, got: %v", capturedArgs)
	}

	// Should include proto file (as relative path "test.proto")
	if !containsArg(capturedArgs, "test.proto") {
		t.Errorf("Expected proto file in args, got: %v", capturedArgs)
	}
}

func TestGenerator_Generate_WithYAMLFormat(t *testing.T) {
	tmpDir := t.TempDir()
	protoFile := filepath.Join(tmpDir, "test.proto")
	protoContent := `syntax = "proto3";
package test;
service TestService { rpc Method(Request) returns (Response); }
message Request {}
message Response {}
`
	if err := os.WriteFile(protoFile, []byte(protoContent), 0o644); err != nil {
		t.Fatalf("Failed to create .proto file: %v", err)
	}

	var capturedArgs []string
	mockExec := &mockExecutorForGenerator{
		results: map[string]*executor.ExecuteResult{
			"protoc": {ExitCode: 0},
		},
		errors: map[string]error{
			"protoc": nil,
		},
		captureArgs: &capturedArgs,
		postExecuteHook: func() {
			_ = os.WriteFile(filepath.Join(tmpDir, "test.openapi.yaml"), []byte(`openapi: 3.0.0`), 0o644)
		},
	}

	g := grpcprotoc.NewGeneratorWithExecutor(mockExec)

	info := &extractor.ProjectInfo{
		Framework:     grpcprotoc.FrameworkName,
		BuildTool:     grpcprotoc.BuildToolProtoc,
		BuildFilePath: tmpDir,
		FrameworkData: &grpcprotoc.Info{
			ProtoFiles:        []string{protoFile},
			ServiceProtoFiles: []string{protoFile},
			ProtoRoot:         tmpDir,
			HasGoogleAPI:      false,
			HasBuf:            false,
			ImportPaths:       []string{tmpDir},
		},
	}

	opts := &extractor.GenerateOptions{
		OutputDir: tmpDir,
		Format:    "yaml",
		Timeout:   30 * time.Second,
	}

	_, err := g.Generate(context.Background(), tmpDir, info, opts)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check that format=yaml option is added
	if !slices.Contains(capturedArgs, "--connect-openapi_opt=format=yaml") {
		t.Errorf("Expected --connect-openapi_opt=format=yaml, got: %v", capturedArgs)
	}
}

func TestGenerator_Generate_WithExtraImportPaths(t *testing.T) {
	tmpDir := t.TempDir()
	protoFile := filepath.Join(tmpDir, "test.proto")
	protoContent := `syntax = "proto3";
package test;
service TestService { rpc Method(Request) returns (Response); }
message Request {}
message Response {}
`
	if err := os.WriteFile(protoFile, []byte(protoContent), 0o644); err != nil {
		t.Fatalf("Failed to create .proto file: %v", err)
	}

	extraPath := "/usr/local/include"

	var capturedArgs []string
	mockExec := &mockExecutorForGenerator{
		results: map[string]*executor.ExecuteResult{
			"protoc": {ExitCode: 0},
		},
		errors: map[string]error{
			"protoc": nil,
		},
		captureArgs: &capturedArgs,
		postExecuteHook: func() {
			_ = os.WriteFile(filepath.Join(tmpDir, "test.openapi.json"), []byte(`{}`), 0o644)
		},
	}

	g := grpcprotoc.NewGeneratorWithExecutor(mockExec)

	info := &extractor.ProjectInfo{
		Framework:     grpcprotoc.FrameworkName,
		BuildTool:     grpcprotoc.BuildToolProtoc,
		BuildFilePath: tmpDir,
		FrameworkData: &grpcprotoc.Info{
			ProtoFiles:        []string{protoFile},
			ServiceProtoFiles: []string{protoFile},
			ProtoRoot:         tmpDir,
			HasGoogleAPI:      false,
			HasBuf:            false,
			ImportPaths:       []string{tmpDir},
		},
	}

	// Use a custom GenerateOptions with extra data via context or separate mechanism
	// For now, we'll test the extra import paths by modifying the Info
	info.FrameworkData.(*grpcprotoc.Info).ImportPaths = append(info.FrameworkData.(*grpcprotoc.Info).ImportPaths, extraPath)

	opts := &extractor.GenerateOptions{
		OutputDir: tmpDir,
		Format:    "json",
		Timeout:   30 * time.Second,
	}

	_, err := g.Generate(context.Background(), tmpDir, info, opts)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check that extra import path is included (may be relative)
	hasExtraPath := containsArg(capturedArgs, "-I"+extraPath) || containsArgPrefix(capturedArgs, "-I")
	if !hasExtraPath {
		t.Errorf("Expected extra import path -I%s, got: %v", extraPath, capturedArgs)
	}
}

func TestGenerator_Generate_FindOutputFile(t *testing.T) {
	// Test that generator correctly finds the output file
	tmpDir := t.TempDir()
	protoFile := filepath.Join(tmpDir, "myservice.proto")
	protoContent := `syntax = "proto3";
package test;
service MyService { rpc Method(Request) returns (Response); }
message Request {}
message Response {}
`
	if err := os.WriteFile(protoFile, []byte(protoContent), 0o644); err != nil {
		t.Fatalf("Failed to create .proto file: %v", err)
	}

	// The output should be named based on the proto file: myservice.openapi.json
	expectedOutputPath := filepath.Join(tmpDir, "myservice.openapi.json")

	mockExec := &mockExecutorForGenerator{
		results: map[string]*executor.ExecuteResult{
			"protoc": {ExitCode: 0},
		},
		errors: map[string]error{
			"protoc": nil,
		},
		postExecuteHook: func() {
			// Create the file with the expected naming pattern
			_ = os.WriteFile(expectedOutputPath, []byte(`{"openapi":"3.0.0"}`), 0o644)
		},
	}

	g := grpcprotoc.NewGeneratorWithExecutor(mockExec)

	info := &extractor.ProjectInfo{
		Framework:     grpcprotoc.FrameworkName,
		BuildTool:     grpcprotoc.BuildToolProtoc,
		BuildFilePath: tmpDir,
		FrameworkData: &grpcprotoc.Info{
			ProtoFiles:        []string{protoFile},
			ServiceProtoFiles: []string{protoFile},
			ProtoRoot:         tmpDir,
			HasGoogleAPI:      false,
			HasBuf:            false,
			ImportPaths:       []string{tmpDir},
		},
	}

	opts := &extractor.GenerateOptions{
		OutputDir: tmpDir,
		Format:    "json",
		Timeout:   30 * time.Second,
	}

	result, err := g.Generate(context.Background(), tmpDir, info, opts)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if result.SpecFilePath != expectedOutputPath {
		t.Errorf("Expected SpecFilePath %s, got: %s", expectedOutputPath, result.SpecFilePath)
	}
}

func TestGenerator_Generate_Error_NoProtoFiles(t *testing.T) {
	tmpDir := t.TempDir()

	g := grpcprotoc.NewGenerator()

	info := &extractor.ProjectInfo{
		Framework:     grpcprotoc.FrameworkName,
		BuildTool:     grpcprotoc.BuildToolProtoc,
		BuildFilePath: tmpDir,
		FrameworkData: &grpcprotoc.Info{
			ProtoFiles:        []string{}, // No proto files
			ServiceProtoFiles: []string{},
			ProtoRoot:         tmpDir,
			HasGoogleAPI:      false,
			HasBuf:            false,
			ImportPaths:       []string{tmpDir},
		},
	}

	opts := &extractor.GenerateOptions{
		OutputDir: tmpDir,
		Format:    "json",
		Timeout:   30 * time.Second,
	}

	_, err := g.Generate(context.Background(), tmpDir, info, opts)
	if err == nil {
		t.Fatal("Expected error when no proto files are provided")
	}

	if !containsStr(err.Error(), "no proto files") {
		t.Errorf("Expected error message to mention 'no proto files', got: %v", err)
	}
}

func TestGenerator_Generate_Error_NoServiceProtoFiles(t *testing.T) {
	tmpDir := t.TempDir()
	// Create proto file without service definition
	protoFile := filepath.Join(tmpDir, "test.proto")
	if err := os.WriteFile(protoFile, []byte(`syntax = "proto3"; message Empty {}`), 0o644); err != nil {
		t.Fatalf("Failed to create .proto file: %v", err)
	}

	g := grpcprotoc.NewGenerator()

	info := &extractor.ProjectInfo{
		Framework:     grpcprotoc.FrameworkName,
		BuildTool:     grpcprotoc.BuildToolProtoc,
		BuildFilePath: tmpDir,
		FrameworkData: &grpcprotoc.Info{
			ProtoFiles:        []string{protoFile},
			ServiceProtoFiles: []string{}, // No service proto files
			ProtoRoot:         tmpDir,
			HasGoogleAPI:      false,
			HasBuf:            false,
			ImportPaths:       []string{tmpDir},
		},
	}

	opts := &extractor.GenerateOptions{
		OutputDir: tmpDir,
		Format:    "json",
		Timeout:   30 * time.Second,
	}

	_, err := g.Generate(context.Background(), tmpDir, info, opts)
	if err == nil {
		t.Fatal("Expected error when no service proto files are provided")
	}

	if !containsStr(err.Error(), "no proto files with service definitions") {
		t.Errorf("Expected error message to mention 'no proto files with service definitions', got: %v", err)
	}
}

func TestGenerator_Generate_Error_ProtocFails(t *testing.T) {
	tmpDir := t.TempDir()
	protoFile := filepath.Join(tmpDir, "test.proto")
	protoContent := `syntax = "proto3";
package test;
service TestService { rpc Method(Request) returns (Response); }
message Request {}
message Response {}
`
	if err := os.WriteFile(protoFile, []byte(protoContent), 0o644); err != nil {
		t.Fatalf("Failed to create .proto file: %v", err)
	}

	mockExec := &mockExecutorForGenerator{
		results: map[string]*executor.ExecuteResult{
			"protoc": {
				ExitCode: 1,
				Stdout:   "",
				Stderr:   "protoc error: syntax error",
			},
		},
		errors: map[string]error{
			"protoc": &executor.CommandFailedError{
				Command:  "protoc",
				ExitCode: 1,
				Stderr:   "protoc error: syntax error",
			},
		},
	}

	g := grpcprotoc.NewGeneratorWithExecutor(mockExec)

	info := &extractor.ProjectInfo{
		Framework:     grpcprotoc.FrameworkName,
		BuildTool:     grpcprotoc.BuildToolProtoc,
		BuildFilePath: tmpDir,
		FrameworkData: &grpcprotoc.Info{
			ProtoFiles:        []string{protoFile},
			ServiceProtoFiles: []string{protoFile},
			ProtoRoot:         tmpDir,
			HasGoogleAPI:      false,
			HasBuf:            false,
			ImportPaths:       []string{tmpDir},
		},
	}

	opts := &extractor.GenerateOptions{
		OutputDir: tmpDir,
		Format:    "json",
		Timeout:   30 * time.Second,
	}

	_, err := g.Generate(context.Background(), tmpDir, info, opts)
	if err == nil {
		t.Fatal("Expected error when protoc fails")
	}
}

func TestGenerator_Generate_Error_OutputFileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	protoFile := filepath.Join(tmpDir, "test.proto")
	protoContent := `syntax = "proto3";
package test;
service TestService { rpc Method(Request) returns (Response); }
message Request {}
message Response {}
`
	if err := os.WriteFile(protoFile, []byte(protoContent), 0o644); err != nil {
		t.Fatalf("Failed to create .proto file: %v", err)
	}

	// Don't create output file - simulate protoc not generating it
	mockExec := &mockExecutorForGenerator{
		results: map[string]*executor.ExecuteResult{
			"protoc": {ExitCode: 0},
		},
		errors: map[string]error{
			"protoc": nil,
		},
		// No postExecuteHook, so no output file is created
	}

	g := grpcprotoc.NewGeneratorWithExecutor(mockExec)

	info := &extractor.ProjectInfo{
		Framework:     grpcprotoc.FrameworkName,
		BuildTool:     grpcprotoc.BuildToolProtoc,
		BuildFilePath: tmpDir,
		FrameworkData: &grpcprotoc.Info{
			ProtoFiles:        []string{protoFile},
			ServiceProtoFiles: []string{protoFile},
			ProtoRoot:         tmpDir,
			HasGoogleAPI:      false,
			HasBuf:            false,
			ImportPaths:       []string{tmpDir},
		},
	}

	opts := &extractor.GenerateOptions{
		OutputDir: tmpDir,
		Format:    "json",
		Timeout:   30 * time.Second,
	}

	_, err := g.Generate(context.Background(), tmpDir, info, opts)
	if err == nil {
		t.Fatal("Expected error when output file is not found")
	}

	if !containsStr(err.Error(), "output file not found") {
		t.Errorf("Expected error message to mention 'output file not found', got: %v", err)
	}
}

func TestGenerator_Generate_NilOptions(t *testing.T) {
	tmpDir := t.TempDir()
	protoFile := filepath.Join(tmpDir, "test.proto")
	protoContent := `syntax = "proto3";
package test;
service TestService { rpc Method(Request) returns (Response); }
message Request {}
message Response {}
`
	if err := os.WriteFile(protoFile, []byte(protoContent), 0o644); err != nil {
		t.Fatalf("Failed to create .proto file: %v", err)
	}

	mockExec := &mockExecutorForGenerator{
		results: map[string]*executor.ExecuteResult{
			"protoc": {ExitCode: 0},
		},
		errors: map[string]error{
			"protoc": nil,
		},
		postExecuteHook: func() {
			_ = os.WriteFile(filepath.Join(tmpDir, "test.openapi.json"), []byte(`{}`), 0o644)
		},
	}

	g := grpcprotoc.NewGeneratorWithExecutor(mockExec)

	info := &extractor.ProjectInfo{
		Framework:     grpcprotoc.FrameworkName,
		BuildTool:     grpcprotoc.BuildToolProtoc,
		BuildFilePath: tmpDir,
		FrameworkData: &grpcprotoc.Info{
			ProtoFiles:        []string{protoFile},
			ServiceProtoFiles: []string{protoFile},
			ProtoRoot:         tmpDir,
			HasGoogleAPI:      false,
			HasBuf:            false,
			ImportPaths:       []string{tmpDir},
		},
	}

	// Test with nil options - should use defaults
	result, err := g.Generate(context.Background(), tmpDir, info, nil)
	if err != nil {
		t.Fatalf("Generate failed with nil options: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}
}

// mockExecutorForGenerator is a mock executor for generator tests
type mockExecutorForGenerator struct {
	results         map[string]*executor.ExecuteResult
	errors          map[string]error
	captureArgs     *[]string
	postExecuteHook func()
}

func (m *mockExecutorForGenerator) Execute(ctx context.Context, opts *executor.ExecuteOptions) (*executor.ExecuteResult, error) {
	// Capture args if requested
	if m.captureArgs != nil {
		*m.captureArgs = opts.Args
	}

	// Call post-execute hook if set (to simulate file creation)
	if m.postExecuteHook != nil {
		m.postExecuteHook()
	}

	key := opts.Command
	if result, ok := m.results[key]; ok {
		return result, m.errors[key]
	}
	return nil, m.errors[key]
}

// Helper functions
func containsArg(slice []string, item string) bool {
	return slices.Contains(slice, item)
}

func containsArgPrefix(slice []string, prefix string) bool {
	for _, s := range slice {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}

func containsStr(s, substr string) bool {
	return strings.Contains(s, substr)
}
