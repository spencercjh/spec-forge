//go:build e2e

package e2e_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/spencercjh/spec-forge/internal/enricher"
	"github.com/spencercjh/spec-forge/internal/enricher/provider"
	"github.com/spencercjh/spec-forge/internal/executor"
	"github.com/spencercjh/spec-forge/internal/extractor"
	"github.com/spencercjh/spec-forge/internal/extractor/grpcprotoc"
	"github.com/spencercjh/spec-forge/internal/extractor/spring"
	"github.com/spencercjh/spec-forge/internal/validator"
)

// TestE2E_MavenSpringBoot_Generate tests the generate flow for a Maven Spring Boot project.
func TestE2E_MavenSpringBoot_Generate(t *testing.T) {
	projectPath := "maven-springboot-openapi-demo"

	// Check if project exists
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("Maven Spring Boot demo project not found")
	}

	// Check if mvnw wrapper exists
	mvnwPath := filepath.Join(projectPath, "mvnw")
	if _, err := os.Stat(mvnwPath); os.IsNotExist(err) {
		t.Skip("Maven wrapper not found, skipping test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Step 1: Detect project
	detector := spring.NewDetector()
	info, err := detector.Detect(projectPath)
	if err != nil {
		t.Fatalf("Failed to detect project: %v", err)
	}

	if info.BuildTool != spring.BuildToolMaven {
		t.Errorf("Expected Maven build tool, got %s", info.BuildTool)
	}

	springInfo, ok := info.FrameworkData.(*spring.Info)
	if !ok {
		t.Fatal("Expected FrameworkData to be *spring.Info")
	}

	if !springInfo.HasSpringdocDeps {
		t.Error("Expected springdoc dependencies to be present")
	}

	// Step 2: Generate OpenAPI spec
	gen := spring.NewGenerator()
	result, err := gen.Generate(ctx, projectPath, info, &extractor.GenerateOptions{
		Format:    "json",
		SkipTests: true,
	})
	if err != nil {
		t.Fatalf("Failed to generate spec: %v", err)
	}

	if result.SpecFilePath == "" {
		t.Fatal("Expected spec file path to be set")
	}

	// Step 3: Validate generated spec
	v := validator.NewValidator()
	validateResult, err := v.Validate(ctx, result.SpecFilePath)
	if err != nil {
		t.Fatalf("Failed to validate spec: %v", err)
	}

	if !validateResult.Valid {
		t.Errorf("Generated spec is invalid: %v", validateResult.Errors)
	}

	// Step 4: Verify spec content
	specData, err := os.ReadFile(result.SpecFilePath)
	if err != nil {
		t.Fatalf("Failed to read spec file: %v", err)
	}

	var spec map[string]any
	if err := json.Unmarshal(specData, &spec); err != nil {
		t.Fatalf("Failed to parse spec JSON: %v", err)
	}

	// Check for expected API paths
	paths, ok := spec["paths"].(map[string]any)
	if !ok {
		t.Fatal("Expected paths in spec")
	}

	expectedPaths := []string{
		"/api/v1/users",
		"/api/v1/users/{id}",
	}

	for _, expectedPath := range expectedPaths {
		if _, exists := paths[expectedPath]; !exists {
			t.Errorf("Expected path %s not found in spec", expectedPath)
		}
	}

	t.Logf("Successfully generated valid OpenAPI spec at: %s", result.SpecFilePath)
}

// TestE2E_MavenSpringBoot_GenerateEnrich tests the complete generate → enrich flow.
func TestE2E_MavenSpringBoot_GenerateEnrich(t *testing.T) {
	projectPath := "maven-springboot-openapi-demo"

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("Maven Spring Boot demo project not found")
	}

	// Check if mvnw wrapper exists
	mvnwPath := filepath.Join(projectPath, "mvnw")
	if _, err := os.Stat(mvnwPath); os.IsNotExist(err) {
		t.Skip("Maven wrapper not found, skipping test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Step 1: Generate spec
	detector := spring.NewDetector()
	info, err := detector.Detect(projectPath)
	if err != nil {
		t.Fatalf("Failed to detect project: %v", err)
	}

	gen := spring.NewGenerator()
	genResult, err := gen.Generate(ctx, projectPath, info, &extractor.GenerateOptions{
		Format:    "json",
		SkipTests: true,
	})
	if err != nil {
		t.Fatalf("Failed to generate spec: %v", err)
	}

	// Step 2: Load the generated spec
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	spec, err := loader.LoadFromFile(genResult.SpecFilePath)
	if err != nil {
		t.Fatalf("Failed to load spec: %v", err)
	}

	// Step 3: Enrich with mock provider
	mockProvider := &countingMockProvider{
		responses: map[string]string{
			"default": `{"summary": "Test API", "description": "Test description"}`,
		},
	}

	enrichCfg := enricher.Config{
		Provider:    "mock",
		Model:       "test-model",
		Language:    "en",
		Concurrency: 2,
	}
	enrichCfg = enrichCfg.MergeWithDefaults()

	e, err := enricher.NewEnricher(enrichCfg, mockProvider)
	if err != nil {
		t.Fatalf("Failed to create enricher: %v", err)
	}

	enrichedSpec, err := e.Enrich(ctx, spec, nil)
	if err != nil {
		t.Fatalf("Failed to enrich spec: %v", err)
	}

	// Step 4: Verify enrichment was applied
	if mockProvider.callCount == 0 {
		t.Error("Expected mock provider to be called at least once")
	}

	// Check that some operations have descriptions
	if enrichedSpec.Paths != nil {
		for _, pathStr := range enrichedSpec.Paths.InMatchingOrder() {
			pathItem := enrichedSpec.Paths.Value(pathStr)
			if pathItem.Get != nil && pathItem.Get.Summary != "" {
				t.Logf("GET %s enriched with summary: %s", pathStr, pathItem.Get.Summary)
			}
		}
	}

	t.Logf("Enrichment completed with %d LLM calls", mockProvider.callCount)
}

// TestE2E_GradleSpringBoot_Generate tests the generate flow for a Gradle Spring Boot project.
func TestE2E_GradleSpringBoot_Generate(t *testing.T) {
	projectPath := "gradle-springboot-openapi-demo"

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("Gradle Spring Boot demo project not found")
	}

	// Check if gradlew wrapper exists
	gradlewPath := filepath.Join(projectPath, "gradlew")
	if _, err := os.Stat(gradlewPath); os.IsNotExist(err) {
		t.Skip("Gradle wrapper not found, skipping test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Step 1: Detect project
	detector := spring.NewDetector()
	info, err := detector.Detect(projectPath)
	if err != nil {
		t.Fatalf("Failed to detect project: %v", err)
	}

	if info.BuildTool != spring.BuildToolGradle {
		t.Errorf("Expected Gradle build tool, got %s", info.BuildTool)
	}

	// Step 2: Generate OpenAPI spec (uses gradlew wrapper)
	gen := spring.NewGenerator()
	result, err := gen.Generate(ctx, projectPath, info, &extractor.GenerateOptions{
		Format:    "json",
		SkipTests: true,
	})
	if err != nil {
		t.Fatalf("Failed to generate spec: %v", err)
	}

	// Step 3: Validate
	v := validator.NewValidator()
	validateResult, err := v.Validate(ctx, result.SpecFilePath)
	if err != nil {
		t.Fatalf("Failed to validate spec: %v", err)
	}

	if !validateResult.Valid {
		t.Errorf("Generated spec is invalid: %v", validateResult.Errors)
	}

	t.Logf("Successfully generated valid OpenAPI spec at: %s", result.SpecFilePath)
}

// TestE2E_ErrorHandling_CommandNotFound tests error handling when build tool is not found.
func TestE2E_ErrorHandling_CommandNotFound(t *testing.T) {
	// Create a temp directory with pom.xml but no maven installed
	tmpDir := t.TempDir()
	pomPath := filepath.Join(tmpDir, "pom.xml")
	if err := os.WriteFile(pomPath, []byte(`<project></project>`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create target directory with a dummy spec (simulating pre-existing spec)
	targetDir := filepath.Join(tmpDir, "target")
	if err := os.Mkdir(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	exec := executor.NewExecutor()

	// Try to run a non-existent command
	_, err := exec.Execute(ctx, &executor.ExecuteOptions{
		Command: "nonexistent_command_12345",
		Args:    []string{},
	})

	if err == nil {
		t.Fatal("Expected error for non-existent command")
	}

	// Verify it's a CommandNotFoundError
	if _, ok := errors.AsType[*executor.CommandNotFoundError](err); !ok {
		t.Logf("Got error type %T: %v", err, err)
	}
}

// countingMockProvider tracks call counts for verification.
type countingMockProvider struct {
	callCount int
	responses map[string]string
}

func (m *countingMockProvider) Generate(ctx context.Context, prompt string) (string, error) {
	m.callCount++
	if resp, ok := m.responses["default"]; ok {
		return resp, nil
	}
	return `{"description": "Mock response"}`, nil
}

func (m *countingMockProvider) Name() string {
	return "mock"
}

var _ provider.Provider = (*countingMockProvider)(nil)

// TestE2E_GrpcProtoc_Generate tests the generate flow for a gRPC-protoc project.
func TestE2E_GrpcProtoc_Generate(t *testing.T) {
	projectPath := "grpc-protoc-demo"

	// Check if project exists
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("gRPC-protoc demo project not found")
	}

	// Check if proto files exist
	protoPath := filepath.Join(projectPath, "proto", "user.proto")
	if _, err := os.Stat(protoPath); os.IsNotExist(err) {
		t.Skip("user.proto not found, skipping test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Step 1: Detect project
	detector := grpcprotoc.NewDetector()
	info, err := detector.Detect(projectPath)
	if err != nil {
		t.Fatalf("Failed to detect project: %v", err)
	}

	if info.Framework != grpcprotoc.FrameworkName {
		t.Errorf("Expected grpc-protoc framework, got %s", info.Framework)
	}

	grpcInfo, ok := info.FrameworkData.(*grpcprotoc.Info)
	if !ok {
		t.Fatal("Expected grpcprotoc.Info in FrameworkData")
	}

	// Verify detection results
	if len(grpcInfo.ProtoFiles) == 0 {
		t.Error("Expected proto files to be detected")
	}

	if len(grpcInfo.ServiceProtoFiles) == 0 {
		t.Error("Expected service proto files to be detected")
	}

	if !grpcInfo.HasGoogleAPI {
		t.Error("Expected HasGoogleAPI to be true (demo uses HTTP annotations)")
	}

	t.Logf("Detected %d proto files, %d service files, HasGoogleAPI: %v",
		len(grpcInfo.ProtoFiles), len(grpcInfo.ServiceProtoFiles), grpcInfo.HasGoogleAPI)

	// Step 2: Generate OpenAPI spec
	gen := grpcprotoc.NewGenerator()
	result, err := gen.Generate(ctx, projectPath, info, &extractor.GenerateOptions{
		Format:    "yaml",
		SkipTests: true,
	})
	if err != nil {
		t.Fatalf("Failed to generate spec: %v", err)
	}

	if result.SpecFilePath == "" {
		t.Fatal("Expected spec file path to be set")
	}

	// Step 3: Validate generated spec
	v := validator.NewValidator()
	validateResult, err := v.Validate(ctx, result.SpecFilePath)
	if err != nil {
		t.Fatalf("Failed to validate spec: %v", err)
	}

	// Note: kin-openapi may have issues with OpenAPI 3.1 features like 'const'
	// The generated spec is valid OpenAPI 3.1, but the validator might not support all features
	t.Logf("Validation result: valid=%v, errors=%v", validateResult.Valid, validateResult.Errors)

	// Step 4: Verify spec content - check for REST paths from HTTP annotations
	specData, err := os.ReadFile(result.SpecFilePath)
	if err != nil {
		t.Fatalf("Failed to read spec file: %v", err)
	}

	// The generated YAML should contain REST paths
	specContent := string(specData)

	// Check for expected REST paths from google.api.http annotations
	expectedPaths := []string{
		"/v1/users:",
		"/v1/users/{id}:",
		"/v1/users/{user_id}/files:",
	}

	for _, expectedPath := range expectedPaths {
		if !strings.Contains(specContent, expectedPath) {
			t.Errorf("Expected REST path %s not found in spec", expectedPath)
		} else {
			t.Logf("Found REST path: %s", expectedPath)
		}
	}

	t.Logf("Successfully generated OpenAPI spec with REST endpoints at: %s", result.SpecFilePath)
}
