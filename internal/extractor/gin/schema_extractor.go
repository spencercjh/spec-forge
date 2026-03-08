package gin

import (
	"fmt"
	"go/ast"
	"go/token"
	"slices"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

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
	// Check cache
	if cached, ok := e.typeCache[typeName]; ok {
		return cached, nil
	}

	// Find the type definition
	typeSpec := e.findTypeSpec(typeName)
	if typeSpec == nil {
		return nil, fmt.Errorf("type %s not found", typeName)
	}

	// Extract schema from struct
	schema, err := e.extractStructSchema(typeSpec)
	if err != nil {
		return nil, err
	}

	ref := &openapi3.SchemaRef{Value: schema}
	e.typeCache[typeName] = ref
	return ref, nil
}

// findTypeSpec finds a type definition by name.
func (e *SchemaExtractor) findTypeSpec(name string) *ast.TypeSpec {
	for _, file := range e.files {
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}
			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if ok && typeSpec.Name.Name == name {
					return typeSpec
				}
			}
		}
	}
	return nil
}

// extractStructSchema extracts schema from a struct type.
func (e *SchemaExtractor) extractStructSchema(typeSpec *ast.TypeSpec) (*openapi3.Schema, error) {
	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		return nil, fmt.Errorf("type %s is not a struct", typeSpec.Name.Name)
	}

	schema := &openapi3.Schema{
		Type:       &openapi3.Types{"object"},
		Properties: make(openapi3.Schemas),
	}

	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 {
			continue // Embedded field - skip for now
		}

		fieldName := field.Names[0].Name
		propertyName := fieldName
		fieldSchemaRef := e.fieldToSchemaRef(field.Type)

		// Parse struct tags
		if field.Tag != nil {
			tag := strings.Trim(field.Tag.Value, "`")
			propertyName = e.applyTags(fieldSchemaRef.Value, tag, fieldName, schema)
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
		if schema.Type != nil && (*schema.Type)[0] != "object" {
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
				Type:                 &openapi3.Types{"object"},
				AdditionalProperties: openapi3.AdditionalProperties{Schema: valueSchemaRef},
			},
		}
	case *ast.SelectorExpr:
		// Package qualified type (e.g., time.Time)
		if x, ok := t.X.(*ast.Ident); ok {
			fullName := x.Name + "." + t.Sel.Name
			return &openapi3.SchemaRef{Value: goTypeToSchema(fullName)}
		}
	}

	return &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"object"}}}
}

// fieldToSchema converts a Go type to OpenAPI schema.
func (e *SchemaExtractor) fieldToSchema(expr ast.Expr) *openapi3.Schema {
	switch t := expr.(type) {
	case *ast.Ident:
		// Check if it's a basic type first
		schema := goTypeToSchema(t.Name)
		if schema.Type != nil && (*schema.Type)[0] != "object" {
			return schema
		}
		// It's a custom type - check if we know about it
		if e.findTypeSpec(t.Name) != nil {
			// Return a placeholder schema with type object
			// The reference will be created by the caller if needed
			return &openapi3.Schema{Type: &openapi3.Types{"object"}}
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
			Type:                 &openapi3.Types{"object"},
			AdditionalProperties: openapi3.AdditionalProperties{Schema: &openapi3.SchemaRef{Value: valueSchema}},
		}
	case *ast.SelectorExpr:
		// Package qualified type (e.g., time.Time)
		if x, ok := t.X.(*ast.Ident); ok {
			fullName := x.Name + "." + t.Sel.Name
			return goTypeToSchema(fullName)
		}
	}

	return &openapi3.Schema{Type: &openapi3.Types{"object"}}
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
		return &openapi3.Schema{Type: &openapi3.Types{"object"}}
	}
}

// applyTags processes struct tags and updates schema.
// Returns the final property name (may be different from fieldName due to json tag).
func (e *SchemaExtractor) applyTags(schema *openapi3.Schema, tag, fieldName string, parentSchema *openapi3.Schema) string {
	propertyName := fieldName
	isOmitEmpty := false

	// Parse json tag
	if jsonTag := extractTagValue(tag, "json"); jsonTag != "" {
		parts := strings.Split(jsonTag, ",")
		if parts[0] != "" && parts[0] != "-" {
			propertyName = parts[0]
		}
		// Check for omitempty
		if slices.Contains(parts[1:], "omitempty") {
			isOmitEmpty = true
		}
	}

	// Skip required check if omitempty
	if isOmitEmpty {
		return propertyName
	}

	// Parse binding tag
	if bindingTag := extractTagValue(tag, "binding"); bindingTag != "" {
		if bindingTag == "required" {
			parentSchema.Required = append(parentSchema.Required, propertyName)
		}
	}

	// Parse validate tag
	if validateTag := extractTagValue(tag, "validate"); validateTag != "" {
		e.applyValidation(schema, validateTag, propertyName, parentSchema)
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
		case "email":
			schema.Format = "email"
		case "url":
			schema.Format = "uri"
		case "uuid":
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
