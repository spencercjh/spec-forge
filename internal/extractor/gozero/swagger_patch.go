package gozero

import (
	"log/slog"
	"regexp"
	"slices"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi3"
)

// PatchSwagger patches known bugs in goctl swagger generation.
// This addresses issues #5426, #5427, and #5428.
func PatchSwagger(doc *openapi2.T) {
	if doc == nil {
		slog.Debug("PatchSwagger called with nil document")
		return
	}

	pathCount := len(doc.Paths)
	definitionCount := len(doc.Definitions)
	slog.Debug("patching swagger document", "paths", pathCount, "definitions", definitionCount)

	// Patch all paths
	for path, pathItem := range doc.Paths {
		if pathItem == nil {
			continue
		}

		// Extract valid path parameters from the URL path
		validPathParams := extractPathParams(path)

		// Patch operations
		patchOperation(pathItem.Get, validPathParams)
		patchOperation(pathItem.Post, validPathParams)
		patchOperation(pathItem.Put, validPathParams)
		patchOperation(pathItem.Delete, validPathParams)
		patchOperation(pathItem.Patch, validPathParams)
		patchOperation(pathItem.Head, validPathParams)
		patchOperation(pathItem.Options, validPathParams)
	}

	// Patch schema definitions
	for _, schemaRef := range doc.Definitions {
		if schemaRef != nil && schemaRef.Value != nil {
			patchSchema(schemaRef.Value)
		}
	}

	slog.Debug("finished patching swagger document")
}

// patchOperation patches a single operation's parameters and responses.
func patchOperation(op *openapi2.Operation, validPathParams []string) {
	if op == nil {
		return
	}

	// Patch parameters
	patchedParams := make(openapi2.Parameters, 0, len(op.Parameters))
	existingPathParams := make(map[string]bool)

	for _, p := range op.Parameters {
		// Patch #5427: Skip parameters with name "-"
		if p.Name == "-" {
			continue
		}

		// Patch #5428: Remove path params not in URL
		if p.In == "path" {
			if !slices.Contains(validPathParams, p.Name) {
				continue
			}
			existingPathParams[p.Name] = true
		}

		patchedParams = append(patchedParams, p)
	}

	// Patch #5428: Add missing path params
	for _, paramName := range validPathParams {
		if !existingPathParams[paramName] {
			patchedParams = append(patchedParams, &openapi2.Parameter{
				Name:     paramName,
				In:       "path",
				Required: true,
				Type:     &openapi3.Types{"string"},
			})
		}
	}

	op.Parameters = patchedParams

	// Patch response schemas
	for _, resp := range op.Responses {
		if resp != nil && resp.Schema != nil {
			patchSchema(resp.Schema.Value)
		}
	}
}

// patchSchema patches a schema and its nested schemas recursively.
func patchSchema(schema *openapi2.Schema) {
	if schema == nil {
		return
	}

	// Patch #5426: Array types missing items
	if schema.Type != nil && schema.Type.Is("array") && schema.Items == nil {
		schema.Items = &openapi2.SchemaRef{
			Value: &openapi2.Schema{
				Type: &openapi3.Types{"object"},
			},
		}
	}

	// Patch nested schemas in properties
	for _, propRef := range schema.Properties {
		if propRef != nil {
			patchSchema(propRef.Value)
		}
	}

	// Patch items schema if present
	if schema.Items != nil {
		patchSchema(schema.Items.Value)
	}

	// Note: AdditionalProperties contains openapi3.Schema types which are incompatible
	// with openapi2.Schema. This is a limitation when patching goctl-generated specs.
	// The main patches (#5426-#5428) focus on array items and parameters.

	// Patch allOf schemas
	for _, allOfRef := range schema.AllOf {
		if allOfRef != nil {
			patchSchema(allOfRef.Value)
		}
	}

	// Patch not schema
	if schema.Not != nil {
		patchSchema(schema.Not.Value)
	}
}

// extractPathParams extracts path parameter names from a URL path using regex.
// Example: "/users/{id}/posts/{postId}" -> ["id", "postId"]
func extractPathParams(path string) []string {
	// Use [^\}]+ to capture path parameters with hyphens and other valid characters
	// (e.g., {user-id}, {api-key})
	re := regexp.MustCompile(`\{([^\}]+)\}`)
	matches := re.FindAllStringSubmatch(path, -1)

	params := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			params = append(params, match[1])
		}
	}

	return params
}
