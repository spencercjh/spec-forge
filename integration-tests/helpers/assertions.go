//go:build e2e

package helpers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// FullValidation performs comprehensive spec validation
func (v *SpecValidator) FullValidation(config ValidationConfig) {
	v.t.Helper()

	v.t.Log("=== OpenAPI Spec Validation ===")

	// Validate version and info
	v.ValidateOpenAPIVersion()
	v.ValidateInfo()

	// Validate paths
	if len(config.ExpectedPaths) > 0 {
		v.ValidatePaths(config.ExpectedPaths)
	}

	// Validate operations
	for _, op := range config.Operations {
		v.validateOperationConfig(op)
	}

	// Validate schemas
	if len(config.ExpectedSchemas) > 0 {
		v.ValidateSchemas(config.ExpectedSchemas)
	}

	// Log summary
	v.logSummary()
}

func (v *SpecValidator) validateOperationConfig(op OperationConfig) {
	v.t.Helper()

	operation := v.ValidateOperation(op.Path, op.Method)
	if operation == nil {
		return
	}

	// Validate operationId
	if op.WantOperationID {
		operationID, ok := operation["operationId"].(string)
		if !ok || operationID == "" {
			v.t.Errorf("expected operationId for %s %s", op.Method, op.Path)
		} else {
			v.t.Logf("Found %s %s", op.Method, op.Path)
			v.t.Logf("  operationId: %s", operationID)
		}
	}

	// Validate response codes
	if len(op.ExpectedResponseCodes) > 0 {
		v.ValidateResponseCodes(op.Path, op.Method, op.ExpectedResponseCodes)
	}

	// Validate parameters
	if len(op.ExpectedParams) > 0 {
		v.ValidateParameterDetails(op.Path, op.Method, paramsToExpectations(op.ExpectedParams))
	}

	// Validate response content
	if op.ValidateResponseContent != "" {
		v.ValidateResponseContent(op.Path, op.Method, op.ValidateResponseContent)
	}

	// Validate request body
	if op.WantRequestBody != "" {
		v.ValidateRequestBodySchema(op.Path, op.Method, op.WantRequestBody)
	}
}

func paramsToExpectations(params []string) []ParameterExpectation {
	expectations := make([]ParameterExpectation, len(params))
	for i, p := range params {
		expectations[i] = ParameterExpectation{Name: p, In: "path", Required: true}
	}
	return expectations
}

// ValidateOperation validates a specific HTTP method exists on a path
func (v *SpecValidator) ValidateOperation(path, method string) map[string]any {
	v.t.Helper()

	operations := v.ValidatePath(path)
	if operations == nil {
		return nil
	}

	operation, exists := operations[method]
	if !exists {
		v.t.Errorf("expected %s method on path %s not found", method, path)
		return nil
	}

	operationMap, ok := operation.(map[string]any)
	if !ok {
		v.t.Errorf("expected %s operation on path %s to be an object", method, path)
		return nil
	}

	return operationMap
}

// ValidateResponseCodes validates that expected response codes exist
func (v *SpecValidator) ValidateResponseCodes(path, method string, expectedCodes []string) {
	v.t.Helper()

	operation := v.ValidateOperation(path, method)
	if operation == nil {
		return
	}

	responses, ok := operation["responses"].(map[string]any)
	if !ok {
		v.t.Errorf("expected responses for %s %s", method, path)
		return
	}

	for _, code := range expectedCodes {
		if _, exists := responses[code]; !exists {
			v.t.Errorf("expected response code %s for %s %s", code, method, path)
		} else {
			v.t.Logf("  response %s: OK", code)
		}
	}
}

// ValidateParameterDetails validates parameter details
func (v *SpecValidator) ValidateParameterDetails(path, method string, expectations []ParameterExpectation) {
	v.t.Helper()

	operation := v.ValidateOperation(path, method)
	if operation == nil {
		return
	}

	params, ok := operation["parameters"].([]any)
	if !ok {
		v.t.Errorf("expected parameters for %s %s", method, path)
		return
	}

	for _, exp := range expectations {
		found := false
		for _, p := range params {
			param, ok := p.(map[string]any)
			if !ok {
				continue
			}

			name, _ := param["name"].(string)
			in, _ := param["in"].(string)
			required, _ := param["required"].(bool)

			if name == exp.Name {
				found = true
				if in != exp.In {
					v.t.Errorf("parameter %s: expected in=%s, got %s", exp.Name, exp.In, in)
				}
				if required != exp.Required {
					v.t.Errorf("parameter %s: expected required=%v, got %v", exp.Name, exp.Required, required)
				}
				break
			}
		}
		if !found {
			v.t.Errorf("parameter %s not found for %s %s", exp.Name, method, path)
		}
	}
}

// ValidateSchemaProperty validates a schema property
func (v *SpecValidator) ValidateSchemaProperty(schemaName string, expectation SchemaPropertyExpectation) {
	v.t.Helper()

	components, ok := v.spec["components"].(map[string]any)
	if !ok {
		v.t.Error("expected components section")
		return
	}

	schemas, ok := components["schemas"].(map[string]any)
	if !ok {
		v.t.Error("expected components.schemas section")
		return
	}

	schema, ok := schemas[schemaName].(map[string]any)
	if !ok {
		v.t.Errorf("schema %s not found", schemaName)
		return
	}

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		v.t.Errorf("schema %s has no properties", schemaName)
		return
	}

	prop, ok := properties[expectation.Name].(map[string]any)
	if !ok {
		v.t.Errorf("property %s not found in schema %s", expectation.Name, schemaName)
		return
	}

	// Validate type
	if expectation.Type != "" {
		propType, _ := prop["type"].(string)
		if propType != expectation.Type {
			v.t.Errorf("schema %s.%s: expected type=%s, got %s", schemaName, expectation.Name, expectation.Type, propType)
		}
	}

	// Validate item type for arrays
	if expectation.ItemType != "" {
		items, ok := prop["items"].(map[string]any)
		if !ok {
			v.t.Errorf("schema %s.%s: expected items for array type", schemaName, expectation.Name)
			return
		}

		// Check if it's a $ref to the expected type
		if ref, hasRef := items["$ref"].(string); hasRef {
			expectedRef := "#/components/schemas/" + expectation.ItemType
			if ref != expectedRef {
				v.t.Errorf("schema %s.%s items: expected $ref=%s, got %s", schemaName, expectation.Name, expectedRef, ref)
			}
		} else {
			// Check type directly
			itemType, _ := items["type"].(string)
			if itemType != expectation.ItemType {
				v.t.Errorf("schema %s.%s items: expected type=%s, got %s", schemaName, expectation.Name, expectation.ItemType, itemType)
			}
		}
	}

	// Validate $ref
	if expectation.Ref != "" {
		ref, hasRef := prop["$ref"].(string)
		if !hasRef {
			v.t.Errorf("schema %s.%s: expected $ref, got none", schemaName, expectation.Name)
			return
		}
		expectedRef := "#/components/schemas/" + expectation.Ref
		if ref != expectedRef {
			v.t.Errorf("schema %s.%s: expected $ref=%s, got %s", schemaName, expectation.Name, expectedRef, ref)
		}
	}

	v.t.Logf("  schema %s.%s: OK", schemaName, expectation.Name)
}

// ValidateResponseSchema validates response schema reference
func (v *SpecValidator) ValidateResponseSchema(path, method string, expectation ResponseSchemaExpectation) {
	v.t.Helper()

	operation := v.ValidateOperation(path, method)
	if operation == nil {
		return
	}

	responses, ok := operation["responses"].(map[string]any)
	if !ok {
		v.t.Errorf("expected responses for %s %s", method, path)
		return
	}

	response, ok := responses[expectation.Code].(map[string]any)
	if !ok {
		v.t.Errorf("response %s not found for %s %s", expectation.Code, method, path)
		return
	}

	content, ok := response["content"].(map[string]any)
	if !ok {
		v.t.Errorf("expected content for response %s %s %s", expectation.Code, method, path)
		return
	}

	mediaType, ok := content[expectation.ContentType].(map[string]any)
	if !ok {
		v.t.Errorf("expected %s content for response %s %s %s", expectation.ContentType, expectation.Code, method, path)
		return
	}

	schema, ok := mediaType["schema"].(map[string]any)
	if !ok {
		v.t.Errorf("expected schema for response %s %s %s", expectation.Code, method, path)
		return
	}

	ref, ok := schema["$ref"].(string)
	if !ok {
		v.t.Errorf("expected $ref in response schema for %s %s %s", expectation.Code, method, path)
		return
	}

	expectedRef := "#/components/schemas/" + expectation.SchemaRef
	if ref != expectedRef {
		v.t.Errorf("response schema $ref mismatch for %s %s %s: expected %s, got %s", expectation.Code, method, path, expectedRef, ref)
	}
}

// ValidateRequestBodySchema validates request body schema reference
func (v *SpecValidator) ValidateRequestBodySchema(path, method, schemaName string) {
	v.t.Helper()

	operation := v.ValidateOperation(path, method)
	if operation == nil {
		return
	}

	requestBody, ok := operation["requestBody"].(map[string]any)
	if !ok {
		v.t.Errorf("expected requestBody for %s %s", method, path)
		return
	}

	content, ok := requestBody["content"].(map[string]any)
	if !ok {
		v.t.Errorf("expected content in requestBody for %s %s", method, path)
		return
	}

	// Try application/json first
	mediaType, ok := content["application/json"].(map[string]any)
	if !ok {
		// Fallback to any content type
		for _, mt := range content {
			mediaType, ok = mt.(map[string]any)
			if ok {
				break
			}
		}
	}

	if !ok {
		v.t.Errorf("expected media type in requestBody for %s %s", method, path)
		return
	}

	schema, ok := mediaType["schema"].(map[string]any)
	if !ok {
		v.t.Errorf("expected schema in requestBody for %s %s", method, path)
		return
	}

	ref, ok := schema["$ref"].(string)
	if !ok {
		v.t.Errorf("expected $ref in requestBody schema for %s %s", method, path)
		return
	}

	expectedRef := "#/components/schemas/" + schemaName
	if ref != expectedRef {
		v.t.Errorf("requestBody schema $ref mismatch for %s %s: expected %s, got %s", method, path, expectedRef, ref)
	}
}

// ValidateResponseContent validates response content type exists
func (v *SpecValidator) ValidateResponseContent(path, method, contentType string) {
	v.t.Helper()

	operation := v.ValidateOperation(path, method)
	if operation == nil {
		return
	}

	responses, ok := operation["responses"].(map[string]any)
	if !ok {
		return
	}

	for _, resp := range responses {
		response, ok := resp.(map[string]any)
		if !ok {
			continue
		}

		content, ok := response["content"].(map[string]any)
		if !ok {
			continue
		}

		if _, exists := content[contentType]; exists {
			return // Found it
		}
	}

	v.t.Errorf("expected %s response content for %s %s", contentType, method, path)
}

// logSummary logs a summary of the spec
func (v *SpecValidator) logSummary() {
	v.t.Helper()

	paths := v.GetPaths()
	v.t.Logf("Spec Summary: %d paths", len(paths))

	for path, ops := range paths {
		if operations, ok := ops.(map[string]any); ok {
			var methods []string
			for method := range operations {
				methods = append(methods, method)
			}
			v.t.Logf("  %s: %v", path, methods)
		}
	}
}

// FindSpecFile finds the generated spec file in a directory
func FindSpecFile(t *testing.T, dir, format string) string {
	t.Helper()

	ext := ".json"
	if format == "yaml" || format == "yml" {
		ext = ".yaml"
	}

	// Look for openapi.{ext} or any {ext} file
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read directory %s: %v", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ext) {
			return filepath.Join(dir, entry.Name())
		}
	}

	t.Fatalf("no spec file found in %s", dir)
	return ""
}

// LogSummary logs a summary of the spec (public method)
func (v *SpecValidator) LogSummary() {
	v.logSummary()
}

// ValidateOperationFields validates operation fields exist
func (v *SpecValidator) ValidateOperationFields(path, method string, wantOperationID, wantSummary bool) {
	v.t.Helper()

	operation := v.ValidateOperation(path, method)
	if operation == nil {
		return
	}

	if wantOperationID {
		if operationID, ok := operation["operationId"].(string); !ok || operationID == "" {
			v.t.Errorf("expected operationId for %s %s", method, path)
		}
	}

	if wantSummary {
		if summary, ok := operation["summary"].(string); !ok || summary == "" {
			v.t.Errorf("expected summary for %s %s", method, path)
		}
	}
}

// ExtractFromPath extracts a value from a nested map using dot-notation path
// Supports OpenAPI-style paths like "paths./api/v1/users.get" or "components.schemas.User"
func ExtractFromPath(t *testing.T, spec map[string]any, path string) any {
	t.Helper()

	// Handle paths with API paths like "paths./api/v1/users.get"
	var parts []string

	if strings.HasPrefix(path, "paths./") {
		// Format: paths./api/v1/users.get or paths./api/v1/users/{id}.get
		lastDot := strings.LastIndex(path, ".")
		if lastDot > 0 {
			prefix := path[:lastDot]   // paths./api/v1/users
			method := path[lastDot+1:] // get
			apiPath := strings.TrimPrefix(prefix, "paths.")
			parts = []string{"paths", apiPath, method}
		} else {
			parts = strings.Split(path, ".")
		}
	} else if strings.HasPrefix(path, "components.schemas.") {
		parts = []string{"components", "schemas", strings.TrimPrefix(path, "components.schemas.")}
	} else {
		parts = strings.Split(path, ".")
	}

	current := any(spec)
	for _, part := range parts {
		if current == nil {
			return nil
		}
		m, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = m[part]
	}
	return current
}
