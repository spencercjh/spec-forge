package publisher

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"gopkg.in/yaml.v3"
)

// ReadMePublisher publishes OpenAPI specs to ReadMe.com using the rdme CLI.
type ReadMePublisher struct{}

// NewReadMePublisher creates a new ReadMePublisher.
func NewReadMePublisher() *ReadMePublisher {
	return &ReadMePublisher{}
}

// Name returns the publisher name.
func (p *ReadMePublisher) Name() string {
	return "readme"
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

	if opts.ReadMe.APIKey == "" {
		return nil, errors.New("readme API key is required")
	}

	// Create temp file for the spec
	tmpDir, err := os.MkdirTemp("", "spec-forge-readme-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

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

	// Execute rdme CLI with API key via environment variable
	// SECURITY: API key is passed via env var to avoid leaking in process listings
	cmd := exec.CommandContext(ctx, "rdme", args...)
	cmd.Env = append(os.Environ(), "README_API_KEY="+opts.ReadMe.APIKey)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("rdme command failed: %w\noutput: %s", err, string(output))
	}

	// Build location identifier for the uploaded spec
	location := p.buildLocation(opts.ReadMe)

	return &PublishResult{
		Path:         location,
		Format:       format,
		BytesWritten: len(data),
		Message:      strings.TrimSpace(string(output)),
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
