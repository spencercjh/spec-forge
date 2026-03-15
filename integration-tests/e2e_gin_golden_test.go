//go:build e2e

package e2e_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spencercjh/spec-forge/cmd"
)

// GoldenSnapshot represents a golden file snapshot
type GoldenSnapshot struct {
	Name     string
	Path     string // Path in OpenAPI spec (e.g., "paths./api/v1/users.get")
	File     string // Golden file path
	Optional bool   // If true, test won't fail if missing (for unstable features)
}

// Golden snapshots for gin-demo
// These verify critical semantic properties without checking the entire spec
var ginDemoGoldenSnapshots = []GoldenSnapshot{
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

// TestE2E_GinDemo_GoldenSnapshots tests against golden files for semantic stability
func TestE2E_GinDemo_GoldenSnapshots(t *testing.T) {
	projectPath := "./gin-demo"

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

	specFile := FindSpecFile(t, outputDir, "json")
	specData, err := os.ReadFile(specFile)
	if err != nil {
		t.Fatalf("failed to read spec file: %v", err)
	}

	var spec map[string]any
	if err := json.Unmarshal(specData, &spec); err != nil {
		t.Fatalf("failed to parse spec JSON: %v", err)
	}

	goldenDir := "./gin-fixtures/golden"

	for _, snapshot := range ginDemoGoldenSnapshots {
		t.Run(snapshot.Name, func(t *testing.T) {
			// Extract value from spec using path
			actual := extractFromPath(t, spec, snapshot.Path)
			if actual == nil {
				t.Fatalf("failed to extract value at path: %s", snapshot.Path)
			}

			// Format actual value
			actualJSON, err := json.MarshalIndent(actual, "", "  ")
			if err != nil {
				t.Fatalf("failed to marshal actual value: %v", err)
			}

			// Read golden file
			goldenPath := filepath.Join(goldenDir, snapshot.File)
			goldenData, err := os.ReadFile(goldenPath)
			if err != nil {
				if snapshot.Optional {
					t.Skipf("Golden file not found (optional): %s", goldenPath)
				}
				t.Fatalf("failed to read golden file %s: %v", goldenPath, err)
			}

			// Parse golden to normalize
			var golden any
			if err := json.Unmarshal(goldenData, &golden); err != nil {
				t.Fatalf("failed to parse golden file: %v", err)
			}

			goldenJSON, err := json.MarshalIndent(golden, "", "  ")
			if err != nil {
				t.Fatalf("failed to marshal golden value: %v", err)
			}

			// Compare
			if string(actualJSON) != string(goldenJSON) {
				t.Errorf("Golden snapshot mismatch for %s\nPath: %s\n\nExpected (golden):\n%s\n\nActual:\n%s",
					snapshot.Name, snapshot.Path, string(goldenJSON), string(actualJSON))
			}
		})
	}
}

// extractFromPath extracts a value from a nested map using dot-notation path
// Supports OpenAPI-style paths like "paths./api/v1/users.get" or "components.schemas.User"
func extractFromPath(t *testing.T, spec map[string]any, path string) any {
	t.Helper()

	// Handle paths with API paths like "paths./api/v1/users.get"
	// by splitting into known prefixes and then the rest
	var parts []string

	if strings.HasPrefix(path, "paths./") {
		// Format: paths./api/v1/users.get or paths./api/v1/users/{id}.get
		// Find the HTTP method suffix (.get, .post, etc.)
		lastDot := strings.LastIndex(path, ".")
		if lastDot > 0 {
			prefix := path[:lastDot] // paths./api/v1/users
			method := path[lastDot+1:] // get

			// Remove "paths." prefix
			apiPath := strings.TrimPrefix(prefix, "paths.")
			parts = []string{"paths", apiPath, method}
		} else {
			parts = strings.Split(path, ".")
		}
	} else if strings.HasPrefix(path, "components.schemas.") {
		// Format: components.schemas.User
		parts = []string{"components", "schemas", strings.TrimPrefix(path, "components.schemas.")}
	} else {
		parts = strings.Split(path, ".")
	}

	current := any(spec)

	for _, part := range parts {
		if current == nil {
			return nil
		}

		switch v := current.(type) {
		case map[string]any:
			current = v[part]
		default:
			return nil
		}
	}

	return current
}

// TestE2E_GinDemo_CriticalInvariants tests critical semantic invariants that must not change
func TestE2E_GinDemo_CriticalInvariants(t *testing.T) {
	projectPath := "./gin-demo"

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

	specFile := FindSpecFile(t, outputDir, "json")
	validator := NewSpecValidator(t, specFile)

	t.Run("User Schema Must Have Required ID", func(t *testing.T) {
		validator.ValidateSchemaProperty("User", SchemaPropertyExpectation{
			Name: "id",
			Type: "integer",
		})
	})

	t.Run("CreateUserRequest Must Have Required Fields", func(t *testing.T) {
		components := validator.spec["components"].(map[string]any)
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
		validator.ValidateSchemaProperty("PageResult", SchemaPropertyExpectation{
			Name:     "content",
			Type:     "array",
			ItemType: "User",
		})
	})

	t.Run("Path Param ID Must Be Required", func(t *testing.T) {
		validator.ValidateParameterDetails("/api/v1/users/{id}", "get", []ParameterExpectation{
			{Name: "id", In: "path", Required: true},
		})
	})

	t.Run("GET Users Response Must Be PageResult", func(t *testing.T) {
		validator.ValidateResponseSchema("/api/v1/users", "get", ResponseSchemaExpectation{
			Code:        "200",
			ContentType: "application/json",
			SchemaRef:   "PageResult",
		})
	})

	t.Run("POST Users Request Must Reference CreateUserRequest", func(t *testing.T) {
		validator.ValidateRequestBodySchema("/api/v1/users", "post", "CreateUserRequest")
	})
}

// TestE2E_GinDemo_RegenerateGolden tests to regenerate golden files (not run by default)
func TestE2E_GinDemo_RegenerateGolden(t *testing.T) {
	// This test is skipped by default. Run with:
	// go test -v -tags=e2e ./integration-tests/... -run TestE2E_GinDemo_RegenerateGolden -regenerate
	if os.Getenv("REGENERATE_GOLDEN") != "true" {
		t.Skip("Set REGENERATE_GOLDEN=true to regenerate golden files")
	}

	projectPath := "./gin-demo"
	goldenDir := "./gin-fixtures/golden"

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

	specFile := FindSpecFile(t, outputDir, "json")
	specData, err := os.ReadFile(specFile)
	if err != nil {
		t.Fatalf("failed to read spec file: %v", err)
	}

	var spec map[string]any
	if err := json.Unmarshal(specData, &spec); err != nil {
		t.Fatalf("failed to parse spec JSON: %v", err)
	}

	// Regenerate each golden file
	for _, snapshot := range ginDemoGoldenSnapshots {
		actual := extractFromPath(t, spec, snapshot.Path)
		if actual == nil {
			t.Logf("Warning: could not extract %s", snapshot.Path)
			continue
		}

		goldenPath := filepath.Join(goldenDir, snapshot.File)

		// Ensure directory exists
		dir := filepath.Dir(goldenPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}

		// Marshal with indentation
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
