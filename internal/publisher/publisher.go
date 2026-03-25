// Package publisher provides interfaces and implementations for publishing OpenAPI specs.
package publisher

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"

	forgeerrors "github.com/spencercjh/spec-forge/internal/errors"
)

// ErrUnknownPublisher is returned when an unknown publisher type is requested.
var ErrUnknownPublisher = errors.New("unknown publisher type")

// PublishOptions configures how the OpenAPI spec is published.
type PublishOptions struct {
	// OutputPath is the destination path for the spec file
	OutputPath string
	// Format is the output format: "yaml" or "json" (default: yaml)
	Format string
	// Overwrite existing file if true
	Overwrite bool

	// ReadMe-specific options
	ReadMe *ReadMeOptions
}

// ReadMeOptions contains ReadMe-specific publishing options.
type ReadMeOptions struct {
	// APIKey is the ReadMe project API key (required)
	//nolint:gosec // G117
	APIKey string `json:"apiKey"`
	// Branch is the ReadMe project version (default: stable)
	Branch string
	// Slug is the unique identifier for the API definition
	Slug string
	// UseSpecVersion uses OpenAPI info.version for ReadMe project version
	UseSpecVersion bool
}

// PublishResult contains the result of publishing an OpenAPI spec.
type PublishResult struct {
	// Path is the absolute path to the published spec file (local publisher)
	// or a URL/identifier for remote publishers (e.g., ReadMe)
	Path string
	// Format is the output format used
	Format string
	// BytesWritten is the number of bytes written
	BytesWritten int
	// Message contains additional info (e.g., CLI output for remote publishers)
	Message string
}

// Publisher defines the interface for publishing OpenAPI specs.
type Publisher interface {
	// Publish writes an OpenAPI spec to a destination.
	Publish(ctx context.Context, spec *openapi3.T, opts *PublishOptions) (*PublishResult, error)

	// Name returns the publisher name (e.g., "local", "apifox", "postman")
	Name() string
}

// Format constants
const (
	formatJSON      = "json"
	formatYAML      = "yaml"
	publisherReadme = "readme"
)

// NewPublisher creates a Publisher based on the destination type.
// Supported types: "readme"
// For unknown types or empty string, returns error.
func NewPublisher(destType string) (Publisher, error) {
	normalizedType := strings.ToLower(strings.TrimSpace(destType))
	switch normalizedType {
	case publisherReadme:
		return NewReadMePublisher(), nil
	case "":
		return nil, forgeerrors.ConfigError("publish target is required (e.g., readme)", nil)
	default:
		return nil, forgeerrors.ConfigError(fmt.Sprintf("unknown publisher type: %q", destType), ErrUnknownPublisher)
	}
}
