package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// OpenAIProviderName is the name of the OpenAI provider.
const OpenAIProviderName = "openai"

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

// Generate generates a response for the given prompt with optional streaming
func (p *OpenAIProvider) Generate(ctx context.Context, prompt string, opts ...Option) (string, error) {
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
		return "", fmt.Errorf("OpenAI generation failed: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", errors.New("OpenAI generation returned no choices")
	}
	return response.Choices[0].Content, nil
}

// Name returns the provider name
func (p *OpenAIProvider) Name() string {
	return OpenAIProviderName
}
