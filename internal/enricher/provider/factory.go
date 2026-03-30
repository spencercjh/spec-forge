package provider

import "context"

// TokenUsage represents token consumption for a single LLM call.
type TokenUsage struct {
	InputTokens  int
	OutputTokens int
}

// Add returns the sum of two TokenUsage values.
func (u *TokenUsage) Add(other *TokenUsage) {
	if other == nil {
		return
	}
	u.InputTokens += other.InputTokens
	u.OutputTokens += other.OutputTokens
}

// Total returns the total number of tokens.
func (u *TokenUsage) Total() int {
	return u.InputTokens + u.OutputTokens
}

// Provider defines the interface for LLM providers
type Provider interface {
	// Generate generates a response for the given prompt
	Generate(ctx context.Context, prompt string, opts ...Option) (string, *TokenUsage, error)
	// Name returns the provider name
	Name() string
}

// GenerateOptions contains options for generation
type GenerateOptions struct {
	StreamingFunc func(ctx context.Context, chunk []byte) error
}

// Option is a functional option for Generate
type Option func(*GenerateOptions)

// WithStreamingFunc sets a streaming callback function
func WithStreamingFunc(fn func(ctx context.Context, chunk []byte) error) Option {
	return func(o *GenerateOptions) {
		o.StreamingFunc = fn
	}
}

// applyOptions applies options and returns the configured GenerateOptions
func applyOptions(opts ...Option) *GenerateOptions {
	cfg := &GenerateOptions{}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// Config contains configuration for creating a provider
// revive:disable-next-line:exported // keeping Config for clarity as provider.Config would be ambiguous
type Config struct {
	Provider string
	Model    string
	APIKey   string //nolint:gosec // Configuration field, not the actual secret
	BaseURL  string
	Name     string
}

// UnsupportedProviderError is returned when an unsupported provider is requested
type UnsupportedProviderError struct {
	Provider string
}

// Error implements the error interface
func (e *UnsupportedProviderError) Error() string {
	return "unsupported provider: " + e.Provider
}

// NewUnsupportedProviderError creates a new UnsupportedProviderError
func NewUnsupportedProviderError(provider string) *UnsupportedProviderError {
	return &UnsupportedProviderError{Provider: provider}
}

// NewProvider creates a provider based on the provider type
func NewProvider(cfg Config) (Provider, error) { //nolint:gocritic // copying config is acceptable
	switch cfg.Provider {
	case OpenAIProviderName:
		return newOpenAIProvider(cfg.APIKey, cfg.Model)
	case AnthropicProviderName:
		return newAnthropicProvider(cfg.APIKey, cfg.Model)
	case OllamaProviderName:
		baseURL := cfg.BaseURL
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
		return newOllamaProvider(baseURL, cfg.Model)
	case CustomProviderName:
		return newCustomProvider(CustomProviderConfig{
			BaseURL: cfg.BaseURL,
			APIKey:  cfg.APIKey,
			Model:   cfg.Model,
			Name:    cfg.Name,
		})
	default:
		return nil, NewUnsupportedProviderError(cfg.Provider)
	}
}
