// Package gozero tests the go-zero extractor implementation.
package gozero

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestNewDetector(t *testing.T) {
	d := NewDetector()
	if d == nil {
		t.Error("NewDetector() should not return nil")
	}
}

func TestDetector_Detect_NoGoMod(t *testing.T) {
	// Create temp dir without go.mod
	tmpDir := t.TempDir()

	detector := NewDetector()
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

	// Create a .api file (required for detection)
	apiFile := filepath.Join(tmpDir, "test.api")
	if err := os.WriteFile(apiFile, []byte("syntax = \"v1\""), 0o644); err != nil {
		t.Fatalf("Failed to create .api file: %v", err)
	}

	detector := NewDetector()
	info, err := detector.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if info.BuildTool != BuildToolGoModules {
		t.Errorf("BuildTool = %s, want %s", info.BuildTool, BuildToolGoModules)
	}

	if info.BuildFilePath != goModPath {
		t.Errorf("BuildFilePath = %s, want %s", info.BuildFilePath, goModPath)
	}

	if info.FrameworkData == nil {
		t.Fatal("FrameworkData should not be nil")
	}

	goZeroInfo, ok := info.FrameworkData.(*Info)
	if !ok {
		t.Fatal("FrameworkData should be *Info")
	}

	if goZeroInfo.ModuleName != "example.com/testproject" {
		t.Errorf("ModuleName = %s, want example.com/testproject", goZeroInfo.ModuleName)
	}

	if goZeroInfo.GoVersion != "1.21" {
		t.Errorf("GoVersion = %s, want 1.21", goZeroInfo.GoVersion)
	}

	if !goZeroInfo.HasGoZeroDeps {
		t.Error("HasGoZeroDeps should be true")
	}

	if goZeroInfo.GoZeroVersion != "v1.6.0" {
		t.Errorf("GoZeroVersion = %s, want v1.6.0", goZeroInfo.GoZeroVersion)
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

	// Create a .api file (required for detection)
	apiFile := filepath.Join(tmpDir, "test.api")
	if err := os.WriteFile(apiFile, []byte("syntax = \"v1\""), 0o644); err != nil {
		t.Fatalf("Failed to create .api file: %v", err)
	}

	detector := NewDetector()
	info, err := detector.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if info.FrameworkData == nil {
		t.Fatal("FrameworkData should not be nil")
	}

	goZeroInfo, ok := info.FrameworkData.(*Info)
	if !ok {
		t.Fatal("FrameworkData should be *Info")
	}

	if !goZeroInfo.HasGoZeroDeps {
		t.Error("HasGoZeroDeps should be true for single-line require")
	}

	if goZeroInfo.GoZeroVersion != "v1.5.0" {
		t.Errorf("GoZeroVersion = %s, want v1.5.0", goZeroInfo.GoZeroVersion)
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

	detector := NewDetector()
	_, err := detector.Detect(tmpDir)
	if err == nil {
		t.Fatal("expected Detect to fail when no go-zero dependencies are present")
	}

	if _, ok := errors.AsType[*ErrNotGoZeroProject](err); !ok {
		t.Errorf("expected error to be *ErrNotGoZeroProject, got: %T", err)
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

	detector := NewDetector()
	info, err := detector.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if info.FrameworkData == nil {
		t.Fatal("FrameworkData should not be nil")
	}

	goZeroInfo, ok := info.FrameworkData.(*Info)
	if !ok {
		t.Fatal("FrameworkData should be *Info")
	}

	if len(goZeroInfo.APIFiles) != 2 {
		t.Errorf("Expected 2 .api files, got %d", len(goZeroInfo.APIFiles))
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

	detector := NewDetector()
	info, err := detector.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if info.FrameworkData == nil {
		t.Fatal("FrameworkData should not be nil")
	}

	goZeroInfo, ok := info.FrameworkData.(*Info)
	if !ok {
		t.Fatal("FrameworkData should be *Info")
	}

	// Should only find the main.api file, not the vendor one
	if len(goZeroInfo.APIFiles) != 1 {
		t.Errorf("Expected 1 .api file (vendor skipped), got %d", len(goZeroInfo.APIFiles))
	}

	if len(goZeroInfo.APIFiles) > 0 && goZeroInfo.APIFiles[0] != apiFile {
		t.Errorf("Expected %s, got %s", apiFile, goZeroInfo.APIFiles[0])
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

	detector := NewDetector()
	info, err := detector.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if info.FrameworkData == nil {
		t.Fatal("FrameworkData should not be nil")
	}

	goZeroInfo, ok := info.FrameworkData.(*Info)
	if !ok {
		t.Fatal("FrameworkData should be *Info")
	}

	// Should only find the main.api file, not the hidden one
	if len(goZeroInfo.APIFiles) != 1 {
		t.Errorf("Expected 1 .api file (hidden skipped), got %d", len(goZeroInfo.APIFiles))
	}
}

func TestDetector_Detect_InvalidPath(t *testing.T) {
	detector := NewDetector()
	_, err := detector.Detect("/nonexistent/path/that/does/not/exist")

	if err == nil {
		t.Error("Expected error for invalid path")
	}
}

func TestDetector_Detect_EmptyProject(t *testing.T) {
	// Create temp dir with empty go.mod (no go-zero dependency)
	tmpDir := t.TempDir()

	goModContent := `module example.com/empty
`
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goModContent), 0o644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	detector := NewDetector()
	_, err := detector.Detect(tmpDir)
	if err == nil {
		t.Fatal("expected Detect to fail when no go-zero dependency is present")
	}

	if _, ok := errors.AsType[*ErrNotGoZeroProject](err); !ok {
		t.Errorf("expected error to be *ErrNotGoZeroProject, got: %T", err)
	}
}

func TestDetector_findAPIFiles_RelativeDotPath(t *testing.T) {
	dir := t.TempDir()

	// Create an .api file in the project root
	if err := os.WriteFile(filepath.Join(dir, "main.api"), []byte(`syntax = "v1"`), 0o644); err != nil {
		t.Fatalf("failed to create main.api: %v", err)
	}

	// Change to temp dir so "." resolves to it
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	}()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	d := NewDetector()
	files, err := d.findAPIFiles(".")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected at least 1 .api file with relative \".\" path, got 0")
	}
}
