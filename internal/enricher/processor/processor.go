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

	// Param group-specific: fields to set descriptions for
	ParamGroupFields []ParamFieldItem
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
	Skipped  struct {
		APIs         int
		Params       int
		SchemaFields int
	}
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
	FieldName           string
	FieldType           string
	Required            bool
	Format              string
	Enum                []string
	Constraints         string
	ExistingDescription string
	SetValue            func(description string)
}

// ParamGroupElement represents a group of parameters from the same API endpoint.
type ParamGroupElement struct {
	Path   string
	Method string
	Params []ParamFieldItem
}

// ParamFieldItem represents a single parameter within a group.
type ParamFieldItem struct {
	ParamName           string
	ParamIn             string
	FieldType           string
	Required            bool
	Format              string
	Enum                []string
	Constraints         string
	ExistingDescription string
	SetValue            func(description string)
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
		SchemaFields: schema.Fields, // Field-level SetValue callbacks
		SetValue: func(_ string) {
			// Schema SetValue is handled by individual field SetValue callbacks
		},
	})
}

// convertFieldElements converts FieldElement slice to FieldContext slice.
func convertFieldElements(fields []FieldElement) []prompt.FieldContext {
	result := make([]prompt.FieldContext, len(fields))
	for i, f := range fields {
		result[i] = prompt.FieldContext{
			Name:                f.FieldName,
			Type:                f.FieldType,
			Required:            f.Required,
			Format:              f.Format,
			Enum:                f.Enum,
			Constraints:         f.Constraints,
			ExistingDescription: f.ExistingDescription,
		}
	}
	return result
}

// convertParamFieldItems converts ParamFieldItem slice to ParamFieldContext slice.
func convertParamFieldItems(items []ParamFieldItem) []prompt.ParamFieldContext {
	result := make([]prompt.ParamFieldContext, len(items))
	for i, p := range items {
		result[i] = prompt.ParamFieldContext{
			Name:                p.ParamName,
			Type:                p.FieldType,
			ParamIn:             p.ParamIn,
			Required:            p.Required,
			Format:              p.Format,
			Enum:                p.Enum,
			Constraints:         p.Constraints,
			ExistingDescription: p.ExistingDescription,
		}
	}
	return result
}

// GetSchemas returns collected schemas.
func (c *SpecCollector) GetSchemas() []SchemaElement {
	return c.schemas
}

// AddParamGroupElement adds a parameter group to the collector.
func (c *SpecCollector) AddParamGroupElement(group ParamGroupElement, language string) {
	c.elements = append(c.elements, EnrichmentElement{
		Type: prompt.TemplateTypeParam,
		Path: group.Method + " " + group.Path + " [params]",
		Context: prompt.TemplateContext{
			Type:        prompt.TemplateTypeParam,
			Language:    language,
			Method:      group.Method,
			Path:        group.Path,
			ParamFields: convertParamFieldItems(group.Params),
		},
		ParamGroupFields: group.Params,
	})
}
