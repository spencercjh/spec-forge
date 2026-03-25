package executor

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	forgeerrors "github.com/spencercjh/spec-forge/internal/errors"
)

func TestExecutor_Execute_Success(t *testing.T) {
	executor := NewExecutor()

	tests := []struct {
		name    string
		command string
		args    []string
	}{
		{"echo", "echo", []string{"hello"}},
		{"ls", "ls", []string{"-la"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executor.Execute(t.Context(), &ExecuteOptions{
				Command: tt.command,
				Args:    tt.args,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.ExitCode != 0 {
				t.Errorf("expected exit code 0, got %d", result.ExitCode)
			}
		})
	}
}

func TestExecutor_Execute_CommandNotFound(t *testing.T) {
	executor := NewExecutor()

	result, err := executor.Execute(t.Context(), &ExecuteOptions{
		Command: "nonexistent-command-12345",
	})

	if err == nil {
		t.Fatal("expected error for nonexistent command")
	}

	if _, ok := errors.AsType[*CommandNotFoundError](err); !ok {
		t.Errorf("expected CommandNotFoundError, got %T", err)
	}

	if result == nil {
		t.Error("result should not be nil even on error")
	}
}

func TestExecutor_Execute_Timeout(t *testing.T) {
	executor := NewExecutor()

	// Use sleep command with very short timeout
	_, err := executor.Execute(t.Context(), &ExecuteOptions{
		Command: "sleep",
		Args:    []string{"10"},
		Timeout: 100 * time.Millisecond,
	})

	if err == nil {
		t.Fatal("expected timeout error")
	}

	// Timeout errors should be classified as SYSTEM
	if !forgeerrors.IsCode(err, forgeerrors.CodeSystem) {
		t.Errorf("expected timeout error to have SYSTEM code, got: %v (code=%q)", err, forgeerrors.GetCode(err))
	}
	if !forgeerrors.IsRetryable(err) {
		t.Error("timeout error should be retryable")
	}

	// Timeout errors should have a recovery hint
	if fe, ok := errors.AsType[*forgeerrors.Error](err); ok && fe.Hint() == "" {
		t.Error("timeout error should have a non-empty recovery hint")
	}
}

func TestExecutor_Execute_Canceled(t *testing.T) {
	executor := NewExecutor()

	ctx, cancel := context.WithCancel(t.Context())
	cancel() // cancel immediately

	_, err := executor.Execute(ctx, &ExecuteOptions{
		Command: "sleep",
		Args:    []string{"10"},
	})

	if err == nil {
		t.Fatal("expected cancellation error")
	}

	// Cancellation errors should be classified as SYSTEM
	if !forgeerrors.IsCode(err, forgeerrors.CodeSystem) {
		t.Errorf("expected cancellation error to have SYSTEM code, got: %v (code=%q)", err, forgeerrors.GetCode(err))
	}
	if !forgeerrors.IsRetryable(err) {
		t.Error("cancellation error should be retryable")
	}
}

func TestExecutor_Execute_WorkingDir(t *testing.T) {
	executor := NewExecutor()

	result, err := executor.Execute(context.Background(), &ExecuteOptions{
		Command:    "pwd",
		WorkingDir: "/tmp",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}

	// Output should contain /tmp
	if result.Stdout == "" {
		t.Error("expected stdout to contain working directory")
	}
}

func TestExecutor_Execute_FailedCommand(t *testing.T) {
	executor := NewExecutor()

	result, err := executor.Execute(context.Background(), &ExecuteOptions{
		Command: "ls",
		Args:    []string{"/nonexistent-directory-12345"},
	})

	if err == nil {
		t.Fatal("expected error for failed command")
	}

	if _, ok := errors.AsType[*CommandFailedError](err); !ok {
		t.Errorf("expected CommandFailedError, got %T", err)
	}

	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code for failed command")
	}
}

func TestExecutor_Execute_NilOptions(t *testing.T) {
	executor := NewExecutor()

	_, err := executor.Execute(context.Background(), nil)

	if err == nil {
		t.Fatal("expected error for nil options")
	}
}

func TestExecutor_Execute_DefaultTimeout(t *testing.T) {
	executor := NewExecutor()

	// Test with zero timeout - should use default
	result, err := executor.Execute(context.Background(), &ExecuteOptions{
		Command: "echo",
		Args:    []string{"test"},
		Timeout: 0, // Should default to 5 minutes
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
}

func TestExecutor_Execute_EnvReplaceMode(t *testing.T) {
	executor := NewExecutor()

	t.Run("no custom env inherits parent", func(t *testing.T) {
		// Use env to verify parent PATH is inherited without assuming it's non-empty
		result, err := executor.Execute(context.Background(), &ExecuteOptions{
			Command: "env",
			Args:    []string{},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result.Stdout, "PATH=") {
			t.Error("expected PATH to exist and be inherited from parent environment")
		}
	})

	t.Run("EnvReplaceMode=false (default) appends to parent", func(t *testing.T) {
		// Default behavior: custom vars are appended to parent environment
		result, err := executor.Execute(context.Background(), &ExecuteOptions{
			Command:        "env",
			Args:           []string{},
			Env:            []string{"CUSTOM_VAR=test_value"},
			EnvReplaceMode: false, // default - append to parent
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Verify custom var is set
		if !strings.Contains(result.Stdout, "CUSTOM_VAR=test_value") {
			t.Errorf("expected CUSTOM_VAR to be set, got: %s", result.Stdout)
		}
		// Verify PATH still exists (parent env inherited)
		if !strings.Contains(result.Stdout, "PATH=") {
			t.Errorf("expected PATH to exist (parent env inherited), got: %s", result.Stdout)
		}
	})

	t.Run("EnvReplaceMode=true replaces parent", func(t *testing.T) {
		// In replace mode, only the provided env vars should exist
		result, err := executor.Execute(context.Background(), &ExecuteOptions{
			Command:        "env",
			Args:           []string{},
			Env:            []string{"CUSTOM_VAR=isolated"},
			EnvReplaceMode: true, // explicit replace - isolate from parent
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Verify custom var is set (unconditionally assert)
		if !strings.Contains(result.Stdout, "CUSTOM_VAR=isolated") {
			t.Errorf("expected CUSTOM_VAR to be set, got: %s", result.Stdout)
		}
		// In replace mode, PATH should NOT exist
		if strings.Contains(result.Stdout, "PATH=") {
			t.Errorf("expected PATH to NOT exist (parent env replaced), got: %s", result.Stdout)
		}
	})
}
