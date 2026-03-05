package processor

import (
	"testing"

	"github.com/spencercjh/spec-forge/internal/enricher/prompt"
)

func TestEnrichmentElement_Fields(t *testing.T) {
	var setValueCalled bool
	element := EnrichmentElement{
		Type:    prompt.TemplateTypeAPI,
		Path:    "GET /users/{id}",
		Context: prompt.TemplateContext{Method: "GET", Path: "/users/{id}"},
		SetValue: func(description string) {
			setValueCalled = true
			if description != "test description" {
				t.Errorf("SetValue called with %q, want %q", description, "test description")
			}
		},
	}

	if element.Type != prompt.TemplateTypeAPI {
		t.Errorf("Type = %v, want %v", element.Type, prompt.TemplateTypeAPI)
	}

	// Test SetValue callback
	element.SetValue("test description")
	if !setValueCalled {
		t.Error("SetValue callback was not called")
	}
}

func TestBatch_Type(t *testing.T) {
	batch := &Batch{
		Type: prompt.TemplateTypeSchema,
		Elements: []EnrichmentElement{
			{Type: prompt.TemplateTypeSchema},
			{Type: prompt.TemplateTypeSchema},
		},
	}

	if batch.Type != prompt.TemplateTypeSchema {
		t.Errorf("Type = %v, want %v", batch.Type, prompt.TemplateTypeSchema)
	}
	if len(batch.Elements) != 2 {
		t.Errorf("Elements count = %d, want %d", len(batch.Elements), 2)
	}
}

func TestSpecCollector_APIOperations(t *testing.T) {
	collector := &SpecCollector{}

	// Test collecting API operation
	element := EnrichmentElement{
		Type: prompt.TemplateTypeAPI,
		Path: "GET /users",
	}

	collector.AddElement(element)

	if len(collector.elements) != 1 {
		t.Errorf("elements count = %d, want %d", len(collector.elements), 1)
	}
}

func TestSpecCollector_GroupByType(t *testing.T) {
	collector := &SpecCollector{}

	// Add different types of elements
	collector.AddElement(EnrichmentElement{Type: prompt.TemplateTypeAPI, Path: "GET /users"})
	collector.AddElement(EnrichmentElement{Type: prompt.TemplateTypeAPI, Path: "POST /users"})
	collector.AddElement(EnrichmentElement{Type: prompt.TemplateTypeSchema, Path: "User"})
	collector.AddElement(EnrichmentElement{Type: prompt.TemplateTypeParam, Path: "GET /users"})

	batches := collector.GroupByType()

	// Should have 3 batches (2 APIs grouped, 1 schema, 1 param)
	if len(batches) != 3 {
		t.Errorf("batches count = %d, want %d", len(batches), 3)
	}

	// Find API batch
	var apiBatch *Batch
	for i := range batches {
		if batches[i].Type == prompt.TemplateTypeAPI {
			apiBatch = batches[i]
			break
		}
	}

	if apiBatch == nil {
		t.Fatal("API batch not found")
	}
	if len(apiBatch.Elements) != 2 {
		t.Errorf("API batch elements = %d, want %d", len(apiBatch.Elements), 2)
	}
}

func TestSchemaElement(t *testing.T) {
	schema := SchemaElement{
		SchemaName: "User",
		Fields: []FieldElement{
			{FieldName: "id", FieldType: "integer", Required: true},
			{FieldName: "name", FieldType: "string", Required: false},
		},
	}

	if schema.SchemaName != "User" {
		t.Errorf("expected SchemaName 'User', got %s", schema.SchemaName)
	}
	if len(schema.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(schema.Fields))
	}
}

func TestFieldElement(t *testing.T) {
	var setCalled bool
	field := FieldElement{
		FieldName: "userId",
		FieldType: "string",
		Required:  true,
		SetValue: func(_ string) {
			setCalled = true
		},
	}

	field.SetValue("test description")
	if !setCalled {
		t.Error("expected SetValue callback to be called")
	}
}

func TestSpecCollector_AddSchemaElement(t *testing.T) {
	collector := &SpecCollector{}

	schema := SchemaElement{
		SchemaName: "User",
		Fields: []FieldElement{
			{FieldName: "id", FieldType: "integer", Required: true},
		},
	}

	collector.AddSchemaElement(schema, "en")

	if len(collector.schemas) != 1 {
		t.Errorf("expected 1 schema, got %d", len(collector.schemas))
	}
}

func TestSpecCollector_AddParamElement(t *testing.T) {
	collector := &SpecCollector{}

	var received string
	collector.AddParamElement("/users/{id}", "GET", "id", "path", "string", true, "en", func(desc string) {
		received = desc
	})

	if len(collector.params) != 1 {
		t.Errorf("expected 1 param, got %d", len(collector.params))
	}

	// Test SetValue callback
	collector.params[0].SetValue("test description")
	if received != "test description" {
		t.Errorf("expected 'test description', got '%s'", received)
	}
}
