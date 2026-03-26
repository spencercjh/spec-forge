//go:build e2e

package grpcprotoc

import (
	"strings"
	"testing"

	"github.com/spencercjh/spec-forge/integration-tests/helpers"
)

// TestGRPCServiceMapping tests that gRPC service methods are correctly mapped to REST endpoints
func TestGRPCServiceMapping(t *testing.T) {
	_, spec := generateSpec(t, "json")

	t.Run("All gRPC Methods Have REST Mappings", func(t *testing.T) {
		// The proto file defines 5 RPC methods with google.api.http annotations
		expectedOperations := []struct {
			path        string
			method      string
			operationID string
		}{
			{"/v1/users", "get", "demo.user.UserService.ListUsers"},
			{"/v1/users", "post", "demo.user.UserService.CreateUser"},
			{"/v1/users/{id}", "get", "demo.user.UserService.GetUser"},
			{"/v1/users/{id}", "put", "demo.user.UserService.UpdateProfile"},
			{"/v1/users/{user_id}/files", "post", "demo.user.UserService.UploadFile"},
		}

		paths, ok := spec["paths"].(map[string]any)
		if !ok {
			t.Fatal("paths not found in spec")
		}

		for _, expected := range expectedOperations {
			pathItem, ok := paths[expected.path].(map[string]any)
			if !ok {
				t.Errorf("path %s not found", expected.path)
				continue
			}

			op, ok := pathItem[expected.method].(map[string]any)
			if !ok {
				t.Errorf("%s %s not found", strings.ToUpper(expected.method), expected.path)
				continue
			}

			opID, ok := op["operationId"].(string)
			if !ok {
				t.Errorf("%s %s has no operationId", strings.ToUpper(expected.method), expected.path)
				continue
			}

			if opID != expected.operationID {
				t.Errorf("%s %s: expected operationId %q, got %q",
					strings.ToUpper(expected.method), expected.path, expected.operationID, opID)
			}
		}
	})

	t.Run("Service Tag Applied to All Operations", func(t *testing.T) {
		paths, ok := spec["paths"].(map[string]any)
		if !ok {
			t.Fatal("paths not found in spec")
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

				tags, ok := op["tags"].([]any)
				if !ok || len(tags) == 0 {
					t.Errorf("%s %s has no tags", strings.ToUpper(method), pathKey)
					continue
				}

				// All operations should be tagged with the service name
				foundServiceTag := false
				for _, tag := range tags {
					if tagStr, ok := tag.(string); ok && strings.Contains(tagStr, "UserService") {
						foundServiceTag = true
						break
					}
				}
				if !foundServiceTag {
					t.Errorf("%s %s should have a UserService tag", strings.ToUpper(method), pathKey)
				}
			}
		}
	})
}

// TestProtoFieldMapping tests that proto field names are correctly mapped to JSON field names
func TestProtoFieldMapping(t *testing.T) {
	_, spec := generateSpec(t, "json")

	t.Run("Snake Case Proto Fields Map to CamelCase JSON", func(t *testing.T) {
		userSchema := helpers.ExtractFromPath(t, spec, "components.schemas.demo.user.User")
		if userSchema == nil {
			t.Fatal("User schema not found")
		}

		schema, ok := userSchema.(map[string]any)
		if !ok {
			t.Fatal("User schema is not an object")
		}

		properties, ok := schema["properties"].(map[string]any)
		if !ok {
			t.Fatal("User schema has no properties")
		}

		// Proto field full_name should map to JSON fullName
		if _, ok := properties["fullName"]; !ok {
			t.Error("User.fullName (from proto full_name) not found in properties")
		}

		// Proto field full_name's title should reflect the original proto field name
		if fullName, ok := properties["fullName"].(map[string]any); ok {
			if title, ok := fullName["title"].(string); ok {
				if title != "full_name" {
					t.Errorf("fullName title should be 'full_name' (proto field name), got %q", title)
				}
			}
		}
	})

	t.Run("Proto Int64 Fields Have Correct Format", func(t *testing.T) {
		userSchema := helpers.ExtractFromPath(t, spec, "components.schemas.demo.user.User")
		if userSchema == nil {
			t.Fatal("User schema not found")
		}

		schema, ok := userSchema.(map[string]any)
		if !ok {
			t.Fatal("User schema is not an object")
		}

		properties, ok := schema["properties"].(map[string]any)
		if !ok {
			t.Fatal("User schema has no properties")
		}

		idProp, ok := properties["id"].(map[string]any)
		if !ok {
			t.Fatal("User.id property not found")
		}

		format, ok := idProp["format"].(string)
		if !ok {
			t.Fatal("User.id has no format")
		}
		if format != "int64" {
			t.Errorf("User.id format should be 'int64', got %q", format)
		}
	})

	t.Run("Proto Int32 Fields Have Correct Format", func(t *testing.T) {
		userSchema := helpers.ExtractFromPath(t, spec, "components.schemas.demo.user.User")
		if userSchema == nil {
			t.Fatal("User schema not found")
		}

		schema, ok := userSchema.(map[string]any)
		if !ok {
			t.Fatal("User schema is not an object")
		}

		properties, ok := schema["properties"].(map[string]any)
		if !ok {
			t.Fatal("User schema has no properties")
		}

		ageProp, ok := properties["age"].(map[string]any)
		if !ok {
			t.Fatal("User.age property not found")
		}

		format, ok := ageProp["format"].(string)
		if !ok {
			t.Fatal("User.age has no format")
		}
		if format != "int32" {
			t.Errorf("User.age format should be 'int32', got %q", format)
		}
	})
}

// TestProtoMessageReferences tests that proto message references ($ref) are correctly generated
func TestProtoMessageReferences(t *testing.T) {
	_, spec := generateSpec(t, "json")

	t.Run("Response References Correct Schema", func(t *testing.T) {
		// POST /v1/users response should reference CreateUserResponse
		postUsers := helpers.ExtractFromPath(t, spec, "paths./v1/users.post")
		if postUsers == nil {
			t.Fatal("POST /v1/users not found")
		}

		op, ok := postUsers.(map[string]any)
		if !ok {
			t.Fatal("POST /v1/users is not an object")
		}

		responses, ok := op["responses"].(map[string]any)
		if !ok {
			t.Fatal("no responses found")
		}

		resp200, ok := responses["200"].(map[string]any)
		if !ok {
			t.Fatal("no 200 response found")
		}

		content, ok := resp200["content"].(map[string]any)
		if !ok {
			t.Fatal("no content in 200 response")
		}

		jsonContent, ok := content["application/json"].(map[string]any)
		if !ok {
			t.Fatal("no application/json content")
		}

		schema, ok := jsonContent["schema"].(map[string]any)
		if !ok {
			t.Fatal("no schema in application/json content")
		}

		ref, ok := schema["$ref"].(string)
		if !ok {
			t.Fatal("schema has no $ref")
		}

		if !strings.Contains(ref, "CreateUserResponse") {
			t.Errorf("POST /v1/users 200 response should reference CreateUserResponse, got %q", ref)
		}
	})

	t.Run("Nested Message References Exist", func(t *testing.T) {
		// ListUsersResponse should reference User in its users field
		listResp := helpers.ExtractFromPath(t, spec, "components.schemas.demo.user.ListUsersResponse")
		if listResp == nil {
			t.Fatal("ListUsersResponse schema not found")
		}

		schema, ok := listResp.(map[string]any)
		if !ok {
			t.Fatal("ListUsersResponse is not an object")
		}

		properties, ok := schema["properties"].(map[string]any)
		if !ok {
			t.Fatal("ListUsersResponse has no properties")
		}

		users, ok := properties["users"].(map[string]any)
		if !ok {
			t.Fatal("ListUsersResponse.users property not found")
		}

		items, ok := users["items"].(map[string]any)
		if !ok {
			t.Fatal("ListUsersResponse.users.items not found")
		}

		ref, ok := items["$ref"].(string)
		if !ok {
			t.Fatal("ListUsersResponse.users.items has no $ref")
		}

		if !strings.Contains(ref, "demo.user.User") {
			t.Errorf("ListUsersResponse.users.items should reference User, got %q", ref)
		}
	})

	t.Run("Cross-Package References Work", func(t *testing.T) {
		// CreateUserResponse should reference ApiResponse from common package
		createResp := helpers.ExtractFromPath(t, spec, "components.schemas.demo.user.CreateUserResponse")
		if createResp == nil {
			t.Fatal("CreateUserResponse schema not found")
		}

		schema, ok := createResp.(map[string]any)
		if !ok {
			t.Fatal("CreateUserResponse is not an object")
		}

		properties, ok := schema["properties"].(map[string]any)
		if !ok {
			t.Fatal("CreateUserResponse has no properties")
		}

		status, ok := properties["status"].(map[string]any)
		if !ok {
			t.Fatal("CreateUserResponse.status property not found")
		}

		ref, ok := status["$ref"].(string)
		if !ok {
			t.Fatal("CreateUserResponse.status has no $ref")
		}

		if !strings.Contains(ref, "demo.common.ApiResponse") {
			t.Errorf("CreateUserResponse.status should reference ApiResponse, got %q", ref)
		}
	})
}

// TestConnectProtocolSupport tests Connect protocol-specific features
func TestConnectProtocolSupport(t *testing.T) {
	_, spec := generateSpec(t, "json")

	t.Run("Error Schema Follows Connect Protocol", func(t *testing.T) {
		// Check that connect.error schema exists with proper gRPC status codes
		components, ok := spec["components"].(map[string]any)
		if !ok {
			t.Fatal("components not found")
		}

		schemas, ok := components["schemas"].(map[string]any)
		if !ok {
			t.Fatal("schemas not found in components")
		}

		// There should be a connect error schema
		var connectErrorSchema map[string]any
		for name, schema := range schemas {
			if strings.Contains(name, "connect") && strings.Contains(name, "error") {
				if s, ok := schema.(map[string]any); ok {
					connectErrorSchema = s
					break
				}
			}
		}

		if connectErrorSchema == nil {
			t.Skip("connect error schema not found (may vary by plugin version)")
		}

		properties, ok := connectErrorSchema["properties"].(map[string]any)
		if !ok {
			t.Fatal("connect error schema has no properties")
		}

		// Should have code and message properties
		if _, ok := properties["code"]; !ok {
			t.Error("connect error schema should have 'code' property")
		}
		if _, ok := properties["message"]; !ok {
			t.Error("connect error schema should have 'message' property")
		}
	})

	t.Run("Success Responses Present", func(t *testing.T) {
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
					t.Errorf("%s %s has no responses", strings.ToUpper(method), pathKey)
					continue
				}

				if _, ok := responses["200"]; !ok {
					t.Errorf("%s %s should have a 200 success response",
						strings.ToUpper(method), pathKey)
				}
			}
		}
	})
}
