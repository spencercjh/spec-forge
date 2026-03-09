//go:build e2e

package e2e_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spencercjh/spec-forge/cmd"
)

// TestE2E_Generate_Help tests the generate command help output.
func TestE2E_Generate_Help(t *testing.T) {
	rootCmd := cmd.NewRootCommand()

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"generate", "--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v\nstderr: %s", err, stderr.String())
	}

	output := stdout.String()

	// Verify help contains expected content
	if !strings.Contains(output, "generate") {
		t.Error("Expected 'generate' in help output")
	}

	if !strings.Contains(output, "--output") {
		t.Error("Expected '--output' flag in help")
	}

	if !strings.Contains(output, "--timeout") {
		t.Error("Expected '--timeout' flag in help")
	}

	t.Log("Generate help test passed!")
}

// TestE2E_Generate_Version tests the version command.
func TestE2E_Generate_Version(t *testing.T) {
	rootCmd := cmd.NewRootCommand()

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"--version"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v\nstderr: %s", err, stderr.String())
	}

	output := stdout.String()

	if !strings.Contains(output, "0.1.0") {
		t.Errorf("Expected version '0.1.0' in output, got: %s", output)
	}

	t.Logf("Version output: %s", output)
}

// TestE2E_Generate_InvalidProject tests error handling for non-existent project.
func TestE2E_Generate_InvalidProject(t *testing.T) {
	rootCmd := cmd.NewRootCommand()

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"generate", "/nonexistent/path/12345"})

	err := rootCmd.Execute()

	// Should fail with error
	if err == nil {
		t.Error("Expected error for non-existent project")
	}

	t.Logf("Got expected error: %v", err)
}
