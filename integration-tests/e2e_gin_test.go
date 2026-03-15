//go:build e2e

package e2e_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spencercjh/spec-forge/cmd"
)

func TestE2E_GinDemo_Generate(t *testing.T) {
	projectPath := "./gin-demo"

	// Check if project exists
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("Gin demo project not found")
	}

	// Check if go.mod exists
	goModPath := filepath.Join(projectPath, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		t.Skip("go.mod not found, skipping test")
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
				Path:                    "/api/v1/users",
				Method:                  "get",
				WantOperationID:         true,
				ExpectedResponseCodes:   []string{"200", "400"},
				ValidateResponseContent: "application/json",
			},
			{
				Path:                  "/api/v1/users",
				Method:                "post",
				WantOperationID:       true,
				ExpectedResponseCodes: []string{"201", "400"},
				WantRequestBody:       "application/json",
			},
			{
				Path:                  "/api/v1/users/{id}",
				Method:                "get",
				WantOperationID:       true,
				ExpectedResponseCodes: []string{"200", "400", "404"},
				ExpectedParams:        []string{"id"},
			},
			{
				Path:                  "/api/v1/users/{id}/profile",
				Method:                "post",
				WantOperationID:       true,
				ExpectedResponseCodes: []string{"200", "400", "404"},
				ExpectedParams:        []string{"id"},
			},
			{
				Path:                    "/api/v1/users/upload",
				Method:                  "post",
				WantOperationID:         true,
				ExpectedResponseCodes:   []string{"200", "400"},
				ValidateResponseContent: "application/json",
			},
		},
		ExpectedSchemas: []string{
			"User",
			"CreateUserRequest",
			"UpdateProfileRequest",
			"ApiResponse",
			"PageResult",
			"FileUploadResult",
		},
	})

	// === Semantic Validation ===
	t.Log("=== Semantic Validation ===")

	// 1. GET /api/v1/users should have query params: page, size, username
	validator.ValidateParameterDetails("/api/v1/users", "get", []ParameterExpectation{
		{Name: "page", In: "query", Required: false},
		{Name: "size", In: "query", Required: false},
		{Name: "username", In: "query", Required: false},
	})

	// 2. POST /api/v1/users requestBody should reference CreateUserRequest
	validator.ValidateRequestBodySchema("/api/v1/users", "post", "CreateUserRequest")

	// 3. GET /api/v1/users/{id} id parameter should be in=path and required=true
	validator.ValidateParameterDetails("/api/v1/users/{id}", "get", []ParameterExpectation{
		{Name: "id", In: "path", Required: true},
	})

	// 4. PageResult.content should be array of User
	validator.ValidateSchemaProperty("PageResult", SchemaPropertyExpectation{
		Name:     "content",
		Type:     "array",
		ItemType: "User",
	})

	// 5. UpdateProfileRequest form parameters should be properly mapped
	validator.ValidateParameterDetails("/api/v1/users/{id}/profile", "post", []ParameterExpectation{
		{Name: "id", In: "path", Required: true},
		{Name: "fullName", In: "query", Required: false},
		{Name: "email", In: "query", Required: false},
		{Name: "age", In: "query", Required: false},
	})

	// 6. Response schemas should be properly defined (not just status codes)
	validator.ValidateResponseSchema("/api/v1/users", "post", ResponseSchemaExpectation{
		Code:        "201",
		ContentType: "application/json",
		SchemaRef:   "ApiResponse",
	})
	validator.ValidateResponseSchema("/api/v1/users", "post", ResponseSchemaExpectation{
		Code:        "400",
		ContentType: "application/json",
		SchemaRef:   "ApiResponse",
	})
	validator.ValidateResponseSchema("/api/v1/users/{id}", "get", ResponseSchemaExpectation{
		Code:        "404",
		ContentType: "application/json",
		SchemaRef:   "ApiResponse",
	})

	t.Log("All validations passed!")
}

func TestE2E_GinDemo_JSONFormat(t *testing.T) {
	projectPath := "./gin-demo"

	// Check if project exists
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("Gin demo project not found")
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
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		t.Fatalf("failed to read output directory: %v", err)
	}

	var specFile string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			specFile = filepath.Join(outputDir, entry.Name())
			break
		}
	}

	if specFile == "" {
		t.Fatal("no JSON spec file found in output directory")
	}

	// Verify JSON content
	specData, err := os.ReadFile(specFile)
	if err != nil {
		t.Fatalf("failed to read spec file: %v", err)
	}

	var spec map[string]any
	if err := json.Unmarshal(specData, &spec); err != nil {
		t.Fatalf("failed to parse spec JSON: %v", err)
	}

	// Verify basic structure
	if spec["openapi"] == nil {
		t.Error("expected openapi field in spec")
	}

	if spec["paths"] == nil {
		t.Error("expected paths field in spec")
	}

	paths, ok := spec["paths"].(map[string]any)
	if !ok {
		t.Fatal("expected paths to be an object")
	}

	// Verify expected paths exist
	expectedPaths := []string{
		"/api/v1/users",
		"/api/v1/users/{id}",
		"/api/v1/users/{id}/profile",
		"/api/v1/users/upload",
	}

	for _, path := range expectedPaths {
		if _, exists := paths[path]; !exists {
			t.Errorf("expected path %s not found in JSON spec", path)
		}
	}

	t.Log("JSON format test passed!")
}

// TestE2E_GinDemo_DefaultOutput tests the generator's behavior when OutputDir
// is set to the project path.
func TestE2E_GinDemo_DefaultOutput(t *testing.T) {
	projectPath := "./gin-demo"

	// Check if project exists
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("Gin demo project not found")
	}

	rootCmd := cmd.NewRootCommand()

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{
		"generate",
		projectPath,
		"--output", "yaml",
		"--output-dir", projectPath,
		"--skip-enrich",
		"--skip-publish",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v\nstderr: %s", err, stderr.String())
	}

	// Verify output is in project root
	expectedPath := filepath.Join(projectPath, "openapi.yaml")
	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf("spec file not found at expected path: %s", expectedPath)
	}

	// Cleanup
	_ = os.Remove(expectedPath)

	t.Logf("Spec correctly output to project root: %s", expectedPath)
}
