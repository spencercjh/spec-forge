package specctx

import (
	stdcontext "context"

	"github.com/getkin/kin-openapi/openapi3"
)

// Extractor extracts specctx information from source code.
type Extractor interface {
	// Extract extracts specctx from a project.
	// The spec parameter is the generated OpenAPI spec, used as a skeleton.
	Extract(ctx stdcontext.Context, projectPath string, spec *openapi3.T) (*EnrichmentContext, error)

	// Name returns the extractor name.
	Name() string
}
