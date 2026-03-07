//go:build e2e

package e2e_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spencercjh/spec-forge/internal/extractor"
	"github.com/spencercjh/spec-forge/internal/extractor/spring"
	"github.com/spencercjh/spec-forge/internal/validator"
)

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
