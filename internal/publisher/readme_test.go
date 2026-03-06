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

func TestReadMePublisher_Publish_ValidSpecWithNilOpts(t *testing.T) {
	p := NewReadMePublisher()
	spec := createTestSpec()
	// This test passes a valid spec but nil options
	_, err := p.Publish(context.Background(), spec, nil)
	if err == nil {
		t.Error("expected error for nil options when spec is valid")
	}
}

func TestReadMePublisher_BuildArgs(t *testing.T) {
	p := NewReadMePublisher()

	tests := []struct {
		name       string
		specPath   string
		opts       *PublishOptions
		expected   []string
		expectSlug bool // if true, slug should be in output
	}{
		{
			name:     "minimal options without overwrite",
			specPath: "/tmp/spec.yaml",
			opts: &PublishOptions{
				ReadMe: &ReadMeOptions{APIKey: "test-key"},
			},
			expected: []string{
				"openapi", "upload", "/tmp/spec.yaml",
				// NO --slug default-slug since we don't auto-add slug
				// NO --confirm-overwrite since Overwrite is false
			},
		},
		{
			name:     "with overwrite flag",
			specPath: "/tmp/spec.yaml",
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
			name:     "with branch",
			specPath: "/tmp/spec.yaml",
			opts: &PublishOptions{
				ReadMe: &ReadMeOptions{APIKey: "test-key", Branch: "v1.0.0"},
			},
			expected: []string{
				"openapi", "upload", "/tmp/spec.yaml",
				"--branch", "v1.0.0",
			},
		},
		{
			name:     "with slug",
			specPath: "/tmp/spec.yaml",
			opts: &PublishOptions{
				ReadMe: &ReadMeOptions{APIKey: "test-key", Slug: "my-api"},
			},
			expected: []string{
				"openapi", "upload", "/tmp/spec.yaml",
				"--slug", "my-api",
			},
		},
		{
			name:     "with useSpecVersion",
			specPath: "/tmp/spec.yaml",
			opts: &PublishOptions{
				ReadMe: &ReadMeOptions{APIKey: "test-key", UseSpecVersion: true},
			},
			expected: []string{
				"openapi", "upload", "/tmp/spec.yaml",
				"--useSpecVersion",
			},
		},
		{
			name:     "full options with overwrite",
			specPath: "/tmp/spec.yaml",
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
		{
			name:     "full options without overwrite",
			specPath: "/tmp/spec.yaml",
			opts: &PublishOptions{
				Overwrite: false,
				ReadMe:    &ReadMeOptions{APIKey: "test-key", Branch: "v1.0.0", Slug: "my-api"},
			},
			expected: []string{
				"openapi", "upload", "/tmp/spec.yaml",
				"--slug", "my-api",
				"--branch", "v1.0.0",
				// NO --confirm-overwrite since Overwrite is false
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := p.buildArgs(tt.specPath, tt.opts)
			if !containsAll(args, tt.expected) {
				t.Errorf("args missing expected elements\ngot:      %v\nexpected: %v", args, tt.expected)
			}
		})
	}
}

func containsAll(args, expected []string) bool {
	for _, exp := range expected {
		found := false
		for _, arg := range args {
			if arg == exp {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
