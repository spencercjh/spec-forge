package enricher

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/spencercjh/spec-forge/internal/enricher/processor"
	"github.com/spencercjh/spec-forge/internal/enricher/prompt"
	"github.com/spencercjh/spec-forge/internal/enricher/provider"
	"github.com/spencercjh/spec-forge/internal/enricher/specctx"
)

// Enricher enriches OpenAPI specs with AI-generated descriptions
type Enricher struct {
	config    Config
	provider  provider.Provider
	extractor specctx.Extractor
}

// EnrichOptions provides runtime options for enrichment
type EnrichOptions struct {
	Language string    // Runtime language override
	Stream   *bool     // Enable streaming output (nil = default true, false = disabled)
	Writer   io.Writer // Custom writer for streaming (default: os.Stdout)
	Force    bool      // Force regeneration of all descriptions, ignoring existing ones
}

// NewEnricher creates a new Enricher
func NewEnricher(cfg Config, p provider.Provider) (*Enricher, error) { //nolint:gocritic // copying config is acceptable
	if err := cfg.Validate(); err != nil {
		return nil, NewEnrichmentError(ErrorTypeConfig, "invalid configuration", err)
	}
	return &Enricher{
		config:    cfg.MergeWithDefaults(),
		provider:  p,
		extractor: &specctx.NoOpExtractor{}, // Default to NoOpExtractor
	}, nil
}

// WithExtractor sets a custom specctx extractor
func (e *Enricher) WithExtractor(extractor specctx.Extractor) *Enricher {
	if extractor != nil {
		e.extractor = extractor
	}
	return e
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

	// Determine streaming settings
	stream := true // default: streaming enabled
	var writer io.Writer = os.Stdout
	force := false
	if opts != nil {
		// Stream is a tri-state (*bool): nil means "use default" (true)
		if opts.Stream != nil {
			stream = *opts.Stream
		}
		if opts.Writer != nil {
			writer = opts.Writer
		}
		force = opts.Force
	}
	if writer == nil {
		writer = os.Stdout
	}

	var streamWriter *processor.StreamWriter
	if stream {
		streamWriter = processor.NewStreamWriter(writer)
	}

	// Extract context from project (if extractor is configured)
	// This is a no-op for NoOpExtractor, but future extractors can provide
	// richer context like Javadoc, Go struct tags, etc.
	enrichCtx, err := e.extractor.Extract(ctx, e.config.ProjectPath, spec)
	if err != nil {
		slog.Warn("failed to extract enrichment context, using empty context", "error", err)
		enrichCtx = &specctx.EnrichmentContext{Schemas: make(map[string]*specctx.SchemaContext)}
	}

	// Collect elements to enrich
	collector := e.collectElements(spec, enrichCtx, language, force)

	// Log skipped items
	if collector.Skipped.APIs > 0 || collector.Skipped.Params > 0 || collector.Skipped.Schemas > 0 {
		slog.Info("Skipped items with existing descriptions",
			"apis", collector.Skipped.APIs,
			"params", collector.Skipped.Params,
			"schemas", collector.Skipped.Schemas,
		)
	}

	// Group elements into batches
	batches := collector.GroupByType()

	if len(batches) == 0 {
		slog.Debug("No elements to enrich")
		return spec, nil
	}

	slog.Info("Enriching spec", "batches", len(batches), "language", language)

	// Process batches
	tmplMgr := prompt.NewTemplateManager()
	batchProcessor := processor.NewBatchProcessor(e.provider, tmplMgr,
		processor.WithStreamWriter(streamWriter))
	concurrentProcessor := processor.NewConcurrentProcessor(batchProcessor, e.config.Concurrency)

	totalUsage, err := concurrentProcessor.ProcessAll(ctx, batches)

	// Display token usage (even on partial error)
	e.displayTokenUsage(totalUsage)

	if err != nil {
		return spec, err
	}

	// Flush any remaining buffered content in StreamWriter.
	// Streaming output is ancillary; a flush failure should not cause enrichment to fail.
	if streamWriter != nil {
		if flushErr := streamWriter.Flush(); flushErr != nil {
			slog.Warn("failed to flush stream writer after successful enrichment", "error", flushErr)
		}
	}

	return spec, nil
}

// displayTokenUsage logs token consumption and estimated cost.
func (e *Enricher) displayTokenUsage(usage *provider.TokenUsage) {
	if usage == nil || usage.Total() == 0 {
		return
	}
	slog.Info("Token usage",
		"input", usage.InputTokens,
		"output", usage.OutputTokens,
		"total", usage.Total(),
	)
	if cost, ok := provider.EstimateCost(e.config.Provider, e.config.Model, usage); ok {
		slog.Info("Estimated cost", "cost", fmt.Sprintf("$%.4f", cost), "model", e.config.Model)
	}
}

// collectElements collects elements from the spec that need enrichment.
// The enrichCtx parameter provides additional context extracted from source code (currently unused, reserved for future enhancement).
func (e *Enricher) collectElements(spec *openapi3.T, _ *specctx.EnrichmentContext, language string, force bool) *processor.SpecCollector {
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
				{http.MethodGet, pathItem.Get},
				{http.MethodPost, pathItem.Post},
				{http.MethodPut, pathItem.Put},
				{http.MethodDelete, pathItem.Delete},
				{http.MethodPatch, pathItem.Patch},
				{http.MethodHead, pathItem.Head},
				{http.MethodOptions, pathItem.Options},
			}

			for _, item := range operations {
				if item.op == nil {
					continue
				}

				// Only enrich if description is missing (or force is true)
				if !force && item.op.Summary != "" && item.op.Description != "" {
					collector.Skipped.APIs++
					continue
				}

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
						if force || op.Summary == "" {
							op.Summary = summary
						}
						if force || op.Description == "" {
							op.Description = description
						}
					},
				})
			}
		}
	}

	// Collect API parameters (grouped by endpoint)
	collectParameterGroups(spec, collector, language, force)

	// Collect schema fields
	processedSchemas := make(map[string]bool)
	collectSchemasFromSpec(spec, collector, processedSchemas, language, force)

	return collector
}

// collectParameterGroups collects parameters from API operations, grouped by endpoint.
func collectParameterGroups(spec *openapi3.T, collector *processor.SpecCollector, language string, force bool) {
	if spec.Paths == nil {
		return
	}

	for _, pathStr := range spec.Paths.InMatchingOrder() {
		pathItem := spec.Paths.Value(pathStr)
		if pathItem == nil {
			continue
		}

		operations := []struct {
			method string
			op     *openapi3.Operation
		}{
			{http.MethodGet, pathItem.Get},
			{http.MethodPost, pathItem.Post},
			{http.MethodPut, pathItem.Put},
			{http.MethodDelete, pathItem.Delete},
			{http.MethodPatch, pathItem.Patch},
			{http.MethodHead, pathItem.Head},
			{http.MethodOptions, pathItem.Options},
		}

		for _, item := range operations {
			if item.op == nil || len(item.op.Parameters) == 0 {
				continue
			}

			var params []processor.ParamFieldItem
			for _, paramRef := range item.op.Parameters {
				if paramRef.Value == nil {
					continue
				}
				if !force && paramRef.Value.Description != "" {
					collector.Skipped.Params++
					continue
				}
				param := paramRef.Value
				fieldType := ""
				if param.Schema != nil && param.Schema.Value != nil {
					fieldType = getSchemaTypeString(param.Schema.Value)
				}

				// Capture for closure
				p := param
				params = append(params, processor.ParamFieldItem{
					ParamName: param.Name,
					ParamIn:   param.In,
					FieldType: fieldType,
					Required:  param.Required,
					SetValue: func(desc string) {
						p.Description = desc
					},
				})
			}
			if len(params) > 0 {
				collector.AddParamGroupElement(processor.ParamGroupElement{
					Path:   pathStr,
					Method: item.method,
					Params: params,
				}, language)
			}
		}
	}
}

// getSchemaTypeString returns a string representation of a schema type.
func getSchemaTypeString(schema *openapi3.Schema) string {
	if schema.Type != nil && len(*schema.Type) > 0 {
		typeStr := (*schema.Type)[0]
		if schema.Format != "" {
			return typeStr + "(" + schema.Format + ")"
		}
		return typeStr
	}
	return "object"
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

// collectSchemasFromSpec collects schema fields from the OpenAPI spec
func collectSchemasFromSpec(spec *openapi3.T, collector *processor.SpecCollector, processed map[string]bool, language string, force bool) {
	if spec.Components == nil || spec.Components.Schemas == nil {
		return
	}

	for schemaName, schemaRef := range spec.Components.Schemas {
		if !processed[schemaName] {
			processor.CollectSchemaFields(schemaName, schemaRef, collector, processed, language, 0, force)
		}
	}
}
