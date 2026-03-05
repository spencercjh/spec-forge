package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestOpenAICompatibleProvider_Generate(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/chat/completions") {
			t.Errorf("expected /chat/completions path, got %s", r.URL.Path)
		}

		// Verify headers
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Authorization header")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type header")
		}

		// Verify request body
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}
		if req["model"] != "gpt-4o" {
			t.Errorf("expected model gpt-4o, got %v", req["model"])
		}

		// Send response
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"role":    "assistant",
						"content": "Hello, this is a test response.",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create provider
	provider := NewOpenAICompatibleProvider(OpenAICompatibleConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "gpt-4o",
	})

	// Test Generate
	ctx := context.Background()
	result, err := provider.Generate(ctx, "Hello")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Hello, this is a test response." {
		t.Errorf("Generate() = %q, want %q", result, "Hello, this is a test response.")
	}
	if provider.Name() != "openai-compatible" {
		t.Errorf("Name() = %q, want %q", provider.Name(), "openai-compatible")
	}
}

func TestOpenAICompatibleProvider_ExtraHeaders(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify extra headers
		if r.Header.Get("X-Tenant-ID") != "my-team" {
			t.Errorf("expected X-Tenant-ID header")
		}

		// Send response
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"role":    "assistant",
						"content": "ok",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create provider with extra headers
	provider := NewOpenAICompatibleProvider(OpenAICompatibleConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "test-model",
		ExtraHeaders: map[string]string{
			"X-Tenant-ID": "my-team",
		},
	})

	ctx := context.Background()
	_, err := provider.Generate(ctx, "test")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpenAICompatibleProvider_Timeout(t *testing.T) {
	// Create slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create provider with short timeout
	provider := NewOpenAICompatibleProvider(OpenAICompatibleConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "test-model",
		Timeout: 10 * time.Millisecond,
	})

	ctx := context.Background()
	_, err := provider.Generate(ctx, "test")

	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestOpenAICompatibleProvider_APIError(t *testing.T) {
	// Create server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		resp := map[string]interface{}{
			"error": map[string]string{
				"message": "Invalid API key",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := NewOpenAICompatibleProvider(OpenAICompatibleConfig{
		BaseURL: server.URL,
		APIKey:  "invalid-key",
		Model:   "test-model",
	})

	ctx := context.Background()
	_, err := provider.Generate(ctx, "test")

	if err == nil {
		t.Fatal("expected error for unauthorized")
	}
}

func TestOpenAICompatibleProvider_EmptyResponse(t *testing.T) {
	// Create server that returns empty choices
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := NewOpenAICompatibleProvider(OpenAICompatibleConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "test-model",
	})

	ctx := context.Background()
	_, err := provider.Generate(ctx, "test")

	if err == nil {
		t.Fatal("expected error for empty response")
	}
}

func TestFactory_OpenAI(t *testing.T) {
	provider := NewOpenAIProvider("test-api-key", "gpt-4o")

	if provider.Name() != "openai" {
		t.Errorf("Name() = %q, want %q", provider.Name(), "openai")
	}
}

func TestFactory_Anthropic(t *testing.T) {
	provider := NewAnthropicProvider("test-api-key", "claude-3-opus")

	if provider.Name() != "anthropic" {
		t.Errorf("Name() = %q, want %q", provider.Name(), "anthropic")
	}
}

func TestFactory_Ollama(t *testing.T) {
	provider := NewOllamaProvider("http://localhost:11434/v1", "llama3")

	if provider.Name() != "ollama" {
		t.Errorf("Name() = %q, want %q", provider.Name(), "ollama")
	}
}
