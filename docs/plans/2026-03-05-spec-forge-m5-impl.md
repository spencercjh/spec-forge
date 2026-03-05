# M5: Schema Field and API Parameter Enrichment Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Extend the Enricher to add AI-generated descriptions for Schema fields and API parameters.

**Architecture:** Create a new `internal/context` package for extensible context extraction. Extend `internal/enricher/processor` to collect schemas and parameters. Modify `enricher.go` to recursively collect all elements. Use hybrid batch strategy (≤10 fields per batch).

**Tech Stack:** Go 1.26, kin-openapi, langchaingo

---

## Task 1: Create Context Package - Types

**Files:**
- Create: `internal/context/types.go`
- Test: `internal/context/types_test.go`

**Step 1: Write the failing test**

```go
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/context/... -v`
Expected: FAIL - package context not found

**Step 3: Create the types file**

```go
// internal/context/types.go
// Package context provides context extraction for OpenAPI spec enrichment.
package context

// EnrichmentContext contains context information for LLM enrichment.
type EnrichmentContext struct {
	// Project-level information
	ProjectName string
	Framework   string // "spring-boot", "gin", etc.

	// Schema contexts
	Schemas map[string]*SchemaContext // key: schema name
}

// SchemaContext contains context for a single Schema.
type SchemaContext struct {
	// From OpenAPI Spec
	Name   string
	Fields []FieldMeta

	// Extended information (filled by ContextExtractor)
	Description string            // Class/struct documentation
	Package     string            // Package name
	Annotations map[string]string // Language-specific annotations
}

// FieldMeta contains field metadata.
type FieldMeta struct {
	Name     string
	Type     string
	Required bool

	// Extended information
	Description string            // Field documentation
	Tags        map[string]string // e.g., json:"user_id", validate:"required"
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/context/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/context/types.go internal/context/types_test.go
git commit -m "$(cat <<'EOF'
feat(context): add EnrichmentContext types for schema field enrichment

- Add EnrichmentContext for project-level context
- Add SchemaContext for schema metadata
- Add FieldMeta for field-level information

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Create Context Package - Extractor Interface

**Files:**
- Create: `internal/context/extractor.go`
- Test: `internal/context/extractor_test.go`

**Step 1: Write the failing test**

```go
// internal/context/extractor_test.go
package context_test

import (
	"context"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/spencercjh/spec-forge/internal/context"
)

func TestContextExtractor_Interface(t *testing.T) {
	// Verify interface compliance
	var _ context.ContextExtractor = (*context.NoOpExtractor)(nil)
}

func TestNoOpExtractor_Name(t *testing.T) {
	extractor := &context.NoOpExtractor{}
	if extractor.Name() != "noop" {
		t.Errorf("expected Name 'noop', got %s", extractor.Name())
	}
}

func TestNoOpExtractor_Extract(t *testing.T) {
	extractor := &context.NoOpExtractor{}

	spec := &openapi3.T{
		Components: &openapi3.Components{
			Schemas: openapi3.Schemas{
				"User": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: "object",
						Properties: openapi3.Schemas{
							"id":   &openapi3.SchemaRef{Value: &openapi3.Schema{Type: "integer"}},
							"name": &openapi3.SchemaRef{Value: &openapi3.Schema{Type: "string"}},
						},
					},
				},
			},
		},
	}

	ctx := context.Background()
	result, err := extractor.Extract(ctx, "/dummy/path", spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Should have extracted the User schema
	if len(result.Schemas) != 1 {
		t.Errorf("expected 1 schema, got %d", len(result.Schemas))
	}

	userSchema, ok := result.Schemas["User"]
	if !ok {
		t.Fatal("expected User schema in result")
	}

	if len(userSchema.Fields) != 2 {
		t.Errorf("expected 2 fields in User schema, got %d", len(userSchema.Fields))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/context/... -v`
Expected: FAIL - ContextExtractor and NoOpExtractor not defined

**Step 3: Create the extractor interface**

```go
// internal/context/extractor.go
package context

import (
	"context"

	"github.com/getkin/kin-openapi/openapi3"
)

// ContextExtractor extracts context information from source code.
type ContextExtractor interface {
	// Extract extracts context from a project.
	// The spec parameter is the generated OpenAPI spec, used as a skeleton.
	Extract(ctx context.Context, projectPath string, spec *openapi3.T) (*EnrichmentContext, error)

	// Name returns the extractor name.
	Name() string
}
```

**Step 4: Run test to verify it fails**

Run: `go test ./internal/context/... -v`
Expected: FAIL - NoOpExtractor not defined

**Step 5: Create the NoOpExtractor**

```go
// internal/context/noop_extractor.go
package context

import (
	"context"
	"log/slog"

	"github.com/getkin/kin-openapi/openapi3"
)

// NoOpExtractor is a default implementation that extracts basic info from the spec only.
// It does not perform source code analysis.
type NoOpExtractor struct{}

// Name returns the extractor name.
func (e *NoOpExtractor) Name() string {
	return "noop"
}

// Extract extracts context from the OpenAPI spec without source code analysis.
func (e *NoOpExtractor) Extract(_ context.Context, _ string, spec *openapi3.T) (*EnrichmentContext, error) {
	if spec == nil {
		slog.Debug("NoOpExtractor: spec is nil, returning empty context")
		return &EnrichmentContext{Schemas: make(map[string]*SchemaContext)}, nil
	}

	result := &EnrichmentContext{
		Schemas: make(map[string]*SchemaContext),
	}

	// Extract schemas from components
	if spec.Components != nil && spec.Components.Schemas != nil {
		for schemaName, schemaRef := range spec.Components.Schemas {
			if schemaRef.Value == nil {
				continue
			}

			schemaCtx := &SchemaContext{
				Name:   schemaName,
				Fields: extractFieldsFromSchema(schemaRef.Value),
			}
			result.Schemas[schemaName] = schemaCtx
		}
	}

	slog.Debug("NoOpExtractor: extracted context from spec", "schemas", len(result.Schemas))
	return result, nil
}

// extractFieldsFromSchema extracts field metadata from an OpenAPI schema.
func extractFieldsFromSchema(schema *openapi3.Schema) []FieldMeta {
	if schema.Properties == nil {
		return nil
	}

	var fields []FieldMeta
	for propName, propRef := range schema.Properties {
		if propRef.Value == nil {
			continue
		}

		field := FieldMeta{
			Name:     propName,
			Type:     getSchemaTypeName(propRef.Value),
			Required: containsString(schema.Required, propName),
		}
		fields = append(fields, field)
	}

	return fields
}

// getSchemaTypeName returns a human-readable type name for a schema.
func getSchemaTypeName(schema *openapi3.Schema) string {
	if schema.Type != "" {
		if schema.Type == "array" && schema.Items != nil && schema.Items.Value != nil {
			return "array of " + getSchemaTypeName(schema.Items.Value)
		}
		return schema.Type
	}
	if schema.Ref != "" {
		// Extract name from $ref
		return schema.Ref
	}
	return "object"
}

// containsString checks if a string is in a slice.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
```

**Step 6: Run test to verify it passes**

Run: `go test ./internal/context/... -v`
Expected: PASS

**Step 7: Commit**

```bash
git add internal/context/extractor.go internal/context/noop_extractor.go internal/context/extractor_test.go
git commit -m "$(cat <<'EOF'
feat(context): add ContextExtractor interface and NoOpExtractor

- Define ContextExtractor interface for extensible context extraction
- Implement NoOpExtractor that extracts basic info from OpenAPI spec
- Add helper functions for schema type name extraction

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Extend Processor - Schema Collector Types

**Files:**
- Modify: `internal/enricher/processor/processor.go`
- Test: `internal/enricher/processor/processor_test.go`

**Step 1: Write the failing test**

```go
// Add to internal/enricher/processor/processor_test.go

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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/enricher/processor/... -v -run TestSchemaElement`
Expected: FAIL - SchemaElement and FieldElement not defined

**Step 3: Add SchemaElement and FieldElement types**

```go
// Add to internal/enricher/processor/processor.go

// SchemaElement represents a schema to be enriched.
type SchemaElement struct {
	SchemaName string
	Fields     []FieldElement
	Context    prompt.TemplateContext
}

// FieldElement represents a field to be enriched.
type FieldElement struct {
	FieldName string
	FieldType string
	Required  bool
	SetValue  func(description string)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/enricher/processor/... -v -run TestSchemaElement`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/enricher/processor/processor.go internal/enricher/processor/processor_test.go
git commit -m "$(cat <<'EOF'
feat(processor): add SchemaElement and FieldElement types for schema enrichment

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Extend Processor - SpecCollector Schema Methods

**Files:**
- Modify: `internal/enricher/processor/processor.go`
- Test: `internal/enricher/processor/processor_test.go`

**Step 1: Write the failing test**

```go
// Add to internal/enricher/processor/processor_test.go

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

	collector.AddParamElement("/users/{id}", "GET", "id", "path", "string", true, "en")

	if len(collector.params) != 1 {
		t.Errorf("expected 1 param, got %d", len(collector.params))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/enricher/processor/... -v -run TestSpecCollector`
Expected: FAIL - AddSchemaElement and AddParamElement not defined, schemas/params fields not defined

**Step 3: Extend SpecCollector**

```go
// Modify internal/enricher/processor/processor.go

// SpecCollector collects elements from an OpenAPI spec for enrichment.
type SpecCollector struct {
	elements []EnrichmentElement
	schemas  []SchemaElement
	params   []ParamElement
}

// ParamElement represents an API parameter to be enriched.
type ParamElement struct {
	Path      string
	Method    string
	ParamName string
	ParamIn   string // path, query, header, cookie
	FieldType string
	Required  bool
	SetValue  func(description string)
}

// AddSchemaElement adds a schema element to the collector.
func (c *SpecCollector) AddSchemaElement(schema SchemaElement, language string) { //nolint:gocritic
	c.schemas = append(c.schemas, schema)

	// Also add as an enrichment element for processing
	c.elements = append(c.elements, EnrichmentElement{
		Type: prompt.TemplateTypeSchema,
		Path: "#/components/schemas/" + schema.SchemaName,
		Context: prompt.TemplateContext{
			Type:       prompt.TemplateTypeSchema,
			Language:   language,
			SchemaName: schema.SchemaName,
			Fields:     convertFieldElements(schema.Fields),
		},
		SetValue: func(_ string) {
			// Schema SetValue is handled by individual field SetValue callbacks
		},
	})
}

// AddParamElement adds a parameter element to the collector.
func (c *SpecCollector) AddParamElement(path, method, paramName, paramIn, fieldType string, required bool, language string) {
	c.params = append(c.params, ParamElement{
		Path:      path,
		Method:    method,
		ParamName: paramName,
		ParamIn:   paramIn,
		FieldType: fieldType,
		Required:  required,
	})
}

// convertFieldElements converts FieldElement slice to FieldContext slice.
func convertFieldElements(fields []FieldElement) []prompt.FieldContext {
	result := make([]prompt.FieldContext, len(fields))
	for i, f := range fields {
		result[i] = prompt.FieldContext{
			Name:     f.FieldName,
			Type:     f.FieldType,
			Required: f.Required,
		}
	}
	return result
}
```

**Step 4: Update prompt/templates.go to add ParamIn field**

```go
// Modify internal/enricher/prompt/templates.go

// TemplateContext provides context for template rendering
type TemplateContext struct {
	Type     TemplateType
	Language string

	// API context
	Path   string
	Method string

	// Schema context
	SchemaName string
	Fields     []FieldContext

	// Field context
	FieldName string
	FieldType string
	Required  bool

	// Parameter context
	ParamName string
	ParamIn   string // path, query, header, cookie

	// Response context
	ResponseCode string
}
```

**Step 5: Run test to verify it passes**

Run: `go test ./internal/enricher/processor/... -v -run TestSpecCollector`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/enricher/processor/processor.go internal/enricher/processor/processor_test.go internal/enricher/prompt/templates.go
git commit -m "$(cat <<'EOF'
feat(processor): extend SpecCollector with schema and param collection

- Add AddSchemaElement method for schema collection
- Add AddParamElement method for API parameter collection
- Add ParamElement type for parameter enrichment
- Add ParamIn field to TemplateContext

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Extend Processor - Schema Recursive Collection

**Files:**
- Create: `internal/enricher/processor/schema.go`
- Test: `internal/enricher/processor/schema_test.go`

**Step 1: Write the failing test**

```go
// internal/enricher/processor/schema_test.go
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
						Type: "object",
						Properties: openapi3.Schemas{
							"id":   &openapi3.SchemaRef{Value: &openapi3.Schema{Type: "integer"}},
							"name": &openapi3.SchemaRef{Value: &openapi3.Schema{Type: "string"}},
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
						Type: "object",
						Properties: openapi3.Schemas{
							"id": &openapi3.SchemaRef{Value: &openapi3.Schema{Type: "integer"}},
							"customer": &openapi3.SchemaRef{
								Value: &openapi3.Schema{
									Type: "object",
									Properties: openapi3.Schemas{
										"name":  &openapi3.SchemaRef{Value: &openapi3.Schema{Type: "string"}},
										"email": &openapi3.SchemaRef{Value: &openapi3.Schema{Type: "string"}},
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
		Type:       "object",
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
						Type: "object",
						Properties: openapi3.Schemas{
							"id":          &openapi3.SchemaRef{Value: &openapi3.Schema{Type: "integer"}},
							"name":        &openapi3.SchemaRef{Value: &openapi3.Schema{Type: "string", Description: "Already described"}},
							"description": &openapi3.SchemaRef{Value: &openapi3.Schema{Type: "string"}},
						},
					},
				},
			},
		},
	}

	collector := &processor.SpecCollector{}
	processed := make(map[string]bool)

	processor.CollectSchemaFields("User", spec.Components.Schemas["User"], collector, processed, "en", 0)

	// Should only collect id and description (name already has description)
	schemas := collector.GetSchemas()
	if len(schemas) != 1 {
		t.Fatalf("expected 1 schema, got %d", len(schemas))
	}

	// Find the User schema
	var userSchema *processor.SchemaElement
	for _, s := range schemas {
		if s.SchemaName == "User" {
			userSchema = &s
			break
		}
	}

	if userSchema == nil {
		t.Fatal("User schema not found")
	}

	// Should have 2 fields (id and description), not 3
	if len(userSchema.Fields) != 2 {
		t.Errorf("expected 2 fields (id, description), got %d", len(userSchema.Fields))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/enricher/processor/... -v -run TestCollectSchemaFields`
Expected: FAIL - CollectSchemaFields not defined

**Step 3: Create the schema collector**

```go
// internal/enricher/processor/schema.go
package processor

import (
	"log/slog"

	"github.com/getkin/kin-openapi/openapi3"
)

// MaxSchemaDepth is the maximum recursion depth for nested schemas.
const MaxSchemaDepth = 5

// CollectSchemaFields recursively collects fields from a schema.
func CollectSchemaFields(
	schemaName string,
	schemaRef *openapi3.SchemaRef,
	collector *SpecCollector,
	processed map[string]bool,
	language string,
	depth int,
) {
	// Prevent infinite recursion
	if depth > MaxSchemaDepth {
		slog.Warn("max schema depth reached", "schema", schemaName, "depth", depth)
		return
	}

	// Prevent duplicate processing
	if processed[schemaName] {
		return
	}
	processed[schemaName] = true

	schema := schemaRef.Value
	if schema == nil {
		return
	}

	// Collect fields from properties
	var fields []FieldElement
	for propName, propRef := range schema.Properties {
		prop := propRef.Value
		if prop == nil {
			continue
		}

		// Skip fields that already have descriptions
		if prop.Description != "" {
			continue
		}

		field := FieldElement{
			FieldName: propName,
			FieldType: getSchemaTypeName(prop),
			Required:  containsString(schema.Required, propName),
		}

		// Capture prop for closure
		propPtr := prop
		field.SetValue = func(desc string) {
			propPtr.Description = desc
		}
		fields = append(fields, field)

		// Recursively process nested objects
		if prop.Type == "object" && prop.Properties != nil {
			nestedName := schemaName + "." + propName
			CollectSchemaFields(nestedName, propRef, collector, processed, language, depth+1)
		}

		// Recursively process array items
		if prop.Type == "array" && prop.Items != nil {
			nestedName := schemaName + "." + propName + "[]"
			CollectSchemaFields(nestedName, prop.Items, collector, processed, language, depth+1)
		}
	}

	// Only add if there are fields to enrich
	if len(fields) > 0 {
		collector.AddSchemaElement(SchemaElement{
			SchemaName: schemaName,
			Fields:     fields,
		}, language)
	}
}

// getSchemaTypeName returns a human-readable type name.
func getSchemaTypeName(schema *openapi3.Schema) string {
	if schema.Type != "" {
		if schema.Type == "array" && schema.Items != nil && schema.Items.Value != nil {
			return "array of " + getSchemaTypeName(schema.Items.Value)
		}
		return schema.Type
	}
	if schema.Ref != "" {
		return schema.Ref
	}
	return "object"
}

// containsString checks if a string is in a slice.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// GetSchemas returns collected schemas.
func (c *SpecCollector) GetSchemas() []SchemaElement {
	return c.schemas
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/enricher/processor/... -v -run TestCollectSchemaFields`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/enricher/processor/schema.go internal/enricher/processor/schema_test.go
git commit -m "$(cat <<'EOF'
feat(processor): add recursive schema field collection

- Implement CollectSchemaFields for recursive schema traversal
- Support nested objects and arrays
- Enforce max depth of 5 to prevent infinite recursion
- Skip fields that already have descriptions

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Extend Batch Processor - Schema Response Parsing

**Files:**
- Modify: `internal/enricher/processor/batch.go`
- Test: `internal/enricher/processor/batch_test.go`

**Step 1: Write the failing test**

```go
// Add to internal/enricher/processor/batch_test.go

func TestParseSchemaResponse(t *testing.T) {
	tests := []struct {
		name         string
		response     string
		fieldCount   int
		expectFields map[string]string
	}{
		{
			name: "simple JSON",
			response: `{
				"id": "The unique identifier of the user",
				"name": "The display name of the user",
				"email": "The user's email address"
			}`,
			fieldCount: 3,
			expectFields: map[string]string{
				"id":    "The unique identifier of the user",
				"name":  "The display name of the user",
				"email": "The user's email address",
			},
		},
		{
			name: "JSON in markdown code block",
			response: "```json\n{\"userId\": \"The user ID\", \"status\": \"The account status\"}\n```",
			fieldCount: 2,
			expectFields: map[string]string{
				"userId": "The user ID",
				"status": "The account status",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseSchemaResponse(tt.response)
			if len(result) != tt.fieldCount {
				t.Errorf("expected %d fields, got %d", tt.fieldCount, len(result))
			}
			for name, expectedDesc := range tt.expectFields {
				if result[name] != expectedDesc {
					t.Errorf("field %s: expected %q, got %q", name, expectedDesc, result[name])
				}
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/enricher/processor/... -v -run TestParseSchemaResponse`
Expected: FAIL - parseSchemaResponse not defined

**Step 3: Add schema response parser**

```go
// Add to internal/enricher/processor/batch.go

// parseSchemaResponse parses a schema field description response.
func parseSchemaResponse(response string) map[string]string {
	response = stripMarkdownCodeBlock(response)

	var result map[string]string
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		slog.Warn("failed to parse schema response as JSON", "error", err)
		return nil
	}

	return result
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/enricher/processor/... -v -run TestParseSchemaResponse`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/enricher/processor/batch.go internal/enricher/processor/batch_test.go
git commit -m "$(cat <<'EOF'
feat(processor): add parseSchemaResponse for schema field descriptions

- Parse JSON response mapping field names to descriptions
- Support markdown code block wrapped JSON
- Log warning on parse failure

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Extend Batch Processor - Schema Batch Processing

**Files:**
- Modify: `internal/enricher/processor/batch.go`

**Step 1: Write the failing test**

```go
// Add to internal/enricher/processor/batch_test.go

func TestBatchProcessor_ProcessSchemaBatch(t *testing.T) {
	mockProvider := &MockProvider{
		GenerateFunc: func(_ context.Context, prompt string) (string, error) {
			// Return mock schema field descriptions
			if strings.Contains(prompt, "User") {
				return `{"id": "User ID", "name": "User name"}`, nil
			}
			return "{}", nil
		},
	}

	tmplMgr := prompt.NewTemplateManager()
	bp := NewBatchProcessor(mockProvider, tmplMgr)

	fields := []FieldElement{
		{FieldName: "id", FieldType: "integer", Required: true},
		{FieldName: "name", FieldType: "string", Required: false},
	}

	var setValues []string
	for i := range fields {
		f := &fields[i]
		f.SetValue = func(desc string) {
			setValues = append(setValues, f.FieldName+":"+desc)
		}
	}

	batch := &Batch{
		Type: prompt.TemplateTypeSchema,
		Elements: []EnrichmentElement{
			{
				Type: prompt.TemplateTypeSchema,
				Path: "#/components/schemas/User",
				Context: prompt.TemplateContext{
					Type:       prompt.TemplateTypeSchema,
					Language:   "en",
					SchemaName: "User",
					Fields:     convertFieldElements(fields),
				},
				SchemaFields: fields,
			},
		},
	}

	ctx := context.Background()
	err := bp.ProcessBatch(ctx, batch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(setValues) != 2 {
		t.Errorf("expected 2 SetValue calls, got %d", len(setValues))
	}
}
```

**Step 2: Modify EnrichmentElement to support SchemaFields**

```go
// Modify internal/enricher/processor/processor.go

// EnrichmentElement represents an element to be enriched
type EnrichmentElement struct {
	Type     prompt.TemplateType
	Path     string // Path in the OpenAPI spec
	Context  prompt.TemplateContext
	SetValue func(description string) // Callback to set the description

	// Schema-specific: fields to set descriptions for
	SchemaFields []FieldElement
}
```

**Step 3: Modify ProcessBatch to handle schema batches**

```go
// Modify internal/enricher/processor/batch.go

// ProcessBatch processes a single batch of elements
func (p *BatchProcessor) ProcessBatch(ctx context.Context, batch *Batch) error {
	tmpl, err := p.templateMgr.Get(batch.Type)
	if err != nil {
		return fmt.Errorf("failed to get template: %w", err)
	}

	for _, elem := range batch.Elements {
		systemPrompt, userPrompt, err := tmpl.Render(elem.Context)
		if err != nil {
			slog.Warn("failed to render template", "error", err, "path", elem.Path)
			continue
		}

		fullPrompt := systemPrompt + "\n\n" + userPrompt
		response, err := p.provider.Generate(ctx, fullPrompt)
		if err != nil {
			return fmt.Errorf("LLM call failed: %w", err)
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
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/enricher/processor/... -v -run TestBatchProcessor_ProcessSchemaBatch`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/enricher/processor/processor.go internal/enricher/processor/batch.go internal/enricher/processor/batch_test.go
git commit -m "$(cat <<'EOF'
feat(processor): support schema batch processing with field-level SetValue

- Add SchemaFields to EnrichmentElement for schema batches
- Modify ProcessBatch to parse schema responses and set field descriptions
- Handle both schema and non-schema batch types

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: Extend Enricher - Collect Parameters

**Files:**
- Modify: `internal/enricher/enricher.go`
- Test: `internal/enricher/enricher_test.go`

**Step 1: Write the failing test**

```go
// Add to internal/enricher/enricher_test.go

func TestEnricher_CollectParameters(t *testing.T) {
	spec := &openapi3.T{
		Paths: openapi3.Paths{
			"/users/{id}": &openapi3.PathItem{
				Get: &openapi3.Operation{
					Parameters: openapi3.Parameters{
						&openapi3.ParameterRef{
							Value: &openapi3.Parameter{
								Name:     "id",
								In:       "path",
								Required: true,
								Schema:   &openapi3.SchemaRef{Value: &openapi3.Schema{Type: "integer"}},
							},
						},
					},
				},
			},
		},
	}

	// collectElements should collect the parameter
	enricher := &Enricher{
		config:   DefaultConfig(),
		provider: &MockProvider{},
	}

	collector := enricher.collectElements(spec, "en")
	params := collector.GetParams()

	if len(params) != 1 {
		t.Errorf("expected 1 parameter, got %d", len(params))
	}
	if params[0].ParamName != "id" {
		t.Errorf("expected param name 'id', got %s", params[0].ParamName)
	}
	if params[0].ParamIn != "path" {
		t.Errorf("expected param in 'path', got %s", params[0].ParamIn)
	}
}
```

**Step 2: Add GetParams method to SpecCollector**

```go
// Add to internal/enricher/processor/processor.go

// GetParams returns collected parameters.
func (c *SpecCollector) GetParams() []ParamElement {
	return c.params
}
```

**Step 3: Add parameter collection to enricher.go**

```go
// Modify internal/enricher/enricher.go collectElements function

// Add this after the operations loop (inside the pathItem loop):

				// Collect API parameters
				for _, paramRef := range item.op.Parameters {
					if paramRef.Value == nil || paramRef.Value.Description != "" {
						continue // Skip if already has description
					}

					param := paramRef.Value
					fieldType := ""
					if param.Schema != nil && param.Schema.Value != nil {
						fieldType = param.Schema.Value.Type
					}

					collector.AddParamElement(pathStr, item.method, param.Name, param.In, fieldType, param.Required, language)

					// Capture param for closure
					p := param
					collector.AddElement(EnrichmentElement{
						Type: prompt.TemplateTypeParam,
						Path: item.method + " " + pathStr + " param:" + param.Name,
						Context: prompt.TemplateContext{
							Type:      prompt.TemplateTypeParam,
							Language:  language,
							Method:    item.method,
							Path:      pathStr,
							ParamName: param.Name,
							ParamIn:   param.In,
							FieldType: fieldType,
							Required:  param.Required,
						},
						SetValue: func(desc string) {
							p.Description = desc
						},
					})
				}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/enricher/... -v -run TestEnricher_CollectParameters`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/enricher/enricher.go internal/enricher/enricher_test.go internal/enricher/processor/processor.go
git commit -m "$(cat <<'EOF'
feat(enricher): collect API parameters for enrichment

- Add parameter collection in collectElements
- Collect path, query, header, and cookie parameters
- Skip parameters that already have descriptions

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 9: Extend Enricher - Collect Schema Fields

**Files:**
- Modify: `internal/enricher/enricher.go`
- Test: `internal/enricher/enricher_test.go`

**Step 1: Write the failing test**

```go
// Add to internal/enricher/enricher_test.go

func TestEnricher_CollectSchemaFields(t *testing.T) {
	spec := &openapi3.T{
		Components: &openapi3.Components{
			Schemas: openapi3.Schemas{
				"User": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: "object",
						Properties: openapi3.Schemas{
							"id":   &openapi3.SchemaRef{Value: &openapi3.Schema{Type: "integer"}},
							"name": &openapi3.SchemaRef{Value: &openapi3.Schema{Type: "string"}},
						},
					},
				},
			},
		},
	}

	enricher := &Enricher{
		config:   DefaultConfig(),
		provider: &MockProvider{},
	}

	collector := enricher.collectElements(spec, "en")
	schemas := collector.GetSchemas()

	if len(schemas) != 1 {
		t.Errorf("expected 1 schema, got %d", len(schemas))
	}

	// Find User schema
	var userSchema *processor.SchemaElement
	for _, s := range schemas {
		if s.SchemaName == "User" {
			userSchema = &s
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
```

**Step 2: Add schema collection to enricher.go**

```go
// Modify internal/enricher/enricher.go

import (
	// Add import
	"github.com/spencercjh/spec-forge/internal/enricher/processor"
)

// Add this to the end of collectElements function (after the paths loop):

	// Collect standalone schemas (not referenced by APIs)
	if spec.Components != nil && spec.Components.Schemas != nil {
		for schemaName, schemaRef := range spec.Components.Schemas {
			processor.CollectSchemaFields(schemaName, schemaRef, collector, processedSchemas, language, 0)
		}
	}
```

**Step 3: Add processedSchemas tracking to prevent duplicates**

```go
// Modify collectElements to track processed schemas

func (e *Enricher) collectElements(spec *openapi3.T, language string) *processor.SpecCollector {
	collector := &processor.SpecCollector{}
	processedSchemas := make(map[string]bool)

	// ... existing code ...

	// When processing response/request body schemas, pass processedSchemas
	// e.collectSchemaFromResponse(response, collector, processedSchemas, language)

	// Collect standalone schemas
	if spec.Components != nil && spec.Components.Schemas != nil {
		for schemaName, schemaRef := range spec.Components.Schemas {
			if !processedSchemas[schemaName] {
				processor.CollectSchemaFields(schemaName, schemaRef, collector, processedSchemas, language, 0)
			}
		}
	}

	return collector
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/enricher/... -v -run TestEnricher_CollectSchemaFields`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/enricher/enricher.go internal/enricher/enricher_test.go
git commit -m "$(cat <<'EOF'
feat(enricher): collect schema fields for enrichment

- Add schema field collection using CollectSchemaFields
- Track processed schemas to prevent duplicates
- Process standalone schemas from components

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 10: Update Prompt Templates

**Files:**
- Modify: `internal/enricher/prompt/templates.go`

**Step 1: Update Schema template**

```go
// Modify TemplateTypeSchema in NewTemplateManager

TemplateTypeSchema: {
    System: `You are an API documentation expert. Generate concise field descriptions.
Respond in {{.Language}} language.
Output format: JSON object mapping field names to their descriptions.
Example: {"userId": "The unique identifier of the user", "email": "The user's email address"}`,
    User: `Schema: {{.SchemaName}}
Fields:
{{range .Fields}}- {{.Name}} ({{.Type}}, {{if .Required}}required{{else}}optional{{end}})
{{end}}

Generate a concise description for each field. Keep descriptions brief (1-2 sentences).`,
},
```

**Step 2: Update Param template**

```go
// Modify TemplateTypeParam in NewTemplateManager

TemplateTypeParam: {
    System: `You are an API documentation expert. Generate concise parameter descriptions.
Respond in {{.Language}} language.
Output format: JSON with "description" field.`,
    User: `API Endpoint: {{.Method}} {{.Path}}
Parameter: {{.ParamName}}
Type: {{.FieldType}}
In: {{.ParamIn}}
Required: {{.Required}}

Generate a brief description for this parameter.`,
},
```

**Step 3: Run tests to verify**

Run: `go test ./internal/enricher/prompt/... -v`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/enricher/prompt/templates.go
git commit -m "$(cat <<'EOF'
feat(prompt): update Schema and Param templates for field enrichment

- Update Schema template with JSON output format example
- Update Param template to include ParamIn context
- Improve prompt clarity for consistent responses

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 11: Integration Test

**Files:**
- Create: `internal/enricher/integration_test.go` (build tag: integration)

**Step 1: Write integration test**

```go
//go:build integration

package enricher_test

import (
	"context"
	"os"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/spencercjh/spec-forge/internal/enricher"
	"github.com/spencercjh/spec-forge/internal/enricher/provider"
)

func TestEnricher_FullEnrichment(t *testing.T) {
	apiKey := os.Getenv("LLM_API_KEY")
	if apiKey == "" {
		t.Skip("LLM_API_KEY not set")
	}

	// Create a test spec with missing descriptions
	spec := &openapi3.T{
		OpenAPI: "3.1.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: openapi3.Paths{
			"/users/{id}": &openapi3.PathItem{
				Get: &openapi3.Operation{
					Parameters: openapi3.Parameters{
						&openapi3.ParameterRef{
							Value: &openapi3.Parameter{
								Name:     "id",
								In:       "path",
								Required: true,
								Schema:   &openapi3.SchemaRef{Value: &openapi3.Schema{Type: "integer"}},
							},
						},
					},
				},
			},
		},
		Components: &openapi3.Components{
			Schemas: openapi3.Schemas{
				"User": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: "object",
						Properties: openapi3.Schemas{
							"id":    &openapi3.SchemaRef{Value: &openapi3.Schema{Type: "integer"}},
							"name":  &openapi3.SchemaRef{Value: &openapi3.Schema{Type: "string"}},
							"email": &openapi3.SchemaRef{Value: &openapi3.Schema{Type: "string"}},
						},
						Required: []string{"id", "email"},
					},
				},
			},
		},
	}

	// Create provider
	cfg := provider.Config{
		Provider: "custom",
		Model:    "deepseek-chat",
		BaseURL:  os.Getenv("LLM_BASE_URL"),
		APIKey:   apiKey,
	}
	p, err := provider.NewProvider(cfg)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	// Create enricher
	enricherCfg := enricher.Config{
		Provider:    "custom",
		Model:       "deepseek-chat",
		Language:    "en",
		Concurrency: 1,
	}
	e, err := enricher.NewEnricher(enricherCfg, p)
	if err != nil {
		t.Fatalf("failed to create enricher: %v", err)
	}

	// Enrich
	ctx := context.Background()
	result, err := e.Enrich(ctx, spec, nil)
	if err != nil {
		t.Fatalf("enrichment failed: %v", err)
	}

	// Verify parameter has description
	if len(result.Paths) > 0 {
		pathItem := result.Paths["/users/{id}"]
		if pathItem.Get != nil && len(pathItem.Get.Parameters) > 0 {
			param := pathItem.Get.Parameters[0].Value
			if param.Description == "" {
				t.Error("expected parameter description to be enriched")
			}
			t.Logf("Parameter description: %s", param.Description)
		}
	}

	// Verify schema fields have descriptions
	if result.Components != nil && result.Components.Schemas != nil {
		userSchema := result.Components.Schemas["User"].Value
		if userSchema != nil {
			for propName, propRef := range userSchema.Properties {
				if propRef.Value != nil && propRef.Value.Description == "" {
					t.Errorf("expected field %s to have description", propName)
				} else if propRef.Value != nil {
					t.Logf("Field %s description: %s", propName, propRef.Value.Description)
				}
			}
		}
	}
}
```

**Step 2: Run integration test**

Run: `go test ./internal/enricher/... -v -tags=integration -run TestEnricher_FullEnrichment`
Expected: PASS (if LLM_API_KEY is set)

**Step 3: Commit**

```bash
git add internal/enricher/integration_test.go
git commit -m "$(cat <<'EOF'
test(enricher): add integration test for full enrichment

- Test API operation, parameter, and schema field enrichment
- Use build tag for optional integration testing
- Verify all element types are enriched with descriptions

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 12: Run Full Test Suite and Lint

**Step 1: Run all tests**

Run: `make test`
Expected: All tests pass

**Step 2: Run linter**

Run: `make lint`
Expected: No lint errors

**Step 3: Fix any issues**

If lint or test failures, fix them and commit.

---

## Summary

**Files Created:**
- `internal/context/types.go`
- `internal/context/types_test.go`
- `internal/context/extractor.go`
- `internal/context/noop_extractor.go`
- `internal/context/extractor_test.go`
- `internal/enricher/processor/schema.go`
- `internal/enricher/processor/schema_test.go`
- `internal/enricher/integration_test.go`

**Files Modified:**
- `internal/enricher/processor/processor.go`
- `internal/enricher/processor/processor_test.go`
- `internal/enricher/processor/batch.go`
- `internal/enricher/processor/batch_test.go`
- `internal/enricher/enricher.go`
- `internal/enricher/enricher_test.go`
- `internal/enricher/prompt/templates.go`
