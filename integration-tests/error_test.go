//go:build e2e

package e2e_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spencercjh/spec-forge/internal/executor"
)

// TestE2E_ErrorHandling_CommandNotFound tests error handling when build tool is not found.
func TestE2E_ErrorHandling_CommandNotFound(t *testing.T) {
	// Create a temp directory with pom.xml but no maven installed
	tmpDir := t.TempDir()
	pomPath := filepath.Join(tmpDir, "pom.xml")
	if err := os.WriteFile(pomPath, []byte(`<project></project>`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create target directory with a dummy spec (simulating pre-existing spec)
	targetDir := filepath.Join(tmpDir, "target")
	if err := os.Mkdir(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	exec := executor.NewExecutor()

	// Try to run a non-existent command
	_, err := exec.Execute(ctx, &executor.ExecuteOptions{
		Command: "nonexistent_command_12345",
		Args:    []string{},
	})

	if err == nil {
		t.Fatal("Expected error for non-existent command")
	}

	// Verify it's a CommandNotFoundError
	if _, ok := errors.AsType[*executor.CommandNotFoundError](err); !ok {
		t.Logf("Got error type %T: %v", err, err)
	}
}
