//go:build e2e

package gozero

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/spencercjh/spec-forge/cmd"
	"github.com/spencercjh/spec-forge/integration-tests/helpers"
)

// goldenSnapshots defines golden file snapshots for semantic stability testing
var goldenSnapshots = []helpers.GoldenSnapshot{
	{
		Name: "Full OpenAPI Spec",
		Path: "",
		File: "openapi.json",
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

// skipIfGoctlNotAvailable skips the test if goctl is not found in PATH.
func skipIfGoctlNotAvailable(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("goctl"); err != nil {
		t.Skip("goctl not found, skipping test")
	}
}

// generateSpec generates an OpenAPI spec from the gozero-demo project.
// Returns the spec file path and parsed spec map.
func generateSpec(t *testing.T, format string) (string, map[string]any) {
	t.Helper()

	projectPath := "../gozero-demo"

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("gozero-demo project not found")
	}

	skipIfGoctlNotAvailable(t)

	outputDir := t.TempDir()

	rootCmd := cmd.NewRootCommand()
	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{
		"generate",
		projectPath,
		"--output-dir", outputDir,
		"--output", format,
		"--skip-enrich",
		"--skip-publish",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v\nstderr: %s", err, stderr.String())
	}

	specFile := helpers.FindSpecFile(t, outputDir, format)
	specData, err := os.ReadFile(specFile)
	if err != nil {
		t.Fatalf("failed to read spec file: %v", err)
	}

	var spec map[string]any
	if err := json.Unmarshal(specData, &spec); err != nil {
		t.Fatalf("failed to parse spec JSON: %v", err)
	}

	return specFile, spec
}

// volatileFields are top-level fields in goctl output that change on every run
// and must be stripped before golden comparison.
var volatileFields = []string{"x-date"}

// stripVolatileFields removes runtime-dependent fields (e.g., x-date) from a
// spec map so that golden comparisons are deterministic.
func stripVolatileFields(spec map[string]any) map[string]any {
	clean := make(map[string]any, len(spec))
	for k, v := range spec {
		skip := false
		for _, vf := range volatileFields {
			if k == vf {
				skip = true
				break
			}
		}
		if !skip {
			clean[k] = v
		}
	}
	return clean
}

// TestGoldenSnapshots tests against golden files
func TestGoldenSnapshots(t *testing.T) {
	_, spec := generateSpec(t, "json")

	goldenDir := "./fixtures/golden"

	for _, snapshot := range goldenSnapshots {
		t.Run(snapshot.Name, func(t *testing.T) {
			var actual any
			if snapshot.Path == "" {
				// Full spec comparison — strip volatile fields
				actual = stripVolatileFields(spec)
			} else {
				actual = helpers.ExtractFromPath(t, spec, snapshot.Path)
			}
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

// TestCriticalInvariants tests critical semantic invariants for go-zero specs
func TestCriticalInvariants(t *testing.T) {
	specFile, spec := generateSpec(t, "json")

	validator := helpers.NewSpecValidator(t, specFile)

	t.Run("OpenAPI Version Must Be 3.0.x", func(t *testing.T) {
		validator.ValidateOpenAPIVersion()
	})

	t.Run("Info Must Have Title and Version", func(t *testing.T) {
		validator.ValidateInfo()
	})

	t.Run("All Expected Paths Must Exist", func(t *testing.T) {
		validator.ValidatePaths([]string{
			"/api/v1/users",
			"/api/v1/users/{id}",
			"/api/v1/users/upload",
			"/api/v1/users/{id}/profile",
		})
	})

	t.Run("GET Users Must Have Pagination Parameters", func(t *testing.T) {
		getUsers := helpers.ExtractFromPath(t, spec, "paths./api/v1/users.get")
		if getUsers == nil {
			t.Fatal("GET /api/v1/users not found")
		}

		op, ok := getUsers.(map[string]any)
		if !ok {
			t.Fatal("GET /api/v1/users is not an object")
		}

		params, ok := op["parameters"].([]any)
		if !ok {
			t.Fatal("GET /api/v1/users has no parameters")
		}

		paramNames := make(map[string]bool)
		for _, p := range params {
			pm, ok := p.(map[string]any)
			if !ok {
				continue
			}
			if name, ok := pm["name"].(string); ok {
				paramNames[name] = true
			}
		}

		for _, expected := range []string{"page", "size"} {
			if !paramNames[expected] {
				t.Errorf("GET /api/v1/users missing expected parameter: %s", expected)
			}
		}
	})

	t.Run("Path Param ID Must Be Required", func(t *testing.T) {
		getUserByID := helpers.ExtractFromPath(t, spec, "paths./api/v1/users/{id}.get")
		if getUserByID == nil {
			t.Fatal("GET /api/v1/users/{id} not found")
		}

		op, ok := getUserByID.(map[string]any)
		if !ok {
			t.Fatal("GET /api/v1/users/{id} is not an object")
		}

		params, ok := op["parameters"].([]any)
		if !ok {
			t.Fatal("GET /api/v1/users/{id} has no parameters")
		}

		found := false
		for _, p := range params {
			pm, ok := p.(map[string]any)
			if !ok {
				continue
			}
			if pm["name"] == "id" && pm["in"] == "path" {
				found = true
				if required, ok := pm["required"].(bool); !ok || !required {
					t.Error("path parameter 'id' must be required")
				}
			}
		}
		if !found {
			t.Error("path parameter 'id' not found in GET /api/v1/users/{id}")
		}
	})

	t.Run("POST Users Must Have Request Body", func(t *testing.T) {
		postUsers := helpers.ExtractFromPath(t, spec, "paths./api/v1/users.post")
		if postUsers == nil {
			t.Fatal("POST /api/v1/users not found")
		}

		op, ok := postUsers.(map[string]any)
		if !ok {
			t.Fatal("POST /api/v1/users is not an object")
		}

		if _, ok := op["requestBody"]; !ok {
			t.Error("POST /api/v1/users must have a requestBody")
		}
	})

	t.Run("Operations Must Have OperationIDs", func(t *testing.T) {
		operations := []struct {
			path   string
			method string
		}{
			{"paths./api/v1/users.get", "GET /api/v1/users"},
			{"paths./api/v1/users.post", "POST /api/v1/users"},
			{"paths./api/v1/users/{id}.get", "GET /api/v1/users/{id}"},
		}

		for _, op := range operations {
			val := helpers.ExtractFromPath(t, spec, op.path)
			if val == nil {
				t.Errorf("%s not found", op.method)
				continue
			}
			m, ok := val.(map[string]any)
			if !ok {
				t.Errorf("%s is not an object", op.method)
				continue
			}
			if _, ok := m["operationId"]; !ok {
				t.Errorf("%s must have an operationId", op.method)
			}
		}
	})

	t.Run("Response Content Type Must Be JSON", func(t *testing.T) {
		getUserByID := helpers.ExtractFromPath(t, spec, "paths./api/v1/users/{id}.get")
		if getUserByID == nil {
			t.Fatal("GET /api/v1/users/{id} not found")
		}

		op, ok := getUserByID.(map[string]any)
		if !ok {
			t.Fatal("GET /api/v1/users/{id} is not an object")
		}

		responses, ok := op["responses"].(map[string]any)
		if !ok {
			t.Fatal("no responses found")
		}

		resp200, ok := responses["200"].(map[string]any)
		if !ok {
			t.Fatal("no 200 response found")
		}

		content, ok := resp200["content"].(map[string]any)
		if !ok {
			t.Fatal("no content found in 200 response")
		}

		if _, ok := content["application/json"]; !ok {
			t.Error("200 response must have application/json content type")
		}
	})
}

// TestRegenerateGolden regenerates golden files (run with REGENERATE_GOLDEN=true)
func TestRegenerateGolden(t *testing.T) {
	if os.Getenv("REGENERATE_GOLDEN") != "true" {
		t.Skip("Set REGENERATE_GOLDEN=true to regenerate golden files")
	}

	_, spec := generateSpec(t, "json")

	goldenDir := "./fixtures/golden"

	// Write golden snapshots (includes full spec and per-operation files)
	for _, snapshot := range goldenSnapshots {
		var actual any
		if snapshot.Path == "" {
			// Strip volatile fields so golden file is deterministic
			actual = stripVolatileFields(spec)
		} else {
			actual = helpers.ExtractFromPath(t, spec, snapshot.Path)
		}
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
