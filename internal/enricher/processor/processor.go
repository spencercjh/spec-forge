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
}

// AddElement adds an element to the collector
func (c *SpecCollector) AddElement(element EnrichmentElement) {
	c.elements = append(c.elements, element)
}

// GroupByType groups elements by their template type
func (c *SpecCollector) GroupByType() []*Batch {
	groups := make(map[TemplateType][]EnrichmentElement)

	for _, elem := range c.elements {
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
