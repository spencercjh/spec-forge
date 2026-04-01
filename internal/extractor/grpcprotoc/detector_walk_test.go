package grpcprotoc

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetector_findProtoFiles_RelativeDotPath(t *testing.T) {
	dir := t.TempDir()

	// Create a .proto file in the project root
	os.WriteFile(filepath.Join(dir, "main.proto"), []byte(`syntax = "proto3"; package main;`), 0o644)

	// Change to temp dir so "." resolves to it
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	d := NewDetector()
	files, err := d.findProtoFiles(".")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected at least 1 .proto file with relative \".\" path, got 0")
	}
}
