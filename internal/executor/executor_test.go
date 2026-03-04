package executor

import (
	"context"
	"errors"
	"testing"
	"time"
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
			result, err := executor.Execute(context.Background(), &ExecuteOptions{
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

	result, err := executor.Execute(context.Background(), &ExecuteOptions{
		Command: "nonexistent-command-12345",
	})

	if err == nil {
		t.Fatal("expected error for nonexistent command")
	}

	var notFoundErr *CommandNotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Errorf("expected CommandNotFoundError, got %T", err)
	}

	if result == nil {
		t.Error("result should not be nil even on error")
	}
}

func TestExecutor_Execute_Timeout(t *testing.T) {
	executor := NewExecutor()

	// Use sleep command with very short timeout
	_, err := executor.Execute(context.Background(), &ExecuteOptions{
		Command: "sleep",
		Args:    []string{"10"},
		Timeout: 100 * time.Millisecond,
	})

	if err == nil {
		t.Fatal("expected timeout error")
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

	var failedErr *CommandFailedError
	if !errors.As(err, &failedErr) {
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
