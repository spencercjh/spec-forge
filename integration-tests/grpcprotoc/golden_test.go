//go:build e2e

// Package grpcprotoc provides integration tests for gRPC-protoc framework support,
// including golden snapshot tests, semantic invariant tests, and edge case tests.
package grpcprotoc

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/spencercjh/spec-forge/cmd"
	"github.com/spencercjh/spec-forge/integration-tests/helpers"
)

// goldenSnapshots defines golden file snapshots for semantic stability testing
var goldenSnapshots = []helpers.GoldenSnapshot{
	{
		Name: "Full OpenAPI Spec",
		Path: "",
		File: "openapi.json",
	},
	{
		Name: "User Schema Structure",
		Path: "components.schemas.demo.user.User",
		File: "schemas/demo-user-User.json",
	},
	{
		Name: "CreateUserRequest Schema Structure",
		Path: "components.schemas.demo.user.CreateUserRequest",
		File: "schemas/demo-user-CreateUserRequest.json",
	},
	{
		Name: "ListUsersResponse Schema Structure",
		Path: "components.schemas.demo.user.ListUsersResponse",
		File: "schemas/demo-user-ListUsersResponse.json",
	},
	{
		Name: "GET /v1/users Operation",
		Path: "paths./v1/users.get",
		File: "paths/v1-users-get.json",
	},
	{
		Name: "POST /v1/users Operation",
		Path: "paths./v1/users.post",
		File: "paths/v1-users-post.json",
	},
	{
		Name: "GET /v1/users/{id} Operation",
		Path: "paths./v1/users/{id}.get",
		File: "paths/v1-users-id-get.json",
	},
	{
		Name: "PUT /v1/users/{id} Operation",
		Path: "paths./v1/users/{id}.put",
		File: "paths/v1-users-id-put.json",
	},
}

// skipIfProtocNotAvailable skips the test if protoc or protoc-gen-connect-openapi is not found.
func skipIfProtocNotAvailable(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found, skipping test")
	}
	if _, err := exec.LookPath("protoc-gen-connect-openapi"); err != nil {
		t.Skip("protoc-gen-connect-openapi not found, skipping test")
	}
}

// generateSpec generates an OpenAPI spec from the grpc-protoc-demo project.
// Returns the spec file path and parsed spec map.
func generateSpec(t *testing.T, format string) (string, map[string]any) {
	t.Helper()

	projectPath := "../grpc-protoc-demo"

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("grpc-protoc-demo project not found")
	}

	skipIfProtocNotAvailable(t)

	outputDir := t.TempDir()

	rootCmd := cmd.NewRootCommand()
	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{
		"generate",
		projectPath,
		"--output-dir", outputDir,
		"--output", format,
		"--skip-enrich",
		"--skip-publish",
		"--skip-validate",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v\nstderr: %s", err, stderr.String())
	}

	specFile := helpers.FindSpecFile(t, outputDir, format)
	specData, err := os.ReadFile(specFile)
	if err != nil {
		t.Fatalf("failed to read spec file: %v", err)
	}

	var spec map[string]any
	if err := json.Unmarshal(specData, &spec); err != nil {
		t.Fatalf("failed to parse spec JSON: %v", err)
	}

	return specFile, spec
}

// TestGoldenSnapshots tests against golden files
func TestGoldenSnapshots(t *testing.T) {
	_, spec := generateSpec(t, "json")

	goldenDir := "./fixtures/golden"

	for _, snapshot := range goldenSnapshots {
		t.Run(snapshot.Name, func(t *testing.T) {
			var actual any
			if snapshot.Path == "" {
				actual = spec
			} else {
				actual = helpers.ExtractFromPath(t, spec, snapshot.Path)
			}
			if actual == nil {
				t.Fatalf("failed to extract value at path: %s", snapshot.Path)
			}

			actualJSON, err := json.MarshalIndent(actual, "", "  ")
			if err != nil {
				t.Fatalf("failed to marshal actual value: %v", err)
			}

			goldenPath := filepath.Join(goldenDir, snapshot.File)
			goldenData, err := os.ReadFile(goldenPath)
			if err != nil {
				if snapshot.Optional {
					t.Skipf("Golden file not found (optional): %s", goldenPath)
				}
				t.Fatalf("failed to read golden file %s: %v", goldenPath, err)
			}

			var golden any
			if err := json.Unmarshal(goldenData, &golden); err != nil {
				t.Fatalf("failed to parse golden file: %v", err)
			}

			goldenJSON, err := json.MarshalIndent(golden, "", "  ")
			if err != nil {
				t.Fatalf("failed to marshal golden value: %v", err)
			}

			if string(actualJSON) != string(goldenJSON) {
				t.Errorf("Golden snapshot mismatch for %s\nPath: %s\n\nExpected (golden):\n%s\n\nActual:\n%s",
					snapshot.Name, snapshot.Path, string(goldenJSON), string(actualJSON))
			}
		})
	}
}

// TestCriticalInvariants tests critical semantic invariants for gRPC-protoc specs
func TestCriticalInvariants(t *testing.T) {
	specFile, spec := generateSpec(t, "json")

	validator := helpers.NewSpecValidator(t, specFile)

	t.Run("OpenAPI Version Must Be 3.x", func(t *testing.T) {
		validator.ValidateOpenAPIVersion()
	})

	t.Run("Info Must Have Title", func(t *testing.T) {
		info, ok := spec["info"].(map[string]any)
		if !ok {
			t.Fatal("info section not found")
		}
		if _, ok := info["title"].(string); !ok {
			t.Error("info must have a title")
		}
	})

	t.Run("All Expected Paths Must Exist", func(t *testing.T) {
		validator.ValidatePaths([]string{
			"/v1/users",
			"/v1/users/{id}",
			"/v1/users/{user_id}/files",
		})
	})

	t.Run("User Schema Must Have Expected Fields", func(t *testing.T) {
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

		expectedFields := []string{"id", "username", "email", "fullName", "age"}
		for _, field := range expectedFields {
			if _, ok := properties[field]; !ok {
				t.Errorf("User schema missing expected field: %s", field)
			}
		}
	})

	t.Run("CreateUserRequest Must Have Expected Fields", func(t *testing.T) {
		createReq := helpers.ExtractFromPath(t, spec, "components.schemas.demo.user.CreateUserRequest")
		if createReq == nil {
			t.Fatal("CreateUserRequest schema not found")
		}

		schema, ok := createReq.(map[string]any)
		if !ok {
			t.Fatal("CreateUserRequest schema is not an object")
		}

		properties, ok := schema["properties"].(map[string]any)
		if !ok {
			t.Fatal("CreateUserRequest schema has no properties")
		}

		expectedFields := []string{"username", "email", "fullName", "age"}
		for _, field := range expectedFields {
			if _, ok := properties[field]; !ok {
				t.Errorf("CreateUserRequest schema missing expected field: %s", field)
			}
		}
	})

	t.Run("ListUsersResponse Must Have Users Array", func(t *testing.T) {
		listResp := helpers.ExtractFromPath(t, spec, "components.schemas.demo.user.ListUsersResponse")
		if listResp == nil {
			t.Fatal("ListUsersResponse schema not found")
		}

		schema, ok := listResp.(map[string]any)
		if !ok {
			t.Fatal("ListUsersResponse schema is not an object")
		}

		properties, ok := schema["properties"].(map[string]any)
		if !ok {
			t.Fatal("ListUsersResponse schema has no properties")
		}

		users, ok := properties["users"].(map[string]any)
		if !ok {
			t.Fatal("ListUsersResponse.users property not found")
		}

		if usersType, ok := users["type"].(string); ok {
			if usersType != "array" {
				t.Errorf("ListUsersResponse.users should be array, got %s", usersType)
			}
		}
	})

	t.Run("Operations Must Have OperationIDs", func(t *testing.T) {
		operations := []struct {
			path   string
			method string
		}{
			{"paths./v1/users.get", "GET /v1/users"},
			{"paths./v1/users.post", "POST /v1/users"},
			{"paths./v1/users/{id}.get", "GET /v1/users/{id}"},
			{"paths./v1/users/{id}.put", "PUT /v1/users/{id}"},
		}

		for _, op := range operations {
			val := helpers.ExtractFromPath(t, spec, op.path)
			if val == nil {
				t.Errorf("%s not found", op.method)
				continue
			}
			m, ok := val.(map[string]any)
			if !ok {
				t.Errorf("%s is not an object", op.method)
				continue
			}
			if _, ok := m["operationId"]; !ok {
				t.Errorf("%s must have an operationId", op.method)
			}
		}
	})

	t.Run("POST Users Must Have Request Body", func(t *testing.T) {
		postUsers := helpers.ExtractFromPath(t, spec, "paths./v1/users.post")
		if postUsers == nil {
			t.Fatal("POST /v1/users not found")
		}

		op, ok := postUsers.(map[string]any)
		if !ok {
			t.Fatal("POST /v1/users is not an object")
		}

		if _, ok := op["requestBody"]; !ok {
			t.Error("POST /v1/users must have a requestBody")
		}
	})

	t.Run("Response Content Type Must Be JSON", func(t *testing.T) {
		getUser := helpers.ExtractFromPath(t, spec, "paths./v1/users/{id}.get")
		if getUser == nil {
			t.Fatal("GET /v1/users/{id} not found")
		}

		op, ok := getUser.(map[string]any)
		if !ok {
			t.Fatal("GET /v1/users/{id} is not an object")
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
			t.Fatal("no content found in 200 response")
		}

		if _, ok := content["application/json"]; !ok {
			t.Error("200 response must have application/json content type")
		}
	})
}

// TestRegenerateGolden regenerates golden files (run with REGENERATE_GOLDEN=true)
func TestRegenerateGolden(t *testing.T) {
	if os.Getenv("REGENERATE_GOLDEN") != "true" {
		t.Skip("Set REGENERATE_GOLDEN=true to regenerate golden files")
	}

	_, spec := generateSpec(t, "json")

	goldenDir := "./fixtures/golden"

	for _, snapshot := range goldenSnapshots {
		var actual any
		if snapshot.Path == "" {
			actual = spec
		} else {
			actual = helpers.ExtractFromPath(t, spec, snapshot.Path)
		}
		if actual == nil {
			t.Logf("Warning: could not extract %s", snapshot.Path)
			continue
		}

		goldenPath := filepath.Join(goldenDir, snapshot.File)

		dir := filepath.Dir(goldenPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}

		data, err := json.MarshalIndent(actual, "", "  ")
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		if err := os.WriteFile(goldenPath, data, 0o644); err != nil {
			t.Fatalf("failed to write golden file: %v", err)
		}

		t.Logf("Regenerated: %s", goldenPath)
	}
}
