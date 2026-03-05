// Package specctx provides specctx extraction for OpenAPI spec enrichment.
package specctx

// EnrichmentContext contains specctx information for LLM enrichment.
type EnrichmentContext struct {
	// Project-level information
	ProjectName string
	Framework   string // "spring-boot", "gin", etc.

	// Schema contexts
	Schemas map[string]*SchemaContext // key: schema name
}

// SchemaContext contains specctx for a single Schema.
type SchemaContext struct {
	// From OpenAPI Spec
	Name   string
	Fields []FieldMeta

	// Extended information (filled by Extractor)
	Description string            // Class/struct documentation
	Package     string            // Package name
	Annotations map[string]string // Language-specific annotations
}

// FieldMeta contains field metadata.
type FieldMeta struct {
	Name     string
	Type     string
	Required bool

	// Extended information
	Description string            // Field documentation
	Tags        map[string]string // e.g., json:"user_id", validate:"required"
}
