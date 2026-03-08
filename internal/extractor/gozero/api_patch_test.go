// Package gozero provides go-zero framework specific extraction functionality.
package gozero

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/mod/semver"
)

func TestAPIFilePatcher_checkNeedsPatch(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "no patch needed - quoted prefix",
			content:  `prefix: "/api/alert-center"`,
			expected: false,
		},
		{
			name:     "no patch needed - single word",
			content:  `prefix: api`,
			expected: false,
		},
		{
			name:     "needs patch - unquoted multi-hyphen",
			content:  `prefix: /api/alert-center`,
			expected: true,
		},
		{
			name:     "no patch needed - single hyphen",
			content:  `prefix: /api/v1`,
			expected: false,
		},
		{
			name: "needs patch - in server block",
			content: `@server (
    prefix: /api/alert-center
    group: alert
)`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			apiFile := filepath.Join(dir, "test.api")
			if err := os.WriteFile(apiFile, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			patcher := NewAPIFilePatcher()
			needsPatch, err := patcher.checkNeedsPatch(apiFile)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if needsPatch != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, needsPatch)
			}
		})
	}
}

func TestAPIFilePatcher_createPatchedFile(t *testing.T) {
	content := `@server (
    prefix: /api/alert-center
    group: alert
)

service test-api {
    @handler TestHandler
    get /test (Request) returns (Response)
}`

	dir := t.TempDir()
	apiFile := filepath.Join(dir, "test.api")
	if err := os.WriteFile(apiFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	patcher := NewAPIFilePatcher()
	patchedPath, err := patcher.createPatchedFile(apiFile)
	if err != nil {
		t.Fatalf("failed to create patched file: %v", err)
	}
	defer os.Remove(patchedPath)

	// Read patched content
	patchedContent, err := os.ReadFile(patchedPath)
	if err != nil {
		t.Fatalf("failed to read patched file: %v", err)
	}

	// Verify prefix is now quoted
	if !strings.Contains(string(patchedContent), `prefix: "/api/alert-center"`) {
		t.Errorf("patched content should have quoted prefix, got:\n%s", string(patchedContent))
	}

	// Verify other content is preserved
	if !strings.Contains(string(patchedContent), "service test-api") {
		t.Error("patched content should preserve other content")
	}
}

func TestAPIFilePatcher_PatchAPIFiles(t *testing.T) {
	// Create test files
	dir := t.TempDir()

	// File that needs patching
	file1 := filepath.Join(dir, "api1.api")
	os.WriteFile(file1, []byte("prefix: /api/alert-center"), 0o644)

	// File that doesn't need patching
	file2 := filepath.Join(dir, "api2.api")
	os.WriteFile(file2, []byte(`prefix: "/api/alert-center"`), 0o644)

	patcher := NewAPIFilePatcher()
	result, err := patcher.PatchAPIFiles([]string{file1, file2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer patcher.Cleanup()

	// First file should be patched (different path)
	if result[file1] == file1 {
		t.Error("file1 should be patched to a different path")
	}

	// Second file should not be patched (same path)
	if result[file2] != file2 {
		t.Error("file2 should not be patched")
	}

	// Verify patched file exists
	if _, err := os.Stat(result[file1]); err != nil {
		t.Errorf("patched file should exist: %v", err)
	}
}

func TestAPIFilePatcher_Cleanup(t *testing.T) {
	dir := t.TempDir()
	apiFile := filepath.Join(dir, "test.api")
	os.WriteFile(apiFile, []byte("prefix: /api/alert-center"), 0o644)

	patcher := NewAPIFilePatcher()
	patcher.PatchAPIFiles([]string{apiFile})

	// Get patched path before cleanup
	patchedPath := patcher.patchedFiles[apiFile]

	// Verify file exists
	if _, err := os.Stat(patchedPath); err != nil {
		t.Fatalf("patched file should exist before cleanup: %v", err)
	}

	// Cleanup
	patcher.Cleanup()

	// Verify file is removed
	if _, err := os.Stat(patchedPath); !os.IsNotExist(err) {
		t.Error("patched file should be removed after cleanup")
	}
}

func TestValidateAPIFile(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "valid - quoted prefix",
			content:     `prefix: "/api/alert-center"`,
			shouldError: false,
		},
		{
			name:        "valid - no hyphen",
			content:     `prefix: api`,
			shouldError: false,
		},
		{
			name:        "valid - single hyphen",
			content:     `prefix: /api/v1`,
			shouldError: false,
		},
		{
			name:        "invalid - unquoted multi-hyphen",
			content:     `prefix: /api/alert-center-service`,
			shouldError: true,
			errorMsg:    "unquoted prefix value with multiple hyphens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			apiFile := filepath.Join(dir, "test.api")
			if err := os.WriteFile(apiFile, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			err := ValidateAPIFile(apiFile)
			if tt.shouldError {
				if err == nil {
					t.Error("expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("error should contain %q, got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestAPIFilePatcher_GetPatchedPath(t *testing.T) {
	patcher := NewAPIFilePatcher()

	// Before patching, should return original
	original := "/path/to/test.api"
	if path := patcher.GetPatchedPath(original); path != original {
		t.Error("should return original path if not patched")
	}

	// After patching (simulated)
	patcher.patchedFiles[original] = "/path/to/test.api.patched"
	if path := patcher.GetPatchedPath(original); path != "/path/to/test.api.patched" {
		t.Error("should return patched path")
	}
}

func TestAPIFilePatcher_HasPatchedFiles(t *testing.T) {
	patcher := NewAPIFilePatcher()

	// Initially false
	if patcher.HasPatchedFiles() {
		t.Error("should be false initially")
	}

	// After adding a patch
	patcher.patchedFiles["/test.api"] = "/test.api.patched"
	if !patcher.HasPatchedFiles() {
		t.Error("should be true after patching")
	}
}

func TestGoctlVersionCompare(t *testing.T) {
	tests := []struct {
		name       string
		version    string
		shouldSkip bool
	}{
		{
			name:       "exactly 1.9.2",
			version:    "1.9.2",
			shouldSkip: true,
		},
		{
			name:       "greater than 1.9.2",
			version:    "1.9.3",
			shouldSkip: true,
		},
		{
			name:       "much greater",
			version:    "1.10.0",
			shouldSkip: true,
		},
		{
			name:       "version 1.9.10 (multi-digit component)",
			version:    "1.9.10",
			shouldSkip: true,
		},
		{
			name:       "version 2.0.0",
			version:    "2.0.0",
			shouldSkip: true,
		},
		{
			name:       "less than 1.9.2",
			version:    "1.9.1",
			shouldSkip: false,
		},
		{
			name:       "much less than 1.9.2",
			version:    "1.8.0",
			shouldSkip: false,
		},
		{
			name:       "version 1.8.10 (multi-digit, but less)",
			version:    "1.8.10",
			shouldSkip: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patcher := NewAPIFilePatcher()
			patcher.skipPatch = shouldSkipPatch(tt.version)
			if patcher.skipPatch != tt.shouldSkip {
				t.Errorf("version %s: expected skipPatch=%v, got %v", tt.version, tt.shouldSkip, patcher.skipPatch)
			}
		})
	}
}

// shouldSkipPatch checks if a given goctl version should skip patching.
// This is a helper function for testing.
func shouldSkipPatch(version string) bool {
	return semver.Compare("v"+version, "v"+minGoctlVersionForPatch) >= 0
}
