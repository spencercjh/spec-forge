package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Default timeout for API calls
const defaultTimeout = 30 * time.Second

// OpenAICompatibleConfig configuration for OpenAI-compatible services
type OpenAICompatibleConfig struct {
	BaseURL      string
	APIKey       string
	Model        string
	ExtraHeaders map[string]string
	Timeout      time.Duration
	Name         string // provider name for display
}

// OpenAICompatibleProvider implements Provider for OpenAI-compatible APIs
type OpenAICompatibleProvider struct {
	config OpenAICompatibleConfig
	client *http.Client
}

// NewOpenAICompatibleProvider creates a new OpenAI-compatible provider
func NewOpenAICompatibleProvider(config OpenAICompatibleConfig) *OpenAICompatibleProvider {
	timeout := config.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	if config.Name == "" {
		config.Name = "openai-compatible"
	}

	return &OpenAICompatibleProvider{
		config: config,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// chatCompletionRequest represents the request body for chat completion API
type chatCompletionRequest struct {
	Model    string          `json:"model"`
	Messages []chatMessage   `json:"messages"`
	Stream   bool            `json:"stream"`
}

// chatMessage represents a single message in the conversation
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatCompletionResponse represents the response from chat completion API
type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Generate generates a response for the given prompt
func (p *OpenAICompatibleProvider) Generate(ctx context.Context, prompt string) (string, error) {
	reqBody := chatCompletionRequest{
		Model: p.config.Model,
		Messages: []chatMessage{
			{Role: "user", Content: prompt},
		},
		Stream: false,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := p.config.BaseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

	// Add extra headers
	for key, value := range p.config.ExtraHeaders {
		req.Header.Set(key, value)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var chatResp chatCompletionResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for API error
	if chatResp.Error != nil {
		return "", fmt.Errorf("API error: %s", chatResp.Error.Message)
	}

	// Check for non-200 status
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Validate response
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// Name returns the provider name
func (p *OpenAICompatibleProvider) Name() string {
	return p.config.Name
}

// WithName sets the provider name
func (c OpenAICompatibleConfig) WithName(name string) OpenAICompatibleConfig {
	c.Name = name
	return c
}
