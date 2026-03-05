package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// CustomProvider implements Provider for custom OpenAI-compatible services
type CustomProvider struct {
	llm     llms.Model
	model   string
	baseURL string
	name    string
}

// CustomProviderConfig configuration for custom OpenAI-compatible services
type CustomProviderConfig struct {
	BaseURL string
	APIKey  string
	Model   string
	Name    string // optional custom name
}

// NewCustomProvider creates a provider for custom OpenAI-compatible services
func NewCustomProvider(cfg CustomProviderConfig) (*CustomProvider, error) {
	if cfg.BaseURL == "" {
		return nil, errors.New("baseURL is required for custom provider")
	}
	if cfg.APIKey == "" {
		return nil, errors.New("apiKey is required for custom provider")
	}

	name := cfg.Name
	if name == "" {
		name = "custom"
	}

	llm, err := openai.New(
		openai.WithToken(cfg.APIKey),
		openai.WithModel(cfg.Model),
		openai.WithBaseURL(cfg.BaseURL),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create custom LLM: %w", err)
	}

	return &CustomProvider{
		llm:     llm,
		model:   cfg.Model,
		baseURL: cfg.BaseURL,
		name:    name,
	}, nil
}

// Generate generates a response for the given prompt
func (p *CustomProvider) Generate(ctx context.Context, prompt string) (string, error) {
	response, err := llms.GenerateFromSinglePrompt(ctx, p.llm, prompt)
	if err != nil {
		return "", fmt.Errorf("custom provider generation failed: %w", err)
	}
	return response, nil
}

// Name returns the provider name
func (p *CustomProvider) Name() string {
	return p.name
}
