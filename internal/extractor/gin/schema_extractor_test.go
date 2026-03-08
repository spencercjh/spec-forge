package gin

import (
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"testing"
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
