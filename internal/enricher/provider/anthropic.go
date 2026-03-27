package provider

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
)

// AnthropicProvider implements Provider for Anthropic Claude using langchaingo
type AnthropicProvider struct {
	llm   llms.Model
	model string
}

// newAnthropicProvider creates a provider configured for Anthropic
func newAnthropicProvider(apiKey, model string) (*AnthropicProvider, error) {
	llm, err := anthropic.New(
		anthropic.WithToken(apiKey),
		anthropic.WithModel(model),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Anthropic LLM: %w", err)
	}

	return &AnthropicProvider{
		llm:   llm,
		model: model,
	}, nil
}

// Generate generates a response for the given prompt
func (p *AnthropicProvider) Generate(ctx context.Context, prompt string, opts ...Option) (string, error) {
	cfg := applyOptions(opts...)
	_ = cfg // Will use in next task for streaming

	response, err := llms.GenerateFromSinglePrompt(ctx, p.llm, prompt)
	if err != nil {
		return "", fmt.Errorf("anthropic generation failed: %w", err)
	}
	return response, nil
}

// Name returns the provider name
func (p *AnthropicProvider) Name() string {
	return "anthropic"
}
