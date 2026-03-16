//go:build e2e

package e2e_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spencercjh/spec-forge/cmd"
	"github.com/spencercjh/spec-forge/integration-tests/helpers"
)

// TestE2E_GradleSpringBoot_Generate tests the generate flow for a Gradle Spring Boot project.
func TestE2E_GradleSpringBoot_Generate(t *testing.T) {
	projectPath := "gradle-springboot-openapi-demo"

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("Gradle Spring Boot demo project not found")
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
	specFile := helpers.FindSpecFile(t, outputDir, "json")

	// Perform comprehensive spec validation
	// Note: springdoc without annotations can generate:
	// - Paths, operationIds, parameters from Spring annotations
	// - Request body content type from @RequestBody/@PostMapping
	// - Basic schema from DTOs (but generates nested types like PageResultUser instead of PageResult)
	// It CANNOT generate without @ApiResponse annotations:
	// - Multiple response codes (only 200 from return type)
	// - Response content types (generates */* instead of application/json)
	// - Explicit error responses (404, 400, etc.)
	validator := helpers.NewSpecValidator(t, specFile)
	validator.FullValidation(helpers.ValidationConfig{
		ExpectedPaths: []string{
			"/api/v1/users",
			"/api/v1/users/{id}",
			"/api/v1/users/{id}/profile",
			"/api/v1/users/upload",
		},
		Operations: []helpers.OperationConfig{
			{
				Path:            "/api/v1/users",
				Method:          "get",
				WantOperationID: true,
			},
			{
				Path:            "/api/v1/users",
				Method:          "post",
				WantOperationID: true,
				WantRequestBody: "User",
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

	// Validate query parameters for GET /api/v1/users
	validator.ValidateParameterDetails("/api/v1/users", "get", []helpers.ParameterExpectation{
		{Name: "page", In: "query", Required: false},
		{Name: "size", In: "query", Required: false},
		{Name: "username", In: "query", Required: false},
	})

	t.Logf("Successfully generated valid OpenAPI spec at: %s", specFile)
}
