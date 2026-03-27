package provider

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

// OllamaProviderName is the name of the Ollama provider.
const OllamaProviderName = "ollama"

// OllamaProvider implements Provider for Ollama local LLM using langchaingo
type OllamaProvider struct {
	llm     llms.Model
	model   string
	baseURL string
}

// newOllamaProvider creates a provider configured for Ollama
func newOllamaProvider(baseURL, model string) (*OllamaProvider, error) {
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

// Generate generates a response for the given prompt with optional streaming
func (p *OllamaProvider) Generate(ctx context.Context, prompt string, opts ...Option) (string, error) {
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
		return "", fmt.Errorf("%s generation failed: %w", OllamaProviderName, err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("%s generation returned no choices", OllamaProviderName)
	}
	return response.Choices[0].Content, nil
}

// Name returns the provider name
func (p *OllamaProvider) Name() string {
	return OllamaProviderName
}
