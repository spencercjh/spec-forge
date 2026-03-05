package provider

// ProviderConfig contains common configuration for creating providers
type ProviderConfig struct {
	Provider string
	Model    string
	APIKey   string
	BaseURL  string
	Name     string
}

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
			Model:   cfg.Model,
			Name:    cfg.Name,
		})
	default:
		return nil, &UnsupportedProviderError{Provider: cfg.Provider}
	}
}

// UnsupportedProviderError is returned when an unknown provider is requested
type UnsupportedProviderError struct {
	Provider string
}

func (e *UnsupportedProviderError) Error() string {
	return "unsupported provider: " + e.Provider
}
