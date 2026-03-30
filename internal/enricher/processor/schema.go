package processor

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/spencercjh/spec-forge/internal/enricher/prompt"
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
	force bool,
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

		// Skip fields that already have descriptions (unless force is true)
		if !force && prop.Description != "" {
			collector.Skipped.SchemaFields++
			continue
		}

		field := FieldElement{
			FieldName:           propName,
			FieldType:           getSchemaTypeName(prop),
			Required:            containsString(schema.Required, propName),
			Format:              prop.Format,
			Enum:                buildEnumStrings(prop.Enum),
			Constraints:         buildConstraintsString(prop),
			ExistingDescription: prop.Description,
		}

		// Capture prop for closure
		propPtr := prop
		field.SetValue = func(desc string) {
			propPtr.Description = desc
		}
		fields = append(fields, field)

		// Recursively process nested objects
		if prop.Type != nil && len(*prop.Type) > 0 && (*prop.Type)[0] == "object" && prop.Properties != nil {
			nestedName := schemaName + "." + propName
			CollectSchemaFields(nestedName, propRef, collector, processed, language, depth+1, force)
		}

		// Recursively process array items
		if prop.Type != nil && len(*prop.Type) > 0 && (*prop.Type)[0] == "array" && prop.Items != nil {
			nestedName := schemaName + "." + propName + "[]"
			CollectSchemaFields(nestedName, prop.Items, collector, processed, language, depth+1, force)
		}
	}

	// Only add if there are fields to enrich
	if len(fields) > 0 {
		collector.AddSchemaElement(SchemaElement{
			SchemaName: schemaName,
			Fields:     fields,
			Context: prompt.TemplateContext{
				Type:       prompt.TemplateTypeSchema,
				SchemaName: schemaName,
				Language:   language,
				Fields:     convertFieldElements(fields),
			},
		}, language)
	}
}

// getSchemaTypeName returns a human-readable type name.
func getSchemaTypeName(schema *openapi3.Schema) string {
	if schema.Type != nil && len(*schema.Type) > 0 {
		if (*schema.Type)[0] == "array" && schema.Items != nil && schema.Items.Value != nil {
			return "array of " + getSchemaTypeName(schema.Items.Value)
		}
		return (*schema.Type)[0]
	}
	return "object"
}

// containsString checks if a string is in a slice.
func containsString(slice []string, s string) bool {
	return slices.Contains(slice, s)
}

// buildConstraintsString builds a human-readable constraints description from a schema.
func buildConstraintsString(schema *openapi3.Schema) string {
	var parts []string
	if schema.Min != nil {
		parts = append(parts, fmt.Sprintf("min: %v", *schema.Min))
	}
	if schema.Max != nil {
		parts = append(parts, fmt.Sprintf("max: %v", *schema.Max))
	}
	if schema.MinLength > 0 {
		parts = append(parts, fmt.Sprintf("minLength: %d", schema.MinLength))
	}
	if schema.MaxLength != nil {
		parts = append(parts, fmt.Sprintf("maxLength: %d", *schema.MaxLength))
	}
	if schema.Pattern != "" {
		parts = append(parts, "pattern: "+schema.Pattern)
	}
	if schema.MinItems > 0 {
		parts = append(parts, fmt.Sprintf("minItems: %d", schema.MinItems))
	}
	if schema.MaxItems != nil {
		parts = append(parts, fmt.Sprintf("maxItems: %d", *schema.MaxItems))
	}
	return strings.Join(parts, ", ")
}

// buildEnumStrings converts []any to []string for enum values.
func buildEnumStrings(enum []any) []string {
	if len(enum) == 0 {
		return nil
	}
	result := make([]string, len(enum))
	for i, v := range enum {
		result[i] = fmt.Sprintf("%v", v)
	}
	return result
}
