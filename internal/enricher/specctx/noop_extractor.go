package specctx

import (
	stdcontext "context"
	"log/slog"
	"slices"

	"github.com/getkin/kin-openapi/openapi3"
)

// NoOpExtractor is a default implementation that extracts basic info from the spec only.
// It does not perform source code analysis.
type NoOpExtractor struct{}

// Name returns the extractor name.
func (e *NoOpExtractor) Name() string {
	return "noop"
}

// Extract extracts specctx from the OpenAPI spec without source code analysis.
func (e *NoOpExtractor) Extract(_ stdcontext.Context, _ string, spec *openapi3.T) (*EnrichmentContext, error) {
	if spec == nil {
		slog.Debug("NoOpExtractor: spec is nil, returning empty specctx")
		return &EnrichmentContext{Schemas: make(map[string]*SchemaContext)}, nil
	}

	result := &EnrichmentContext{
		Schemas: make(map[string]*SchemaContext),
	}

	// Extract schemas from components
	if spec.Components != nil && spec.Components.Schemas != nil {
		for schemaName, schemaRef := range spec.Components.Schemas {
			if schemaRef.Value == nil {
				continue
			}

			schemaCtx := &SchemaContext{
				Name:   schemaName,
				Fields: extractFieldsFromSchema(schemaRef.Value),
			}
			result.Schemas[schemaName] = schemaCtx
		}
	}

	slog.Debug("NoOpExtractor: extracted specctx from spec", "schemas", len(result.Schemas))
	return result, nil
}

// extractFieldsFromSchema extracts field metadata from an OpenAPI schema.
func extractFieldsFromSchema(schema *openapi3.Schema) []FieldMeta {
	if schema.Properties == nil {
		return nil
	}

	var fields []FieldMeta
	for propName, propRef := range schema.Properties {
		if propRef.Value == nil {
			continue
		}

		field := FieldMeta{
			Name:     propName,
			Type:     getSchemaTypeName(propRef.Value),
			Required: containsString(schema.Required, propName),
		}
		fields = append(fields, field)
	}

	return fields
}

// getSchemaTypeName returns a human-readable type name for a schema.
func getSchemaTypeName(schema *openapi3.Schema) string {
	if schema.Type != nil && len(*schema.Type) > 0 {
		typeStr := (*schema.Type)[0]
		if typeStr == "array" && schema.Items != nil && schema.Items.Value != nil {
			return "array of " + getSchemaTypeName(schema.Items.Value)
		}
		return typeStr
	}
	// If no type is specified, default to object
	return "object"
}

// containsString checks if a string is in a slice.
func containsString(slice []string, s string) bool {
	return slices.Contains(slice, s)
}
