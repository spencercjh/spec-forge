package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// CustomProviderName is the default name of the custom provider.
const CustomProviderName = "custom"

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
	APIKey  string //nolint:gosec // Configuration field, not the actual secret
	Model   string
	Name    string // optional custom name
}

// newCustomProvider creates a provider for custom OpenAI-compatible services
func newCustomProvider(cfg CustomProviderConfig) (*CustomProvider, error) {
	if cfg.BaseURL == "" {
		return nil, errors.New("baseURL is required for custom provider")
	}
	if cfg.APIKey == "" {
		return nil, errors.New("apiKey is required for custom provider")
	}

	name := cfg.Name
	if name == "" {
		name = CustomProviderName
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

// Generate generates a response with optional streaming
func (p *CustomProvider) Generate(ctx context.Context, prompt string, opts ...Option) (string, error) {
	cfg := applyOptions(opts...)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	var callOpts []llms.CallOption
	if cfg.StreamingFunc != nil {
		callOpts = append(callOpts, llms.WithStreamingFunc(cfg.StreamingFunc))
	}

	response, err := p.llm.GenerateContent(ctx, messages, callOpts...)
	if err != nil {
		return "", fmt.Errorf("%s provider generation failed: %w", CustomProviderName, err)
	}

	if len(response.Choices) == 0 {
		return "", errors.New("custom provider generation returned no choices")
	}
	return response.Choices[0].Content, nil
}

// Name returns the provider name
func (p *CustomProvider) Name() string {
	return p.name
}
