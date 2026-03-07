//go:build e2e

package e2e_test

import (
	"context"
	"os"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/spencercjh/spec-forge/internal/extractor"
	"github.com/spencercjh/spec-forge/internal/extractor/gin"
)

func TestGinDemo(t *testing.T) {
	projectPath := "./gin-demo"

	// Detect project
	detector := gin.NewDetector()
	info, err := detector.Detect(projectPath)
	if err != nil {
		t.Fatalf("failed to detect project: %v", err)
	}

	if info.Framework != "gin" {
		t.Errorf("expected framework 'gin', got %s", info.Framework)
	}

	// Generate OpenAPI spec
	generator := gin.NewGenerator()
	ctx := context.Background()
	opts := &extractor.GenerateOptions{
		OutputDir:  t.TempDir(),
		OutputFile: "openapi",
		Format:     "yaml",
	}

	result, err := generator.Generate(ctx, projectPath, info, opts)
	if err != nil {
		t.Fatalf("failed to generate spec: %v", err)
	}

	// Verify output file exists
	if _, err := os.Stat(result.SpecFilePath); err != nil {
		t.Fatalf("output file not created: %s", result.SpecFilePath)
	}

	// Load and validate the spec
	loader := openapi3.NewLoader()
	spec, err := loader.LoadFromFile(result.SpecFilePath)
	if err != nil {
		t.Fatalf("failed to load spec: %v", err)
	}

	// Validate spec
	if err := spec.Validate(loader.Context); err != nil {
		t.Errorf("spec validation failed: %v", err)
	}

	// Check paths
	if spec.Paths.Len() == 0 {
		t.Error("expected at least one path")
	}

	// Check schemas
	if spec.Components == nil || spec.Components.Schemas == nil {
		t.Error("expected Components.Schemas to be defined")
	}

	// Check if User schema exists
	if spec.Components.Schemas["User"] == nil {
		t.Error("expected User schema to be defined")
	}

	// Check if CreateUserRequest schema exists
	if spec.Components.Schemas["CreateUserRequest"] == nil {
		t.Error("expected CreateUserRequest schema to be defined")
	}
}
