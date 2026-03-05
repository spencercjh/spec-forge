package enricher

import (
	"context"
	"errors"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/spencercjh/spec-forge/internal/enricher/processor"
	"github.com/spencercjh/spec-forge/internal/enricher/provider"
)

// mockProvider for testing
type mockProvider struct {
	response string
	err      error
}

func (m *mockProvider) Generate(ctx context.Context, prompt string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
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
	var partialErr *processor.PartialEnrichmentError
	if !errors.As(err, &partialErr) {
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
