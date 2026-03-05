package specctx_test

import (
	stdcontext "context"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/spencercjh/spec-forge/internal/enricher/specctx"
)

func TestExtractor_Interface(t *testing.T) {
	// Verify interface compliance
	var _ specctx.Extractor = (*specctx.NoOpExtractor)(nil)
}

func TestNoOpExtractor_Name(t *testing.T) {
	extractor := &specctx.NoOpExtractor{}
	if extractor.Name() != "noop" {
		t.Errorf("expected Name 'noop', got %s", extractor.Name())
	}
}

func TestNoOpExtractor_Extract(t *testing.T) {
	extractor := &specctx.NoOpExtractor{}
	ctx := stdcontext.Background()
	spec := &openapi3.T{
		Components: &openapi3.Components{
			Schemas: openapi3.Schemas{
				"User": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: &openapi3.Types{"object"},
						Properties: openapi3.Schemas{
							"id": &openapi3.SchemaRef{
								Value: &openapi3.Schema{
									Type: &openapi3.Types{"integer"},
								},
							},
						},
					},
				},
			},
		},
	}

	result, err := extractor.Extract(ctx, "/dummy/path", spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Should have extracted the User schema
	if len(result.Schemas) != 1 {
		t.Errorf("expected 1 schema, got %d", len(result.Schemas))
	}

	userSchema, ok := result.Schemas["User"]
	if !ok {
		t.Fatal("expected User schema in result")
	}

	if len(userSchema.Fields) != 1 {
		t.Errorf("expected 1 field in User schema, got %d", len(userSchema.Fields))
	}
}
