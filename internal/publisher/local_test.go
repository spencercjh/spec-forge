package publisher

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestLocalPublisher_Name(t *testing.T) {
	p := NewLocalPublisher()
	if p.Name() != "local" {
		t.Errorf("expected name 'local', got %s", p.Name())
	}
}

func TestLocalPublisher_Publish_YAML(t *testing.T) {
	p := NewLocalPublisher()
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "openapi.yaml")

	spec := &openapi3.T{
		OpenAPI: "3.1.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
	}

	result, err := p.Publish(context.Background(), spec, &PublishOptions{
		OutputPath: outputPath,
		Format:     "yaml",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Path != outputPath {
		t.Errorf("expected path %s, got %s", outputPath, result.Path)
	}

	if result.Format != "yaml" {
		t.Errorf("expected format yaml, got %s", result.Format)
	}

	if result.BytesWritten == 0 {
		t.Error("expected bytes to be written")
	}

	// Verify file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("expected output file to exist")
	}
}

func TestLocalPublisher_Publish_JSON(t *testing.T) {
	p := NewLocalPublisher()
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "openapi.json")

	spec := &openapi3.T{
		OpenAPI: "3.1.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
	}

	result, err := p.Publish(context.Background(), spec, &PublishOptions{
		OutputPath: outputPath,
		Format:     "json",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Format != "json" {
		t.Errorf("expected format json, got %s", result.Format)
	}
}

func TestLocalPublisher_Publish_AutoFormat(t *testing.T) {
	p := NewLocalPublisher()
	tmpDir := t.TempDir()

	// Test JSON extension auto-detection
	jsonPath := filepath.Join(tmpDir, "spec.json")
	spec := &openapi3.T{OpenAPI: "3.1.0"}

	result, err := p.Publish(context.Background(), spec, &PublishOptions{
		OutputPath: jsonPath,
		// Format not specified, should auto-detect from extension
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Format != "json" {
		t.Errorf("expected auto-detected format json, got %s", result.Format)
	}
}

func TestLocalPublisher_Publish_NilSpec(t *testing.T) {
	p := NewLocalPublisher()

	_, err := p.Publish(context.Background(), nil, &PublishOptions{
		OutputPath: "test.yaml",
	})

	if err == nil {
		t.Error("expected error for nil spec")
	}
}

func TestLocalPublisher_Publish_NilOptions(t *testing.T) {
	p := NewLocalPublisher()
	spec := &openapi3.T{OpenAPI: "3.1.0"}

	_, err := p.Publish(context.Background(), spec, nil)

	if err == nil {
		t.Error("expected error for nil options")
	}
}

func TestLocalPublisher_Publish_EmptyPath(t *testing.T) {
	p := NewLocalPublisher()
	spec := &openapi3.T{OpenAPI: "3.1.0"}

	_, err := p.Publish(context.Background(), spec, &PublishOptions{
		OutputPath: "",
	})

	if err == nil {
		t.Error("expected error for empty output path")
	}
}

func TestLocalPublisher_Publish_CreatesDirectory(t *testing.T) {
	p := NewLocalPublisher()
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "subdir", "nested", "openapi.yaml")

	spec := &openapi3.T{
		OpenAPI: "3.1.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
	}

	result, err := p.Publish(context.Background(), spec, &PublishOptions{
		OutputPath: outputPath,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify nested directories were created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("expected output file to exist in nested directory")
	}

	if result.Path != outputPath {
		t.Errorf("expected path %s, got %s", outputPath, result.Path)
	}
}

func TestLocalPublisher_Publish_Overwrite(t *testing.T) {
	p := NewLocalPublisher()
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "openapi.yaml")

	spec := &openapi3.T{
		OpenAPI: "3.1.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
	}

	// First write should succeed
	_, err := p.Publish(context.Background(), spec, &PublishOptions{
		OutputPath: outputPath,
	})
	if err != nil {
		t.Fatalf("first publish failed: %v", err)
	}

	// Second write without overwrite should fail
	_, err = p.Publish(context.Background(), spec, &PublishOptions{
		OutputPath: outputPath,
	})
	if err == nil {
		t.Error("expected error for existing file without overwrite")
	}

	// Third write with overwrite should succeed
	_, err = p.Publish(context.Background(), spec, &PublishOptions{
		OutputPath: outputPath,
		Overwrite:  true,
	})
	if err != nil {
		t.Errorf("overwrite publish failed: %v", err)
	}
}
