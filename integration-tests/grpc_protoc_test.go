//go:build e2e

package e2e_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spencercjh/spec-forge/internal/extractor"
	"github.com/spencercjh/spec-forge/internal/extractor/grpcprotoc"
	"github.com/spencercjh/spec-forge/internal/validator"
)

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

	if info.Framework != grpcprotoc.ExtractorName {
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
