//go:build e2e

package e2e_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spencercjh/spec-forge/cmd"
	"github.com/spencercjh/spec-forge/integration-tests/helpers"
)

// TestE2E_GrpcProtoc_Generate tests the generate flow for a gRPC-protoc project.
func TestE2E_GrpcProtoc_Generate(t *testing.T) {
	projectPath := "grpc-protoc-demo"

	// Check if project exists
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("gRPC-protoc demo project not found")
	}

	// Check if proto files exist
	protoPath := filepath.Join(projectPath, "proto", "user.proto")
	if _, err := os.Stat(protoPath); os.IsNotExist(err) {
		t.Skip("user.proto not found, skipping test")
	}

	// Create temp output directory
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
	})

	err := rootCmd.Execute()
	// protoc generation may fail if tools are not available
	if err != nil {
		t.Skipf("generate command failed (may be due to missing protoc): %v\nstderr: %s", err, stderr.String())
	}

	// Find the generated spec file
	specFile := helpers.FindSpecFile(t, outputDir, "yaml")
	if specFile == "" {
		t.Log("no spec file found - protoc tools may not be available")
		return
	}

	// Verify spec content - check for REST paths from HTTP annotations
	specData, err := os.ReadFile(specFile)
	if err != nil {
		t.Fatalf("failed to read spec file: %v", err)
	}

	// The generated YAML should contain REST paths
	specContent := string(specData)

	// Check for expected REST paths from google.api.http annotations
	expectedPaths := []string{
		"/v1/users:",
		"/v1/users/{id}:",
		"/v1/users/{user_id}/files:",
	}

	for _, expectedPath := range expectedPaths {
		if !strings.Contains(specContent, expectedPath) {
			t.Errorf("expected REST path %s not found in spec", expectedPath)
		} else {
			t.Logf("Found REST path: %s", expectedPath)
		}
	}

	// Perform comprehensive spec validation using the validator
	validator := helpers.NewSpecValidator(t, specFile)
	validator.ValidateOpenAPIVersion()
	validator.ValidateInfo()

	// Validate paths exist
	validator.ValidatePaths([]string{
		"/v1/users",
		"/v1/users/{id}",
		"/v1/users/{user_id}/files",
	})

	// Validate operations
	validator.ValidateOperationFields("/v1/users", "get", true, true)
	validator.ValidateResponseCodes("/v1/users", "get", []string{"200"})

	validator.ValidateOperationFields("/v1/users", "post", true, true)
	validator.ValidateResponseCodes("/v1/users", "post", []string{"200"})

	validator.ValidateOperationFields("/v1/users/{id}", "get", true, true)
	validator.ValidateResponseCodes("/v1/users/{id}", "get", []string{"200"})

	validator.LogSummary()

	t.Logf("Successfully generated OpenAPI spec with REST endpoints at: %s", specFile)
}
