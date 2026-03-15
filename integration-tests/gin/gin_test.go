//go:build e2e

package gin

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spencercjh/spec-forge/cmd"
	"github.com/spencercjh/spec-forge/integration-tests/helpers"
)

// TestGenerate tests basic generation from gin-demo
func TestGenerate(t *testing.T) {
	projectPath := "./fixtures/gin-demo"

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("Gin demo project not found")
	}

	goModPath := filepath.Join(projectPath, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		t.Skip("go.mod not found, skipping test")
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
	if err != nil {
		t.Fatalf("command failed: %v\nstderr: %s", err, stderr.String())
	}

	specFile := helpers.FindSpecFile(t, outputDir, "json")
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
				WantRequestBody:       "CreateUserRequest",
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

	// Semantic validations
	t.Log("=== Semantic Validation ===")

	validator.ValidateParameterDetails("/api/v1/users", "get", []helpers.ParameterExpectation{
		{Name: "page", In: "query", Required: false},
		{Name: "size", In: "query", Required: false},
		{Name: "username", In: "query", Required: false},
	})

	validator.ValidateRequestBodySchema("/api/v1/users", "post", "CreateUserRequest")

	validator.ValidateParameterDetails("/api/v1/users/{id}", "get", []helpers.ParameterExpectation{
		{Name: "id", In: "path", Required: true},
	})

	validator.ValidateSchemaProperty("PageResult", helpers.SchemaPropertyExpectation{
		Name:     "content",
		Type:     "array",
		ItemType: "User",
	})

	validator.ValidateParameterDetails("/api/v1/users/{id}/profile", "post", []helpers.ParameterExpectation{
		{Name: "id", In: "path", Required: true},
		{Name: "fullName", In: "query", Required: false},
		{Name: "email", In: "query", Required: false},
		{Name: "age", In: "query", Required: false},
	})

	validator.ValidateResponseSchema("/api/v1/users", "post", helpers.ResponseSchemaExpectation{
		Code:        "201",
		ContentType: "application/json",
		SchemaRef:   "User",
	})
	validator.ValidateResponseSchema("/api/v1/users", "post", helpers.ResponseSchemaExpectation{
		Code:        "400",
		ContentType: "application/json",
		SchemaRef:   "ApiResponse",
	})
	validator.ValidateResponseSchema("/api/v1/users/{id}", "get", helpers.ResponseSchemaExpectation{
		Code:        "404",
		ContentType: "application/json",
		SchemaRef:   "ApiResponse",
	})

	t.Log("All validations passed!")
}

// TestJSONFormat tests JSON output format
func TestJSONFormat(t *testing.T) {
	projectPath := "./fixtures/gin-demo"

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("Gin demo project not found")
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
	if err != nil {
		t.Fatalf("command failed: %v\nstderr: %s", err, stderr.String())
	}

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

	specData, err := os.ReadFile(specFile)
	if err != nil {
		t.Fatalf("failed to read spec file: %v", err)
	}

	var spec map[string]any
	if err := json.Unmarshal(specData, &spec); err != nil {
		t.Fatalf("failed to parse spec JSON: %v", err)
	}

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

// TestDefaultOutput tests output to project directory
func TestDefaultOutput(t *testing.T) {
	projectPath := "./fixtures/gin-demo"

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

	expectedPath := filepath.Join(projectPath, "openapi.yaml")
	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf("spec file not found at expected path: %s", expectedPath)
	}

	_ = os.Remove(expectedPath)

	t.Logf("Spec correctly output to project root: %s", expectedPath)
}
