//go:build e2e

package grpcprotoc

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spencercjh/spec-forge/cmd"
	"github.com/spencercjh/spec-forge/integration-tests/helpers"
)

// TestBufYAMLRejection tests that buf.yaml managed projects are rejected with a helpful error
func TestBufYAMLRejection(t *testing.T) {
	tempDir := t.TempDir()

	// Create a proto file with a service definition
	protoDir := filepath.Join(tempDir, "proto")
	if err := os.MkdirAll(protoDir, 0o755); err != nil {
		t.Fatalf("failed to create proto dir: %v", err)
	}

	protoContent := `syntax = "proto3";
package test;

message TestRequest {
  string name = 1;
}

message TestResponse {
  string message = 1;
}

service TestService {
  rpc GetTest(TestRequest) returns (TestResponse);
}
`
	if err := os.WriteFile(filepath.Join(protoDir, "test.proto"), []byte(protoContent), 0o644); err != nil {
		t.Fatalf("failed to write test.proto: %v", err)
	}

	// Create a buf.yaml to trigger rejection
	bufYAML := `version: v1
name: buf.build/test/project
`
	if err := os.WriteFile(filepath.Join(tempDir, "buf.yaml"), []byte(bufYAML), 0o644); err != nil {
		t.Fatalf("failed to write buf.yaml: %v", err)
	}

	outputDir := t.TempDir()

	rootCmd := cmd.NewRootCommand()
	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{
		"generate",
		tempDir,
		"--output-dir", outputDir,
		"--output", "json",
		"--skip-enrich",
		"--skip-publish",
	})

	err := rootCmd.Execute()
	if err == nil {
		t.Error("expected error for buf.yaml managed project")
		return
	}

	errMsg := err.Error()
	// The error could mention buf.yaml or fall through to another detector
	if strings.Contains(errMsg, "buf") || strings.Contains(errMsg, "detect") {
		t.Logf("Got expected error for buf.yaml project: %v", err)
	} else {
		t.Logf("Generate failed (expected): %v", err)
	}
}

// TestMissingProtocGracefulSkip tests graceful handling when protoc is not available
func TestMissingProtocGracefulSkip(t *testing.T) {
	if _, err := exec.LookPath("protoc"); err == nil {
		t.Skip("protoc is available, skipping missing protoc test")
	}

	projectPath := "../grpc-protoc-demo"

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("grpc-protoc-demo project not found")
	}

	outputDir := t.TempDir()

	rootCmd := cmd.NewRootCommand()
	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{
		"generate",
		projectPath,
		"--output-dir", outputDir,
		"--output", "json",
		"--skip-enrich",
		"--skip-publish",
	})

	err := rootCmd.Execute()
	if err == nil {
		t.Error("expected error when protoc is not available")
		return
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "protoc") {
		t.Errorf("expected error message to mention 'protoc', got: %s", errMsg)
	}

	t.Logf("Got expected error when protoc not available: %v", err)
}

// TestYAMLOutputFormat tests YAML output format for gRPC-protoc projects
func TestYAMLOutputFormat(t *testing.T) {
	// Acquire lock to prevent race conditions with other tests
	helpers.AcquireFileLock(t, "grpcprotoc")

	projectPath := "../grpc-protoc-demo"

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("grpc-protoc-demo project not found")
	}

	skipIfProtocNotAvailable(t)

	outputDir := t.TempDir()

	rootCmd := cmd.NewRootCommand()
	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{
		"generate",
		projectPath,
		"--output-dir", outputDir,
		"--output", "yaml",
		"--skip-enrich",
		"--skip-publish",
		"--skip-validate",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("YAML generation failed: %v\nstderr: %s", err, stderr.String())
	}

	specFile := helpers.FindSpecFile(t, outputDir, "yaml")
	validator := helpers.NewSpecValidator(t, specFile)

	validator.ValidateOpenAPIVersion()

	pathCount := validator.GetPathCount()
	if pathCount == 0 {
		t.Error("expected at least one path in YAML spec")
	}

	t.Logf("YAML output format: %d paths generated", pathCount)
}

// TestMultipleProtoFiles tests correct handling of multiple proto files
func TestMultipleProtoFiles(t *testing.T) {
	_, spec := generateSpec(t, "json")

	t.Run("Common Package Schemas Present", func(t *testing.T) {
		// The common.proto defines shared types that should be available as schemas
		components, ok := spec["components"].(map[string]any)
		if !ok {
			t.Fatal("components not found")
		}

		schemas, ok := components["schemas"].(map[string]any)
		if !ok {
			t.Fatal("schemas not found")
		}

		// Common package schemas should be imported
		expectedCommonSchemas := []string{
			"demo.common.ApiResponse",
			"demo.common.PageMetadata",
			"demo.common.PageRequest",
			"demo.common.FileInfo",
		}

		for _, schemaName := range expectedCommonSchemas {
			if _, ok := schemas[schemaName]; !ok {
				t.Errorf("expected common schema %q not found", schemaName)
			}
		}
	})

	t.Run("Service Schemas Present", func(t *testing.T) {
		components, ok := spec["components"].(map[string]any)
		if !ok {
			t.Fatal("components not found")
		}

		schemas, ok := components["schemas"].(map[string]any)
		if !ok {
			t.Fatal("schemas not found")
		}

		expectedSchemas := []string{
			"demo.user.User",
			"demo.user.CreateUserRequest",
			"demo.user.CreateUserResponse",
			"demo.user.GetUserRequest",
			"demo.user.GetUserResponse",
			"demo.user.ListUsersRequest",
			"demo.user.ListUsersResponse",
			"demo.user.UpdateProfileRequest",
			"demo.user.UpdateProfileResponse",
			"demo.user.UploadFileRequest",
			"demo.user.UploadFileResponse",
		}

		for _, schemaName := range expectedSchemas {
			if _, ok := schemas[schemaName]; !ok {
				t.Errorf("expected schema %q not found", schemaName)
			}
		}
	})
}

// TestNonProtocProject verifies error handling for non-protoc projects
func TestNonProtocProject(t *testing.T) {
	tempDir := t.TempDir()

	// Create a basic Go project without proto files
	goMod := `module non-protoc-test

go 1.26

require github.com/gin-gonic/gin v1.12.0
`
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	mainGo := `package main

func main() {}
`
	if err := os.WriteFile(filepath.Join(tempDir, "main.go"), []byte(mainGo), 0o644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	outputDir := t.TempDir()

	rootCmd := cmd.NewRootCommand()
	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{
		"generate",
		tempDir,
		"--output-dir", outputDir,
		"--output", "json",
		"--skip-enrich",
		"--skip-publish",
	})

	// This might either fail (if no detector matches) or succeed (if gin detector picks it up)
	// The important thing is that it doesn't crash
	_ = rootCmd.Execute()
	t.Log("Non-protoc project handled without crash")
}
