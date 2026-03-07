// Package executor provides command execution capabilities for running external tools.
package executor

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Default timeout for command execution.
const defaultTimeout = 5 * time.Minute

// ExecuteOptions configures command execution.
type ExecuteOptions struct {
	Command    string        // "mvn" or "gradle"
	Args       []string      // Command arguments
	WorkingDir string        // Working directory for the command
	Timeout    time.Duration // Execution timeout (default: 5 minutes)
	Env        []string      // Environment variables (replaces entire env if set)
	// EnvAppendMode determines how Env is applied:
	// - false (default): Env replaces the entire environment (useful for security-sensitive vars)
	// - true: Env is appended to the current environment
	EnvAppendMode bool
}

// ExecuteResult contains the result of command execution.
type ExecuteResult struct {
	ExitCode int    // Process exit code
	Stdout   string // Standard output
	Stderr   string // Standard error
}

// Interface defines the interface for command execution.
type Interface interface {
	Execute(ctx context.Context, opts *ExecuteOptions) (*ExecuteResult, error)
}

// Executor executes external commands.
type Executor struct{}

// NewExecutor creates a new Executor instance.
func NewExecutor() *Executor {
	return &Executor{}
}

// Execute runs a command with the given options.
//
//nolint:gosec // G204: Subprocess launched with variable command - this is the intended behavior
func (e *Executor) Execute(ctx context.Context, opts *ExecuteOptions) (*ExecuteResult, error) {
	if opts == nil {
		return nil, errors.New("execute options cannot be nil")
	}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, opts.Command, opts.Args...)

	if opts.WorkingDir != "" {
		cmd.Dir = opts.WorkingDir
	}

	if len(opts.Env) > 0 {
		if opts.EnvAppendMode {
			cmd.Env = append(cmd.Environ(), opts.Env...)
		} else {
			cmd.Env = opts.Env
		}
	}

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := &ExecuteResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	// Check if context was canceled/timed out
	if ctx.Err() != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return result, fmt.Errorf("command '%s' timed out after %v", opts.Command, timeout)
		}
		return result, fmt.Errorf("command '%s' canceled: %w", opts.Command, ctx.Err())
	}

	// Get exit code
	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}

	// Handle command not found
	if execErr, ok := errors.AsType[*exec.Error](err); ok {
		if errors.Is(execErr.Err, exec.ErrNotFound) {
			return result, &CommandNotFoundError{
				Command: opts.Command,
			}
		}
	}

	// Command executed but failed (non-zero exit code)
	if err != nil {
		return result, &CommandFailedError{
			Command:  opts.Command,
			Args:     opts.Args,
			ExitCode: result.ExitCode,
			Stdout:   result.Stdout,
			Stderr:   result.Stderr,
			Err:      err,
		}
	}

	return result, nil
}

// CommandNotFoundError indicates the command was not found in PATH.
type CommandNotFoundError struct {
	Command string
}

func (e *CommandNotFoundError) Error() string {
	return fmt.Sprintf("command '%s' not found in PATH", e.Command)
}

func (e *CommandNotFoundError) Unwrap() error {
	return exec.ErrNotFound
}

// CommandFailedError indicates the command executed but returned non-zero exit code.
type CommandFailedError struct {
	Command  string
	Args     []string
	ExitCode int
	Stdout   string
	Stderr   string
	Err      error
}

func (e *CommandFailedError) Error() string {
	// Build tools like Maven/Gradle often output errors to stdout, not stderr
	// Combine both for better error messages
	var output strings.Builder
	if e.Stdout != "" {
		output.WriteString(e.Stdout)
	}
	if e.Stderr != "" {
		if output.Len() > 0 {
			output.WriteString("\n")
		}
		output.WriteString(e.Stderr)
	}

	combined := strings.TrimSpace(output.String())
	if combined != "" {
		return fmt.Sprintf("command '%s' failed with exit code %d:\n%s", e.Command, e.ExitCode, combined)
	}
	if e.Err != nil {
		return fmt.Sprintf("command '%s' failed with exit code %d (no output): %v", e.Command, e.ExitCode, e.Err)
	}
	return fmt.Sprintf("command '%s' failed with exit code %d (no output)", e.Command, e.ExitCode)
}

func (e *CommandFailedError) Unwrap() error {
	return e.Err
}
