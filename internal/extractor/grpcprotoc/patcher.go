// Package grpcprotoc provides gRPC-protoc framework extraction functionality.
package grpcprotoc

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	forgeerrors "github.com/spencercjh/spec-forge/internal/errors"
	"github.com/spencercjh/spec-forge/internal/executor"
)

// ErrProtocNotInstalled indicates protoc is not found.
var ErrProtocNotInstalled = errors.New(
	"protoc not found. Install from: https://github.com/protocolbuffers/protobuf/releases")

// ErrProtocGenConnectOpenAPINotInstalled indicates the plugin is not found.
var ErrProtocGenConnectOpenAPINotInstalled = errors.New(
	"protoc-gen-connect-openapi not found. Install with: " +
		"go install github.com/sudorandom/protoc-gen-connect-openapi@latest")

// Patcher checks and ensures protoc tools are available for gRPC projects.
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
	ProtocInstalled                  bool   // Whether protoc is available
	ProtocVersion                    string // Version of protoc if available
	ProtocGenConnectOpenAPIInstalled bool   // Whether protoc-gen-connect-openapi is available
	ProtocGenConnectOpenAPIVersion   string // Version of protoc-gen-connect-openapi if available
}

// Patch checks if protoc and protoc-gen-connect-openapi are installed and available.
// For gRPC-protoc projects, patching primarily involves verifying that the required
// tools are available for OpenAPI generation.
func (p *Patcher) Patch(projectPath string) (*PatchResult, error) {
	slog.Debug("checking protoc tools availability", "path", projectPath)
	ctx := context.Background()

	// Check protoc
	protocVersion, err := p.checkProtoc(ctx)
	if err != nil {
		return nil, err
	}

	// Check protoc-gen-connect-openapi
	pluginVersion, err := p.checkProtocGenConnectOpenAPI(ctx)
	if err != nil {
		return nil, err
	}

	slog.Info("protoc tools are available",
		"protocVersion", protocVersion,
		"pluginVersion", pluginVersion)

	return &PatchResult{
		ProtocInstalled:                  true,
		ProtocVersion:                    protocVersion,
		ProtocGenConnectOpenAPIInstalled: true,
		ProtocGenConnectOpenAPIVersion:   pluginVersion,
	}, nil
}

// checkProtoc verifies that protoc is installed and returns its version.
func (p *Patcher) checkProtoc(ctx context.Context) (string, error) {
	opts := &executor.ExecuteOptions{
		Command: "protoc",
		Args:    []string{"--version"},
	}

	result, err := p.exec.Execute(ctx, opts)
	if err != nil {
		// Check if the command was not found
		//nolint:errcheck // errors.AsType only returns (T, bool), no error to check
		if _, ok := errors.AsType[*executor.CommandNotFoundError](err); ok {
			slog.Error("protoc not found", "error", err)
			return "", forgeerrors.PatchError(ErrProtocNotInstalled.Error(), nil)
		}

		// Command failed (non-zero exit code)
		slog.Error("protoc version check failed", "error", err)
		return "", forgeerrors.PatchError("protoc version check failed", err)
	}

	version := strings.TrimSpace(result.Stdout)
	slog.Debug("protoc is available", "version", version)
	return version, nil
}

// checkProtocGenConnectOpenAPI verifies that protoc-gen-connect-openapi is installed and returns its version.
func (p *Patcher) checkProtocGenConnectOpenAPI(ctx context.Context) (string, error) {
	opts := &executor.ExecuteOptions{
		Command: "protoc-gen-connect-openapi",
		Args:    []string{"--version"},
	}

	result, err := p.exec.Execute(ctx, opts)
	if err != nil {
		// Check if the command was not found
		//nolint:errcheck // errors.AsType only returns (T, bool), no error to check
		if _, ok := errors.AsType[*executor.CommandNotFoundError](err); ok {
			slog.Error("protoc-gen-connect-openapi not found", "error", err)
			return "", forgeerrors.PatchError(ErrProtocGenConnectOpenAPINotInstalled.Error(), nil)
		}

		// Command failed (non-zero exit code)
		slog.Error("protoc-gen-connect-openapi version check failed", "error", err)
		return "", forgeerrors.PatchError("protoc-gen-connect-openapi version check failed", err)
	}

	version := strings.TrimSpace(result.Stdout)
	slog.Debug("protoc-gen-connect-openapi is available", "version", version)
	return version, nil
}

// NeedsPatch always returns true for protoc projects since we need to verify tool availability.
func (p *Patcher) NeedsPatch(_ *Info) bool {
	return true
}
