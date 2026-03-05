package provider

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// OpenAIProvider implements Provider for OpenAI using langchaingo
type OpenAIProvider struct {
	llm   llms.Model
	model string
}

// newOpenAIProvider creates a provider configured for OpenAI
func newOpenAIProvider(apiKey, model string) (*OpenAIProvider, error) {
	llm, err := openai.New(
		openai.WithToken(apiKey),
		openai.WithModel(model),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI LLM: %w", err)
	}

	return &OpenAIProvider{
		llm:   llm,
		model: model,
	}, nil
}

// Generate generates a response for the given prompt
func (p *OpenAIProvider) Generate(ctx context.Context, prompt string) (string, error) {
	response, err := llms.GenerateFromSinglePrompt(ctx, p.llm, prompt)
	if err != nil {
		return "", fmt.Errorf("OpenAI generation failed: %w", err)
	}
	return response, nil
}

// Name returns the provider name
func (p *OpenAIProvider) Name() string {
	return "openai"
}
