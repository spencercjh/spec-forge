package enricher

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/spencercjh/spec-forge/internal/enricher/processor"
	"github.com/spencercjh/spec-forge/internal/enricher/prompt"
	"github.com/spencercjh/spec-forge/internal/enricher/provider"
	"github.com/spencercjh/spec-forge/internal/enricher/specctx"
)

// mockProvider for testing
type mockProvider struct {
	response string
	err      error
}

func (m *mockProvider) Generate(_ context.Context, _ string, _ ...provider.Option) (string, *provider.TokenUsage, error) {
	if m.err != nil {
		return "", nil, m.err
	}
	return m.response, nil, nil
}

func (m *mockProvider) Name() string {
	return "mock"
}

var _ provider.Provider = (*mockProvider)(nil)

func TestNewEnricher(t *testing.T) {
	cfg := Config{
		Provider: "openai",
		Model:    "gpt-4o",
	}
	cfg = cfg.MergeWithDefaults()

	enricher, err := NewEnricher(cfg, &mockProvider{response: "{}"})
	if err != nil {
		t.Fatalf("NewEnricher() error = %v", err)
	}
	if enricher == nil {
		t.Fatal("NewEnricher() returned nil")
	}
}

func TestEnricher_Enrich_EmptySpec(t *testing.T) {
	cfg := Config{
		Provider:    "openai",
		Model:       "gpt-4o",
		Concurrency: 1,
	}
	cfg = cfg.MergeWithDefaults()

	enricher, _ := NewEnricher(cfg, &mockProvider{response: `{"description": "test"}`})

	spec := &openapi3.T{}
	ctx := context.Background()

	result, err := enricher.Enrich(ctx, spec, nil)
	if err != nil {
		t.Fatalf("Enrich() error = %v", err)
	}
	if result == nil {
		t.Fatal("Enrich() returned nil spec")
	}
}

func TestEnricher_Enrich_APIOperations(t *testing.T) {
	cfg := Config{
		Provider:    "openai",
		Model:       "gpt-4o",
		Concurrency: 1,
	}
	cfg = cfg.MergeWithDefaults()

	enricher, _ := NewEnricher(cfg, &mockProvider{
		response: `{"summary": "Get users", "description": "Retrieves all users"}`,
	})

	// Create spec with API operation
	paths := openapi3.NewPaths()
	paths.Set("/users", &openapi3.PathItem{
		Get: &openapi3.Operation{
			Summary:     "", // Empty - should be enriched
			Description: "",
		},
	})
	spec := &openapi3.T{
		Paths: paths,
	}

	ctx := context.Background()
	result, err := enricher.Enrich(ctx, spec, nil)
	if err != nil {
		t.Fatalf("Enrich() error = %v", err)
	}

	// Check that operation was enriched
	op := result.Paths.Value("/users").Get
	if op.Summary == "" {
		t.Error("Expected operation summary to be enriched")
	}
}

func TestEnricher_Enrich_ProviderError(t *testing.T) {
	cfg := Config{
		Provider:    "openai",
		Model:       "gpt-4o",
		Concurrency: 1,
	}
	cfg = cfg.MergeWithDefaults()

	enricher, _ := NewEnricher(cfg, &mockProvider{
		err: errors.New("API error"),
	})

	paths := openapi3.NewPaths()
	paths.Set("/users", &openapi3.PathItem{
		Get: &openapi3.Operation{},
	})
	spec := &openapi3.T{
		Paths: paths,
	}

	ctx := context.Background()
	_, err := enricher.Enrich(ctx, spec, nil)

	if err == nil {
		t.Fatal("Expected error for provider failure")
	}

	// Should return PartialEnrichmentError
	if _, ok := errors.AsType[*processor.PartialEnrichmentError](err); !ok {
		t.Errorf("Expected PartialEnrichmentError, got %T", err)
	}
}

func TestEnricher_Enrich_WithOptions(t *testing.T) {
	cfg := Config{
		Provider:    "openai",
		Model:       "gpt-4o",
		Language:    "en",
		Concurrency: 1,
	}
	cfg = cfg.MergeWithDefaults()

	enricher, _ := NewEnricher(cfg, &mockProvider{
		response: `{"description": "test"}`,
	})

	spec := &openapi3.T{}
	ctx := context.Background()

	// Test with options
	opts := &EnrichOptions{
		Language: "zh",
	}

	result, err := enricher.Enrich(ctx, spec, opts)
	if err != nil {
		t.Fatalf("Enrich() error = %v", err)
	}
	if result == nil {
		t.Fatal("Enrich() returned nil")
	}
}

func TestEnricher_CollectParameters(t *testing.T) {
	paths := openapi3.NewPaths()
	paths.Set("/users/{id}", &openapi3.PathItem{
		Get: &openapi3.Operation{
			Parameters: openapi3.Parameters{
				&openapi3.ParameterRef{
					Value: &openapi3.Parameter{
						Name:     "id",
						In:       "path",
						Required: true,
						Schema:   &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"integer"}}},
					},
				},
			},
		},
	})
	spec := &openapi3.T{
		Paths: paths,
	}

	collector := &processor.SpecCollector{}
	collectParameterGroups(spec, collector, "en", false)

	// Verify parameter group was added as an enrichment element
	batches := collector.GroupByType()
	paramBatches := 0
	for _, batch := range batches {
		if batch.Type == "param" {
			paramBatches += len(batch.Elements)
		}
	}
	if paramBatches != 1 {
		t.Fatalf("expected 1 parameter group element, got %d", paramBatches)
	}

	// Test that the callback works by finding the param group element
	for _, batch := range batches {
		if batch.Type == "param" {
			for _, elem := range batch.Elements {
				if len(elem.ParamGroupFields) > 0 {
					elem.ParamGroupFields[0].SetValue("User ID")
					if spec.Paths.Value("/users/{id}").Get.Parameters[0].Value.Description != "User ID" {
						t.Errorf("expected description to be set via callback")
					}
				}
			}
		}
	}
}

func TestEnricher_CollectSchemaFields(t *testing.T) {
	spec := &openapi3.T{
		Components: &openapi3.Components{
			Schemas: openapi3.Schemas{
				"User": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: &openapi3.Types{"object"},
						Properties: openapi3.Schemas{
							"id":   &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"integer"}}},
							"name": &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"string"}}},
						},
					},
				},
			},
		},
	}

	collector := &processor.SpecCollector{}
	processed := make(map[string]bool)

	collectSchemasFromSpec(spec, collector, processed, "en", false)

	schemas := collector.GetSchemas()
	if len(schemas) != 1 {
		t.Errorf("expected 1 schema, got %d", len(schemas))
	}

	// Find User schema
	var userSchema *processor.SchemaElement
	for i := range schemas {
		if schemas[i].SchemaName == "User" {
			userSchema = &schemas[i]
			break
		}
	}

	if userSchema == nil {
		t.Fatal("User schema not found")
	}

	if len(userSchema.Fields) != 2 {
		t.Errorf("expected 2 fields in User schema, got %d", len(userSchema.Fields))
	}
}

func TestEnricher_WithStreaming(t *testing.T) {
	// Create a mock provider that simulates streaming
	chunks := []string{"Hello", " ", "World"}
	mockProvider := &mockStreamingProvider{
		chunks:   chunks,
		response: `{"summary": "Get test", "description": "Test endpoint"}`,
	}

	var buf bytes.Buffer
	cfg := Config{
		Provider:    "mock",
		Model:       "mock-model",
		Language:    "en",
		Concurrency: 1,
		Timeout:     5 * time.Second,
	}

	e, err := NewEnricher(cfg, mockProvider)
	require.NoError(t, err)

	paths := openapi3.NewPaths()
	paths.Set("/test", &openapi3.PathItem{
		Get: &openapi3.Operation{
			Summary:     "", // Empty to trigger enrichment
			Description: "",
		},
	})
	spec := &openapi3.T{
		Paths: paths,
	}

	streamEnabled := true
	_, err = e.Enrich(context.Background(), spec, &EnrichOptions{
		Language: "en",
		Stream:   &streamEnabled,
		Writer:   &buf,
	})
	require.NoError(t, err)

	// Verify streaming output was written
	// Note: TemplateType is lowercase, so prefix is "[api]" not "[API]"
	output := buf.String()
	assert.Contains(t, output, "[api]", "Expected [api] prefix in streaming output")
}

// mockStreamingProvider simulates streaming behavior
type mockStreamingProvider struct {
	chunks   []string
	response string
}

func (m *mockStreamingProvider) Generate(ctx context.Context, _ string, opts ...provider.Option) (string, *provider.TokenUsage, error) {
	// Apply options to get streaming config
	cfg := &provider.GenerateOptions{}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.StreamingFunc != nil {
		for _, chunk := range m.chunks {
			if err := cfg.StreamingFunc(ctx, []byte(chunk)); err != nil {
				return "", nil, err
			}
		}
	}
	return m.response, nil, nil
}

func (m *mockStreamingProvider) Name() string {
	return "mock-streaming"
}

// mockStreamingDisabledProvider tracks whether streaming was attempted
type mockStreamingDisabledProvider struct {
	response        string
	streamingCalled bool
}

func (m *mockStreamingDisabledProvider) Generate(_ context.Context, _ string, opts ...provider.Option) (string, *provider.TokenUsage, error) {
	cfg := &provider.GenerateOptions{}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.StreamingFunc != nil {
		m.streamingCalled = true
	}
	return m.response, nil, nil
}

func (m *mockStreamingDisabledProvider) Name() string {
	return "mock-streaming-disabled"
}

func TestEnricher_ForceFlag_CollectsAllElements(t *testing.T) {
	// Test that Force=true collects elements that would otherwise be skipped
	paths := openapi3.NewPaths()
	paths.Set("/users", &openapi3.PathItem{
		Get: &openapi3.Operation{
			Summary:     "Existing summary",
			Description: "Existing description",
		},
	})
	spec := &openapi3.T{
		Paths: paths,
	}

	// With force=false: collector should be empty (API already has descriptions)
	cfg := Config{
		Provider:    "openai",
		Model:       "gpt-4o",
		Concurrency: 1,
	}
	cfg = cfg.MergeWithDefaults()

	e, _ := NewEnricher(cfg, &mockProvider{response: `{"summary": "test", "description": "test"}`})

	// Force=false: should skip already-described API
	collector1 := e.collectElements(spec, &specctx.EnrichmentContext{}, "en", false)
	batches1 := collector1.GroupByType()
	// API should NOT be in batches (already has description)
	for _, b := range batches1 {
		if b.Type == prompt.TemplateTypeAPI {
			t.Error("expected no API batch when Force=false and descriptions exist")
		}
	}

	// Force=true: should include the API
	collector2 := e.collectElements(spec, &specctx.EnrichmentContext{}, "en", true)
	batches2 := collector2.GroupByType()
	found := false
	for _, b := range batches2 {
		if b.Type == prompt.TemplateTypeAPI {
			found = true
		}
	}
	if !found {
		t.Error("expected API batch when Force=true")
	}
}

func TestEnricher_WithStreamingDisabled(t *testing.T) {
	mockProvider := &mockStreamingDisabledProvider{
		response: `{"summary": "Get test", "description": "Test endpoint"}`,
	}

	var buf bytes.Buffer
	cfg := Config{
		Provider:    "mock",
		Model:       "mock-model",
		Language:    "en",
		Concurrency: 1,
		Timeout:     5 * time.Second,
	}

	e, err := NewEnricher(cfg, mockProvider)
	require.NoError(t, err)

	paths := openapi3.NewPaths()
	paths.Set("/test", &openapi3.PathItem{
		Get: &openapi3.Operation{
			Summary:     "",
			Description: "",
		},
	})
	spec := &openapi3.T{
		Paths: paths,
	}

	streamDisabled := false
	_, err = e.Enrich(context.Background(), spec, &EnrichOptions{
		Language: "en",
		Stream:   &streamDisabled, // Disable streaming
		Writer:   &buf,
	})
	require.NoError(t, err)

	// Verify streaming was NOT called
	assert.False(t, mockProvider.streamingCalled, "Streaming function should not be called when Stream: false")
	// Buffer should be empty since no streaming occurred
	assert.Empty(t, buf.String(), "Buffer should be empty when streaming is disabled")
}

func TestEnricher_CollectParameters_EnrichedContext(t *testing.T) {
	maxLen := uint64(50)
	paths := openapi3.NewPaths()
	paths.Set("/users", &openapi3.PathItem{
		Get: &openapi3.Operation{
			Parameters: openapi3.Parameters{
				&openapi3.ParameterRef{
					Value: &openapi3.Parameter{
						Name:     "status",
						In:       "query",
						Required: false,
						Schema: &openapi3.SchemaRef{Value: &openapi3.Schema{
							Type:      &openapi3.Types{"string"},
							Enum:      []any{"active", "inactive"},
							MaxLength: &maxLen,
						}},
					},
				},
			},
		},
	})
	spec := &openapi3.T{Paths: paths}

	collector := &processor.SpecCollector{}
	collectParameterGroups(spec, collector, "en", false)

	batches := collector.GroupByType()
	for _, batch := range batches {
		if batch.Type == prompt.TemplateTypeParam {
			for _, elem := range batch.Elements {
				if len(elem.Context.ParamFields) > 0 {
					pf := elem.Context.ParamFields[0]
					if len(pf.Enum) != 2 {
						t.Errorf("param enum count = %d, want 2", len(pf.Enum))
					}
					if pf.Constraints == "" {
						t.Error("param constraints should not be empty")
					}
				}
			}
		}
	}
}

func TestEnricher_CustomPrompts(t *testing.T) {
	var capturedPrompt string
	mp := &trackingMockProvider{
		response: `{"summary": "Custom", "description": "Custom desc"}`,
		capture:  &capturedPrompt,
	}

	cfg := Config{
		Provider:    "openai",
		Model:       "gpt-4o",
		Concurrency: 1,
		CustomPrompts: map[string]CustomPromptConfig{
			"api": {
				System: "Custom system prompt for {{.Language}}.",
				User:   "Custom user prompt: {{.Method}} {{.Path}}",
			},
		},
	}
	cfg = cfg.MergeWithDefaults()

	e, err := NewEnricher(cfg, mp)
	if err != nil {
		t.Fatalf("NewEnricher() error = %v", err)
	}

	paths := openapi3.NewPaths()
	paths.Set("/test", &openapi3.PathItem{
		Get: &openapi3.Operation{},
	})
	spec := &openapi3.T{Paths: paths}

	_, err = e.Enrich(context.Background(), spec, nil)
	if err != nil {
		t.Fatalf("Enrich() error = %v", err)
	}

	if !strings.Contains(capturedPrompt, "Custom system prompt") {
		t.Errorf("expected custom system prompt, got: %s", capturedPrompt)
	}
	if !strings.Contains(capturedPrompt, "Custom user prompt: GET /test") {
		t.Errorf("expected custom user prompt, got: %s", capturedPrompt)
	}
}

// trackingMockProvider captures the prompt sent to Generate.
type trackingMockProvider struct {
	response string
	capture  *string
}

func (m *trackingMockProvider) Generate(_ context.Context, p string, _ ...provider.Option) (string, *provider.TokenUsage, error) {
	*m.capture = p
	return m.response, nil, nil
}

func (m *trackingMockProvider) Name() string { return "tracking-mock" }

func TestEnricher_CollectElements_APITags(t *testing.T) {
	paths := openapi3.NewPaths()
	paths.Set("/users/{id}", &openapi3.PathItem{
		Get: &openapi3.Operation{
			Tags:    []string{"users", "admin"},
			Summary: "",
		},
	})
	spec := &openapi3.T{Paths: paths}

	cfg := Config{Provider: "openai", Model: "gpt-4o", Concurrency: 1}
	cfg = cfg.MergeWithDefaults()
	e, _ := NewEnricher(cfg, &mockProvider{response: `{"summary": "test"}`})

	collector := e.collectElements(spec, &specctx.EnrichmentContext{Schemas: make(map[string]*specctx.SchemaContext)}, "en", false)
	batches := collector.GroupByType()

	for _, batch := range batches {
		if batch.Type == prompt.TemplateTypeAPI {
			for _, elem := range batch.Elements {
				if elem.Context.Path == "/users/{id}" {
					if len(elem.Context.Tags) != 2 {
						t.Errorf("Tags = %d, want 2", len(elem.Context.Tags))
					}
				}
			}
		}
	}
}
