//go:build e2e

// Package helpers provides test utilities for E2E tests
package helpers

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// SpecValidator provides comprehensive OpenAPI spec validation helpers
type SpecValidator struct {
	spec map[string]any
	t    *testing.T
}

// NewSpecValidator creates a new spec validator from a JSON/YAML spec file
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

// GetSpec returns the underlying spec map
func (v *SpecValidator) GetSpec() map[string]any {
	return v.spec
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

// GetPathCount returns the number of paths in the spec
func (v *SpecValidator) GetPathCount() int {
	v.t.Helper()
	return len(v.GetPaths())
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

// ValidateSchemas validates that expected schemas exist
func (v *SpecValidator) ValidateSchemas(expectedSchemas []string) {
	v.t.Helper()

	components, ok := v.spec["components"].(map[string]any)
	if !ok {
		v.t.Error("expected components section to exist")
		return
	}

	schemas, ok := components["schemas"].(map[string]any)
	if !ok {
		v.t.Error("expected components.schemas to exist")
		return
	}

	for _, schemaName := range expectedSchemas {
		if _, exists := schemas[schemaName]; !exists {
			v.t.Errorf("expected schema %s not found", schemaName)
		} else {
			v.t.Logf("Found schema: %s", schemaName)
		}
	}
}
