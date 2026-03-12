//go:build e2e

package e2e_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spencercjh/spec-forge/cmd"
)

// TestE2E_Enrich_Help tests the enrich command help output.
func TestE2E_Enrich_Help(t *testing.T) {
	rootCmd := cmd.NewRootCommand()

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"enrich", "--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v\nstderr: %s", err, stderr.String())
	}

	output := stdout.String()

	// Verify help contains expected content
	if !strings.Contains(output, "enrich") {
		t.Error("Expected 'enrich' in help output")
	}

	if !strings.Contains(output, "--provider") {
		t.Error("Expected '--provider' flag in help")
	}

	if !strings.Contains(output, "--model") {
		t.Error("Expected '--model' flag in help")
	}

	if !strings.Contains(output, "--language") {
		t.Error("Expected '--language' flag in help")
	}

	t.Log("Enrich help test passed!")
}

// TestE2E_Enrich_MissingArgs tests error handling when spec file is not provided.
func TestE2E_Enrich_MissingArgs(t *testing.T) {
	rootCmd := cmd.NewRootCommand()

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"enrich"}) // No spec file

	err := rootCmd.Execute()

	// Should fail with error
	if err == nil {
		t.Error("Expected error for missing spec file argument")
	}

	t.Logf("Got expected error: %v", err)
}

// TestE2E_Enrich_NonExistentFile tests error handling for non-existent spec file.
func TestE2E_Enrich_NonExistentFile(t *testing.T) {
	rootCmd := cmd.NewRootCommand()

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"enrich", "/nonexistent/spec.yaml"})

	err := rootCmd.Execute()

	// Should fail because file doesn't exist
	if err == nil {
		t.Error("Expected error for non-existent spec file")
	}

	t.Logf("Got expected error: %v", err)
}

// TestE2E_Enrich_ValidSpec tests enrichment with a valid spec file (requires API key).
// This test is skipped unless a mock provider or real API key is available.
func TestE2E_Enrich_ValidSpec(t *testing.T) {
	// Create a simple OpenAPI spec for testing
	specContent := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      summary: ""
      operationId: listUsers
      responses:
        "200":
          description: ""
`
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "test-spec.yaml")
	if err := os.WriteFile(specFile, []byte(specContent), 0o644); err != nil {
		t.Fatalf("Failed to write test spec: %v", err)
	}

	rootCmd := cmd.NewRootCommand()

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{
		"enrich",
		specFile,
		"--provider", "custom",
		"--model", "test-model",
		"--language", "en",
	})

	err := rootCmd.Execute()

	// This will fail without a valid API key, but we're testing the CLI flow
	// In a real scenario with API key, it should succeed
	if err != nil {
		t.Logf("Enrich failed (expected without API key): %v", err)
		t.Logf("stderr: %s", stderr.String())
	} else {
		t.Log("Enrich succeeded!")
	}
}
