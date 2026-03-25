//go:build e2e

package gozero

import (
	"strings"
	"testing"

	"github.com/spencercjh/spec-forge/integration-tests/helpers"
)

// TestAPITypeDefinitions tests that go-zero API type definitions are correctly parsed
func TestAPITypeDefinitions(t *testing.T) {
	_, spec := generateSpec(t, "json")

	t.Run("User Fields Inline in Response", func(t *testing.T) {
		// go-zero inlines schemas in responses (no $ref to components)
		userResp := helpers.ExtractFromPath(t, spec, "paths./api/v1/users/{id}.get")
		if userResp == nil {
			t.Fatal("GET /api/v1/users/{id} not found")
		}

		op, ok := userResp.(map[string]any)
		if !ok {
			t.Fatal("operation is not an object")
		}

		responses, ok := op["responses"].(map[string]any)
		if !ok {
			t.Fatal("responses is not an object")
		}
		resp200, ok := responses["200"].(map[string]any)
		if !ok {
			t.Fatal("200 response is not an object")
		}
		content, ok := resp200["content"].(map[string]any)
		if !ok {
			t.Fatal("content is not an object")
		}
		jsonContent, ok := content["application/json"].(map[string]any)
		if !ok {
			t.Fatal("application/json content is not an object")
		}
		schema, ok := jsonContent["schema"].(map[string]any)
		if !ok {
			t.Fatal("schema is not an object")
		}
		properties, ok := schema["properties"].(map[string]any)
		if !ok {
			t.Fatal("properties is not an object")
		}

		// Check the data property contains User fields
		data, ok := properties["data"].(map[string]any)
		if !ok {
			t.Fatal("data property is not an object")
		}
		dataProps, ok := data["properties"].(map[string]any)
		if !ok {
			t.Fatal("data properties is not an object")
		}

		expectedFields := []string{"id", "username", "email", "fullName", "age"}
		for _, field := range expectedFields {
			if _, ok := dataProps[field]; !ok {
				t.Errorf("User data missing expected field: %s", field)
			}
		}
	})

	t.Run("CreateUser Request Has Required Fields", func(t *testing.T) {
		postUsers := helpers.ExtractFromPath(t, spec, "paths./api/v1/users.post")
		if postUsers == nil {
			t.Fatal("POST /api/v1/users not found")
		}

		op, ok := postUsers.(map[string]any)
		if !ok {
			t.Fatal("operation is not an object")
		}

		reqBody, ok := op["requestBody"].(map[string]any)
		if !ok {
			t.Fatal("requestBody is not an object")
		}
		content, ok := reqBody["content"].(map[string]any)
		if !ok {
			t.Fatal("content is not an object")
		}
		jsonContent, ok := content["application/json"].(map[string]any)
		if !ok {
			t.Fatal("application/json content is not an object")
		}
		schema, ok := jsonContent["schema"].(map[string]any)
		if !ok {
			t.Fatal("schema is not an object")
		}

		// Check required fields
		required, ok := schema["required"].([]any)
		if !ok {
			t.Fatal("CreateUser request schema has no required fields")
		}

		requiredFields := make(map[string]bool)
		for _, r := range required {
			if s, ok := r.(string); ok {
				requiredFields[s] = true
			}
		}

		for _, field := range []string{"username", "email", "fullName", "age"} {
			if !requiredFields[field] {
				t.Errorf("CreateUser request should require field: %s", field)
			}
		}
	})

	t.Run("ListUsers Response Has Pagination Fields", func(t *testing.T) {
		listUsers := helpers.ExtractFromPath(t, spec, "paths./api/v1/users.get")
		if listUsers == nil {
			t.Fatal("GET /api/v1/users not found")
		}

		op, ok := listUsers.(map[string]any)
		if !ok {
			t.Fatal("operation is not an object")
		}

		responses, ok := op["responses"].(map[string]any)
		if !ok {
			t.Fatal("responses is not an object")
		}
		resp200, ok := responses["200"].(map[string]any)
		if !ok {
			t.Fatal("200 response is not an object")
		}
		content, ok := resp200["content"].(map[string]any)
		if !ok {
			t.Fatal("content is not an object")
		}
		jsonContent, ok := content["application/json"].(map[string]any)
		if !ok {
			t.Fatal("application/json content is not an object")
		}
		schema, ok := jsonContent["schema"].(map[string]any)
		if !ok {
			t.Fatal("schema is not an object")
		}
		properties, ok := schema["properties"].(map[string]any)
		if !ok {
			t.Fatal("properties is not an object")
		}
		data, ok := properties["data"].(map[string]any)
		if !ok {
			t.Fatal("data property is not an object")
		}
		dataProps, ok := data["properties"].(map[string]any)
		if !ok {
			t.Fatal("data properties is not an object")
		}

		paginationFields := []string{"content", "pageNumber", "pageSize", "total", "totalPages"}
		for _, field := range paginationFields {
			if _, ok := dataProps[field]; !ok {
				t.Errorf("ListUsers response data missing pagination field: %s", field)
			}
		}
	})
}

// TestGoSwaggerFormatCompatibility tests go-swagger format compatibility
func TestGoSwaggerFormatCompatibility(t *testing.T) {
	_, spec := generateSpec(t, "json")

	t.Run("Spec Has Valid OpenAPI Version", func(t *testing.T) {
		version, ok := spec["openapi"].(string)
		if !ok {
			t.Fatal("openapi version not found")
		}
		if !strings.HasPrefix(version, "3.") {
			t.Errorf("expected OpenAPI version 3.x, got %s", version)
		}
	})

	t.Run("Components Section Exists", func(t *testing.T) {
		// go-zero inlines schemas, so components may be empty but should exist
		if _, ok := spec["components"]; !ok {
			t.Error("components section should exist in spec")
		}
	})

	t.Run("JSON Response Content Type", func(t *testing.T) {
		paths, ok := spec["paths"].(map[string]any)
		if !ok {
			t.Fatal("paths not found")
		}

		for pathKey, pathValue := range paths {
			pathItem, ok := pathValue.(map[string]any)
			if !ok {
				continue
			}

			for method, opValue := range pathItem {
				op, ok := opValue.(map[string]any)
				if !ok {
					continue
				}

				responses, ok := op["responses"].(map[string]any)
				if !ok {
					continue
				}

				for code, respValue := range responses {
					resp, ok := respValue.(map[string]any)
					if !ok {
						continue
					}

					content, ok := resp["content"].(map[string]any)
					if !ok {
						continue
					}

					if _, hasJSON := content["application/json"]; !hasJSON {
						t.Errorf("path %s %s response %s should have application/json content type",
							pathKey, method, code)
					}
				}
			}
		}
	})
}

// TestRouteGeneration tests that routes are correctly generated from .api files
func TestRouteGeneration(t *testing.T) {
	specFile, _ := generateSpec(t, "json")

	validator := helpers.NewSpecValidator(t, specFile)

	t.Run("All API Routes Present", func(t *testing.T) {
		validator.ValidatePaths([]string{
			"/api/v1/users",
			"/api/v1/users/{id}",
			"/api/v1/users/upload",
			"/api/v1/users/{id}/profile",
		})
	})

	t.Run("HTTP Methods Correct", func(t *testing.T) {
		validator.ValidateOperation("/api/v1/users", "get")
		validator.ValidateOperation("/api/v1/users", "post")
		validator.ValidateOperation("/api/v1/users/{id}", "get")
		validator.ValidateOperation("/api/v1/users/upload", "post")
		validator.ValidateOperation("/api/v1/users/{id}/profile", "post")
	})

	t.Run("Path Count Matches Expected", func(t *testing.T) {
		pathCount := validator.GetPathCount()
		if pathCount != 4 {
			t.Errorf("expected 4 paths, got %d", pathCount)
		}
	})
}
