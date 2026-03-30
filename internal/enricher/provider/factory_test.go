package provider

import (
	"context"
	"testing"
)

// MockProvider for testing
type MockProvider struct {
	GenerateFunc func(ctx context.Context, prompt string, opts ...Option) (string, *TokenUsage, error)
	name         string
}

func (m *MockProvider) Generate(ctx context.Context, prompt string, opts ...Option) (string, *TokenUsage, error) {
	if m.GenerateFunc != nil {
		return m.GenerateFunc(ctx, prompt, opts...)
	}
	return "", nil, nil
}

func (m *MockProvider) Name() string {
	return m.name
}

func TestProvider_Interface(t *testing.T) {
	// Verify MockProvider implements Provider interface
	var _ Provider = (*MockProvider)(nil)
}

func TestMockProvider_Generate(t *testing.T) {
	usage := &TokenUsage{InputTokens: 10, OutputTokens: 20}
	mock := &MockProvider{
		name: "test",
		GenerateFunc: func(ctx context.Context, prompt string, opts ...Option) (string, *TokenUsage, error) {
			return "response: " + prompt, usage, nil
		},
	}

	ctx := context.Background()
	result, gotUsage, err := mock.Generate(ctx, "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "response: hello" {
		t.Errorf("Generate() = %q, want %q", result, "response: hello")
	}
	if gotUsage.InputTokens != 10 || gotUsage.OutputTokens != 20 {
		t.Errorf("usage = %+v, want InputTokens=10 OutputTokens=20", gotUsage)
	}
	if mock.Name() != "test" {
		t.Errorf("Name() = %q, want %q", mock.Name(), "test")
	}
}

func TestNewOpenAIProvider(t *testing.T) {
	provider, err := newOpenAIProvider("test-api-key", "gpt-4o")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.Name() != "openai" {
		t.Errorf("Name() = %q, want %q", provider.Name(), "openai")
	}
}

func TestNewAnthropicProvider(t *testing.T) {
	provider, err := newAnthropicProvider("test-api-key", "claude-3-opus")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.Name() != "anthropic" {
		t.Errorf("Name() = %q, want %q", provider.Name(), "anthropic")
	}
}

func TestNewOllamaProvider(t *testing.T) {
	provider, err := newOllamaProvider("http://localhost:11434", "llama3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.Name() != "ollama" {
		t.Errorf("Name() = %q, want %q", provider.Name(), "ollama")
	}
}

func TestNewOllamaProvider_EmptyBaseURL(t *testing.T) {
	provider, err := newOllamaProvider("", "llama3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.Name() != "ollama" {
		t.Errorf("Name() = %q, want %q", provider.Name(), "ollama")
	}
}

func TestNewCustomProvider(t *testing.T) {
	provider, err := newCustomProvider(CustomProviderConfig{
		BaseURL: "https://api.example.com/v1",
		APIKey:  "test-key",
		Model:   "custom-model",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.Name() != "custom" {
		t.Errorf("Name() = %q, want %q", provider.Name(), "custom")
	}
}

func TestNewCustomProvider_WithCustomName(t *testing.T) {
	provider, err := newCustomProvider(CustomProviderConfig{
		BaseURL: "https://api.example.com/v1",
		APIKey:  "test-key",
		Model:   "custom-model",
		Name:    "my-provider",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.Name() != "my-provider" {
		t.Errorf("Name() = %q, want %q", provider.Name(), "my-provider")
	}
}

func TestNewCustomProvider_MissingBaseURL(t *testing.T) {
	_, err := newCustomProvider(CustomProviderConfig{
		APIKey: "test-key",
		Model:  "custom-model",
	})
	if err == nil {
		t.Fatal("expected error for missing baseURL")
	}
}

func TestNewCustomProvider_MissingAPIKey(t *testing.T) {
	_, err := newCustomProvider(CustomProviderConfig{
		BaseURL: "https://api.example.com/v1",
		Model:   "custom-model",
	})
	if err == nil {
		t.Fatal("expected error for missing apiKey")
	}
}

func TestNewProvider(t *testing.T) {
	tests := []struct {
		name       string
		cfg        Config
		wantName   string
		wantErr    bool
		errContain string
	}{
		{
			name:     "openai",
			cfg:      Config{Provider: "openai", Model: "gpt-4o", APIKey: "test"},
			wantName: "openai",
		},
		{
			name:     "anthropic",
			cfg:      Config{Provider: "anthropic", Model: "claude-3-opus", APIKey: "test"},
			wantName: "anthropic",
		},
		{
			name:     "ollama with baseURL",
			cfg:      Config{Provider: "ollama", Model: "llama3", BaseURL: "http://localhost:11434"},
			wantName: "ollama",
		},
		{
			name:     "ollama without baseURL",
			cfg:      Config{Provider: "ollama", Model: "llama3"},
			wantName: "ollama",
		},
		{
			name:     "custom",
			cfg:      Config{Provider: "custom", Model: "test", BaseURL: "https://api.example.com/v1", APIKey: "test"},
			wantName: "custom",
		},
		{
			name:       "unsupported",
			cfg:        Config{Provider: "unsupported", Model: "test"},
			wantErr:    true,
			errContain: "unsupported provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewProvider(tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				if tt.errContain != "" && !containsString(err.Error(), tt.errContain) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tt.errContain)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if provider.Name() != tt.wantName {
				t.Errorf("Name() = %q, want %q", provider.Name(), tt.wantName)
			}
		})
	}
}

func TestUnsupportedProviderError(t *testing.T) {
	err := &UnsupportedProviderError{Provider: "unknown"}
	if err.Error() != "unsupported provider: unknown" {
		t.Errorf("Error() = %q, want %q", err.Error(), "unsupported provider: unknown")
	}
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestTokenUsage_Add(t *testing.T) {
	u := &TokenUsage{InputTokens: 10, OutputTokens: 20}
	u.Add(&TokenUsage{InputTokens: 5, OutputTokens: 10})
	if u.InputTokens != 15 || u.OutputTokens != 30 {
		t.Errorf("Add() = %+v, want InputTokens=15 OutputTokens=30", u)
	}
}

func TestTokenUsage_Add_Nil(t *testing.T) {
	u := &TokenUsage{InputTokens: 10, OutputTokens: 20}
	u.Add(nil)
	if u.InputTokens != 10 || u.OutputTokens != 20 {
		t.Errorf("Add(nil) should be no-op, got %+v", u)
	}
}

func TestTokenUsage_Total(t *testing.T) {
	u := &TokenUsage{InputTokens: 100, OutputTokens: 50}
	if u.Total() != 150 {
		t.Errorf("Total() = %d, want 150", u.Total())
	}
}

func TestWithStreamingFunc(t *testing.T) {
	var called bool
	fn := func(ctx context.Context, chunk []byte) error {
		called = true
		return nil
	}

	opts := applyOptions(WithStreamingFunc(fn))
	if opts.StreamingFunc == nil {
		t.Fatal("StreamingFunc should not be nil")
	}
	if called {
		t.Fatal("StreamingFunc should not be called yet")
	}

	// Call the function to verify it was set correctly
	_ = opts.StreamingFunc(context.Background(), []byte("test"))
	if !called {
		t.Fatal("StreamingFunc should have been called")
	}
}

func TestApplyOptionsEmpty(t *testing.T) {
	opts := applyOptions()
	if opts == nil {
		t.Fatal("opts should not be nil")
	}
	if opts.StreamingFunc != nil {
		t.Fatal("StreamingFunc should be nil when no options are provided")
	}
}
