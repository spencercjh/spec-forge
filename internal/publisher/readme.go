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

	// Build rdme command
	args := p.buildArgs(tmpFile, opts.ReadMe)

	// Execute rdme CLI
	cmd := exec.CommandContext(ctx, "rdme", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("rdme command failed: %w\noutput: %s", err, string(output))
	}

	return &PublishResult{
		Path:         strings.TrimSpace(string(output)),
		Format:       format,
		BytesWritten: len(data),
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

func (p *ReadMePublisher) buildArgs(specPath string, opts *ReadMeOptions) []string {
	args := []string{
		"openapi", "upload",
		specPath,
		"--key", opts.APIKey,
		"--confirm-overwrite",
	}

	if opts.Branch != "" {
		args = append(args, "--branch", opts.Branch)
	}

	if opts.Slug != "" {
		args = append(args, "--slug", opts.Slug)
	}

	if opts.UseSpecVersion {
		args = append(args, "--useSpecVersion")
	}

	return args
}
