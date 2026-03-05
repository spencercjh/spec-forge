package provider

// NewOpenAIProvider creates a provider configured for OpenAI
func NewOpenAIProvider(apiKey, model string) *OpenAICompatibleProvider {
	return NewOpenAICompatibleProvider(OpenAICompatibleConfig{
		BaseURL: "https://api.openai.com/v1",
		APIKey:  apiKey,
		Model:   model,
	}.WithName("openai"))
}

// NewAnthropicProvider creates a provider configured for Anthropic
func NewAnthropicProvider(apiKey, model string) *OpenAICompatibleProvider {
	return NewOpenAICompatibleProvider(OpenAICompatibleConfig{
		BaseURL: "https://api.anthropic.com/v1",
		APIKey:  apiKey,
		Model:   model,
	}.WithName("anthropic"))
}

// NewOllamaProvider creates a provider configured for Ollama
func NewOllamaProvider(baseURL, model string) *OpenAICompatibleProvider {
	return NewOpenAICompatibleProvider(OpenAICompatibleConfig{
		BaseURL: baseURL,
		APIKey:  "ollama", // Ollama doesn't need a real key
		Model:   model,
	}.WithName("ollama"))
}

// NewCustomProvider creates a provider for custom OpenAI-compatible services
func NewCustomProvider(baseURL, apiKey, model string, headers map[string]string) *OpenAICompatibleProvider {
	return NewOpenAICompatibleProvider(OpenAICompatibleConfig{
		BaseURL:      baseURL,
		APIKey:       apiKey,
		Model:        model,
		ExtraHeaders: headers,
	}.WithName("custom"))
}
