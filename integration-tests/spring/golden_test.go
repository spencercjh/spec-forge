//go:build e2e

package spring

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spencercjh/spec-forge/cmd"
	"github.com/spencercjh/spec-forge/integration-tests/helpers"
)

// skipIfProjectMissing skips the test if the Maven project or wrapper is not found.
func skipIfProjectMissing(t *testing.T, projectPath string) {
	t.Helper()

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skipf("Project not found: %s", projectPath)
	}

	mvnwPath := filepath.Join(projectPath, "mvnw")
	if _, err := os.Stat(mvnwPath); os.IsNotExist(err) {
		t.Skip("Maven wrapper not found, skipping test")
	}
}

// generateSpec generates an OpenAPI spec from the Spring Boot demo project.
func generateSpec(t *testing.T, projectPath, outputDir string) string {
	t.Helper()

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

	return helpers.FindSpecFile(t, outputDir, "json")
}

// goldenSnapshots defines golden file snapshots for semantic stability testing
var goldenSnapshots = []helpers.GoldenSnapshot{
	{
		Name: "User Schema Structure",
		Path: "components.schemas.User",
		File: "schemas/User.json",
	},
	{
		Name: "PageResultUser Schema Structure",
		Path: "components.schemas.PageResultUser",
		File: "schemas/PageResultUser.json",
	},
	{
		Name: "FileUploadResult Schema Structure",
		Path: "components.schemas.FileUploadResult",
		File: "schemas/FileUploadResult.json",
	},
	{
		Name: "ApiResponseUser Schema Structure",
		Path: "components.schemas.ApiResponseUser",
		File: "schemas/ApiResponseUser.json",
	},
	{
		Name: "ApiResponsePageResultUser Schema Structure",
		Path: "components.schemas.ApiResponsePageResultUser",
		File: "schemas/ApiResponsePageResultUser.json",
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
		Name: "GET /api/v1/users/{id} Operation",
		Path: "paths./api/v1/users/{id}.get",
		File: "paths/api-v1-users-id-get.json",
	},
}

// TestGoldenSnapshots tests against golden files
func TestGoldenSnapshots(t *testing.T) {
	projectPath := "../maven-springboot-openapi-demo"

	skipIfProjectMissing(t, projectPath)

	outputDir := t.TempDir()
	specFile := generateSpec(t, projectPath, outputDir)

	specData, err := os.ReadFile(specFile)
	if err != nil {
		t.Fatalf("failed to read spec file: %v", err)
	}

	var spec map[string]any
	if err := json.Unmarshal(specData, &spec); err != nil {
		t.Fatalf("failed to parse spec JSON: %v", err)
	}

	goldenDir := "./fixtures/golden"

	// Validate the full generated OpenAPI spec against the golden openapi.json.
	t.Run("full_spec", func(t *testing.T) {
		goldenPath := filepath.Join(goldenDir, "openapi.json")

		goldenData, err := os.ReadFile(goldenPath)
		if err != nil {
			t.Fatalf("failed to read golden spec file %s: %v", goldenPath, err)
		}

		var goldenSpec any
		if err := json.Unmarshal(goldenData, &goldenSpec); err != nil {
			t.Fatalf("failed to parse golden spec JSON: %v", err)
		}

		actualJSON, err := json.MarshalIndent(spec, "", "  ")
		if err != nil {
			t.Fatalf("failed to marshal actual spec: %v", err)
		}

		goldenJSON, err := json.MarshalIndent(goldenSpec, "", "  ")
		if err != nil {
			t.Fatalf("failed to marshal golden spec: %v", err)
		}

		if !bytes.Equal(actualJSON, goldenJSON) {
			t.Errorf("Full spec golden mismatch\n\nExpected (golden):\n%s\n\nActual:\n%s",
				string(goldenJSON), string(actualJSON))
		}
	})

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

			if !bytes.Equal(actualJSON, goldenJSON) {
				t.Errorf("Golden snapshot mismatch for %s\nPath: %s\n\nExpected (golden):\n%s\n\nActual:\n%s",
					snapshot.Name, snapshot.Path, string(goldenJSON), string(actualJSON))
			}
		})
	}
}

// TestRegenerateGolden regenerates golden files (run with REGENERATE_GOLDEN=true)
func TestRegenerateGolden(t *testing.T) {
	if os.Getenv("REGENERATE_GOLDEN") != "true" {
		t.Skip("Set REGENERATE_GOLDEN=true to regenerate golden files")
	}

	projectPath := "../maven-springboot-openapi-demo"
	goldenDir := "./fixtures/golden"

	skipIfProjectMissing(t, projectPath)

	outputDir := t.TempDir()
	specFile := generateSpec(t, projectPath, outputDir)

	specData, err := os.ReadFile(specFile)
	if err != nil {
		t.Fatalf("failed to read spec file: %v", err)
	}

	var spec map[string]any
	if err := json.Unmarshal(specData, &spec); err != nil {
		t.Fatalf("failed to parse spec JSON: %v", err)
	}

	// Regenerate full spec golden file
	fullGoldenPath := filepath.Join(goldenDir, "openapi.json")
	fullData, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal full spec: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(fullGoldenPath), 0o755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	if err := os.WriteFile(fullGoldenPath, fullData, 0o644); err != nil {
		t.Fatalf("failed to write golden file: %v", err)
	}
	t.Logf("Regenerated: %s", fullGoldenPath)

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
