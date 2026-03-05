package enricher

import (
	"context"
	"log/slog"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/spencercjh/spec-forge/internal/enricher/processor"
	"github.com/spencercjh/spec-forge/internal/enricher/prompt"
	"github.com/spencercjh/spec-forge/internal/enricher/provider"
)

// Enricher enriches OpenAPI specs with AI-generated descriptions
type Enricher struct {
	config   Config
	provider provider.Provider
}

// EnrichOptions provides runtime options for enrichment
type EnrichOptions struct {
	Language string // Runtime language override
}

// NewEnricher creates a new Enricher
func NewEnricher(cfg Config, p provider.Provider) (*Enricher, error) {
	if err := cfg.Validate(); err != nil {
		return nil, NewEnrichmentError(ErrorTypeConfig, "invalid configuration", err)
	}
	return &Enricher{
		config:   cfg.MergeWithDefaults(),
		provider: p,
	}, nil
}

// Enrich enriches an OpenAPI spec with AI-generated descriptions
func (e *Enricher) Enrich(ctx context.Context, spec *openapi3.T, opts *EnrichOptions) (*openapi3.T, error) {
	if spec == nil {
		return nil, NewEnrichmentError(ErrorTypeConfig, "spec is nil", nil)
	}

	language := e.config.Language
	if opts != nil && opts.Language != "" {
		language = opts.Language
	}

	// Collect elements to enrich
	collector := e.collectElements(spec, language)

	// Group elements into batches
	batches := collector.GroupByType()

	if len(batches) == 0 {
		slog.Debug("No elements to enrich")
		return spec, nil
	}

	slog.Info("Enriching spec", "batches", len(batches), "language", language)

	// Process batches
	tmplMgr := prompt.NewTemplateManager()
	batchProcessor := processor.NewBatchProcessor(e.provider, tmplMgr)
	concurrentProcessor := processor.NewConcurrentProcessor(batchProcessor, e.config.Concurrency)

	if err := concurrentProcessor.ProcessAll(ctx, batches); err != nil {
		return spec, err
	}

	return spec, nil
}

// collectElements collects elements from the spec that need enrichment
func (e *Enricher) collectElements(spec *openapi3.T, language string) *processor.SpecCollector {
	collector := &processor.SpecCollector{}

	// Collect API operations
	if spec.Paths != nil {
		for _, pathStr := range spec.Paths.InMatchingOrder() {
			pathItem := spec.Paths.Value(pathStr)
			if pathItem == nil {
				continue
			}

			operations := []struct {
				method string
				op     *openapi3.Operation
			}{
				{"GET", pathItem.Get},
				{"POST", pathItem.Post},
				{"PUT", pathItem.Put},
				{"DELETE", pathItem.Delete},
				{"PATCH", pathItem.Patch},
				{"HEAD", pathItem.Head},
				{"OPTIONS", pathItem.Options},
			}

			for _, item := range operations {
				if item.op == nil {
					continue
				}

				// Only enrich if description is missing
				if item.op.Summary == "" || item.op.Description == "" {
					op := item.op // Capture for closure
					collector.AddElement(processor.EnrichmentElement{
						Type: prompt.TemplateTypeAPI,
						Path: item.method + " " + pathStr,
						Context: prompt.TemplateContext{
							Type:     prompt.TemplateTypeAPI,
							Language: language,
							Method:   item.method,
							Path:     pathStr,
						},
						SetValue: func(desc string) {
							// Parse response and set summary/description
							summary, description := parseSummaryDescription(desc)
							if op.Summary == "" {
								op.Summary = summary
							}
							if op.Description == "" {
								op.Description = description
							}
						},
					})
				}
			}
		}
	}

	// TODO: Collect schemas, parameters, responses

	return collector
}

// parseSummaryDescription splits a description into summary and full description
func parseSummaryDescription(desc string) (summary, description string) {
	// If the description contains double newline, split it
	for i := 0; i < len(desc)-1; i++ {
		if desc[i] == '\n' && desc[i+1] == '\n' {
			return desc[:i], desc[i+2:]
		}
	}
	// Otherwise, use the whole thing as summary
	return desc, ""
}
