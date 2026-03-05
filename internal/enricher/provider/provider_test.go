package provider

import (
	"context"
	"testing"
)

// MockProvider for testing
type MockProvider struct {
	GenerateFunc func(ctx context.Context, prompt string) (string, error)
	name         string
}

func (m *MockProvider) Generate(ctx context.Context, prompt string) (string, error) {
	if m.GenerateFunc != nil {
		return m.GenerateFunc(ctx, prompt)
	}
	return "", nil
}

func (m *MockProvider) Name() string {
	return m.name
}

func TestProvider_Interface(t *testing.T) {
	// Verify MockProvider implements Provider interface
	var _ Provider = (*MockProvider)(nil)
}

func TestMockProvider_Generate(t *testing.T) {
	mock := &MockProvider{
		name: "test",
		GenerateFunc: func(ctx context.Context, prompt string) (string, error) {
			return "response: " + prompt, nil
		},
	}

	ctx := context.Background()
	result, err := mock.Generate(ctx, "hello")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "response: hello" {
		t.Errorf("Generate() = %q, want %q", result, "response: hello")
	}
	if mock.Name() != "test" {
		t.Errorf("Name() = %q, want %q", mock.Name(), "test")
	}
}
