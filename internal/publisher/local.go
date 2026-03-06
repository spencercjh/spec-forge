package publisher

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/getkin/kin-openapi/openapi3"
	"gopkg.in/yaml.v3"
)

// LocalPublisher publishes OpenAPI specs to local files.
type LocalPublisher struct{}

// NewLocalPublisher creates a new LocalPublisher.
func NewLocalPublisher() *LocalPublisher {
	return &LocalPublisher{}
}

const publisherLocal = "local"

// Name returns the publisher name.
func (p *LocalPublisher) Name() string {
	return publisherLocal
}

// Publish writes an OpenAPI spec to a local file.
func (p *LocalPublisher) Publish(ctx context.Context, spec *openapi3.T, opts *PublishOptions) (*PublishResult, error) {
	_ = ctx // Context reserved for future use (e.g., validation, cancellation)
	if spec == nil {
		return nil, errors.New("spec is nil")
	}

	if opts == nil {
		return nil, errors.New("options are required")
	}

	if opts.OutputPath == "" {
		return nil, errors.New("output path is required")
	}

	// Determine format from extension or options
	format := opts.Format
	if format == "" {
		if filepath.Ext(opts.OutputPath) == ".json" {
			format = formatJSON
		} else {
			format = formatYAML
		}
	}

	// Marshal spec to bytes
	var data []byte
	var marshalErr error

	switch format {
	case formatJSON:
		data, marshalErr = spec.MarshalJSON()
	default:
		var yamlData any
		yamlData, marshalErr = spec.MarshalYAML()
		if marshalErr == nil {
			data, marshalErr = yaml.Marshal(yamlData)
		}
	}

	if marshalErr != nil {
		return nil, errors.New("failed to marshal spec: " + marshalErr.Error())
	}

	// Create parent directory if needed
	dir := filepath.Dir(opts.OutputPath)
	if dir != "" && dir != "." {
		if mkdirErr := os.MkdirAll(dir, 0o755); mkdirErr != nil {
			return nil, errors.New("failed to create directory: " + mkdirErr.Error())
		}
	}

	// Check if file exists and overwrite is not set
	if !opts.Overwrite {
		if _, statErr := os.Stat(opts.OutputPath); statErr == nil {
			return nil, errors.New("file already exists: " + opts.OutputPath + " (use overwrite=true to replace)")
		}
	}

	// Write file
	if writeErr := os.WriteFile(opts.OutputPath, data, 0o600); writeErr != nil {
		return nil, errors.New("failed to write file: " + writeErr.Error())
	}

	// Get absolute path
	absPath, absErr := filepath.Abs(opts.OutputPath)
	if absErr != nil {
		absPath = opts.OutputPath
	}

	return &PublishResult{
		Path:         absPath,
		Format:       format,
		BytesWritten: len(data),
	}, nil
}
