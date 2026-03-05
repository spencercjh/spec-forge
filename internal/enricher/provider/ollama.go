package provider

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

// OllamaProvider implements Provider for Ollama local LLM using langchaingo
type OllamaProvider struct {
	llm     llms.Model
	model   string
	baseURL string
}

// NewOllamaProvider creates a provider configured for Ollama
func NewOllamaProvider(baseURL, model string) (*OllamaProvider, error) {
	opts := []ollama.Option{ollama.WithModel(model)}

	if baseURL != "" {
		opts = append(opts, ollama.WithServerURL(baseURL))
	}

	llm, err := ollama.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Ollama LLM: %w", err)
	}

	return &OllamaProvider{
		llm:     llm,
		model:   model,
		baseURL: baseURL,
	}, nil
}

// Generate generates a response for the given prompt
func (p *OllamaProvider) Generate(ctx context.Context, prompt string) (string, error) {
	response, err := llms.GenerateFromSinglePrompt(ctx, p.llm, prompt)
	if err != nil {
		return "", fmt.Errorf("Ollama generation failed: %w", err)
	}
	return response, nil
}

// Name returns the provider name
func (p *OllamaProvider) Name() string {
	return "ollama"
}
