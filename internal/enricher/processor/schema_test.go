package processor_test

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/spencercjh/spec-forge/internal/enricher/processor"
)

func TestCollectSchemaFields_SimpleSchema(t *testing.T) {
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
						Required: []string{"id"},
					},
				},
			},
		},
	}

	collector := &processor.SpecCollector{}
	processed := make(map[string]bool)

	processor.CollectSchemaFields("User", spec.Components.Schemas["User"], collector, processed, "en", 0)

	if len(processed) != 1 {
		t.Errorf("expected 1 processed schema, got %d", len(processed))
	}
	if !processed["User"] {
		t.Error("expected User to be processed")
	}
}

func TestCollectSchemaFields_NestedSchema(t *testing.T) {
	spec := &openapi3.T{
		Components: &openapi3.Components{
			Schemas: openapi3.Schemas{
				"Order": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: &openapi3.Types{"object"},
						Properties: openapi3.Schemas{
							"id": &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"integer"}}},
							"customer": &openapi3.SchemaRef{
								Value: &openapi3.Schema{
									Type: &openapi3.Types{"object"},
									Properties: openapi3.Schemas{
										"name":  &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"string"}}},
										"email": &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"string"}}},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	collector := &processor.SpecCollector{}
	processed := make(map[string]bool)

	processor.CollectSchemaFields("Order", spec.Components.Schemas["Order"], collector, processed, "en", 0)

	// Should process both Order and nested Order.customer
	if len(processed) < 2 {
		t.Errorf("expected at least 2 processed schemas, got %d", len(processed))
	}
}

func TestCollectSchemaFields_MaxDepth(t *testing.T) {
	// Create deeply nested schema
	nestedSchema := &openapi3.Schema{
		Type:       &openapi3.Types{"object"},
		Properties: openapi3.Schemas{},
	}
	nestedSchema.Properties["level"] = &openapi3.SchemaRef{Value: nestedSchema}

	spec := &openapi3.T{
		Components: &openapi3.Components{
			Schemas: openapi3.Schemas{
				"Deep": &openapi3.SchemaRef{Value: nestedSchema},
			},
		},
	}

	collector := &processor.SpecCollector{}
	processed := make(map[string]bool)

	processor.CollectSchemaFields("Deep", spec.Components.Schemas["Deep"], collector, processed, "en", 0)

	// Should stop at max depth (5)
	if len(processed) > 6 {
		t.Errorf("expected max 6 processed schemas (depth 5), got %d", len(processed))
	}
}

func TestCollectSchemaFields_SkipFieldsWithDescription(t *testing.T) {
	spec := &openapi3.T{
		Components: &openapi3.Components{
			Schemas: openapi3.Schemas{
				"User": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: &openapi3.Types{"object"},
						Properties: openapi3.Schemas{
							"id":          &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"integer"}}},
							"name":        &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"string"}, Description: "Already described"}},
							"description": &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"string"}}},
						},
					},
				},
			},
		},
	}

	collector := &processor.SpecCollector{}
	processed := make(map[string]bool)

	processor.CollectSchemaFields("User", spec.Components.Schemas["User"], collector, processed, "en", 0)

	// Find the User schema
	schemas := collector.GetSchemas()
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

	// Should have 2 fields (id and description), not 3 (name already has description)
	if len(userSchema.Fields) != 2 {
		t.Errorf("expected 2 fields (id, description), got %d", len(userSchema.Fields))
	}
}
