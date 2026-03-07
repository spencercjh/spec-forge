package provider

import "context"

// Provider defines the interface for LLM providers
type Provider interface {
	// Generate generates a response for the given prompt
	Generate(ctx context.Context, prompt string) (string, error)
	// Name returns the provider name
	Name() string
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
	case "anthropic":
		return newAnthropicProvider(cfg.APIKey, cfg.Model)
	case OllamaProviderName:
		baseURL := cfg.BaseURL
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
		return newOllamaProvider(baseURL, cfg.Model)
	case "custom":
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
