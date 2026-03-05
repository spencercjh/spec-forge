package provider

import (
	"errors"

	"github.com/spencercjh/spec-forge/internal/enricher/errors"
)

// NewProvider creates a provider based on the provider type
func NewProvider(cfg ProviderConfig) (Provider, error) {
	switch cfg.Provider {
	case "openai":
		return NewOpenAIProvider(cfg.APIKey, cfg.Model)
	case "anthropic":
		return NewAnthropicProvider(cfg.APIKey, cfg.Model)
	case "ollama":
		baseURL := cfg.BaseURL
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
		return NewOllamaProvider(baseURL, cfg.Model)
	case "custom":
		return NewCustomProvider(CustomProviderConfig{
			BaseURL: cfg.BaseURL,
			APIKey:  cfg.APIKey,
			Model:  cfg.Model,
			Name:    cfg.Name,
		})
	default:
		return nil, enricher.NewUnsupportedProviderError(cfg.Provider)
	}
}

// NewProvider creates a provider based on the provider type with direct parameter mapping
// This avoids type alias issues between provider.Config and enricher.Config
func NewProviderDirect(provider string, model string, language string, baseURL string, apiKey string, customHeaders map[string]string, name string) (Provider, error) {
	switch provider {
	case "openai":
		return NewOpenAIProvider(apiKey, model)
	case "anthropic":
		return NewAnthropicProvider(apiKey, model)
	case "ollama":
		return NewOllamaProvider(baseURL, model)
	case "custom":
		return NewCustomProvider(CustomProviderConfig{
			BaseURL: baseURL,
			APIKey:  apiKey,
			Model:   model,
			Name:    name,
		})
	default:
		return nil, enricher.NewUnsupportedProviderError(provider)
	}
}
