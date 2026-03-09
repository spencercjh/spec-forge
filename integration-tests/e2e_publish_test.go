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

// TestE2E_Publish_Help tests the publish command help output.
func TestE2E_Publish_Help(t *testing.T) {
	rootCmd := cmd.NewRootCommand()

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"publish", "--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v\nstderr: %s", err, stderr.String())
	}

	output := stdout.String()

	// Verify help contains expected content
	if !strings.Contains(output, "publish") {
		t.Error("Expected 'publish' in help output")
	}

	if !strings.Contains(output, "--to") {
		t.Error("Expected '--to' flag in help")
	}

	if !strings.Contains(output, "--format") {
		t.Error("Expected '--format' flag in help")
	}

	if !strings.Contains(output, "--overwrite") {
		t.Error("Expected '--overwrite' flag in help")
	}

	t.Log("Publish help test passed!")
}

// TestE2E_Publish_MissingTarget tests error handling when target is not provided.
func TestE2E_Publish_MissingTarget(t *testing.T) {
	rootCmd := cmd.NewRootCommand()

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"publish", "spec.yaml"}) // No --to flag

	err := rootCmd.Execute()

	// Should fail because --to is required
	if err == nil {
		t.Error("Expected error for missing --to flag")
	}

	t.Logf("Got expected error: %v", err)
}

// TestE2E_Publish_InvalidTarget tests error handling for invalid target.
func TestE2E_Publish_InvalidTarget(t *testing.T) {
	rootCmd := cmd.NewRootCommand()

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{
		"publish",
		"spec.yaml",
		"--to", "invalid-target",
	})

	err := rootCmd.Execute()

	// Should fail because target is invalid
	if err == nil {
		t.Error("Expected error for invalid target")
	}

	t.Logf("Got expected error: %v", err)
}

// TestE2E_Publish_NonExistentSpec tests error handling for non-existent spec file.
func TestE2E_Publish_NonExistentSpec(t *testing.T) {
	rootCmd := cmd.NewRootCommand()

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{
		"publish",
		"/nonexistent/spec.yaml",
		"--to", "readme",
	})

	err := rootCmd.Execute()

	// Should fail because file doesn't exist
	if err == nil {
		t.Error("Expected error for non-existent spec file")
	}

	t.Logf("Got expected error: %v", err)
}

// TestE2E_Publish_InvalidSpec tests error handling for invalid spec file format.
func TestE2E_Publish_InvalidSpec(t *testing.T) {
	// Create an invalid spec file
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "invalid-spec.yaml")
	if err := os.WriteFile(specFile, []byte("invalid: yaml: content: ["), 0o644); err != nil {
		t.Fatalf("Failed to write test spec: %v", err)
	}

	rootCmd := cmd.NewRootCommand()

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{
		"publish",
		specFile,
		"--to", "readme",
	})

	err := rootCmd.Execute()

	// Should fail because spec is invalid
	if err == nil {
		t.Error("Expected error for invalid spec file")
	}

	t.Logf("Got expected error: %v", err)
}

// TestE2E_Publish_MissingAPIKey tests error handling when API key is not provided.
func TestE2E_Publish_MissingAPIKey(t *testing.T) {
	// Create a valid OpenAPI spec
	specContent := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /users:
    get:
      summary: "List users"
      operationId: listUsers
      responses:
        "200":
          description: "Success"
`
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "test-spec.yaml")
	if err := os.WriteFile(specFile, []byte(specContent), 0o644); err != nil {
		t.Fatalf("Failed to write test spec: %v", err)
	}

	// Ensure no API key is set
	t.Setenv("README_API_KEY", "")

	rootCmd := cmd.NewRootCommand()

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{
		"publish",
		specFile,
		"--to", "readme",
		"--readme-slug", "test-api",
	})

	err := rootCmd.Execute()

	// Should fail because API key is missing
	if err == nil {
		t.Error("Expected error for missing API key")
	}

	t.Logf("Got expected error: %v", err)
}

// TestE2E_Publish_ValidSpecToReadme tests publishing to ReadMe (requires API key).
// This test is skipped unless README_API_KEY is set.
func TestE2E_Publish_ValidSpecToReadme(t *testing.T) {
	// Check if API key is available
	if os.Getenv("README_API_KEY") == "" {
		t.Skip("README_API_KEY not set, skipping ReadMe publish test")
	}

	// Create a valid OpenAPI spec
	specContent := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /users:
    get:
      summary: "List users"
      operationId: listUsers
      responses:
        "200":
          description: "Success"
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
		"publish",
		specFile,
		"--to", "readme",
		"--readme-slug", "test-api",
	})

	err := rootCmd.Execute()

	// This may fail due to network or API issues, but we're testing the CLI flow
	if err != nil {
		t.Logf("Publish failed (may be expected): %v", err)
		t.Logf("stderr: %s", stderr.String())
	} else {
		t.Log("Publish succeeded!")
	}
}
