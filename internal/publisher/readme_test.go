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

	// Ensure README_API_KEY is not set in environment
	t.Setenv("README_API_KEY", "")

	_, err := p.Publish(context.Background(), spec, &PublishOptions{ReadMe: &ReadMeOptions{APIKey: ""}})
	if err == nil {
		t.Error("expected error for missing API key")
	}
}

func TestReadMePublisher_Publish_EnvVarFallback(t *testing.T) {
	p := NewReadMePublisher()
	spec := createTestSpec()

	// Set API key via environment variable
	t.Setenv("README_API_KEY", "env-api-key")

	// Empty APIKey in options, should fallback to env var
	// This will fail at command execution because rdme is not available in test,
	// but we can verify it doesn't fail at the API key validation stage
	_, err := p.Publish(context.Background(), spec, &PublishOptions{ReadMe: &ReadMeOptions{APIKey: ""}})

	// Should fail at command execution (rdme not found), not at API key validation
	if err != nil && err.Error() == "readme API key is required (set --readme-api-key flag or README_API_KEY env var)" {
		t.Error("should have used env var fallback, but failed with missing API key error")
	}
}

func TestReadMePublisher_Publish_OptionsKeyPriority(t *testing.T) {
	p := NewReadMePublisher()
	spec := createTestSpec()

	// Set different keys in env and options
	t.Setenv("README_API_KEY", "env-api-key")

	// Options key should take priority over env var
	// This will fail at command execution, but we verify it doesn't fail at validation
	_, err := p.Publish(context.Background(), spec, &PublishOptions{ReadMe: &ReadMeOptions{APIKey: "options-api-key"}})

	// Should not fail at API key validation
	if err != nil && err.Error() == "readme API key is required (set --readme-api-key flag or README_API_KEY env var)" {
		t.Error("should have used options API key, but failed with missing API key error")
	}
}

func TestReadMePublisher_BuildArgs(t *testing.T) {
	p := NewReadMePublisher()

	tests := []struct {
		name     string
		path     string
		opts     *PublishOptions
		expected []string
	}{
		{
			name: "minimal options without overwrite",
			path: "/tmp/spec.yaml",
			opts: &PublishOptions{
				ReadMe: &ReadMeOptions{APIKey: "test-key"},
			},
			expected: []string{
				"openapi", "upload", "/tmp/spec.yaml",
			},
		},
		{
			name: "minimal options with overwrite",
			path: "/tmp/spec.yaml",
			opts: &PublishOptions{
				Overwrite: true,
				ReadMe:    &ReadMeOptions{APIKey: "test-key"},
			},
			expected: []string{
				"openapi", "upload", "/tmp/spec.yaml",
				"--confirm-overwrite",
			},
		},
		{
			name: "with branch",
			path: "/tmp/spec.yaml",
			opts: &PublishOptions{
				ReadMe: &ReadMeOptions{APIKey: "test-key", Branch: "v1.0.0"},
			},
			expected: []string{
				"openapi", "upload", "/tmp/spec.yaml",
				"--branch", "v1.0.0",
			},
		},
		{
			name: "with slug",
			path: "/tmp/spec.yaml",
			opts: &PublishOptions{
				ReadMe: &ReadMeOptions{APIKey: "test-key", Slug: "my-api"},
			},
			expected: []string{
				"openapi", "upload", "/tmp/spec.yaml",
				"--slug", "my-api",
			},
		},
		{
			name: "with useSpecVersion",
			path: "/tmp/spec.yaml",
			opts: &PublishOptions{
				ReadMe: &ReadMeOptions{APIKey: "test-key", UseSpecVersion: true},
			},
			expected: []string{
				"openapi", "upload", "/tmp/spec.yaml",
				"--useSpecVersion",
			},
		},
		{
			name: "full options",
			path: "/tmp/spec.yaml",
			opts: &PublishOptions{
				Overwrite: true,
				ReadMe:    &ReadMeOptions{APIKey: "test-key", Branch: "v1.0.0", Slug: "my-api"},
			},
			expected: []string{
				"openapi", "upload", "/tmp/spec.yaml",
				"--slug", "my-api",
				"--branch", "v1.0.0",
				"--confirm-overwrite",
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
