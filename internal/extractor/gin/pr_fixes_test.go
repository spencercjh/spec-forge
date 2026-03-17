package gin

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

// TestFormTagSchemaSupport tests that form tags are used for property naming in schemas
// Fix for: Form tag schema mismatch - SchemaExtractor only uses json tags
func TestFormTagSchemaSupport(t *testing.T) {
	src := `package main

type UpdateProfileRequest struct {
	FullName string ` + "`" + `form:"full_name" json:"fullName"` + "`" + `
	Email    string ` + "`" + `form:"email" json:"emailAddress"` + "`" + `
	Age      int    ` + "`" + `form:"age" json:"user_age"` + "`" + `
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	files := map[string]*ast.File{"test.go": file}

	extractor := NewSchemaExtractor(files)
	schema, err := extractor.ExtractSchema("UpdateProfileRequest")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that form tag names are used as property names
	// Before fix: would use json tag names (fullName, emailAddress, user_age)
	// After fix: should use form tag names (full_name, email, age)
	expectedProps := map[string]bool{
		"full_name": false,
		"email":     false,
		"age":       false,
	}

	for propName := range schema.Value.Properties {
		if _, exists := expectedProps[propName]; exists {
			expectedProps[propName] = true
		} else {
			t.Errorf("unexpected property: %s", propName)
		}
	}

	for name, found := range expectedProps {
		if !found {
			t.Errorf("expected property %s not found in schema", name)
		}
	}
}

// TestTypeAliasToCustomType tests that type aliases to custom types create $ref
// Fix for: Type alias to non-primitive types produces empty object
func TestTypeAliasToCustomType(t *testing.T) {
	src := `package main

type User struct {
	ID   int    ` + "`" + `json:"id"` + "`" + `
	Name string ` + "`" + `json:"name"` + "`" + `
}

// UserID is an alias to User - should create $ref to User, not empty object
type UserID User
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	files := map[string]*ast.File{"test.go": file}

	extractor := NewSchemaExtractor(files)
	schema, err := extractor.ExtractSchema("UserID")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have $ref to User, not be an empty object
	if schema.Ref == "" {
		t.Errorf("expected $ref for type alias to custom type, got inline schema: %+v", schema.Value)
	} else if schema.Ref != "#/components/schemas/User" {
		t.Errorf("expected $ref '#/components/schemas/User', got %s", schema.Ref)
	}
}

// TestCrossPackageTypeResolution tests that findTypeSpec handles package-qualified names
// Fix for: Qualified type params missing - findTypeSpec only matches exact local type names
func TestCrossPackageTypeResolution(t *testing.T) {
	src := `package main

type Query struct {
	Page int    ` + "`" + `form:"page"` + "`" + `
	Size int    ` + "`" + `form:"size"` + "`" + `
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	files := map[string]*ast.File{"test.go": file}

	analyzer := NewHandlerAnalyzer(fset, files)

	// Test with package-qualified name (as would come from varTypeMap)
	// Before fix: findTypeSpec("models.Query") would fail to find "Query"
	// After fix: should strip package prefix and find the type
	typeSpec := analyzer.findTypeSpec("models.Query")
	if typeSpec == nil {
		t.Error("findTypeSpec failed to resolve cross-package type 'models.Query'")
	} else if typeSpec.Name.Name != "Query" {
		t.Errorf("expected type name 'Query', got %s", typeSpec.Name.Name)
	}

	// Also test with local name (should still work)
	typeSpec = analyzer.findTypeSpec("Query")
	if typeSpec == nil {
		t.Error("findTypeSpec failed to resolve local type 'Query'")
	}
}

// TestGinHTypeNormalization tests that gin.H is normalized to ginHType in response extraction
// Fix for: gin.H type handling inconsistency
func TestGinHTypeNormalization(t *testing.T) {
	src := `package main

import "github.com/gin-gonic/gin"

func handler(c *gin.Context) {
	c.JSON(200, gin.H{"message": "ok"})
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	files := map[string]*ast.File{"test.go": file}

	analyzer := NewHandlerAnalyzer(fset, files)

	// Find the handler function
	var handlerDecl *ast.FuncDecl
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "handler" {
			handlerDecl = fn
			break
		}
	}

	if handlerDecl == nil {
		t.Fatal("handler function not found")
	}

	info, err := analyzer.AnalyzeHandler(handlerDecl)
	if err != nil {
		t.Fatalf("failed to analyze handler: %v", err)
	}

	// Check that gin.H response is normalized to ginHType
	if len(info.Responses) == 0 {
		t.Fatal("expected at least one response")
	}

	resp := info.Responses[0]
	if resp.GoType != ginHType {
		t.Errorf("expected GoType to be normalized to '%s', got '%s'", ginHType, resp.GoType)
	}
}

// TestIsFormRelatedWord tests that isFormRelatedWord uses whole-word matching
// Fix for: hasFormContext substring matching ("performAction" matched "form")
func TestIsFormRelatedWord(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		// Should match (whole words - with word boundaries)
		{name: "just form", text: "form", expected: true},
		{name: "uppercase FORM", text: "UPDATE_FORM", expected: true},
		{name: "underscore prefix", text: "form_update", expected: true},
		{name: "underscore suffix", text: "update_form", expected: true},
		{name: "dash separated", text: "form-data", expected: true},
		{name: "formData", text: "formData", expected: true},
		{name: "multipart", text: "multipart", expected: true},
		// Should NOT match (substring only - false positives)
		{name: "platform", text: "platform", expected: false},
		{name: "information", text: "information", expected: false},
		{name: "performAction", text: "performAction", expected: false},
		{name: "format", text: "format", expected: false},
		{name: "conform", text: "conform", expected: false},
		{name: "formal", text: "formal", expected: false},
		{name: "formation", text: "formation", expected: false},
		// Edge cases
		{name: "empty", text: "", expected: false},
		{name: "form in middle", text: "platformData", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isFormRelatedWord(tt.text)
			if result != tt.expected {
				t.Errorf("isFormRelatedWord(%q) = %v, expected %v", tt.text, result, tt.expected)
			}
		})
	}
}

// TestExtractAllSchemasNestedRefs tests that ExtractAllSchemas follows refs in nested structures
// Fix for: ExtractAllSchemas $ref traversal incompleteness
func TestExtractAllSchemasNestedRefs(t *testing.T) {
	src := `package main

type Address struct {
	Street string ` + "`" + `json:"street"` + "`" + `
	City   string ` + "`" + `json:"city"` + "`" + `
}

type Company struct {
	Name    string  ` + "`" + `json:"name"` + "`" + `
	Address Address ` + "`" + `json:"address"` + "`" + `
}

type Employee struct {
	ID      int     ` + "`" + `json:"id"` + "`" + `
	Company Company ` + "`" + `json:"company"` + "`" + `
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	files := map[string]*ast.File{"test.go": file}

	extractor := NewSchemaExtractor(files)
	schemas, err := extractor.ExtractAllSchemas("Employee")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should extract all nested types: Employee, Company, Address
	expectedTypes := []string{"Employee", "Company", "Address"}
	for _, typeName := range expectedTypes {
		if _, exists := schemas[typeName]; !exists {
			t.Errorf("expected schema %s to be extracted", typeName)
		}
	}

	if len(schemas) != len(expectedTypes) {
		t.Errorf("expected %d schemas, got %d: %v", len(expectedTypes), len(schemas), getSchemaKeys(schemas))
	}
}

// TestExtractAllSchemasMapAdditionalProperties tests refs in map value types
func TestExtractAllSchemasMapAdditionalProperties(t *testing.T) {
	src := `package main

type Metadata struct {
	Key   string ` + "`" + `json:"key"` + "`" + `
	Value string ` + "`" + `json:"value"` + "`" + `
}

type Resource struct {
	Name     string              ` + "`" + `json:"name"` + "`" + `
	Metadata map[string]Metadata ` + "`" + `json:"metadata"` + "`" + `
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	files := map[string]*ast.File{"test.go": file}

	extractor := NewSchemaExtractor(files)
	schemas, err := extractor.ExtractAllSchemas("Resource")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should extract Resource and Metadata (used in map)
	if _, exists := schemas["Resource"]; !exists {
		t.Error("expected schema Resource to be extracted")
	}
	if _, exists := schemas["Metadata"]; !exists {
		t.Error("expected schema Metadata to be extracted (used in map value type)")
	}
}

// TestExtractAllSchemasCompositionKeywords tests refs in allOf/oneOf/anyOf
// Note: This tests the extraction logic, though Go structs don't directly map to composition
func TestExtractAllSchemasNestedArray(t *testing.T) {
	src := `package main

type Tag struct {
	Name string ` + "`" + `json:"name"` + "`" + `
}

type Item struct {
	Tags [][]Tag ` + "`" + `json:"tags"` + "`" + `
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	files := map[string]*ast.File{"test.go": file}

	extractor := NewSchemaExtractor(files)
	schemas, err := extractor.ExtractAllSchemas("Item")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should extract Item and Tag (used in nested array)
	if _, exists := schemas["Item"]; !exists {
		t.Error("expected schema Item to be extracted")
	}
	if _, exists := schemas["Tag"]; !exists {
		t.Error("expected schema Tag to be extracted (used in nested array)")
	}
}

// Helper function to get schema keys for error messages
func getSchemaKeys(schemas openapi3.Schemas) []string {
	keys := make([]string, 0, len(schemas))
	for k := range schemas {
		keys = append(keys, k)
	}
	return keys
}
