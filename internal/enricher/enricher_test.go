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

	cfg := &processor.SpecCollector{}
	collectParametersFromSpec(spec, cfg, "en")

	params := cfg.GetParams()
	if len(params) != 1 {
		t.Fatalf("expected 1 parameter, got %d", len(params))
	}
	if params[0].ParamName != "id" {
		t.Errorf("expected param name 'id', got %s", params[0].ParamName)
	}
	if params[0].ParamIn != "path" {
		t.Errorf("expected param in 'path', got %s", params[0].ParamIn)
	}
	// Test SetValue callback
	params[0].SetValue("User ID")
	if spec.Paths.Value("/users/{id}").Get.Parameters[0].Value.Description != "User ID" {
		t.Errorf("expected description to be set via callback")
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

	collectSchemasFromSpec(spec, collector, processed, "en")

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
