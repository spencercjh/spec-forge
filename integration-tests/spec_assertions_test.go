//go:build e2e

package e2e_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// SpecValidator provides comprehensive OpenAPI spec validation helpers
type SpecValidator struct {
	spec map[string]any
	t    *testing.T
}

// NewSpecValidator creates a new spec validator from a JSON spec file
func NewSpecValidator(t *testing.T, specFile string) *SpecValidator {
	t.Helper()

	specData, err := os.ReadFile(specFile)
	if err != nil {
		t.Fatalf("failed to read spec file: %v", err)
	}

	var spec map[string]any

	if strings.HasSuffix(specFile, ".yaml") || strings.HasSuffix(specFile, ".yml") {
		if err := yaml.Unmarshal(specData, &spec); err != nil {
			t.Fatalf("failed to parse spec YAML: %v", err)
		}
	} else {
		if err := json.Unmarshal(specData, &spec); err != nil {
			t.Fatalf("failed to parse spec JSON: %v", err)
		}
	}

	return &SpecValidator{spec: spec, t: t}
}

// ValidateOpenAPIVersion validates the OpenAPI version field
func (v *SpecValidator) ValidateOpenAPIVersion() {
	v.t.Helper()

	version, ok := v.spec["openapi"].(string)
	if !ok || version == "" {
		v.t.Error("expected openapi version field to be a non-empty string")
		return
	}

	// Should start with 3.
	if !strings.HasPrefix(version, "3.") {
		v.t.Errorf("expected OpenAPI version to start with '3.', got %s", version)
	}
	v.t.Logf("OpenAPI version: %s", version)
}

// ValidateInfo validates the info section
func (v *SpecValidator) ValidateInfo() {
	v.t.Helper()

	info, ok := v.spec["info"].(map[string]any)
	if !ok {
		v.t.Error("expected info section to be an object")
		return
	}

	// Validate title
	title, ok := info["title"].(string)
	if !ok || title == "" {
		v.t.Error("expected info.title to be a non-empty string")
	} else {
		v.t.Logf("API Title: %s", title)
	}

	// Validate version
	version, ok := info["version"].(string)
	if !ok || version == "" {
		v.t.Error("expected info.version to be a non-empty string")
	} else {
		v.t.Logf("API Version: %s", version)
	}
}

// GetPaths returns the paths map
func (v *SpecValidator) GetPaths() map[string]any {
	v.t.Helper()

	paths, ok := v.spec["paths"].(map[string]any)
	if !ok {
		v.t.Fatal("expected paths to be an object")
	}
	return paths
}

// ValidatePaths validates that expected paths exist
func (v *SpecValidator) ValidatePaths(expectedPaths []string) {
	v.t.Helper()

	paths := v.GetPaths()
	for _, path := range expectedPaths {
		if _, exists := paths[path]; !exists {
			v.t.Errorf("expected path %s not found in spec", path)
		} else {
			v.t.Logf("Found path: %s", path)
		}
	}
}

// ValidatePath validates a specific path exists and returns its operations
func (v *SpecValidator) ValidatePath(path string) map[string]any {
	v.t.Helper()

	paths := v.GetPaths()
	pathItem, exists := paths[path]
	if !exists {
		v.t.Errorf("expected path %s not found in spec", path)
		return nil
	}

	operations, ok := pathItem.(map[string]any)
	if !ok {
		v.t.Errorf("expected path %s to have operations object", path)
		return nil
	}

	return operations
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

	v.t.Logf("Found %s %s", method, path)
	return operationMap
}

// ValidateOperationFields validates common operation fields
func (v *SpecValidator) ValidateOperationFields(path, method string, wantOperationID, wantSummary bool) {
	v.t.Helper()

	operation := v.ValidateOperation(path, method)
	if operation == nil {
		return
	}

	// Validate operationId
	if wantOperationID {
		operationID, ok := operation["operationId"].(string)
		if !ok || operationID == "" {
			v.t.Errorf("expected operationId for %s %s", method, path)
		} else {
			v.t.Logf("  operationId: %s", operationID)
		}
	}

	// Validate summary
	if wantSummary {
		summary, ok := operation["summary"].(string)
		if !ok || summary == "" {
			v.t.Errorf("expected summary for %s %s", method, path)
		} else {
			v.t.Logf("  summary: %s", summary)
		}
	}
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

// ValidateResponseContent validates response content type and schema
func (v *SpecValidator) ValidateResponseContent(path, method, code, contentType string) {
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

	response, exists := responses[code]
	if !exists {
		v.t.Errorf("expected response code %s for %s %s", code, method, path)
		return
	}

	responseMap, ok := response.(map[string]any)
	if !ok {
		v.t.Errorf("expected response %s to be an object", code)
		return
	}

	content, ok := responseMap["content"].(map[string]any)
	if !ok {
		v.t.Errorf("expected content for response %s %s %s", method, path, code)
		return
	}

	if _, exists := content[contentType]; !exists {
		v.t.Errorf("expected content type %s for response %s %s %s", contentType, method, path, code)
	} else {
		v.t.Logf("  content type %s for %s: OK", contentType, code)
	}
}

// ValidateRequestBody validates request body exists and has content
func (v *SpecValidator) ValidateRequestBody(path, method, contentType string) {
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

	if _, exists := content[contentType]; !exists {
		v.t.Errorf("expected content type %s in requestBody for %s %s", contentType, method, path)
	} else {
		v.t.Logf("  requestBody content type %s: OK", contentType)
	}
}

// ValidateParameters validates parameters exist
func (v *SpecValidator) ValidateParameters(path, method string, expectedParams []string) {
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

	paramNames := make(map[string]bool)
	for _, p := range params {
		paramMap, ok := p.(map[string]any)
		if !ok {
			continue
		}
		name, ok := paramMap["name"].(string)
		if ok {
			paramNames[name] = true
		}
	}

	for _, param := range expectedParams {
		if !paramNames[param] {
			v.t.Errorf("expected parameter %s for %s %s", param, method, path)
		} else {
			v.t.Logf("  parameter %s: OK", param)
		}
	}
}

// ParameterExpectation defines expected parameter properties
type ParameterExpectation struct {
	Name     string
	In       string // path, query, header, formData
	Required bool
	Type     string // string, integer, boolean, etc.
}

// ValidateParameterDetails validates parameter details including 'in' location and 'required'
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

	paramMap := make(map[string]map[string]any)
	for _, p := range params {
		param, ok := p.(map[string]any)
		if !ok {
			continue
		}
		name, ok := param["name"].(string)
		if ok {
			paramMap[name] = param
		}
	}

	for _, exp := range expectations {
		param, exists := paramMap[exp.Name]
		if !exists {
			v.t.Errorf("expected parameter %s for %s %s", exp.Name, method, path)
			continue
		}

		// Validate 'in' location
		if exp.In != "" {
			paramIn, ok := param["in"].(string)
			if !ok || paramIn != exp.In {
				v.t.Errorf("expected parameter %s to have in=%s, got %s", exp.Name, exp.In, paramIn)
			} else {
				v.t.Logf("  parameter %s in=%s: OK", exp.Name, exp.In)
			}
		}

		// Validate 'required'
		if exp.Required {
			required, ok := param["required"].(bool)
			if !ok || !required {
				v.t.Errorf("expected parameter %s to be required", exp.Name)
			} else {
				v.t.Logf("  parameter %s required=true: OK", exp.Name)
			}
		}
	}
}

// ValidateRequestBodySchema validates that request body references the expected schema
func (v *SpecValidator) ValidateRequestBodySchema(path, method, expectedSchema string) {
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

	// Check all content types for schema reference
	for contentType, mediaType := range content {
		mediaTypeMap, ok := mediaType.(map[string]any)
		if !ok {
			continue
		}

		schema, ok := mediaTypeMap["schema"].(map[string]any)
		if !ok {
			continue
		}

		schemaRef, ok := schema["$ref"].(string)
		if ok {
			expectedRef := "#/components/schemas/" + expectedSchema
			if schemaRef == expectedRef {
				v.t.Logf("  requestBody schema ref %s: OK (content-type: %s)", expectedSchema, contentType)
				return
			}
		}
	}

	v.t.Errorf("expected requestBody to reference schema %s for %s %s", expectedSchema, method, path)
}

// SchemaPropertyExpectation defines expected schema property
type SchemaPropertyExpectation struct {
	Name     string
	Type     string
	ItemType string // For array types, the item schema reference
}

// ValidateSchemaProperty validates a specific schema property
func (v *SpecValidator) ValidateSchemaProperty(schemaName string, prop SchemaPropertyExpectation) {
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
		v.t.Errorf("expected schema %s not found", schemaName)
		return
	}

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		v.t.Errorf("expected properties in schema %s", schemaName)
		return
	}

	propData, ok := properties[prop.Name].(map[string]any)
	if !ok {
		v.t.Errorf("expected property %s in schema %s", prop.Name, schemaName)
		return
	}

	// Validate type
	if prop.Type != "" {
		propType, ok := propData["type"].(string)
		if !ok || propType != prop.Type {
			v.t.Errorf("expected property %s to have type=%s, got %s", prop.Name, prop.Type, propType)
		} else {
			v.t.Logf("  schema %s.%s type=%s: OK", schemaName, prop.Name, prop.Type)
		}
	}

	// For array types, validate items
	if prop.Type == "array" && prop.ItemType != "" {
		items, ok := propData["items"].(map[string]any)
		if !ok {
			v.t.Errorf("expected items for array property %s", prop.Name)
			return
		}

		itemRef, ok := items["$ref"].(string)
		if ok {
			expectedRef := "#/components/schemas/" + prop.ItemType
			if itemRef == expectedRef {
				v.t.Logf("  schema %s.%s items=$ref:%s: OK", schemaName, prop.Name, prop.ItemType)
			} else {
				v.t.Errorf("expected array items to reference %s, got %s", expectedRef, itemRef)
			}
		}
	}
}

// ResponseSchemaExpectation defines expected response schema structure
type ResponseSchemaExpectation struct {
	Code       string
	ContentType string
	SchemaRef  string // Expected schema reference (e.g., "User", "ApiResponse")
}

// ValidateResponseSchema validates response has proper schema reference
func (v *SpecValidator) ValidateResponseSchema(path, method string, exp ResponseSchemaExpectation) {
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

	response, ok := responses[exp.Code].(map[string]any)
	if !ok {
		v.t.Errorf("expected response code %s for %s %s", exp.Code, method, path)
		return
	}

	content, ok := response["content"].(map[string]any)
	if !ok {
		v.t.Errorf("expected content for response %s %s %s", method, path, exp.Code)
		return
	}

	mediaType, ok := content[exp.ContentType].(map[string]any)
	if !ok {
		v.t.Errorf("expected content type %s for response %s %s %s", exp.ContentType, method, path, exp.Code)
		return
	}

	schema, ok := mediaType["schema"].(map[string]any)
	if !ok {
		v.t.Errorf("expected schema for response %s %s %s", method, path, exp.Code)
		return
	}

	schemaRef, ok := schema["$ref"].(string)
	if !ok {
		// Check if it's an inline object schema
		schemaType, typeOk := schema["type"].(string)
		if typeOk && schemaType == "object" {
			v.t.Logf("  response %s schema type=object (inline): OK", exp.Code)
			return
		}
		v.t.Errorf("expected schema reference or object type for response %s %s %s", method, path, exp.Code)
		return
	}

	expectedRef := "#/components/schemas/" + exp.SchemaRef
	if schemaRef == expectedRef {
		v.t.Logf("  response %s schema=$ref:%s: OK", exp.Code, exp.SchemaRef)
	} else {
		v.t.Errorf("expected response schema %s, got %s", expectedRef, schemaRef)
	}
}

// ValidateSchemas validates that expected schemas exist in components
func (v *SpecValidator) ValidateSchemas(expectedSchemas []string) {
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

	for _, schema := range expectedSchemas {
		if _, exists := schemas[schema]; !exists {
			v.t.Errorf("expected schema %s not found", schema)
		} else {
			v.t.Logf("Found schema: %s", schema)
		}
	}
}

// GetPathCount returns the number of paths in the spec
func (v *SpecValidator) GetPathCount() int {
	v.t.Helper()

	paths, ok := v.spec["paths"].(map[string]any)
	if !ok {
		return 0
	}
	return len(paths)
}

// LogSummary logs a summary of the spec
func (v *SpecValidator) LogSummary() {
	v.t.Helper()

	paths := v.GetPaths()
	v.t.Logf("Spec Summary: %d paths", len(paths))

	for path, ops := range paths {
		operations, ok := ops.(map[string]any)
		if !ok {
			continue
		}
		methods := make([]string, 0)
		for method := range operations {
			methods = append(methods, method)
		}
		v.t.Logf("  %s: %v", path, methods)
	}
}

// FullValidation performs a complete validation of the spec
func (v *SpecValidator) FullValidation(config ValidationConfig) {
	v.t.Helper()

	v.t.Log("=== OpenAPI Spec Validation ===")

	// Basic structure
	v.ValidateOpenAPIVersion()
	v.ValidateInfo()

	// Paths
	if len(config.ExpectedPaths) > 0 {
		v.ValidatePaths(config.ExpectedPaths)
	}

	// Operations
	for _, op := range config.Operations {
		v.ValidateOperationFields(op.Path, op.Method, op.WantOperationID, op.WantSummary)
		if len(op.ExpectedResponseCodes) > 0 {
			v.ValidateResponseCodes(op.Path, op.Method, op.ExpectedResponseCodes)
		}
		if op.ValidateResponseContent != "" {
			// Use first expected response code, or default to "200"
			responseCode := "200"
			if len(op.ExpectedResponseCodes) > 0 {
				responseCode = op.ExpectedResponseCodes[0]
			}
			v.ValidateResponseContent(op.Path, op.Method, responseCode, op.ValidateResponseContent)
		}
		if op.WantRequestBody != "" {
			v.ValidateRequestBody(op.Path, op.Method, op.WantRequestBody)
		}
		if len(op.ExpectedParams) > 0 {
			v.ValidateParameters(op.Path, op.Method, op.ExpectedParams)
		}
	}

	// Schemas
	if len(config.ExpectedSchemas) > 0 {
		v.ValidateSchemas(config.ExpectedSchemas)
	}

	v.LogSummary()
}

// ValidationConfig defines the configuration for full validation
type ValidationConfig struct {
	ExpectedPaths   []string
	Operations      []OperationConfig
	ExpectedSchemas []string
}

// OperationConfig defines validation config for a single operation
type OperationConfig struct {
	Path                    string
	Method                  string
	WantOperationID         bool
	WantSummary             bool
	ExpectedResponseCodes   []string
	ValidateResponseContent string
	WantRequestBody         string
	ExpectedParams          []string
}

// FindSpecFile finds a spec file in the output directory
func FindSpecFile(t *testing.T, outputDir, format string) string {
	t.Helper()

	entries, err := os.ReadDir(outputDir)
	if err != nil {
		t.Fatalf("failed to read output directory: %v", err)
	}

	switch format {
	case "json":
		for _, entry := range entries {
			if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
				return filepath.Join(outputDir, entry.Name())
			}
		}
	case "yaml":
		for _, entry := range entries {
			if !entry.IsDir() && (filepath.Ext(entry.Name()) == ".yaml" || filepath.Ext(entry.Name()) == ".yml") {
				return filepath.Join(outputDir, entry.Name())
			}
		}
	default:
		// Accept any format
		for _, entry := range entries {
			if !entry.IsDir() {
				ext := filepath.Ext(entry.Name())
				if ext == ".json" || ext == ".yaml" || ext == ".yml" {
					return filepath.Join(outputDir, entry.Name())
				}
			}
		}
	}

	t.Fatal("no spec file found in output directory")
	return ""
}
