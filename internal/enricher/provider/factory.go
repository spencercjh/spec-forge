package provider

// ProviderConfig contains configuration for creating a provider
// revive:disable-next-line:exported // keeping ProviderConfig for clarity as provider.Config would be ambiguous
type ProviderConfig struct {
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
func NewProvider(cfg ProviderConfig) (Provider, error) { //nolint:gocritic // copying config is acceptable
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
			Model:   cfg.Model,
			Name:    cfg.Name,
		})
	default:
		return nil, NewUnsupportedProviderError(cfg.Provider)
	}
}

// NewProviderDirect creates a provider based on the provider type with direct parameter mapping
// This avoids type alias issues between provider.Config and enricher.Config
func NewProviderDirect(provider, model, _, baseURL, apiKey string, _ map[string]string, name string) (Provider, error) {
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
		return nil, NewUnsupportedProviderError(provider)
	}
}
