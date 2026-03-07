package gin

// SchemaExtractor extracts OpenAPI schemas from Go structs.
type SchemaExtractor struct{}

// NewSchemaExtractor creates a new SchemaExtractor instance.
func NewSchemaExtractor() *SchemaExtractor {
	return &SchemaExtractor{}
}
