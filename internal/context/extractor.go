// internal/context/extractor.go
package context

import (
	stdcontext "context"

	"github.com/getkin/kin-openapi/openapi3"
)

// ContextExtractor extracts context information from source code.
type ContextExtractor interface {
	// Extract extracts context from a project.
	// The spec parameter is the generated OpenAPI spec, used as a skeleton.
	Extract(ctx stdcontext.Context, projectPath string, spec *openapi3.T) (*EnrichmentContext, error)

	// Name returns the extractor name.
	Name() string
}
