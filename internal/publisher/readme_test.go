package publisher

import (
	"context"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

// createTestSpec creates a minimal OpenAPI spec for testing
func createTestSpec() *openapi3.T {
	return &openapi3.T{
		OpenAPI: "3.1.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
	}
}

func TestReadMePublisher_Name(t *testing.T) {
	p := NewReadMePublisher()
	if p.Name() != "readme" {
		t.Errorf("expected name 'readme', got %s", p.Name())
	}
}

func TestReadMePublisher_Publish_NilSpec(t *testing.T) {
	p := NewReadMePublisher()
	_, err := p.Publish(context.Background(), nil, &PublishOptions{ReadMe: &ReadMeOptions{APIKey: "test"}})
	if err == nil {
		t.Error("expected error for nil spec")
	}
}

func TestReadMePublisher_Publish_NilOptions(t *testing.T) {
	p := NewReadMePublisher()
	_, err := p.Publish(context.Background(), nil, nil)
	if err == nil {
		t.Error("expected error for nil options")
	}
}

func TestReadMePublisher_Publish_NilReadMeOptions(t *testing.T) {
	p := NewReadMePublisher()
	spec := createTestSpec()
	_, err := p.Publish(context.Background(), spec, &PublishOptions{})
	if err == nil {
		t.Error("expected error for nil readme options")
	}
}

func TestReadMePublisher_Publish_MissingAPIKey(t *testing.T) {
	p := NewReadMePublisher()
	spec := createTestSpec()
	_, err := p.Publish(context.Background(), spec, &PublishOptions{ReadMe: &ReadMeOptions{APIKey: ""}})
	if err == nil {
		t.Error("expected error for missing API key")
	}
}

func TestReadMePublisher_BuildArgs(t *testing.T) {
	p := NewReadMePublisher()

	tests := []struct {
		name     string
		path     string
		opts     *ReadMeOptions
		expected []string
	}{
		{
			name: "minimal options",
			path: "/tmp/spec.yaml",
			opts: &ReadMeOptions{APIKey: "test-key"},
			expected: []string{
				"openapi", "upload", "/tmp/spec.yaml",
				"--key", "test-key",
				"--confirm-overwrite",
			},
		},
		{
			name: "with branch",
			path: "/tmp/spec.yaml",
			opts: &ReadMeOptions{APIKey: "test-key", Branch: "v1.0.0"},
			expected: []string{
				"openapi", "upload", "/tmp/spec.yaml",
				"--key", "test-key",
				"--confirm-overwrite",
				"--branch", "v1.0.0",
			},
		},
		{
			name: "with slug",
			path: "/tmp/spec.yaml",
			opts: &ReadMeOptions{APIKey: "test-key", Slug: "my-api"},
			expected: []string{
				"openapi", "upload", "/tmp/spec.yaml",
				"--key", "test-key",
				"--confirm-overwrite",
				"--slug", "my-api",
			},
		},
		{
			name: "with useSpecVersion",
			path: "/tmp/spec.yaml",
			opts: &ReadMeOptions{APIKey: "test-key", UseSpecVersion: true},
			expected: []string{
				"openapi", "upload", "/tmp/spec.yaml",
				"--key", "test-key",
				"--confirm-overwrite",
				"--useSpecVersion",
			},
		},
		{
			name: "full options",
			path: "/tmp/spec.yaml",
			opts: &ReadMeOptions{APIKey: "test-key", Branch: "v1.0.0", Slug: "my-api"},
			expected: []string{
				"openapi", "upload", "/tmp/spec.yaml",
				"--key", "test-key",
				"--confirm-overwrite",
				"--branch", "v1.0.0",
				"--slug", "my-api",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := p.buildArgs(tt.path, tt.opts)
			if !equalArgs(args, tt.expected) {
				t.Errorf("args mismatch\ngot:      %v\nexpected: %v", args, tt.expected)
			}
		})
	}
}

func equalArgs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
