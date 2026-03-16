package gin

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// TestArrayTypeWithCustomElement tests that arrays of custom types use $ref instead of inline objects
func TestArrayTypeWithCustomElement(t *testing.T) {
	src := `package main

type User struct {
	ID   int    ` + "`" + `json:"id"` + "`" + `
	Name string ` + "`" + `json:"name"` + "`" + `
}

type UserList struct {
	Users []User ` + "`" + `json:"users"` + "`" + `
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	files := map[string]*ast.File{"test.go": file}

	extractor := NewSchemaExtractor(files)
	schema, err := extractor.ExtractSchema("UserList")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check Users field
	usersProp := schema.Value.Properties["users"]
	if usersProp == nil {
		t.Fatal("expected 'users' property")
	}

	// Check it's an array
	if usersProp.Value.Type == nil || len(*usersProp.Value.Type) == 0 || (*usersProp.Value.Type)[0] != "array" {
		t.Errorf("expected array type, got %v", usersProp.Value.Type)
	}

	// Check items - should use $ref, not inline object
	if usersProp.Value.Items == nil {
		t.Fatal("expected items schema")
	}

	// The issue is that for custom types, Items should have Ref pointing to the type
	// not be an inline object schema
	if usersProp.Value.Items.Ref == "" {
		t.Errorf("expected $ref for array items, got inline object. Items.Value = %+v", usersProp.Value.Items.Value)
	} else if usersProp.Value.Items.Ref != "#/components/schemas/User" {
		t.Errorf("expected $ref '#/components/schemas/User', got %s", usersProp.Value.Items.Ref)
	}
}

// TestArrayTypeAlias tests type aliases for arrays with custom element types
func TestArrayTypeAlias(t *testing.T) {
	src := `package main

type User struct {
	ID   int    ` + "`" + `json:"id"` + "`" + `
	Name string ` + "`" + `json:"name"` + "`" + `
}

// UserList is a type alias for []User
type UserList []User
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	files := map[string]*ast.File{"test.go": file}

	extractor := NewSchemaExtractor(files)
	schema, err := extractor.ExtractSchema("UserList")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check it's an array
	if schema.Value.Type == nil || len(*schema.Value.Type) == 0 || (*schema.Value.Type)[0] != "array" {
		t.Errorf("expected array type, got %v", schema.Value.Type)
	}

	// Check items - should use $ref for custom element type
	if schema.Value.Items == nil {
		t.Fatal("expected items schema")
	}

	// This is the bug: array type aliases with custom elements create inline objects
	// instead of $ref schemas
	if schema.Value.Items.Ref == "" {
		t.Errorf("expected $ref for array items, got inline object. Items.Value = %+v", schema.Value.Items.Value)
	} else if schema.Value.Items.Ref != "#/components/schemas/User" {
		t.Errorf("expected $ref '#/components/schemas/User', got %s", schema.Value.Items.Ref)
	}
}

// TestNestedArrayType tests arrays with primitive element types
func TestNestedArrayType(t *testing.T) {
	src := `package main

type StringList struct {
	Items []string ` + "`" + `json:"items"` + "`" + `
}
`
	fset := token.NewFileSet()
	file, _ := parser.ParseFile(fset, "test.go", src, 0)
	files := map[string]*ast.File{"test.go": file}

	extractor := NewSchemaExtractor(files)
	schema, err := extractor.ExtractSchema("StringList")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	itemsProp := schema.Value.Properties["items"]
	if itemsProp == nil {
		t.Fatal("expected 'items' property")
	}

	if itemsProp.Value.Items == nil {
		t.Fatal("expected items schema")
	}

	// For primitives, Items should have inline schema with type
	if itemsProp.Value.Items.Value == nil || itemsProp.Value.Items.Value.Type == nil {
		t.Fatal("expected inline schema for primitive array items")
	}

	itemType := (*itemsProp.Value.Items.Value.Type)[0]
	if itemType != "string" {
		t.Errorf("expected string item type, got %s", itemType)
	}
}
