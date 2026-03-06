// Package gozero_test tests the go-zero extractor implementation.
package gozero_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spencercjh/spec-forge/internal/extractor/gozero"
)

func TestNewDetector(t *testing.T) {
	d := gozero.NewDetector()
	if d == nil {
		t.Error("NewDetector() should not return nil")
	}
}

func TestDetector_Detect_NoGoMod(t *testing.T) {
	// Create temp dir without go.mod
	tmpDir := t.TempDir()

	detector := gozero.NewDetector()
	_, err := detector.Detect(tmpDir)

	if err == nil {
		t.Error("Expected error when no go.mod found")
	}

	if err != nil && err.Error() == "" {
		t.Error("Expected meaningful error message")
	}
}

func TestDetector_Detect_ValidGoMod(t *testing.T) {
	// Create temp dir with go.mod
	tmpDir := t.TempDir()

	goModContent := `module example.com/testproject

go 1.21

require (
	github.com/zeromicro/go-zero v1.6.0
)
`
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goModContent), 0o644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	detector := gozero.NewDetector()
	info, err := detector.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if info.BuildTool != gozero.BuildToolGoModules {
		t.Errorf("BuildTool = %s, want %s", info.BuildTool, gozero.BuildToolGoModules)
	}

	if info.BuildFilePath != goModPath {
		t.Errorf("BuildFilePath = %s, want %s", info.BuildFilePath, goModPath)
	}

	if info.GoZero == nil {
		t.Fatal("GoZero should not be nil")
	}

	if info.GoZero.ModuleName != "example.com/testproject" {
		t.Errorf("ModuleName = %s, want example.com/testproject", info.GoZero.ModuleName)
	}

	if info.GoZero.GoVersion != "1.21" {
		t.Errorf("GoVersion = %s, want 1.21", info.GoZero.GoVersion)
	}

	if !info.GoZero.HasGoZeroDeps {
		t.Error("HasGoZeroDeps should be true")
	}

	if info.GoZero.GoZeroVersion != "v1.6.0" {
		t.Errorf("GoZeroVersion = %s, want v1.6.0", info.GoZero.GoZeroVersion)
	}
}

func TestDetector_Detect_SingleLineRequire(t *testing.T) {
	// Create temp dir with go.mod using single-line require
	tmpDir := t.TempDir()

	goModContent := `module example.com/singleline

go 1.20

require github.com/zeromicro/go-zero v1.5.0
`
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goModContent), 0o644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	detector := gozero.NewDetector()
	info, err := detector.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if info.GoZero == nil {
		t.Fatal("GoZero should not be nil")
	}

	if !info.GoZero.HasGoZeroDeps {
		t.Error("HasGoZeroDeps should be true for single-line require")
	}

	if info.GoZero.GoZeroVersion != "v1.5.0" {
		t.Errorf("GoZeroVersion = %s, want v1.5.0", info.GoZero.GoZeroVersion)
	}
}

func TestDetector_Detect_NoGoZeroDeps(t *testing.T) {
	// Create temp dir with go.mod without go-zero
	tmpDir := t.TempDir()

	goModContent := `module example.com/nogozero

go 1.21

require (
	github.com/some/other v1.0.0
)
`
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goModContent), 0o644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	detector := gozero.NewDetector()
	info, err := detector.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if info.GoZero == nil {
		t.Fatal("GoZero should not be nil")
	}

	if info.GoZero.HasGoZeroDeps {
		t.Error("HasGoZeroDeps should be false")
	}

	if info.GoZero.GoZeroVersion != "" {
		t.Errorf("GoZeroVersion should be empty, got %s", info.GoZero.GoZeroVersion)
	}
}

func TestDetector_Detect_WithAPIFiles(t *testing.T) {
	// Create temp dir with go.mod and .api files
	tmpDir := t.TempDir()

	goModContent := `module example.com/apiproject

go 1.21

require github.com/zeromicro/go-zero v1.6.0
`
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goModContent), 0o644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create .api files in different directories
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("Failed to create api dir: %v", err)
	}

	apiFile1 := filepath.Join(apiDir, "user.api")
	if err := os.WriteFile(apiFile1, []byte("syntax = \"v1\""), 0o644); err != nil {
		t.Fatalf("Failed to create .api file: %v", err)
	}

	apiFile2 := filepath.Join(tmpDir, "order.api")
	if err := os.WriteFile(apiFile2, []byte("syntax = \"v1\""), 0o644); err != nil {
		t.Fatalf("Failed to create .api file: %v", err)
	}

	detector := gozero.NewDetector()
	info, err := detector.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if info.GoZero == nil {
		t.Fatal("GoZero should not be nil")
	}

	if len(info.GoZero.APIFiles) != 2 {
		t.Errorf("Expected 2 .api files, got %d", len(info.GoZero.APIFiles))
	}
}

func TestDetector_Detect_SkipVendor(t *testing.T) {
	// Create temp dir with go.mod and vendor directory
	tmpDir := t.TempDir()

	goModContent := `module example.com/vendorproject

go 1.21

require github.com/zeromicro/go-zero v1.6.0
`
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goModContent), 0o644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create .api file in main directory
	apiFile := filepath.Join(tmpDir, "main.api")
	if err := os.WriteFile(apiFile, []byte("syntax = \"v1\""), 0o644); err != nil {
		t.Fatalf("Failed to create .api file: %v", err)
	}

	// Create vendor directory with .api file (should be skipped)
	vendorDir := filepath.Join(tmpDir, "vendor", "github.com", "example")
	if err := os.MkdirAll(vendorDir, 0o755); err != nil {
		t.Fatalf("Failed to create vendor dir: %v", err)
	}

	vendorAPIFile := filepath.Join(vendorDir, "vendor.api")
	if err := os.WriteFile(vendorAPIFile, []byte("syntax = \"v1\""), 0o644); err != nil {
		t.Fatalf("Failed to create vendor .api file: %v", err)
	}

	detector := gozero.NewDetector()
	info, err := detector.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if info.GoZero == nil {
		t.Fatal("GoZero should not be nil")
	}

	// Should only find the main.api file, not the vendor one
	if len(info.GoZero.APIFiles) != 1 {
		t.Errorf("Expected 1 .api file (vendor skipped), got %d", len(info.GoZero.APIFiles))
	}

	if len(info.GoZero.APIFiles) > 0 && info.GoZero.APIFiles[0] != apiFile {
		t.Errorf("Expected %s, got %s", apiFile, info.GoZero.APIFiles[0])
	}
}

func TestDetector_Detect_SkipHiddenDirs(t *testing.T) {
	// Create temp dir with go.mod and hidden directory
	tmpDir := t.TempDir()

	goModContent := `module example.com/hiddenproject

go 1.21

require github.com/zeromicro/go-zero v1.6.0
`
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goModContent), 0o644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create .api file in main directory
	apiFile := filepath.Join(tmpDir, "main.api")
	if err := os.WriteFile(apiFile, []byte("syntax = \"v1\""), 0o644); err != nil {
		t.Fatalf("Failed to create .api file: %v", err)
	}

	// Create hidden directory with .api file (should be skipped)
	hiddenDir := filepath.Join(tmpDir, ".git", "hooks")
	if err := os.MkdirAll(hiddenDir, 0o755); err != nil {
		t.Fatalf("Failed to create hidden dir: %v", err)
	}

	hiddenAPIFile := filepath.Join(hiddenDir, "hidden.api")
	if err := os.WriteFile(hiddenAPIFile, []byte("syntax = \"v1\""), 0o644); err != nil {
		t.Fatalf("Failed to create hidden .api file: %v", err)
	}

	detector := gozero.NewDetector()
	info, err := detector.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if info.GoZero == nil {
		t.Fatal("GoZero should not be nil")
	}

	// Should only find the main.api file, not the hidden one
	if len(info.GoZero.APIFiles) != 1 {
		t.Errorf("Expected 1 .api file (hidden skipped), got %d", len(info.GoZero.APIFiles))
	}
}

func TestDetector_Detect_InvalidPath(t *testing.T) {
	detector := gozero.NewDetector()
	_, err := detector.Detect("/nonexistent/path/that/does/not/exist")

	if err == nil {
		t.Error("Expected error for invalid path")
	}
}

func TestDetector_Detect_EmptyProject(t *testing.T) {
	// Create temp dir with empty go.mod
	tmpDir := t.TempDir()

	goModContent := `module example.com/empty
`
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goModContent), 0o644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	detector := gozero.NewDetector()
	info, err := detector.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if info.GoZero == nil {
		t.Fatal("GoZero should not be nil")
	}

	if info.GoZero.ModuleName != "example.com/empty" {
		t.Errorf("ModuleName = %s, want example.com/empty", info.GoZero.ModuleName)
	}

	if info.GoZero.GoVersion != "" {
		t.Errorf("GoVersion should be empty for empty go.mod, got %s", info.GoZero.GoVersion)
	}

	if info.GoZero.HasGoZeroDeps {
		t.Error("HasGoZeroDeps should be false for empty go.mod")
	}
}
