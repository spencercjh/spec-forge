package errors_test

import (
	"errors"
	"testing"

	forgeerrors "github.com/spencercjh/spec-forge/internal/errors"
	"github.com/spencercjh/spec-forge/internal/executor"
)

func TestBackwardCompat_ExecutorErrors(t *testing.T) {
	t.Run("CommandNotFoundError - errors.As works", func(t *testing.T) {
		err := &executor.CommandNotFoundError{Command: "test-command"}

		// Old-style assertion should still work
		var cmdErr *executor.CommandNotFoundError
		if !errors.As(err, &cmdErr) {
			t.Error("errors.As for CommandNotFoundError should still work")
		}
		if cmdErr.Command != "test-command" {
			t.Errorf("expected command 'test-command', got %q", cmdErr.Command)
		}
	})

	t.Run("CommandFailedError - errors.As works", func(t *testing.T) {
		err := &executor.CommandFailedError{
			Command:  "test",
			ExitCode: 1,
		}

		var cmdErr *executor.CommandFailedError
		if !errors.As(err, &cmdErr) {
			t.Error("errors.As for CommandFailedError should still work")
		}
	})
}

func TestBackwardCompat_WrappedErrors(t *testing.T) {
	t.Run("wrapped classified error", func(t *testing.T) {
		inner := forgeerrors.SystemError("inner error", nil)
		wrapped := forgeerrors.DetectError("outer error", inner)

		// Should find SYSTEM code inside
		if !forgeerrors.IsCode(wrapped, forgeerrors.CodeSystem) {
			t.Error("IsCode should find SYSTEM code nested inside DETECT error")
		}

		// Should also find DETECT code in outer error
		if !forgeerrors.IsCode(wrapped, forgeerrors.CodeDetect) {
			t.Error("IsCode should find DETECT code in outer error")
		}
	})

	t.Run("wrapped with custom type", func(t *testing.T) {
		inner := forgeerrors.LLMError("rate limited", nil)
		wrapped := wrappedError{err: inner}

		// errors.As should work through wrapping
		var fe *forgeerrors.Error
		if !errors.As(wrapped, &fe) {
			t.Error("errors.As should find *Error through custom wrapper")
		}
		if fe.Code != forgeerrors.CodeLLM {
			t.Errorf("expected code %q, got %q", forgeerrors.CodeLLM, fe.Code)
		}
	})
}

// wrappedError is a custom error type for testing error chain traversal.
type wrappedError struct {
	err error
}

func (e wrappedError) Error() string {
	return "wrapped: " + e.err.Error()
}

func (e wrappedError) Unwrap() error {
	return e.err
}
