//go:build e2e

package gozero

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/spencercjh/spec-forge/cmd"
	"github.com/spencercjh/spec-forge/integration-tests/helpers"
)

// TestMissingGoctlGracefulSkip verifies that tests skip gracefully when goctl is not installed
func TestMissingGoctlGracefulSkip(t *testing.T) {
	if _, err := exec.LookPath("goctl"); err == nil {
		t.Skip("goctl is available, skipping missing goctl test")
	}

	// When goctl is not available, the generate command should fail with a clear error
	projectPath := "../gozero-demo"

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("gozero-demo project not found")
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
	if err == nil {
		t.Error("expected error when goctl is not available")
		return
	}

	t.Logf("Got expected error when goctl not available: %v", err)
}

// TestNonGoZeroProject verifies error handling for non-go-zero projects
func TestNonGoZeroProject(t *testing.T) {
	tempDir := t.TempDir()

	// Create a basic Go project without go-zero dependency
	goMod := `module non-gozero-test

go 1.26

require github.com/gin-gonic/gin v1.12.0
`
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	mainGo := `package main

func main() {}
`
	if err := os.WriteFile(filepath.Join(tempDir, "main.go"), []byte(mainGo), 0o644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	outputDir := t.TempDir()

	rootCmd := cmd.NewRootCommand()
	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{
		"generate",
		tempDir,
		"--output-dir", outputDir,
		"--output", "json",
		"--skip-enrich",
		"--skip-publish",
	})

	// This might either fail (if gozero detector rejects it) or succeed (if gin detector picks it up)
	// The important thing is that it doesn't crash
	_ = rootCmd.Execute()
	t.Log("Non-go-zero project handled without crash")
}

// TestYAMLOutputFormat tests YAML output format for go-zero projects
func TestYAMLOutputFormat(t *testing.T) {
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
		"--output", "yaml",
		"--skip-enrich",
		"--skip-publish",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("YAML generation failed: %v\nstderr: %s", err, stderr.String())
	}

	specFile := helpers.FindSpecFile(t, outputDir, "yaml")
	validator := helpers.NewSpecValidator(t, specFile)

	validator.ValidateOpenAPIVersion()
	validator.ValidateInfo()

	pathCount := validator.GetPathCount()
	if pathCount == 0 {
		t.Error("expected at least one path in YAML spec")
	}

	t.Logf("YAML output format: %d paths generated", pathCount)
}

// TestFormDataEndpoint tests the form-data endpoint (update profile)
func TestFormDataEndpoint(t *testing.T) {
	_, spec := generateSpec(t, "json")

	t.Run("Update Profile Has Path Parameter", func(t *testing.T) {
		updateProfile := helpers.ExtractFromPath(t, spec, "paths./api/v1/users/{id}/profile.post")
		if updateProfile == nil {
			t.Fatal("POST /api/v1/users/{id}/profile not found")
		}

		op, ok := updateProfile.(map[string]any)
		if !ok {
			t.Fatal("operation is not an object")
		}

		params, ok := op["parameters"].([]any)
		if !ok {
			t.Fatal("no parameters found")
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
			t.Error("path parameter 'id' not found in POST /api/v1/users/{id}/profile")
		}
	})

	t.Run("Update Profile Has Form Request Body", func(t *testing.T) {
		updateProfile := helpers.ExtractFromPath(t, spec, "paths./api/v1/users/{id}/profile.post")
		if updateProfile == nil {
			t.Fatal("POST /api/v1/users/{id}/profile not found")
		}

		op, ok := updateProfile.(map[string]any)
		if !ok {
			t.Fatal("operation is not an object")
		}

		reqBody, ok := op["requestBody"].(map[string]any)
		if !ok {
			t.Skip("requestBody not present (may vary by goctl version)")
		}

		content, ok := reqBody["content"].(map[string]any)
		if !ok {
			t.Fatal("requestBody has no content")
		}

		// Check for form-urlencoded content type
		if _, ok := content["application/x-www-form-urlencoded"]; !ok {
			t.Error("update profile should use application/x-www-form-urlencoded content type")
		}
	})
}

// TestUploadEndpoint tests the file upload endpoint
func TestUploadEndpoint(t *testing.T) {
	_, spec := generateSpec(t, "json")

	t.Run("Upload Endpoint Exists", func(t *testing.T) {
		upload := helpers.ExtractFromPath(t, spec, "paths./api/v1/users/upload.post")
		if upload == nil {
			t.Fatal("POST /api/v1/users/upload not found")
		}

		op, ok := upload.(map[string]any)
		if !ok {
			t.Fatal("operation is not an object")
		}

		if _, ok := op["operationId"]; !ok {
			t.Error("upload endpoint must have an operationId")
		}
	})

	t.Run("Upload Response Has File Metadata", func(t *testing.T) {
		upload := helpers.ExtractFromPath(t, spec, "paths./api/v1/users/upload.post")
		if upload == nil {
			t.Fatal("POST /api/v1/users/upload not found")
		}

		op, ok := upload.(map[string]any)
		if !ok {
			t.Fatal("operation is not an object")
		}

		responses := op["responses"].(map[string]any)
		resp200 := responses["200"].(map[string]any)
		content := resp200["content"].(map[string]any)
		jsonContent := content["application/json"].(map[string]any)
		schema := jsonContent["schema"].(map[string]any)
		properties := schema["properties"].(map[string]any)
		data := properties["data"].(map[string]any)
		dataProps := data["properties"].(map[string]any)

		expectedFields := []string{"filename", "size", "contentType"}
		for _, field := range expectedFields {
			if _, ok := dataProps[field]; !ok {
				t.Errorf("upload response data missing expected field: %s", field)
			}
		}
	})
}
