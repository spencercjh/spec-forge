package gin

import (
	"fmt"
	"go/ast"
	"go/token"
	"log/slog"
	"slices"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// Validation rule constants for goconst compliance
const (
	validateRuleEmail = "email"
	validateRuleURL   = "url"
	validateRuleUUID  = "uuid"
)

// Schema type constants for goconst compliance
const schemaTypeObject = "object"

// SchemaExtractor extracts OpenAPI schemas from Go structs.
type SchemaExtractor struct {
	files     map[string]*ast.File
	typeCache map[string]*openapi3.SchemaRef
}

// NewSchemaExtractor creates a new SchemaExtractor instance.
func NewSchemaExtractor(files map[string]*ast.File) *SchemaExtractor {
	return &SchemaExtractor{
		files:     files,
		typeCache: make(map[string]*openapi3.SchemaRef),
	}
}

// ExtractSchema extracts an OpenAPI schema from a Go type.
func (e *SchemaExtractor) ExtractSchema(typeName string) (*openapi3.SchemaRef, error) {
	slog.Debug("Extracting schema", "type", typeName)

	// Check cache
	if cached, ok := e.typeCache[typeName]; ok {
		slog.Debug("Using cached schema", "type", typeName)
		return cached, nil
	}

	// Find the type definition
	typeSpec := e.findTypeSpec(typeName)
	if typeSpec == nil {
		slog.Warn("Type not found", "type", typeName)
		return nil, fmt.Errorf("type %s not found", typeName)
	}

	// Extract schema based on underlying type
	var schema *openapi3.Schema
	var err error

	switch underlying := typeSpec.Type.(type) {
	case *ast.StructType:
		schema, err = e.extractStructSchema(typeSpec, underlying)
	case *ast.Ident:
		// Type alias (e.g., type CustomString string)
		schema = goTypeToSchema(underlying.Name)
	case *ast.ArrayType:
		// Type alias for array (e.g., type IDs []int or type Users []User)
		// Use fieldToSchemaRef to preserve $ref for custom element types
		itemSchemaRef := e.fieldToSchemaRef(underlying.Elt)
		schema = &openapi3.Schema{
			Type:  &openapi3.Types{"array"},
			Items: itemSchemaRef,
		}
	default:
		// Fallback to object for unknown types
		schema = &openapi3.Schema{Type: &openapi3.Types{schemaTypeObject}}
	}

	if err != nil {
		slog.Error("Failed to extract schema", "type", typeName, "error", err)
		return nil, err
	}

	ref := &openapi3.SchemaRef{Value: schema}
	e.typeCache[typeName] = ref
	slog.Debug("Extracted schema", "type", typeName)
	return ref, nil
}

// ExtractAllSchemas extracts a type and all its nested referenced types.
// This ensures that when Company is extracted, Address (if referenced) is also extracted.
func (e *SchemaExtractor) ExtractAllSchemas(typeName string) (openapi3.Schemas, error) {
	schemas := make(openapi3.Schemas)
	visited := make(map[string]bool)

	var extractRecursive func(string) error
	extractRecursive = func(name string) error {
		// Skip already visited or builtin types
		if visited[name] {
			return nil
		}
		visited[name] = true

		// Skip primitive types
		if isPrimitiveType(name) {
			return nil
		}

		// Extract this type
		schemaRef, err := e.ExtractSchema(name)
		if err != nil {
			// Log warning but continue - type might be from external package
			slog.Warn("Failed to extract nested schema", "type", name, "error", err)
			return nil
		}

		schemas[name] = schemaRef

		// Find and extract all referenced types from this schema
		if schemaRef.Value != nil && schemaRef.Value.Properties != nil {
			for _, propRef := range schemaRef.Value.Properties {
				if propRef.Ref != "" {
					// Extract the type name from $ref (e.g., "#/components/schemas/Address" -> "Address")
					refName := strings.TrimPrefix(propRef.Ref, "#/components/schemas/")
					if err := extractRecursive(refName); err != nil {
						return err
					}
				}
				// Also check array items
				if propRef.Value != nil && propRef.Value.Items != nil && propRef.Value.Items.Ref != "" {
					refName := strings.TrimPrefix(propRef.Value.Items.Ref, "#/components/schemas/")
					if err := extractRecursive(refName); err != nil {
						return err
					}
				}
			}
		}

		return nil
	}

	if err := extractRecursive(typeName); err != nil {
		return nil, err
	}

	return schemas, nil
}

// isPrimitiveType checks if a type name is a primitive Go type.
func isPrimitiveType(name string) bool {
	primitives := []string{
		"string", "int", "int32", "int64", "uint", "uint32", "uint64",
		"float32", "float64", "bool", "byte", "rune", "time.Time",
	}
	return slices.Contains(primitives, name)
}

// findTypeSpec finds a type definition by name.
// Supports both local types ("User") and cross-package types ("models.User").
func (e *SchemaExtractor) findTypeSpec(name string) *ast.TypeSpec {
	// For cross-package references (e.g., "models.User"), try to find by short name
	shortName := name
	if idx := strings.LastIndex(name, "."); idx != -1 {
		shortName = name[idx+1:]
	}

	// Try full name first, then short name
	for _, tryName := range []string{name, shortName} {
		for _, file := range e.files {
			for _, decl := range file.Decls {
				genDecl, ok := decl.(*ast.GenDecl)
				if !ok || genDecl.Tok != token.TYPE {
					continue
				}
				for _, spec := range genDecl.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if ok && typeSpec.Name.Name == tryName {
						return typeSpec
					}
				}
			}
		}
	}
	return nil
}

// extractStructSchema extracts schema from a struct type.
func (e *SchemaExtractor) extractStructSchema(_ *ast.TypeSpec, structType *ast.StructType) (*openapi3.Schema, error) {
	schema := &openapi3.Schema{
		Type:       &openapi3.Types{schemaTypeObject},
		Properties: make(openapi3.Schemas),
	}

	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 {
			continue // Embedded field - skip for now
		}

		fieldName := field.Names[0].Name
		propertyName := fieldName
		fieldSchemaRef := e.fieldToSchemaRef(field.Type)

		// Parse struct tags - apply regardless of whether field is inline or $ref
		if field.Tag != nil {
			tag := strings.Trim(field.Tag.Value, "`")
			propertyName = e.applyTags(fieldSchemaRef, tag, fieldName, schema)
		}

		// Skip fields with json:"-"
		if propertyName == "" {
			continue
		}

		schema.Properties[propertyName] = fieldSchemaRef
	}

	return schema, nil
}

// fieldToSchemaRef converts a Go type to OpenAPI schema reference.
// It returns a $ref for custom types and inline schemas for primitive types.
func (e *SchemaExtractor) fieldToSchemaRef(expr ast.Expr) *openapi3.SchemaRef {
	switch t := expr.(type) {
	case *ast.Ident:
		// Check if it's a basic type first
		schema := goTypeToSchema(t.Name)
		if schema.Type != nil && (*schema.Type)[0] != schemaTypeObject {
			return &openapi3.SchemaRef{Value: schema}
		}
		// It's a custom type - create a reference
		if e.findTypeSpec(t.Name) != nil {
			ref := "#/components/schemas/" + t.Name
			return &openapi3.SchemaRef{Ref: ref}
		}
		return &openapi3.SchemaRef{Value: schema}
	case *ast.StarExpr:
		// Pointer - unwrap and process underlying type
		return e.fieldToSchemaRef(t.X)
	case *ast.ArrayType:
		itemSchemaRef := e.fieldToSchemaRef(t.Elt)
		return &openapi3.SchemaRef{
			Value: &openapi3.Schema{
				Type:  &openapi3.Types{"array"},
				Items: itemSchemaRef,
			},
		}
	case *ast.MapType:
		valueSchemaRef := e.fieldToSchemaRef(t.Value)
		return &openapi3.SchemaRef{
			Value: &openapi3.Schema{
				Type:                 &openapi3.Types{schemaTypeObject},
				AdditionalProperties: openapi3.AdditionalProperties{Schema: valueSchemaRef},
			},
		}
	case *ast.SelectorExpr:
		// Package qualified type (e.g., time.Time, models.User)
		if x, ok := t.X.(*ast.Ident); ok {
			fullName := x.Name + "." + t.Sel.Name
			shortName := t.Sel.Name

			// Check if it's a known primitive type (like time.Time)
			schema := goTypeToSchema(fullName)
			if schema.Type != nil && (*schema.Type)[0] != schemaTypeObject {
				return &openapi3.SchemaRef{Value: schema}
			}

			// It's a custom type - try to find it and create a reference
			// Try both full name (models.User) and short name (User)
			if e.findTypeSpec(fullName) != nil || e.findTypeSpec(shortName) != nil {
				// Use short name for schema reference
				ref := "#/components/schemas/" + shortName
				return &openapi3.SchemaRef{Ref: ref}
			}
			return &openapi3.SchemaRef{Value: schema}
		}
	}

	return &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{schemaTypeObject}}}
}

// fieldToSchema converts a Go type to OpenAPI schema.
func (e *SchemaExtractor) fieldToSchema(expr ast.Expr) *openapi3.Schema {
	switch t := expr.(type) {
	case *ast.Ident:
		// Check if it's a basic type first
		schema := goTypeToSchema(t.Name)
		if schema.Type != nil && (*schema.Type)[0] != schemaTypeObject {
			return schema
		}
		// It's a custom type - check if we know about it
		if e.findTypeSpec(t.Name) != nil {
			// Return a placeholder schema with type object
			// The reference will be created by the caller if needed
			return &openapi3.Schema{Type: &openapi3.Types{schemaTypeObject}}
		}
		return schema
	case *ast.StarExpr:
		// Pointer - unwrap and process underlying type
		return e.fieldToSchema(t.X)
	case *ast.ArrayType:
		itemSchema := e.fieldToSchema(t.Elt)
		return &openapi3.Schema{
			Type:  &openapi3.Types{"array"},
			Items: &openapi3.SchemaRef{Value: itemSchema},
		}
	case *ast.MapType:
		valueSchema := e.fieldToSchema(t.Value)
		return &openapi3.Schema{
			Type:                 &openapi3.Types{schemaTypeObject},
			AdditionalProperties: openapi3.AdditionalProperties{Schema: &openapi3.SchemaRef{Value: valueSchema}},
		}
	case *ast.SelectorExpr:
		// Package qualified type (e.g., time.Time)
		if x, ok := t.X.(*ast.Ident); ok {
			fullName := x.Name + "." + t.Sel.Name
			return goTypeToSchema(fullName)
		}
	}

	return &openapi3.Schema{Type: &openapi3.Types{schemaTypeObject}}
}

// goTypeToSchema converts a Go type name to OpenAPI schema.
func goTypeToSchema(goType string) *openapi3.Schema {
	switch goType {
	case "string":
		return &openapi3.Schema{Type: &openapi3.Types{"string"}}
	case "int", "int32":
		return &openapi3.Schema{Type: &openapi3.Types{"integer"}, Format: "int32"}
	case "int64":
		return &openapi3.Schema{Type: &openapi3.Types{"integer"}, Format: "int64"}
	case "uint", "uint32":
		return &openapi3.Schema{Type: &openapi3.Types{"integer"}}
	case "float32":
		return &openapi3.Schema{Type: &openapi3.Types{"number"}, Format: "float"}
	case "float64":
		return &openapi3.Schema{Type: &openapi3.Types{"number"}, Format: "double"}
	case "bool":
		return &openapi3.Schema{Type: &openapi3.Types{"boolean"}}
	case "time.Time":
		return &openapi3.Schema{Type: &openapi3.Types{"string"}, Format: "date-time"}
	default:
		return &openapi3.Schema{Type: &openapi3.Types{schemaTypeObject}}
	}
}

// applyTags processes struct tags and updates schema.
// Returns the final property name (may be different from fieldName due to json tag).
// Returns empty string if the field should be skipped (json:"-").
func (e *SchemaExtractor) applyTags(schemaRef *openapi3.SchemaRef, tag, fieldName string, parentSchema *openapi3.Schema) string {
	propertyName := fieldName
	isOmitEmpty := false

	// Parse json tag
	if jsonTag := extractTagValue(tag, "json"); jsonTag != "" {
		parts := strings.Split(jsonTag, ",")
		// Handle json:"-" - skip field entirely
		if parts[0] == "-" {
			return ""
		}
		if parts[0] != "" {
			propertyName = parts[0]
		}
		// Check for omitempty
		if slices.Contains(parts[1:], "omitempty") {
			isOmitEmpty = true
		}
	}

	// Parse binding tag (handle multiple comma-separated rules like "required,email")
	if bindingTag := extractTagValue(tag, "binding"); bindingTag != "" {
		if strings.Contains(bindingTag, "required") && !isOmitEmpty {
			parentSchema.Required = append(parentSchema.Required, propertyName)
		}
	}

	// Parse validate tag - always apply validation even with omitempty
	if validateTag := extractTagValue(tag, "validate"); validateTag != "" {
		// Only apply validation if we have a non-nil schema value
		if schemaRef.Value != nil {
			e.applyValidation(schemaRef.Value, validateTag, propertyName, parentSchema)
		}
	}

	return propertyName
}

// extractTagValue extracts a specific tag value.
func extractTagValue(tag, key string) string {
	prefix := key + ":\""
	start := strings.Index(tag, prefix)
	if start == -1 {
		return ""
	}
	start += len(prefix)
	end := strings.Index(tag[start:], "\"")
	if end == -1 {
		return ""
	}
	return tag[start : start+end]
}

// applyValidation applies validation rules to schema.
func (e *SchemaExtractor) applyValidation(schema *openapi3.Schema, validateTag, fieldName string, parentSchema *openapi3.Schema) {
	for rule := range strings.SplitSeq(validateTag, ",") {
		if rule == "required" {
			parentSchema.Required = append(parentSchema.Required, fieldName)
			continue
		}

		// Parse min/max for numeric types
		if valStr, ok := strings.CutPrefix(rule, "min="); ok {
			if val, err := strconv.ParseFloat(valStr, 64); err == nil {
				schema.Min = &val
			}
		}
		if valStr, ok := strings.CutPrefix(rule, "max="); ok {
			if val, err := strconv.ParseFloat(valStr, 64); err == nil {
				schema.Max = &val
			}
		}

		// Parse minLength/maxLength for strings
		if valStr, ok := strings.CutPrefix(rule, "minLength="); ok {
			if val, err := strconv.ParseUint(valStr, 10, 64); err == nil {
				schema.MinLength = val
			}
		}
		if valStr, ok := strings.CutPrefix(rule, "maxLength="); ok {
			if val, err := strconv.ParseUint(valStr, 10, 64); err == nil {
				ui := uint64(val)
				schema.MaxLength = &ui
			}
		}

		// Parse format validators
		switch rule {
		case validateRuleEmail:
			schema.Format = validateRuleEmail
		case validateRuleURL:
			schema.Format = "uri"
		case validateRuleUUID:
			schema.Format = "uuid"
		}

		// Parse oneof enum
		if valStr, ok := strings.CutPrefix(rule, "oneof="); ok {
			for v := range strings.SplitSeq(valStr, " ") {
				schema.Enum = append(schema.Enum, v)
			}
		}
	}
}
