package publisher

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/spencercjh/spec-forge/internal/executor"
)

// mockExecutor is a mock implementation of executor.Interface for testing
type mockExecutor struct {
	result *executor.ExecuteResult
	err    error
}

func (m *mockExecutor) Execute(ctx context.Context, opts *executor.ExecuteOptions) (*executor.ExecuteResult, error) {
	return m.result, m.err
}

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
	// Use mock executor to ensure test doesn't depend on external rdme CLI
	mock := &mockExecutor{}
	p := NewReadMePublisherWithExecutor(mock)
	_, err := p.Publish(context.Background(), nil, &PublishOptions{ReadMe: &ReadMeOptions{APIKey: "test"}})
	if err == nil {
		t.Error("expected error for nil spec")
	}
}

func TestReadMePublisher_Publish_NilOptions(t *testing.T) {
	// Use mock executor to ensure test doesn't depend on external rdme CLI
	mock := &mockExecutor{}
	p := NewReadMePublisherWithExecutor(mock)
	_, err := p.Publish(context.Background(), nil, nil)
	if err == nil {
		t.Error("expected error for nil options")
	}
}

func TestReadMePublisher_Publish_NilReadMeOptions(t *testing.T) {
	// Use mock executor to ensure test doesn't depend on external rdme CLI
	mock := &mockExecutor{}
	p := NewReadMePublisherWithExecutor(mock)
	spec := createTestSpec()
	_, err := p.Publish(context.Background(), spec, &PublishOptions{})
	if err == nil {
		t.Error("expected error for nil readme options")
	}
}

func TestReadMePublisher_Publish_MissingAPIKey(t *testing.T) {
	mock := &mockExecutor{}
	p := NewReadMePublisherWithExecutor(mock)
	spec := createTestSpec()

	// Ensure README_API_KEY is not set in environment
	t.Setenv("README_API_KEY", "")

	_, err := p.Publish(context.Background(), spec, &PublishOptions{ReadMe: &ReadMeOptions{APIKey: ""}})
	if err == nil {
		t.Error("expected error for missing API key")
	}
}

func TestReadMePublisher_Publish_EnvVarFallback(t *testing.T) {
	mock := &mockExecutor{
		result: &executor.ExecuteResult{
			ExitCode: 0,
			Stdout:   "Successfully uploaded",
		},
	}
	p := NewReadMePublisherWithExecutor(mock)
	spec := createTestSpec()

	// Set API key via environment variable
	t.Setenv("README_API_KEY", "env-api-key")

	// Empty APIKey in options, should fallback to env var
	_, err := p.Publish(context.Background(), spec, &PublishOptions{ReadMe: &ReadMeOptions{APIKey: ""}})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestReadMePublisher_Publish_OptionsKeyPriority(t *testing.T) {
	mock := &mockExecutor{
		result: &executor.ExecuteResult{
			ExitCode: 0,
			Stdout:   "Successfully uploaded",
		},
	}
	p := NewReadMePublisherWithExecutor(mock)
	spec := createTestSpec()

	// Set different keys in env and options
	t.Setenv("README_API_KEY", "env-api-key")

	// Options key should take priority - verify no error
	_, err := p.Publish(context.Background(), spec, &PublishOptions{ReadMe: &ReadMeOptions{APIKey: "options-api-key"}})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestReadMePublisher_Publish_CommandNotFound(t *testing.T) {
	mock := &mockExecutor{
		err: &executor.CommandNotFoundError{Command: "rdme", Hint: "Install rdme"},
	}
	p := NewReadMePublisherWithExecutor(mock)
	spec := createTestSpec()

	_, err := p.Publish(context.Background(), spec, &PublishOptions{ReadMe: &ReadMeOptions{APIKey: "test-key"}})
	if err == nil {
		t.Error("expected error for command not found")
	}
	// Check that error message contains expected text (error is wrapped)
	if !strings.Contains(err.Error(), "rdme CLI not found") {
		t.Errorf("expected error to contain 'rdme CLI not found', got: %v", err)
	}
}

func TestReadMePublisher_Publish_CommandFailed(t *testing.T) {
	mock := &mockExecutor{
		result: &executor.ExecuteResult{
			ExitCode: 1,
			Stdout:   "",
			Stderr:   "API error: Invalid API key",
		},
		err: &executor.CommandFailedError{
			Command:  "rdme",
			ExitCode: 1,
			Stderr:   "API error: Invalid API key",
		},
	}
	p := NewReadMePublisherWithExecutor(mock)
	spec := createTestSpec()

	_, err := p.Publish(context.Background(), spec, &PublishOptions{ReadMe: &ReadMeOptions{APIKey: "test-key"}})
	if err == nil {
		t.Error("expected error for command failure")
	}
	// Check that error message contains expected text (error is wrapped)
	if !strings.Contains(err.Error(), "rdme command failed") {
		t.Errorf("expected error to contain 'rdme command failed', got: %v", err)
	}
	if !strings.Contains(err.Error(), "API error: Invalid API key") {
		t.Errorf("expected error to contain output from command, got: %v", err)
	}
}

func TestReadMePublisher_Publish_Success(t *testing.T) {
	mock := &mockExecutor{
		result: &executor.ExecuteResult{
			ExitCode: 0,
			Stdout:   "✔ Successfully uploaded to ReadMe",
		},
	}
	p := NewReadMePublisherWithExecutor(mock)
	spec := createTestSpec()

	result, err := p.Publish(context.Background(), spec, &PublishOptions{
		ReadMe: &ReadMeOptions{
			APIKey: "test-key",
			Slug:   "my-api",
			Branch: "v1.0.0",
		},
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result.Path != "readme.com/slug:my-api/branch:v1.0.0" {
		t.Errorf("unexpected path: %s", result.Path)
	}
	if result.Message != "✔ Successfully uploaded to ReadMe" {
		t.Errorf("unexpected message: %s", result.Message)
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

func TestReadMePublisher_BuildEnv(t *testing.T) {
	p := NewReadMePublisher()

	// Set some env vars including README_API_KEY
	t.Setenv("README_API_KEY", "existing-key")
	t.Setenv("OTHER_VAR", "other-value")

	env := p.buildEnv("new-key")

	// Check that new key is present
	if !slices.Contains(env, "README_API_KEY=new-key") {
		t.Error("expected README_API_KEY=new-key in env")
	}

	// Check that old key is not present
	if slices.Contains(env, "README_API_KEY=existing-key") {
		t.Error("old README_API_KEY should have been filtered out")
	}

	// Check that other vars are preserved
	if !slices.Contains(env, "OTHER_VAR=other-value") {
		t.Error("expected OTHER_VAR to be preserved")
	}
}

func TestReadMePublisher_BuildLocation(t *testing.T) {
	p := NewReadMePublisher()

	tests := []struct {
		name     string
		opts     *ReadMeOptions
		expected string
	}{
		{
			name:     "empty options",
			opts:     &ReadMeOptions{},
			expected: "readme.com (default)",
		},
		{
			name:     "with slug",
			opts:     &ReadMeOptions{Slug: "my-api"},
			expected: "readme.com/slug:my-api",
		},
		{
			name:     "with branch",
			opts:     &ReadMeOptions{Branch: "v1.0.0"},
			expected: "readme.com/branch:v1.0.0",
		},
		{
			name:     "with slug and branch",
			opts:     &ReadMeOptions{Slug: "my-api", Branch: "v1.0.0"},
			expected: "readme.com/slug:my-api/branch:v1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			location := p.buildLocation(tt.opts)
			if location != tt.expected {
				t.Errorf("buildLocation() = %q, want %q", location, tt.expected)
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
