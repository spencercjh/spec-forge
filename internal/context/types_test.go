// internal/context/types_test.go
package context_test

import (
	"testing"

	"github.com/spencercjh/spec-forge/internal/context"
)

func TestEnrichmentContext(t *testing.T) {
	ctx := &context.EnrichmentContext{
		ProjectName: "test-project",
		Framework:   "spring-boot",
		Schemas:     make(map[string]*context.SchemaContext),
	}

	if ctx.ProjectName != "test-project" {
		t.Errorf("expected ProjectName 'test-project', got %s", ctx.ProjectName)
	}
	if ctx.Framework != "spring-boot" {
		t.Errorf("expected Framework 'spring-boot', got %s", ctx.Framework)
	}
}

func TestSchemaContext(t *testing.T) {
	schemaCtx := &context.SchemaContext{
		Name:        "User",
		Description: "A user entity",
		Package:     "com.example.model",
		Fields: []context.FieldMeta{
			{Name: "id", Type: "integer", Required: true},
			{Name: "name", Type: "string", Required: false},
		},
	}

	if len(schemaCtx.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(schemaCtx.Fields))
	}
}

func TestFieldMeta(t *testing.T) {
	field := context.FieldMeta{
		Name:        "userId",
		Type:        "string",
		Required:    true,
		Description: "The unique identifier",
		Tags:        map[string]string{"json": "user_id"},
	}

	if field.Name != "userId" {
		t.Errorf("expected Name 'userId', got %s", field.Name)
	}
	if field.Tags["json"] != "user_id" {
		t.Errorf("expected json tag 'user_id', got %s", field.Tags["json"])
	}
}
