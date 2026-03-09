//go:build e2e

package e2e_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spencercjh/spec-forge/cmd"
)

// TestE2E_MavenMultiModule_Generate tests the generate flow for a Maven multi-module project.
func TestE2E_MavenMultiModule_Generate(t *testing.T) {
	projectPath := "maven-multi-module-demo"

	// Check if project exists
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("Maven multi-module demo project not found")
	}

	// Check if mvnw wrapper exists
	mvnwPath := filepath.Join(projectPath, "mvnw")
	if _, err := os.Stat(mvnwPath); os.IsNotExist(err) {
		t.Skip("Maven wrapper not found, skipping test")
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
		"--output", "json",
		"--skip-enrich",
		"--skip-publish",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v\nstderr: %s", err, stderr.String())
	}

	// Find the generated spec file
	specFile := FindSpecFile(t, outputDir, "json")

	// Perform comprehensive spec validation
	// Note: springdoc without annotations can generate:
	// - Paths, operationIds, parameters from Spring annotations
	// - Request body content type from @RequestBody/@PostMapping
	// - Basic schema from DTOs (but generates nested types like PageResultUser instead of PageResult)
	// It CANNOT generate without @ApiResponse annotations:
	// - Multiple response codes (only 200 from return type)
	// - Response content types (generates */* instead of application/json)
	// - Explicit error responses (404, 400, etc.)
	validator := NewSpecValidator(t, specFile)
	validator.FullValidation(ValidationConfig{
		ExpectedPaths: []string{
			"/api/v1/users",
			"/api/v1/users/{id}",
			"/api/v1/users/{id}/profile",
			"/api/v1/users/upload",
		},
		Operations: []OperationConfig{
			{
				Path:            "/api/v1/users",
				Method:          "get",
				WantOperationID: true,
				ExpectedParams:  []string{"page", "size", "username"},
			},
			{
				Path:            "/api/v1/users",
				Method:          "post",
				WantOperationID: true,
				WantRequestBody: "application/json",
			},
			{
				Path:            "/api/v1/users/{id}",
				Method:          "get",
				WantOperationID: true,
				ExpectedParams:  []string{"id"},
			},
			{
				Path:            "/api/v1/users/{id}/profile",
				Method:          "post",
				WantOperationID: true,
				ExpectedParams:  []string{"id"},
			},
			{
				Path:            "/api/v1/users/upload",
				Method:          "post",
				WantOperationID: true,
			},
		},
		ExpectedSchemas: []string{
			"User",
			"FileUploadResult",
		},
	})

	t.Logf("Successfully generated spec from multi-module project at: %s", specFile)
}

// TestE2E_GradleMultiModule_Generate tests the generate flow for a Gradle multi-module project.
func TestE2E_GradleMultiModule_Generate(t *testing.T) {
	projectPath := "gradle-multi-module-demo"

	// Check if project exists
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("Gradle multi-module demo project not found")
	}

	// Check if gradlew wrapper exists
	gradlewPath := filepath.Join(projectPath, "gradlew")
	if _, err := os.Stat(gradlewPath); os.IsNotExist(err) {
		t.Skip("Gradle wrapper not found, skipping test")
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
		"--output", "json",
		"--skip-enrich",
		"--skip-publish",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v\nstderr: %s", err, stderr.String())
	}

	// Find the generated spec file
	specFile := FindSpecFile(t, outputDir, "json")

	// Perform comprehensive spec validation
	// Note: springdoc without annotations can generate:
	// - Paths, operationIds, parameters from Spring annotations
	// - Request body content type from @RequestBody/@PostMapping
	// - Basic schema from DTOs (but generates nested types like PageResultUser instead of PageResult)
	// It CANNOT generate without @ApiResponse annotations:
	// - Multiple response codes (only 200 from return type)
	// - Response content types (generates */* instead of application/json)
	// - Explicit error responses (404, 400, etc.)
	validator := NewSpecValidator(t, specFile)
	validator.FullValidation(ValidationConfig{
		ExpectedPaths: []string{
			"/api/v1/users",
			"/api/v1/users/{id}",
			"/api/v1/users/{id}/profile",
			"/api/v1/users/upload",
		},
		Operations: []OperationConfig{
			{
				Path:            "/api/v1/users",
				Method:          "get",
				WantOperationID: true,
				ExpectedParams:  []string{"page", "size", "username"},
			},
			{
				Path:            "/api/v1/users",
				Method:          "post",
				WantOperationID: true,
				WantRequestBody: "application/json",
			},
			{
				Path:            "/api/v1/users/{id}",
				Method:          "get",
				WantOperationID: true,
				ExpectedParams:  []string{"id"},
			},
			{
				Path:            "/api/v1/users/{id}/profile",
				Method:          "post",
				WantOperationID: true,
				ExpectedParams:  []string{"id"},
			},
			{
				Path:            "/api/v1/users/upload",
				Method:          "post",
				WantOperationID: true,
			},
		},
		ExpectedSchemas: []string{
			"User",
			"FileUploadResult",
		},
	})

	t.Logf("Successfully generated spec from Gradle multi-module project at: %s", specFile)
}
