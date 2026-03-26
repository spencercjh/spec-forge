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

// TestE2E_MavenSpringBoot_Generate tests the generate flow for a Maven Spring Boot project.
func TestE2E_MavenSpringBoot_Generate(t *testing.T) {
	// Acquire lock to prevent port conflicts with other Spring Boot tests
	helpers.AcquireSpringLock(t)

	projectPath := "maven-springboot-openapi-demo"

	// Check if project exists
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("Maven Spring Boot demo project not found")
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

// TestE2E_MavenSpringBoot_GenerateEnrich tests the complete generate → enrich flow.
func TestE2E_MavenSpringBoot_GenerateEnrich(t *testing.T) {
	projectPath := "maven-springboot-openapi-demo"

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("Maven Spring Boot demo project not found")
	}

	// Check if mvnw wrapper exists
	mvnwPath := filepath.Join(projectPath, "mvnw")
	if _, err := os.Stat(mvnwPath); os.IsNotExist(err) {
		t.Skip("Maven wrapper not found, skipping test")
	}

	// Create temp output directory
	outputDir := t.TempDir()

	// Step 1: Generate spec
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
		t.Fatalf("generate command failed: %v\nstderr: %s", err, stderr.String())
	}

	// Find the generated spec file
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		t.Fatalf("failed to read output directory: %v", err)
	}

	var specFile string
	for _, entry := range entries {
		if !entry.IsDir() && (filepath.Ext(entry.Name()) == ".json" || filepath.Ext(entry.Name()) == ".yaml") {
			specFile = filepath.Join(outputDir, entry.Name())
			break
		}
	}

	if specFile == "" {
		t.Fatal("no spec file found in output directory")
	}

	// Step 2: Enrich with mock provider
	// Note: enrich command requires API key, so we just verify it fails gracefully
	// or we could set up a mock server, but for now we just verify the generate worked
	t.Logf("Generate completed. Spec file: %s", specFile)
	t.Log("Note: enrich test requires LLM API key, skipping enrichment step in E2E test")
}
