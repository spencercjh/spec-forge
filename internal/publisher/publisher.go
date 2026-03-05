// Package publisher provides interfaces and implementations for publishing OpenAPI specs.
package publisher

import (
	"context"

	"github.com/getkin/kin-openapi/openapi3"
)

// PublishOptions configures how the OpenAPI spec is published.
type PublishOptions struct {
	// OutputPath is the destination path for the spec file
	OutputPath string
	// Format is the output format: "yaml" or "json" (default: yaml)
	Format string
	// Overwrite existing file if true
	Overwrite bool
}

// PublishResult contains the result of publishing an OpenAPI spec.
type PublishResult struct {
	// Path is the absolute path to the published spec file
	Path string
	// Format is the output format used
	Format string
	// BytesWritten is the number of bytes written
	BytesWritten int
}

// Publisher defines the interface for publishing OpenAPI specs.
type Publisher interface {
	// Publish writes an OpenAPI spec to a destination.
	Publish(ctx context.Context, spec *openapi3.T, opts *PublishOptions) (*PublishResult, error)

	// Name returns the publisher name (e.g., "local", "apifox", "postman")
	Name() string
}
