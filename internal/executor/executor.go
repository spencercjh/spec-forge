// Package executor provides command execution capabilities for running external tools.
package executor

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/spencercjh/spec-forge/internal/extractor"
)

// Default timeout for command execution.
const defaultTimeout = 5 * time.Minute

// ExecuteOptions configures command execution.
type ExecuteOptions struct {
	Command    string        // "mvn" or "gradle"
	Args       []string      // Command arguments
	WorkingDir string        // Working directory for the command
	Timeout    time.Duration // Execution timeout (default: 5 minutes)
	Env        []string      // Additional environment variables
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
		cmd.Env = append(cmd.Environ(), opts.Env...)
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
				Hint:    getInstallHint(opts.Command),
			}
		}
	}

	// Command executed but failed (non-zero exit code)
	if err != nil {
		return result, &CommandFailedError{
			Command:  opts.Command,
			Args:     opts.Args,
			ExitCode: result.ExitCode,
			Stderr:   result.Stderr,
			Err:      err,
		}
	}

	return result, nil
}

// CommandNotFoundError indicates the command was not found in PATH.
type CommandNotFoundError struct {
	Command string
	Hint    string
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
	Stderr   string
	Err      error
}

func (e *CommandFailedError) Error() string {
	if e.Stderr != "" {
		return fmt.Sprintf("command '%s' failed with exit code %d: %s", e.Command, e.ExitCode, e.Stderr)
	}
	return fmt.Sprintf("command '%s' failed with exit code %d", e.Command, e.ExitCode)
}

func (e *CommandFailedError) Unwrap() error {
	return e.Err
}

// getInstallHint returns installation hints for common build tools.
func getInstallHint(command string) string {
	switch command {
	case "mvn":
		return "Install Maven from https://maven.apache.org/install.html or use your package manager"
	case "gradle":
		return "Install Gradle from https://gradle.org/install/ or use your package manager"
	default:
		return ""
	}
}

// Compile-time assertion that Executor implements Interface.
var _ Interface = (*Executor)(nil)

// Compile-time assertion for extractor types usage.
var _ = extractor.GenerateOptions{}
