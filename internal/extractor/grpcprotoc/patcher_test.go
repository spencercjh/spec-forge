// Package grpcprotoc_test tests the gRPC-protoc extractor implementation.
package grpcprotoc_test

import (
	"context"
	"testing"

	"github.com/spencercjh/spec-forge/internal/executor"
	"github.com/spencercjh/spec-forge/internal/extractor/grpcprotoc"
)

// mockExecutor is a mock implementation of executor.Interface for testing.
type mockExecutor struct {
	results map[string]*executor.ExecuteResult
	errors  map[string]error
}

func (m *mockExecutor) Execute(ctx context.Context, opts *executor.ExecuteOptions) (*executor.ExecuteResult, error) {
	key := opts.Command
	if result, ok := m.results[key]; ok {
		return result, m.errors[key]
	}
	return nil, m.errors[key]
}

func TestPatcher_Patch_BothToolsInstalled(t *testing.T) {
	mockExec := &mockExecutor{
		results: map[string]*executor.ExecuteResult{
			"protoc": {
				ExitCode: 0,
				Stdout:   "libprotoc 25.1",
				Stderr:   "",
			},
			"protoc-gen-connect-openapi": {
				ExitCode: 0,
				Stdout:   "protoc-gen-connect-openapi version 0.5.0",
				Stderr:   "",
			},
		},
		errors: map[string]error{
			"protoc":                     nil,
			"protoc-gen-connect-openapi": nil,
		},
	}

	patcher := grpcprotoc.NewPatcherWithExecutor(mockExec)
	result, err := patcher.Patch("/some/path")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if !result.ProtocInstalled {
		t.Error("Expected ProtocInstalled to be true")
	}

	if result.ProtocVersion != "libprotoc 25.1" {
		t.Errorf("Expected ProtocVersion 'libprotoc 25.1', got: %s", result.ProtocVersion)
	}

	if !result.ProtocGenConnectOpenAPIInstalled {
		t.Error("Expected ProtocGenConnectOpenAPIInstalled to be true")
	}

	if result.ProtocGenConnectOpenAPIVersion != "protoc-gen-connect-openapi version 0.5.0" {
		t.Errorf("Expected ProtocGenConnectOpenAPIVersion 'protoc-gen-connect-openapi version 0.5.0', got: %s", result.ProtocGenConnectOpenAPIVersion)
	}
}

func TestPatcher_Patch_ProtocNotInstalled(t *testing.T) {
	mockExec := &mockExecutor{
		results: map[string]*executor.ExecuteResult{
			"protoc": {
				ExitCode: -1,
				Stdout:   "",
				Stderr:   "",
			},
		},
		errors: map[string]error{
			"protoc": &executor.CommandNotFoundError{
				Command: "protoc",
			},
		},
	}

	patcher := grpcprotoc.NewPatcherWithExecutor(mockExec)
	result, err := patcher.Patch("/some/path")

	if err == nil {
		t.Error("Expected error when protoc not found, got nil")
	}

	if result != nil {
		t.Error("Expected nil result when protoc not found")
	}

	// Check that the error contains the installation hint
	expectedMsg := "protoc not found"
	if err != nil && !contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error message to contain '%s', got: %s", expectedMsg, err.Error())
	}

	// Verify the installation hint is included
	installHint := "github.com/protocolbuffers/protobuf/releases"
	if err != nil && !contains(err.Error(), installHint) {
		t.Errorf("Expected error message to contain installation hint '%s', got: %s", installHint, err.Error())
	}
}

func TestPatcher_Patch_ProtocGenConnectOpenAPINotInstalled(t *testing.T) {
	mockExec := &mockExecutor{
		results: map[string]*executor.ExecuteResult{
			"protoc": {
				ExitCode: 0,
				Stdout:   "libprotoc 25.1",
				Stderr:   "",
			},
			"protoc-gen-connect-openapi": {
				ExitCode: -1,
				Stdout:   "",
				Stderr:   "",
			},
		},
		errors: map[string]error{
			"protoc": nil,
			"protoc-gen-connect-openapi": &executor.CommandNotFoundError{
				Command: "protoc-gen-connect-openapi",
			},
		},
	}

	patcher := grpcprotoc.NewPatcherWithExecutor(mockExec)
	result, err := patcher.Patch("/some/path")

	if err == nil {
		t.Error("Expected error when protoc-gen-connect-openapi not found, got nil")
	}

	if result != nil {
		t.Error("Expected nil result when protoc-gen-connect-openapi not found")
	}

	// Check that the error contains the installation hint
	expectedMsg := "protoc-gen-connect-openapi not found"
	if err != nil && !contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error message to contain '%s', got: %s", expectedMsg, err.Error())
	}

	// Verify the installation hint is included
	installHint := "go install github.com/sudorandom/protoc-gen-connect-openapi@latest"
	if err != nil && !contains(err.Error(), installHint) {
		t.Errorf("Expected error message to contain installation hint '%s', got: %s", installHint, err.Error())
	}
}

func TestPatcher_Patch_ProtocCommandFailed(t *testing.T) {
	mockExec := &mockExecutor{
		results: map[string]*executor.ExecuteResult{
			"protoc": {
				ExitCode: 1,
				Stdout:   "",
				Stderr:   "some error output",
			},
		},
		errors: map[string]error{
			"protoc": &executor.CommandFailedError{
				Command:  "protoc",
				Args:     []string{"--version"},
				ExitCode: 1,
				Stderr:   "some error output",
			},
		},
	}

	patcher := grpcprotoc.NewPatcherWithExecutor(mockExec)
	result, err := patcher.Patch("/some/path")

	if err == nil {
		t.Error("Expected error when protoc command fails, got nil")
	}

	if result != nil {
		t.Error("Expected nil result when protoc command fails")
	}

	expectedMsg := "protoc version check failed"
	if err != nil && !contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error message to contain '%s', got: %s", expectedMsg, err.Error())
	}
}

func TestPatcher_Patch_ProtocGenConnectOpenAPICommandFailed(t *testing.T) {
	mockExec := &mockExecutor{
		results: map[string]*executor.ExecuteResult{
			"protoc": {
				ExitCode: 0,
				Stdout:   "libprotoc 25.1",
				Stderr:   "",
			},
			"protoc-gen-connect-openapi": {
				ExitCode: 1,
				Stdout:   "",
				Stderr:   "plugin error",
			},
		},
		errors: map[string]error{
			"protoc": nil,
			"protoc-gen-connect-openapi": &executor.CommandFailedError{
				Command:  "protoc-gen-connect-openapi",
				Args:     []string{"--version"},
				ExitCode: 1,
				Stderr:   "plugin error",
			},
		},
	}

	patcher := grpcprotoc.NewPatcherWithExecutor(mockExec)
	result, err := patcher.Patch("/some/path")

	if err == nil {
		t.Error("Expected error when protoc-gen-connect-openapi command fails, got nil")
	}

	if result != nil {
		t.Error("Expected nil result when protoc-gen-connect-openapi command fails")
	}

	expectedMsg := "protoc-gen-connect-openapi version check failed"
	if err != nil && !contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error message to contain '%s', got: %s", expectedMsg, err.Error())
	}
}

func TestPatcher_NeedsPatch(t *testing.T) {
	patcher := grpcprotoc.NewPatcher()

	// NeedsPatch should always return true for protoc projects
	info := &grpcprotoc.Info{
		ProtoFiles:   []string{"/path/to/test.proto"},
		ProtoRoot:    "/path/to",
		HasGoogleAPI: false,
		HasBuf:       false,
		ImportPaths:  []string{"/path/to"},
	}

	if !patcher.NeedsPatch(info) {
		t.Error("Expected NeedsPatch to return true")
	}
}

func TestNewPatcher(t *testing.T) {
	patcher := grpcprotoc.NewPatcher()
	if patcher == nil {
		t.Fatal("Expected patcher to not be nil")
	}
}

func TestNewPatcherWithExecutor(t *testing.T) {
	mockExec := &mockExecutor{
		results: map[string]*executor.ExecuteResult{
			"protoc": {
				ExitCode: 0,
				Stdout:   "libprotoc 25.1",
			},
			"protoc-gen-connect-openapi": {
				ExitCode: 0,
				Stdout:   "protoc-gen-connect-openapi version 0.5.0",
			},
		},
		errors: map[string]error{
			"protoc":                     nil,
			"protoc-gen-connect-openapi": nil,
		},
	}

	patcher := grpcprotoc.NewPatcherWithExecutor(mockExec)
	if patcher == nil {
		t.Fatal("Expected patcher to not be nil")
	}

	// Verify the custom executor is used by calling Patch
	result, err := patcher.Patch("/some/path")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if result == nil || !result.ProtocInstalled {
		t.Error("Expected protoc to be installed")
	}
}

// Helper function
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
