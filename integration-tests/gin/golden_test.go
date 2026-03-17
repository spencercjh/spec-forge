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

// goldenSnapshots defines golden file snapshots for semantic stability testing
var goldenSnapshots = []helpers.GoldenSnapshot{
	{
		Name: "User Schema Structure",
		Path: "components.schemas.User",
		File: "schemas/User.json",
	},
	{
		Name: "CreateUserRequest Schema Structure",
		Path: "components.schemas.CreateUserRequest",
		File: "schemas/CreateUserRequest.json",
	},
	{
		Name: "PageResult Schema with Array Content",
		Path: "components.schemas.PageResult",
		File: "schemas/PageResult.json",
	},
	{
		Name: "GET /api/v1/users Operation",
		Path: "paths./api/v1/users.get",
		File: "paths/api-v1-users-get.json",
	},
	{
		Name: "POST /api/v1/users Operation",
		Path: "paths./api/v1/users.post",
		File: "paths/api-v1-users-post.json",
	},
	{
		Name: "GET /api/v1/users/{id} Operation with Path Param",
		Path: "paths./api/v1/users/{id}.get",
		File: "paths/api-v1-users-id-get.json",
	},
}

// TestGoldenSnapshots tests against golden files
func TestGoldenSnapshots(t *testing.T) {
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

	specFile := helpers.FindSpecFile(t, outputDir, "json")
	specData, err := os.ReadFile(specFile)
	if err != nil {
		t.Fatalf("failed to read spec file: %v", err)
	}

	var spec map[string]any
	if err := json.Unmarshal(specData, &spec); err != nil {
		t.Fatalf("failed to parse spec JSON: %v", err)
	}

	goldenDir := "./fixtures/golden"

	for _, snapshot := range goldenSnapshots {
		t.Run(snapshot.Name, func(t *testing.T) {
			actual := helpers.ExtractFromPath(t, spec, snapshot.Path)
			if actual == nil {
				t.Fatalf("failed to extract value at path: %s", snapshot.Path)
			}

			actualJSON, err := json.MarshalIndent(actual, "", "  ")
			if err != nil {
				t.Fatalf("failed to marshal actual value: %v", err)
			}

			goldenPath := filepath.Join(goldenDir, snapshot.File)
			goldenData, err := os.ReadFile(goldenPath)
			if err != nil {
				if snapshot.Optional {
					t.Skipf("Golden file not found (optional): %s", goldenPath)
				}
				t.Fatalf("failed to read golden file %s: %v", goldenPath, err)
			}

			var golden any
			if err := json.Unmarshal(goldenData, &golden); err != nil {
				t.Fatalf("failed to parse golden file: %v", err)
			}

			goldenJSON, err := json.MarshalIndent(golden, "", "  ")
			if err != nil {
				t.Fatalf("failed to marshal golden value: %v", err)
			}

			if string(actualJSON) != string(goldenJSON) {
				t.Errorf("Golden snapshot mismatch for %s\nPath: %s\n\nExpected (golden):\n%s\n\nActual:\n%s",
					snapshot.Name, snapshot.Path, string(goldenJSON), string(actualJSON))
			}
		})
	}
}

// TestCriticalInvariants tests critical semantic invariants
func TestCriticalInvariants(t *testing.T) {
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

	specFile := helpers.FindSpecFile(t, outputDir, "json")
	validator := helpers.NewSpecValidator(t, specFile)

	t.Run("User Schema Must Have Required ID", func(t *testing.T) {
		validator.ValidateSchemaProperty("User", helpers.SchemaPropertyExpectation{
			Name: "id",
			Type: "integer",
		})
	})

	t.Run("CreateUserRequest Must Have Required Fields", func(t *testing.T) {
		spec := validator.GetSpec()
		components := spec["components"].(map[string]any)
		schemas := components["schemas"].(map[string]any)
		createReq := schemas["CreateUserRequest"].(map[string]any)
		required := createReq["required"].([]any)

		requiredFields := []string{"username", "email", "fullName"}
		for _, field := range requiredFields {
			found := false
			for _, r := range required {
				if r == field {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("CreateUserRequest should require field: %s", field)
			}
		}
	})

	t.Run("PageResult Content Must Be User Array", func(t *testing.T) {
		validator.ValidateSchemaProperty("PageResult", helpers.SchemaPropertyExpectation{
			Name:     "content",
			Type:     "array",
			ItemType: "User",
		})
	})

	t.Run("Path Param ID Must Be Required", func(t *testing.T) {
		validator.ValidateParameterDetails("/api/v1/users/{id}", "get", []helpers.ParameterExpectation{
			{Name: "id", In: "path", Required: true},
		})
	})

	t.Run("GET Users Response Must Be PageResult", func(t *testing.T) {
		validator.ValidateResponseSchema("/api/v1/users", "get", helpers.ResponseSchemaExpectation{
			Code:        "200",
			ContentType: "application/json",
			SchemaRef:   "PageResult",
		})
	})

	t.Run("POST Users Request Must Reference CreateUserRequest", func(t *testing.T) {
		validator.ValidateRequestBodySchema("/api/v1/users", "post", "CreateUserRequest")
	})
}

// TestRegenerateGolden regenerates golden files (run with REGENERATE_GOLDEN=true)
func TestRegenerateGolden(t *testing.T) {
	if os.Getenv("REGENERATE_GOLDEN") != "true" {
		t.Skip("Set REGENERATE_GOLDEN=true to regenerate golden files")
	}

	projectPath := "./fixtures/gin-demo"
	goldenDir := "./fixtures/golden"

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
	specData, err := os.ReadFile(specFile)
	if err != nil {
		t.Fatalf("failed to read spec file: %v", err)
	}

	var spec map[string]any
	if err := json.Unmarshal(specData, &spec); err != nil {
		t.Fatalf("failed to parse spec JSON: %v", err)
	}

	for _, snapshot := range goldenSnapshots {
		actual := helpers.ExtractFromPath(t, spec, snapshot.Path)
		if actual == nil {
			t.Logf("Warning: could not extract %s", snapshot.Path)
			continue
		}

		goldenPath := filepath.Join(goldenDir, snapshot.File)

		dir := filepath.Dir(goldenPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}

		data, err := json.MarshalIndent(actual, "", "  ")
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		if err := os.WriteFile(goldenPath, data, 0o644); err != nil {
			t.Fatalf("failed to write golden file: %v", err)
		}

		t.Logf("Regenerated: %s", goldenPath)
	}
}
