package publisher

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"gopkg.in/yaml.v3"

	"github.com/spencercjh/spec-forge/internal/executor"
)

// ReadMePublisher publishes OpenAPI specs to ReadMe.com using the rdme CLI.
type ReadMePublisher struct {
	exec executor.Interface
}

// NewReadMePublisher creates a new ReadMePublisher with default executor.
func NewReadMePublisher() *ReadMePublisher {
	return NewReadMePublisherWithExecutor(executor.NewExecutor())
}

// NewReadMePublisherWithExecutor creates a new ReadMePublisher with the given executor.
// This is useful for testing with a mock executor.
func NewReadMePublisherWithExecutor(exec executor.Interface) *ReadMePublisher {
	return &ReadMePublisher{exec: exec}
}

// Name returns the publisher name.
func (p *ReadMePublisher) Name() string {
	return publisherReadme
}

// Publish uploads an OpenAPI spec to ReadMe.com.
// The API key is passed via README_API_KEY environment variable to avoid leaking in process listings.
func (p *ReadMePublisher) Publish(ctx context.Context, spec *openapi3.T, opts *PublishOptions) (*PublishResult, error) {
	if spec == nil {
		return nil, errors.New("spec is nil")
	}

	if opts == nil || opts.ReadMe == nil {
		return nil, errors.New("readme options are required")
	}

	// Resolve API key from options or environment variable
	apiKey := opts.ReadMe.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("README_API_KEY")
	}
	if apiKey == "" {
		return nil, errors.New("readme API key is required (set --readme-api-key flag or README_API_KEY env var)")
	}

	// Create temp file for the spec
	tmpDir, err := os.MkdirTemp("", "spec-forge-readme-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(tmpDir)

	// Determine format
	format := opts.Format
	if format == "" {
		format = formatYAML
	}

	// Write spec to temp file
	tmpFile := filepath.Join(tmpDir, "openapi."+format)
	data, err := p.marshalSpec(spec, format)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal spec: %w", err)
	}

	if writeErr := os.WriteFile(tmpFile, data, 0o600); writeErr != nil {
		return nil, fmt.Errorf("failed to write temp file: %w", writeErr)
	}

	// Build rdme command args (without API key)
	args := p.buildArgs(tmpFile, opts)

	// Build sanitized environment with API key
	// SECURITY: API key is passed via env var to avoid leaking in process listings
	env := p.buildEnv(apiKey)

	// Execute rdme CLI via executor
	result, err := p.exec.Execute(ctx, &executor.ExecuteOptions{
		Command: "rdme",
		Args:    args,
		Env:     env,
	})
	if err != nil {
		return nil, p.wrapExecuteError(err, result)
	}

	// Build location identifier for the uploaded spec
	location := p.buildLocation(opts.ReadMe)

	return &PublishResult{
		Path:         location,
		Format:       format,
		BytesWritten: len(data),
		Message:      strings.TrimSpace(result.Stdout),
	}, nil
}

func (p *ReadMePublisher) marshalSpec(spec *openapi3.T, format string) ([]byte, error) {
	switch format {
	case formatJSON:
		return spec.MarshalJSON()
	default:
		yamlData, err := spec.MarshalYAML()
		if err != nil {
			return nil, err
		}
		return yaml.Marshal(yamlData)
	}
}

// buildArgs constructs rdme CLI arguments.
// The API key is read from README_API_KEY environment variable.
func (p *ReadMePublisher) buildArgs(specPath string, opts *PublishOptions) []string {
	args := []string{
		"openapi", "upload",
		specPath,
	}

	readmeOpts := opts.ReadMe

	// Add slug if specified
	if readmeOpts.Slug != "" {
		args = append(args, "--slug", readmeOpts.Slug)
	}

	// Add branch if specified
	if readmeOpts.Branch != "" {
		args = append(args, "--branch", readmeOpts.Branch)
	}

	// Add useSpecVersion if enabled
	if readmeOpts.UseSpecVersion {
		args = append(args, "--useSpecVersion")
	}

	// Only add confirm-overwrite if Overwrite is explicitly set to true
	// This respects the PublishOptions.Overwrite safety control
	if opts.Overwrite {
		args = append(args, "--confirm-overwrite")
	}

	return args
}

// buildEnv creates a sanitized environment with README_API_KEY set deterministically.
// It filters out any existing README_API_KEY entries to avoid duplicates.
func (p *ReadMePublisher) buildEnv(apiKey string) []string {
	env := make([]string, 0, len(os.Environ())+1)
	for _, v := range os.Environ() {
		if !strings.HasPrefix(v, "README_API_KEY=") {
			env = append(env, v)
		}
	}
	env = append(env, "README_API_KEY="+apiKey)
	return env
}

// buildLocation creates a human-readable location string for the uploaded spec.
func (p *ReadMePublisher) buildLocation(opts *ReadMeOptions) string {
	var parts []string

	if opts.Slug != "" {
		parts = append(parts, "slug:"+opts.Slug)
	}

	if opts.Branch != "" {
		parts = append(parts, "branch:"+opts.Branch)
	}

	if len(parts) == 0 {
		return "readme.com (default)"
	}

	return "readme.com/" + strings.Join(parts, "/")
}

// wrapExecuteError wraps executor errors with appropriate context.
func (p *ReadMePublisher) wrapExecuteError(err error, result *executor.ExecuteResult) error {
	// Handle command not found
	//nolint:errcheck // errors.AsType only returns (T, bool), no error to check
	if _, ok := errors.AsType[*executor.CommandNotFoundError](err); ok {
		return fmt.Errorf("rdme CLI not found: %w\nTo install rdme, run: npm install -g rdme", err)
	}

	// Handle command failure - include output for debugging
	//nolint:errcheck // errors.AsType only returns (T, bool), no error to check
	if cmdFailed, ok := errors.AsType[*executor.CommandFailedError](err); ok {
		output := cmdFailed.Stdout
		if cmdFailed.Stderr != "" {
			output += "\n" + cmdFailed.Stderr
		}
		return fmt.Errorf("rdme command failed: %w\noutput: %s", err, strings.TrimSpace(output))
	}

	// Other errors (timeout, etc.)
	if result != nil {
		output := result.Stdout
		if result.Stderr != "" {
			output += "\n" + result.Stderr
		}
		if output != "" {
			return fmt.Errorf("rdme command failed: %w\noutput: %s", err, strings.TrimSpace(output))
		}
	}

	return fmt.Errorf("rdme command failed: %w", err)
}
