// Package gozero provides go-zero framework specific extraction functionality.
package gozero

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/spencercjh/spec-forge/internal/executor"
)

// Patcher checks and ensures goctl tool is available for go-zero projects.
type Patcher struct {
	exec executor.Interface
}

// NewPatcher creates a new Patcher instance with the default executor.
func NewPatcher() *Patcher {
	return &Patcher{
		exec: executor.NewExecutor(),
	}
}

// NewPatcherWithExecutor creates a new Patcher instance with a custom executor.
// This is primarily used for testing.
func NewPatcherWithExecutor(exec executor.Interface) *Patcher {
	return &Patcher{
		exec: exec,
	}
}

// PatchResult contains the result of a patch operation.
type PatchResult struct {
	GoctlAvailable bool   // Whether goctl is available
	GoctlVersion   string // Version of goctl if available
}

// Patch checks if goctl is installed and available.
// For go-zero projects, patching primarily involves verifying that goctl
// (the go-zero toolchain) is available for API processing.
func (p *Patcher) Patch(_ string) (*PatchResult, error) {
	slog.Debug("checking goctl availability")
	ctx := context.Background()

	opts := &executor.ExecuteOptions{
		Command: "goctl",
		Args:    []string{"--version"},
		Timeout: versionCheckTimeout,
	}

	result, err := p.exec.Execute(ctx, opts)
	if err != nil {
		// Check if the command was not found
		//nolint:errcheck // errors.AsType only returns (T, bool), no error to check
		if _, ok := errors.AsType[*executor.CommandNotFoundError](err); ok {
			slog.Error("goctl not found", "error", err)
			return nil, errors.New(
				"goctl is not installed. goctl is required for processing go-zero API files.\n" +
					"To install goctl, run:\n" +
					"  go install github.com/zeromicro/go-zero/tools/goctl@latest",
			)
		}

		// Command failed (non-zero exit code)
		slog.Error("goctl version check failed", "error", err)
		return nil, fmt.Errorf("goctl version check failed: %w", err)
	}

	version := strings.TrimSpace(result.Stdout)
	slog.Info("goctl is available", "version", version)

	// goctl is available
	return &PatchResult{
		GoctlAvailable: true,
		GoctlVersion:   version,
	}, nil
}

// NeedsPatch always returns true for go-zero projects since we need to verify goctl availability.
func (p *Patcher) NeedsPatch(_ *Info) bool {
	return true
}
