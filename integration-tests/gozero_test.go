//go:build e2e

package e2e_test

import (
	"context"
	"os"
	"path/filepath"
	 "testing"
   "time"

  "github.com/spencercjh/spec-forge/internal/extractor"
  "github.com/spencercjh/spec-forge/internal/extractor/gozero"
  "github.com/spencercjh/spec-forge/internal/validator"
)


// TestE2E_GoZero_Generate tests the generate flow for a go-zero project.
func TestE2E_GoZero_Generate(t *testing.T) {
    projectPath := "gozero-demo"

    // Check if project exists
    if _, err := os.Stat(projectPath); os.IsNotExist(err) {
        t.Skip("go-zero demo project not found")
    }

    // Check if go.mod exists
    goModPath := filepath.Join(projectPath, "go.mod")
    if _, err := os.Stat(goModPath); os.IsNotExist(err) {
        t.Skip("go.mod not found, skipping test")
    }

    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()

    // Step 1: Detect project
    detector := gozero.NewDetector()
    info, err := detector.Detect(projectPath)
    if err != nil {
        t.Fatalf("Failed to detect project: %v", err)
    }

    // Verify it's detected as go-zero
    if info.Framework != gozero.FrameworkGoZero {
        t.Errorf("Expected framework %s, got %s", gozero.FrameworkGoZero, info.Framework)
    }

    // Verify go-zero specific info
    gozeroInfo, ok := info.FrameworkData.(*gozero.Info)
    if !ok {
        t.Fatal("Expected FrameworkData to be *gozero.Info")
    }

    if !gozeroInfo.HasGoZeroDeps {
        t.Error("Expected go-zero dependencies to be present")
    }

    if len(gozeroInfo.APIFiles) == 0 {
        t.Error("Expected at least one .api file to be found")
    }

    t.Logf("Detected go-zero project with %d API files: %v", len(gozeroInfo.APIFiles), gozeroInfo.APIFiles)

    // Step 2: Patch (verify goctl availability)
    patcher := gozero.NewPatcher()
    patchResult, err := patcher.Patch(projectPath)
    if err != nil {
        t.Fatalf("Failed to patch project: %v", err)
    }

    if !patchResult.GoctlAvailable {
        t.Skip("goctl not found in PATH, skipping generation test")
    }

    t.Logf("goctl is available: %s", patchResult.GoctlVersion)

    // Step 3: Generate OpenAPI spec
    gen := gozero.NewGenerator()
    result, err := gen.Generate(ctx, projectPath, info, &extractor.GenerateOptions{
        Format: "yaml",
    })
    if err != nil {
        t.Fatalf("Failed to generate spec: %v", err)
    }

    if result.SpecFilePath == "" {
        t.Fatal("Expected spec file path to be set")
    }

    // Step 4: Validate generated spec
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

// TestE2E_GoZero_Detect tests the detection of go-zero project characteristics.
func TestE2E_GoZero_Detect(t *testing.T) {
    projectPath := "gozero-demo"

    if _, err := os.Stat(projectPath); os.IsNotExist(err) {
        t.Skip("go-zero demo project not found")
    }

    detector := gozero.NewDetector()
    info, err := detector.Detect(projectPath)
    if err != nil {
        t.Fatalf("Failed to detect project: %v", err)
    }

    gozeroInfo, ok := info.FrameworkData.(*gozero.Info)
    if !ok {
        t.Fatal("Expected FrameworkData to be *gozero.Info")
    }

    if !gozeroInfo.HasGoZeroDeps {
        t.Error("Expected go-zero dependencies to be present")
    }

    if len(gozeroInfo.APIFiles) == 0 {
        t.Error("Expected at least one .api file to be found")
    }

    // Verify go-zero specific detection
    t.Logf("Go Version: %s", gozeroInfo.GoVersion)
    t.Logf("Module Name: %s", gozeroInfo.ModuleName)
    t.Logf("Has go-zero deps: %v", gozeroInfo.HasGoZeroDeps)
    t.Logf("go-zero version: %s", gozeroInfo.GoZeroVersion)
    t.Logf("API files: %v", gozeroInfo.APIFiles)
    t.Logf("Has goctl: %v", gozeroInfo.HasGoctl)
}

// TestE2E_GoZero_NoGoctl tests graceful handling when goctl is not available.
func TestE2E_GoZero_NoGoctl(t *testing.T) {
    projectPath := "gozero-demo"

    if _, err := os.Stat(projectPath); os.IsNotExist(err) {
        t.Skip("go-zero demo project not found")
    }

    // Check if goctl is available
    patcher := gozero.NewPatcher()
    patchResult, err := patcher.Patch(projectPath)
    if err != nil {
        t.Fatalf("Failed to check goctl: %v", err)
    }

    if patchResult.GoctlAvailable {
        t.Skip("goctl is available, skip this test")
    }

    // When goctl is not available, generation should fail with a clear error
    ctx := context.Background()
    detector := gozero.NewDetector()
    info, err := detector.Detect(projectPath)
    if err != nil {
        t.Fatalf("Failed to detect project: %v", err)
    }

    gen := gozero.NewGenerator()
    _, err = gen.Generate(ctx, projectPath, info, &extractor.GenerateOptions{
        Format: "yaml",
    })

    if err == nil {
        t.Error("Expected error when goctl is not available")
    }

    t.Logf("Got expected error when goctl not available: %v", err)
    }
