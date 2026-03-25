//go:build e2e

package e2e_test

import (
	"testing"

	forgeerrors "github.com/spencercjh/spec-forge/internal/errors"
	"github.com/spencercjh/spec-forge/internal/executor"
)

func TestExecutorErrorClassification(t *testing.T) {
	exec := executor.NewExecutor()

	t.Run("command not found returns SYSTEM error", func(t *testing.T) {
		_, err := exec.Execute(t.Context(), &executor.ExecuteOptions{
			Command: "nonexistent_command_12345",
		})

		if err == nil {
			t.Fatal("expected error for nonexistent command")
		}

		if !forgeerrors.IsCode(err, forgeerrors.CodeSystem) {
			t.Errorf("command not found should be SYSTEM error, got code: %s", forgeerrors.GetCode(err))
		}
	})

	t.Run("command failed returns SYSTEM error", func(t *testing.T) {
		_, err := exec.Execute(t.Context(), &executor.ExecuteOptions{
			Command: "ls",
			Args:    []string{"/nonexistent/path/that/does/not/exist"},
		})

		if err == nil {
			t.Fatal("expected error for failed command")
		}

		if !forgeerrors.IsCode(err, forgeerrors.CodeSystem) {
			t.Errorf("command failed should be SYSTEM error, got code: %s", forgeerrors.GetCode(err))
		}
	})
}

func TestErrorRetryable(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name:      "LLM error is retryable",
			err:       forgeerrors.LLMError("rate limited", nil),
			retryable: true,
		},
		{
			name:      "Publish error is retryable",
			err:       forgeerrors.PublishError("network error", nil),
			retryable: true,
		},
		{
			name:      "System error is retryable",
			err:       forgeerrors.SystemError("timeout", nil),
			retryable: true,
		},
		{
			name:      "Config error is not retryable",
			err:       forgeerrors.ConfigError("bad config", nil),
			retryable: false,
		},
		{
			name:      "Detect error is not retryable",
			err:       forgeerrors.DetectError("not found", nil),
			retryable: false,
		},
		{
			name:      "Patch error is not retryable",
			err:       forgeerrors.PatchError("patch failed", nil),
			retryable: false,
		},
		{
			name:      "Generate error is not retryable",
			err:       forgeerrors.GenerateError("build failed", nil),
			retryable: false,
		},
		{
			name:      "Validate error is not retryable",
			err:       forgeerrors.ValidateError("invalid spec", nil),
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := forgeerrors.IsRetryable(tt.err); got != tt.retryable {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.retryable)
			}
		})
	}
}
