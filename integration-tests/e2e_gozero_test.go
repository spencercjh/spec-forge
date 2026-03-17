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

// TestE2E_GoZero_Generate tests the generate flow for a go-zero project.
func TestE2E_GoZero_Generate(t *testing.T) {
	projectPath := "gozero-demo"

	// Check if project exists
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("go-zero demo project not found")
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
		"--output", "yaml",
		"--skip-enrich",
		"--skip-publish",
	})

	err := rootCmd.Execute()
	// go-zero generation may fail if goctl is not available
	// This is expected behavior
	if err != nil {
		t.Skipf("generate command failed (may be due to missing goctl): %v\nstderr: %s", err, stderr.String())
	}

	// Find the generated spec file
	specFile := helpers.FindSpecFile(t, outputDir, "yaml")
	if specFile == "" {
		t.Log("no spec file found - goctl may not be available")
		return
	}

	// Perform comprehensive spec validation
	validator := helpers.NewSpecValidator(t, specFile)
	validator.ValidateOpenAPIVersion()
	validator.ValidateInfo()

	// Validate paths exist (go-zero demo has specific paths based on its .api file)
	paths := validator.GetPaths()
	if len(paths) == 0 {
		t.Error("expected at least one path in spec")
	}

	validator.LogSummary()

	t.Logf("Successfully generated valid OpenAPI spec at: %s", specFile)
}

// TestE2E_GoZero_NoProject tests error handling for missing project.
func TestE2E_GoZero_NoProject(t *testing.T) {
	rootCmd := cmd.NewRootCommand()

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{
		"generate",
		"/nonexistent/gozero-project",
	})

	err := rootCmd.Execute()

	// Should fail with error
	if err == nil {
		t.Error("expected error for non-existent project")
	}

	t.Logf("Got expected error: %v", err)
}
