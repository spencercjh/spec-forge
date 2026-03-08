//go:build e2e

package e2e_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/spencercjh/spec-forge/internal/extractor"
	"github.com/spencercjh/spec-forge/internal/extractor/gin"
	"github.com/spencercjh/spec-forge/internal/validator"
)

func TestGinDemo(t *testing.T) {
	projectPath := "./gin-demo"

	// Check if project exists
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("Gin demo project not found")
	}

	// Check if go.mod exists
	goModPath := filepath.Join(projectPath, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		t.Skip("go.mod not found, skipping test")
	}

	ctx := context.Background()

	// Step 1: Detect project
	detector := gin.NewDetector()
	info, err := detector.Detect(projectPath)
	if err != nil {
		t.Fatalf("failed to detect project: %v", err)
	}

	if info.Framework != "gin" {
		t.Errorf("expected framework 'gin', got %s", info.Framework)
	}

	if info.BuildTool != "gomodules" {
		t.Errorf("expected build tool 'gomodules', got %s", info.BuildTool)
	}

	// Check FrameworkData
	ginInfo, ok := info.FrameworkData.(*gin.Info)
	if !ok {
		t.Fatal("expected FrameworkData to be *gin.Info")
	}

	if !ginInfo.HasGin {
		t.Error("expected HasGin to be true")
	}

	if ginInfo.GinVersion == "" {
		t.Error("expected GinVersion to be set")
	}

	t.Logf("Detected Gin project: module=%s, version=%s", ginInfo.ModuleName, ginInfo.GinVersion)

	// Step 2: Generate OpenAPI spec (YAML)
	gen := gin.NewGenerator()
	result, err := gen.Generate(ctx, projectPath, info, &extractor.GenerateOptions{
		OutputDir:  t.TempDir(),
		OutputFile: "openapi",
		Format:     "yaml",
	})
	if err != nil {
		t.Fatalf("failed to generate spec: %v", err)
	}

	if result.SpecFilePath == "" {
		t.Fatal("expected spec file path to be set")
	}

	// Verify output file exists
	if _, err := os.Stat(result.SpecFilePath); err != nil {
		t.Fatalf("output file not created: %s", result.SpecFilePath)
	}

	t.Logf("Generated spec: %s", result.SpecFilePath)

	// Step 3: Validate generated spec
	v := validator.NewValidator()
	validateResult, err := v.Validate(ctx, result.SpecFilePath)
	if err != nil {
		t.Fatalf("failed to validate spec: %v", err)
	}

	if !validateResult.Valid {
		t.Errorf("generated spec is invalid: %v", validateResult.Errors)
	}

	// Step 4: Load and verify spec content
	loader := openapi3.NewLoader()
	spec, err := loader.LoadFromFile(result.SpecFilePath)
	if err != nil {
		t.Fatalf("failed to load spec: %v", err)
	}

	// Step 5: Verify paths
	if spec.Paths.Len() == 0 {
		t.Fatal("expected at least one path")
	}

	expectedPaths := map[string][]string{
		"/api/v1/users":              {"GET", "POST"},
		"/api/v1/users/{id}":         {"GET"},
		"/api/v1/users/{id}/profile": {"POST"},
		"/api/v1/users/upload":       {"POST"},
	}

	for expectedPath, methods := range expectedPaths {
		pathItem := spec.Paths.Find(expectedPath)
		if pathItem == nil {
			t.Errorf("expected path %s not found", expectedPath)
			continue
		}

		for _, method := range methods {
			var operation *openapi3.Operation
			switch method {
			case "GET":
				operation = pathItem.Get
			case "POST":
				operation = pathItem.Post
			case "PUT":
				operation = pathItem.Put
			case "DELETE":
				operation = pathItem.Delete
			case "PATCH":
				operation = pathItem.Patch
			}

			if operation == nil {
				t.Errorf("expected %s operation for path %s", method, expectedPath)
			}
		}
	}

	// Step 6: Verify schemas
	if spec.Components == nil || spec.Components.Schemas == nil {
		t.Fatal("expected Components.Schemas to be defined")
	}

	// The extractor now supports extracting types wrapped in ApiResponse.Data
	// by tracking variable assignments and resolving composite literal fields
	// e.g., c.JSON(200, ApiResponse{Data: user}) extracts User as the response type
	expectedSchemas := []string{
		"CreateUserRequest",
		"ListUsersRequest",
		"UpdateProfileRequest",
		"ApiResponse",
		"User",
		"PageResult",
		"FileUploadResult",
	}

	for _, schemaName := range expectedSchemas {
		if spec.Components.Schemas[schemaName] == nil {
			t.Errorf("expected schema %s to be defined", schemaName)
		}
	}

	// Log available schemas for debugging
	t.Logf("Available schemas: %v", getSchemaNames(spec.Components.Schemas))

	// Step 7: Verify specific schema properties
	apiResponseSchema := spec.Components.Schemas["ApiResponse"]
	if apiResponseSchema != nil {
		apiProps := apiResponseSchema.Value.Properties
		expectedProps := []string{"code", "message", "data"}
		for _, prop := range expectedProps {
			if _, exists := apiProps[prop]; !exists {
				t.Errorf("expected ApiResponse schema to have property %s", prop)
			}
		}
	}

	createUserSchema := spec.Components.Schemas["CreateUserRequest"]
	if createUserSchema != nil {
		props := createUserSchema.Value.Properties
		expectedProps := []string{"username", "email", "fullName", "age"}
		for _, prop := range expectedProps {
			if _, exists := props[prop]; !exists {
				t.Errorf("expected CreateUserRequest schema to have property %s", prop)
			}
		}
	}

	// Step 8: Verify request body types
	createUserPath := spec.Paths.Find("/api/v1/users")
	if createUserPath != nil && createUserPath.Post != nil {
		if createUserPath.Post.RequestBody != nil &&
			createUserPath.Post.RequestBody.Value != nil {
			content := createUserPath.Post.RequestBody.Value.Content
			if jsonContent, exists := content["application/json"]; exists {
				if jsonContent.Schema != nil &&
					jsonContent.Schema.Ref != "#/components/schemas/CreateUserRequest" {
					t.Errorf("expected CreateUser request body to reference CreateUserRequest, got %s",
						jsonContent.Schema.Ref)
				}
			}
		}
	}

	// Step 9: Verify path parameters
	getUserPath := spec.Paths.Find("/api/v1/users/{id}")
	if getUserPath != nil && getUserPath.Get != nil {
		params := getUserPath.Get.Parameters
		foundPathParam := false
		for _, param := range params {
			if param.Value.In == "path" && param.Value.Name == "id" {
				foundPathParam = true
				if !param.Value.Required {
					t.Error("expected path parameter 'id' to be required")
				}
			}
		}
		if !foundPathParam {
			t.Error("expected path parameter 'id' for GET /api/v1/users/{id}")
		}
	}

	t.Log("All validations passed!")
}

// getSchemaNames returns a slice of schema names for debugging
func getSchemaNames(schemas openapi3.Schemas) []string {
	names := make([]string, 0, len(schemas))
	for name := range schemas {
		names = append(names, name)
	}
	return names
}

func TestGinDemo_JSONFormat(t *testing.T) {
	projectPath := "./gin-demo"

	// Check if project exists
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("Gin demo project not found")
	}

	ctx := context.Background()

	// Detect project
	detector := gin.NewDetector()
	info, err := detector.Detect(projectPath)
	if err != nil {
		t.Fatalf("failed to detect project: %v", err)
	}

	// Generate OpenAPI spec (JSON)
	gen := gin.NewGenerator()
	result, err := gen.Generate(ctx, projectPath, info, &extractor.GenerateOptions{
		OutputDir:  t.TempDir(),
		OutputFile: "openapi",
		Format:     "json",
	})
	if err != nil {
		t.Fatalf("failed to generate spec: %v", err)
	}

	// Verify JSON content
	specData, err := os.ReadFile(result.SpecFilePath)
	if err != nil {
		t.Fatalf("failed to read spec file: %v", err)
	}

	var spec map[string]any
	if err := json.Unmarshal(specData, &spec); err != nil {
		t.Fatalf("failed to parse spec JSON: %v", err)
	}

	// Verify basic structure
	if spec["openapi"] == nil {
		t.Error("expected openapi field in spec")
	}

	if spec["paths"] == nil {
		t.Error("expected paths field in spec")
	}

	paths, ok := spec["paths"].(map[string]any)
	if !ok {
		t.Fatal("expected paths to be an object")
	}

	// Verify expected paths exist
	expectedPaths := []string{
		"/api/v1/users",
		"/api/v1/users/{id}",
		"/api/v1/users/{id}/profile",
		"/api/v1/users/upload",
	}

	for _, path := range expectedPaths {
		if _, exists := paths[path]; !exists {
			t.Errorf("expected path %s not found in JSON spec", path)
		}
	}

	// Verify components/schemas exist
	components, ok := spec["components"].(map[string]any)
	if !ok {
		t.Fatal("expected components field in spec")
	}

	schemas, ok := components["schemas"].(map[string]any)
	if !ok {
		t.Fatal("expected components.schemas to be an object")
	}

	expectedSchemas := []string{
		"CreateUserRequest",
		"ListUsersRequest",
		"UpdateProfileRequest",
		"ApiResponse",
	}

	for _, schema := range expectedSchemas {
		if _, exists := schemas[schema]; !exists {
			t.Errorf("expected schema %s not found in JSON spec", schema)
		}
	}

	t.Log("JSON format test passed!")
}

// TestGinDemo_DefaultOutput tests the generator's behavior when OutputDir
// is set to the project path. Note: This test bypasses the CLI layer and
// tests the extractor directly. The CLI layer (cmd/generate.go) handles
// config precedence and passes the resolved outputDir to the generator.
func TestGinDemo_DefaultOutput(t *testing.T) {
	projectPath := "./gin-demo"

	// Check if project exists
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("Gin demo project not found")
	}

	ctx := context.Background()

	// Detect project
	detector := gin.NewDetector()
	info, err := detector.Detect(projectPath)
	if err != nil {
		t.Fatalf("failed to detect project: %v", err)
	}

	// Generate with OutputDir set to project path
	// The CLI layer (cmd/generate.go) resolves outputDir via: flag > config > projectPath
	gen := gin.NewGenerator()
	result, err := gen.Generate(ctx, projectPath, info, &extractor.GenerateOptions{
		OutputDir: projectPath,
		Format:    "yaml",
	})
	if err != nil {
		t.Fatalf("failed to generate spec: %v", err)
	}

	// Verify output is in project root
	expectedPath := filepath.Join(projectPath, "openapi.yaml")
	absResultPath, _ := filepath.Abs(result.SpecFilePath)
	absExpectedPath, _ := filepath.Abs(expectedPath)

	if absResultPath != absExpectedPath {
		t.Errorf("expected spec at %s, got %s", absExpectedPath, absResultPath)
	}

	// Verify file exists
	if _, err := os.Stat(result.SpecFilePath); err != nil {
		t.Fatalf("spec file not found: %s", result.SpecFilePath)
	}

	// Cleanup
	_ = os.Remove(result.SpecFilePath)

	t.Logf("Spec correctly output to project root: %s", result.SpecFilePath)
}

func TestGinDemo_AutoDetection(t *testing.T) {
	projectPath := "./gin-demo"

	// Check if project exists
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("Gin demo project not found")
	}

	// Test using Extractor interface (like CLI does)
	ext := gin.NewExtractor()

	if ext.Name() != "gin" {
		t.Errorf("expected extractor name 'gin', got %s", ext.Name())
	}

	// Detect
	info, err := ext.Detect(projectPath)
	if err != nil {
		t.Fatalf("failed to detect: %v", err)
	}

	if info.Framework != "gin" {
		t.Errorf("expected framework 'gin', got %s", info.Framework)
	}

	// Patch (no-op for Gin)
	patchOpts := &extractor.PatchOptions{}
	patchResult, err := ext.Patch(projectPath, patchOpts)
	if err != nil {
		t.Fatalf("failed to patch: %v", err)
	}

	if patchResult == nil {
		t.Error("expected patch result")
	}

	// Generate
	ctx := context.Background()
	genOpts := &extractor.GenerateOptions{
		OutputDir:  t.TempDir(),
		OutputFile: "openapi",
		Format:     "yaml",
	}
	result, err := ext.Generate(ctx, projectPath, info, genOpts)
	if err != nil {
		t.Fatalf("failed to generate: %v", err)
	}

	if result.SpecFilePath == "" {
		t.Error("expected spec file path")
	}

	// Verify file exists
	if _, err := os.Stat(result.SpecFilePath); err != nil {
		t.Errorf("spec file not found: %s", result.SpecFilePath)
	}

	t.Log("Auto-detection test passed!")
}
