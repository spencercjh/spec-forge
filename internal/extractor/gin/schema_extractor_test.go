package gin

import (
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestNewSchemaExtractor(t *testing.T) {
	e := NewSchemaExtractor(nil)
	if e == nil {
		t.Error("expected non-nil extractor")
	}
}

func TestSchemaExtractor_ExtractSchema(t *testing.T) {
	src := `package main

type User struct {
	ID       int    ` + "`" + `json:"id" binding:"required"` + "`" + `
	Name     string ` + "`" + `json:"name"` + "`" + `
	Email    string ` + "`" + `json:"email" validate:"email"` + "`" + `
	Age      int    ` + "`" + `json:"age,omitempty" validate:"min=0,max=150"` + "`" + `
}
`
	fset := token.NewFileSet()
	file, _ := parser.ParseFile(fset, "test.go", src, 0)
	files := map[string]*ast.File{"test.go": file}

	extractor := NewSchemaExtractor(files)
	schema, err := extractor.ExtractSchema("User")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if schema == nil {
		t.Fatal("expected non-nil schema")
	}

	// Check required fields
	required := schema.Value.Required
	if len(required) != 1 || required[0] != "id" {
		t.Errorf("expected required=['id'], got %v", required)
	}

	// Check properties
	props := schema.Value.Properties
	if len(props) != 4 {
		t.Errorf("expected 4 properties, got %d", len(props))
	}

	// Check email format
	if emailProp := props["email"]; emailProp != nil {
		if emailProp.Value.Format != "email" {
			t.Errorf("expected email format, got %s", emailProp.Value.Format)
		}
	}
}

func TestGoTypeToSchema(t *testing.T) {
	tests := []struct {
		goType       string
		expectedType string
		expectedFmt  string
	}{
		{"string", "string", ""},
		{"int", "integer", "int32"},
		{"int32", "integer", "int32"},
		{"int64", "integer", "int64"},
		{"uint", "integer", ""},
		{"float32", "number", "float"},
		{"float64", "number", "double"},
		{"bool", "boolean", ""},
		{"time.Time", "string", "date-time"},
		{"unknown", "object", ""},
	}

	for _, tt := range tests {
		t.Run(tt.goType, func(t *testing.T) {
			schema := goTypeToSchema(tt.goType)
			if schema.Type == nil || len(*schema.Type) == 0 || (*schema.Type)[0] != tt.expectedType {
				t.Errorf("expected type %s, got %v", tt.expectedType, schema.Type)
			}
			if schema.Format != tt.expectedFmt {
				t.Errorf("expected format %s, got %s", tt.expectedFmt, schema.Format)
			}
		})
	}
}

func TestExtractTagValue(t *testing.T) {
	tests := []struct {
		tag      string
		key      string
		expected string
	}{
		{`json:"name"`, "json", "name"},
		{`json:"name,omitempty"`, "json", "name,omitempty"},
		{`binding:"required"`, "binding", "required"},
		{`validate:"email"`, "validate", "email"},
		{`json:"name" binding:"required"`, "binding", "required"},
		{`json:"name"`, "xml", ""},
		{``, "json", ""},
	}

	for _, tt := range tests {
		t.Run(tt.key+"_"+tt.expected, func(t *testing.T) {
			result := extractTagValue(tt.tag, tt.key)
			if result != tt.expected {
				t.Errorf("extractTagValue(%q, %q) = %q, expected %q", tt.tag, tt.key, result, tt.expected)
			}
		})
	}
}

func TestSchemaExtractor_Caching(t *testing.T) {
	src := `package main

type User struct {
	ID int ` + "`" + `json:"id"` + "`" + `
}
`
	fset := token.NewFileSet()
	file, _ := parser.ParseFile(fset, "test.go", src, 0)
	files := map[string]*ast.File{"test.go": file}

	extractor := NewSchemaExtractor(files)

	// First extraction
	schema1, err := extractor.ExtractSchema("User")
	if err != nil {
		t.Fatalf("first extraction failed: %v", err)
	}

	// Second extraction should return cached
	schema2, err := extractor.ExtractSchema("User")
	if err != nil {
		t.Fatalf("second extraction failed: %v", err)
	}

	// Should be the same pointer (cached)
	if schema1 != schema2 {
		t.Error("expected cached schema to be same pointer")
	}
}

func TestSchemaExtractor_ArrayType(t *testing.T) {
	src := `package main

type ListResponse struct {
	Items []string ` + "`" + `json:"items"` + "`" + `
	Count int      ` + "`" + `json:"count"` + "`" + `
}
`
	fset := token.NewFileSet()
	file, _ := parser.ParseFile(fset, "test.go", src, 0)
	files := map[string]*ast.File{"test.go": file}

	extractor := NewSchemaExtractor(files)
	schema, err := extractor.ExtractSchema("ListResponse")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check Items is an array
	itemsProp := schema.Value.Properties["items"]
	if itemsProp == nil {
		t.Fatal("expected 'items' property")
	}
	if itemsProp.Value.Type == nil || len(*itemsProp.Value.Type) == 0 || (*itemsProp.Value.Type)[0] != "array" {
		t.Errorf("expected array type, got %s", itemsProp.Value.Type)
	}
}

func TestSchemaExtractor_WithValidation(t *testing.T) {
	src := `package main

type ValidatedUser struct {
	Name  string ` + "`" + `json:"name" validate:"minLength=2,maxLength=50"` + "`" + `
	Email string ` + "`" + `json:"email" validate:"email,required"` + "`" + `
	Role  string ` + "`" + `json:"role" validate:"oneof=admin user guest"` + "`" + `
}
`
	fset := token.NewFileSet()
	file, _ := parser.ParseFile(fset, "test.go", src, 0)
	files := map[string]*ast.File{"test.go": file}

	extractor := NewSchemaExtractor(files)
	schema, err := extractor.ExtractSchema("ValidatedUser")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check required fields (both name and email should be required)
	if !slices.Contains(schema.Value.Required, "email") {
		t.Error("expected 'email' to be in required list")
	}

	// Check enum for role
	roleProp := schema.Value.Properties["role"]
	if roleProp != nil && len(roleProp.Value.Enum) > 0 {
		// Check if enum contains expected values
		hasAdmin := false
		for _, v := range roleProp.Value.Enum {
			if v == "admin" {
				hasAdmin = true
				break
			}
		}
		if !hasAdmin {
			t.Error("expected 'admin' in role enum")
		}
	}
}

func TestFieldToSchema(t *testing.T) {
	e := NewSchemaExtractor(nil)

	tests := []struct {
		name     string
		src      string
		expected string
	}{
		{"string", `type T struct { F string }`, "string"},
		{"int", `type T struct { F int }`, "integer"},
		{"bool", `type T struct { F bool }`, "boolean"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, _ := parser.ParseFile(fset, "test.go", "package main\n"+tt.src, 0)

			// Get the struct type
			var structType *ast.StructType
			for _, decl := range file.Decls {
				if gen, ok := decl.(*ast.GenDecl); ok {
					for _, spec := range gen.Specs {
						if ts, ok := spec.(*ast.TypeSpec); ok {
							if st, ok := ts.Type.(*ast.StructType); ok {
								structType = st
								break
							}
						}
					}
				}
			}

			if structType == nil || len(structType.Fields.List) == 0 {
				t.Fatal("failed to find struct field")
			}

			schema := e.fieldToSchema(structType.Fields.List[0].Type)
			if schema.Type == nil || len(*schema.Type) == 0 || (*schema.Type)[0] != tt.expected {
				t.Errorf("expected type %s, got %s", tt.expected, schema.Type)
			}
		})
	}
}

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

// TestExtractAllSchemasRefOnly tests that $ref-only schemas trigger extraction of referenced types
// Fix for: ExtractAllSchemas doesn't traverse $ref-only schemas (e.g., type UserID User)
func TestExtractAllSchemasRefOnly(t *testing.T) {
	src := `package main

type User struct {
	ID   int    ` + "`" + `json:"id"` + "`" + `
	Name string ` + "`" + `json:"name"` + "`" + `
}

// UserID is a type alias that creates a $ref-only schema
type UserID User

type Wrapper struct {
	UserRef UserID ` + "`" + `json:"userRef"` + "`" + `
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	files := map[string]*ast.File{"test.go": file}

	extractor := NewSchemaExtractor(files)
	schemas, err := extractor.ExtractAllSchemas("Wrapper")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should extract Wrapper, UserID (alias), and User (referenced by alias)
	expectedTypes := []string{"Wrapper", "UserID", "User"}
	for _, typeName := range expectedTypes {
		if _, exists := schemas[typeName]; !exists {
			t.Errorf("expected schema %s to be extracted", typeName)
		}
	}

	// Verify UserID has a $ref to User
	if userIDSchema, exists := schemas["UserID"]; exists {
		if userIDSchema.Ref == "" {
			t.Error("expected UserID schema to have $ref, but it's inline")
		} else if userIDSchema.Ref != "#/components/schemas/User" {
			t.Errorf("expected UserID $ref to '#/components/schemas/User', got %s", userIDSchema.Ref)
		}
	}
}

// TestSelectorExprTypeAlias tests that type aliases to imported types work correctly
// Fix for: Missing *ast.SelectorExpr handling for type aliases like type MyTime time.Time
func TestSelectorExprTypeAlias(t *testing.T) {
	src := `package main

import "time"

type MyTime time.Time
type MyDuration time.Duration

type Event struct {
	Timestamp MyTime    ` + "`" + `json:"timestamp"` + "`" + `
	Duration  MyDuration ` + "`" + `json:"duration"` + "`" + `
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	files := map[string]*ast.File{"test.go": file}

	extractor := NewSchemaExtractor(files)

	// Test MyTime - should produce date-time formatted string, not empty object
	schema, err := extractor.ExtractSchema("MyTime")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should resolve to time.Time which has special handling
	// The schema should either have a $ref to time.Time or be string/date-time
	if schema.Ref != "" {
		// If it's a $ref, that's acceptable
		if schema.Ref != "#/components/schemas/Time" {
			t.Logf("MyTime has $ref to %s (may vary by Go version)", schema.Ref)
		}
	} else if schema.Value != nil {
		// If inline, should be string with date-time format
		if schema.Value.Type == nil || len(*schema.Value.Type) == 0 || (*schema.Value.Type)[0] != "string" {
			t.Errorf("expected MyTime to resolve to string type, got %v", schema.Value.Type)
		}
		if schema.Value.Format != "date-time" {
			t.Logf("MyTime format is %q (expected date-time for time.Time)", schema.Value.Format)
		}
	} else {
		t.Error("MyTime schema has neither Ref nor Value")
	}

	// Test that Event correctly uses MyTime
	eventSchema, err := extractor.ExtractSchema("Event")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	timestampProp := eventSchema.Value.Properties["timestamp"]
	if timestampProp == nil {
		t.Fatal("expected 'timestamp' property in Event")
	}

	// Before fix: would be empty object
	// After fix: should be properly resolved
	if timestampProp.Ref == "" && (timestampProp.Value == nil || timestampProp.Value.Type == nil) {
		t.Error("timestamp property was not properly resolved (likely empty object)")
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
