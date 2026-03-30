package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/spencercjh/spec-forge/internal/enricher/prompt"
	"github.com/spencercjh/spec-forge/internal/enricher/provider"
)

// BatchProcessor processes batches of elements
type BatchProcessor struct {
	provider     provider.Provider
	templateMgr  *prompt.TemplateManager
	streamWriter *StreamWriter // New: optional streaming writer
}

// BatchProcessorOption for configuring BatchProcessor
type BatchProcessorOption func(*BatchProcessor)

// WithStreamWriter sets streaming writer for BatchProcessor
func WithStreamWriter(sw *StreamWriter) BatchProcessorOption {
	return func(bp *BatchProcessor) {
		bp.streamWriter = sw
	}
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(p provider.Provider, tm *prompt.TemplateManager, opts ...BatchProcessorOption) *BatchProcessor {
	bp := &BatchProcessor{
		provider:    p,
		templateMgr: tm,
	}
	for _, opt := range opts {
		opt(bp)
	}
	return bp
}

// HasStreamWriter returns true if streaming output is configured
func (p *BatchProcessor) HasStreamWriter() bool {
	return p.streamWriter != nil
}

// ProcessBatch processes a single batch of elements
func (p *BatchProcessor) ProcessBatch(ctx context.Context, batch *Batch) error {
	tmpl, err := p.templateMgr.Get(batch.Type)
	if err != nil {
		return fmt.Errorf("failed to get template: %w", err)
	}

	for _, elem := range batch.Elements { //nolint:gocritic // copying elements is acceptable here
		systemPrompt, userPrompt, err := tmpl.Render(elem.Context)
		if err != nil {
			slog.Warn("failed to render template", "error", err, "path", elem.Path)
			continue
		}

		fullPrompt := systemPrompt + "\n\n" + userPrompt

		// Prepare options for provider
		var genOpts []provider.Option
		if p.streamWriter != nil {
			prefix := string(batch.Type) // e.g., "api", "schema", "param" (lowercase from TemplateType)
			genOpts = append(genOpts, provider.WithStreamingFunc(func(_ context.Context, chunk []byte) error {
				return p.streamWriter.WriteWithPrefix(prefix, chunk)
			}))
		}

		response, err := p.provider.Generate(ctx, fullPrompt, genOpts...)
		if err != nil {
			return fmt.Errorf("LLM call failed: %w", err)
		}

		// Flush any buffered streamed output after each LLM call completes
		// This ensures real-time output is visible even for short responses
		if p.streamWriter != nil {
			if flushErr := p.streamWriter.Flush(); flushErr != nil {
				slog.Warn("failed to flush stream writer", "error", flushErr)
			}
		}

		// Handle schema-specific response parsing
		if batch.Type == prompt.TemplateTypeSchema && len(elem.SchemaFields) > 0 {
			fieldDescriptions := parseSchemaResponse(response)
			for _, field := range elem.SchemaFields {
				if desc, ok := fieldDescriptions[field.FieldName]; ok {
					field.SetValue(desc)
				}
			}
		} else {
			description := parseDescriptionResponse(response)
			elem.SetValue(description)
		}
	}

	return nil
}

// parseDescriptionResponse extracts description from LLM response
func parseDescriptionResponse(response string) string {
	// Strip markdown code blocks if present
	response = stripMarkdownCodeBlock(response)

	var result map[string]string
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return response // Return as-is if not JSON
	}

	// Combine summary and description if both exist
	if summary, ok := result["summary"]; ok {
		if desc, ok := result["description"]; ok && desc != "" {
			return summary + "\n\n" + desc
		}
		return summary
	}

	if desc, ok := result["description"]; ok {
		return desc
	}

	return response
}

// stripMarkdownCodeBlock removes markdown code block wrappers
func stripMarkdownCodeBlock(s string) string {
	// Match ```json ... ``` or ``` ... ```
	re := regexp.MustCompile("(?s)^```(?:json)?\\s*\n?(.*?)\\s*```$")
	matches := re.FindStringSubmatch(strings.TrimSpace(s))
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return s
}

// parseSchemaResponse parses a schema field description response.
// Returns a map of field names to descriptions.
func parseSchemaResponse(response string) map[string]string {
	response = stripMarkdownCodeBlock(response)

	var result map[string]string
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		slog.Warn("failed to parse schema response as JSON", "error", err, "response", response)
		return nil
	}

	return result
}
