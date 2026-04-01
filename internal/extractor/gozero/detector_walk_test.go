package gozero

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetector_findAPIFiles_RelativeDotPath(t *testing.T) {
	dir := t.TempDir()

	// Create an .api file in the project root
	os.WriteFile(filepath.Join(dir, "main.api"), []byte(`syntax = "v1"`), 0o644)

	// Change to temp dir so "." resolves to it
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
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
