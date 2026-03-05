package processor

import (
	"fmt"

	"github.com/spencercjh/spec-forge/internal/enricher/prompt"
)

// EnrichmentElement represents an element to be enriched
type EnrichmentElement struct {
	Type     prompt.TemplateType
	Path     string // Path in the OpenAPI spec
	Context  prompt.TemplateContext
	SetValue func(description string) // Callback to set the description

	// Schema-specific: fields to set descriptions for
	SchemaFields []FieldElement
}

// Batch represents a group of elements to process together
type Batch struct {
	Type     TemplateType
	Elements []EnrichmentElement
}

// TemplateType alias for prompt.TemplateType
type TemplateType = prompt.TemplateType

// PartialEnrichmentError indicates that some batches failed during enrichment
type PartialEnrichmentError struct {
	TotalBatches  int
	FailedBatches int
	Errors        []error
}

// Error implements the error interface
func (e *PartialEnrichmentError) Error() string {
	return fmt.Sprintf("partial enrichment: %d/%d batches failed", e.FailedBatches, e.TotalBatches)
}

// SpecCollector collects elements from an OpenAPI spec for enrichment
type SpecCollector struct {
	elements []EnrichmentElement
	schemas  []SchemaElement
	params   []ParamElement
}

// AddElement adds an element to the collector
func (c *SpecCollector) AddElement(element EnrichmentElement) { //nolint:gocritic // copying element is acceptable
	c.elements = append(c.elements, element)
}

// GroupByType groups elements by their template type
func (c *SpecCollector) GroupByType() []*Batch {
	groups := make(map[TemplateType][]EnrichmentElement)

	for _, elem := range c.elements { //nolint:gocritic // copying elements is acceptable here
		groups[elem.Type] = append(groups[elem.Type], elem)
	}

	batches := make([]*Batch, 0, len(groups))
	for ttype, elements := range groups {
		batches = append(batches, &Batch{
			Type:     ttype,
			Elements: elements,
		})
	}

	return batches
}

// SchemaElement represents a schema to be enriched.
type SchemaElement struct {
	SchemaName string
	Fields     []FieldElement
	Context    prompt.TemplateContext
}

// FieldElement represents a field to be enriched.
type FieldElement struct {
	FieldName string
	FieldType string
	Required  bool
	SetValue  func(description string)
}

// ParamElement represents an API parameter to be enriched.
type ParamElement struct {
	Path      string
	Method    string
	ParamName string
	ParamIn   string // path, query, header, cookie
	FieldType string
	Required  bool
	SetValue  func(description string)
}

// AddSchemaElement adds a schema element to the collector.
func (c *SpecCollector) AddSchemaElement(schema SchemaElement, language string) { //nolint:gocritic
	c.schemas = append(c.schemas, schema)

	// Also add as an enrichment element for processing
	c.elements = append(c.elements, EnrichmentElement{
		Type: prompt.TemplateTypeSchema,
		Path: "#/components/schemas/" + schema.SchemaName,
		Context: prompt.TemplateContext{
			Type:       prompt.TemplateTypeSchema,
			Language:   language,
			SchemaName: schema.SchemaName,
			Fields:     convertFieldElements(schema.Fields),
		},
		SetValue: func(_ string) {
			// Schema SetValue is handled by individual field SetValue callbacks
		},
	})
}

// AddParamElement adds a parameter element to the collector.
func (c *SpecCollector) AddParamElement(path, method, paramName, paramIn, fieldType string, required bool, language string) {
	c.params = append(c.params, ParamElement{
		Path:      path,
		Method:    method,
		ParamName: paramName,
		ParamIn:   paramIn,
		FieldType: fieldType,
		Required:  required,
	})
}

// convertFieldElements converts FieldElement slice to FieldContext slice.
func convertFieldElements(fields []FieldElement) []prompt.FieldContext {
	result := make([]prompt.FieldContext, len(fields))
	for i, f := range fields {
		result[i] = prompt.FieldContext{
			Name:     f.FieldName,
			Type:     f.FieldType,
			Required: f.Required,
		}
	}
	return result
}

// GetSchemas returns collected schemas.
func (c *SpecCollector) GetSchemas() []SchemaElement {
	return c.schemas
}

// GetParams returns collected parameters.
func (c *SpecCollector) GetParams() []ParamElement {
	return c.params
}
