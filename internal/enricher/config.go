package enricher

import (
	"errors"
	"time"
)

// CustomPromptConfig holds custom prompt overrides for a template type.
type CustomPromptConfig struct {
	System string
	User   string
}

// Config Enricher configuration
type Config struct {
	// Provider type: "openai", "anthropic", "ollama", "custom"
	Provider string

	// Common configuration
	Model    string
	Language string
	// Number of concurrent LLM calls. Only effective with --no-stream;
	// streaming mode processes batches sequentially for readable output.
	Concurrency int
	MaxRetries  int
	Timeout     time.Duration

	// Project path for context extraction (used by specctx extractors)
	ProjectPath string

	// Custom provider configuration (used when Provider == "custom")
	CustomBaseURL   string
	CustomAPIKeyEnv string
	CustomHeaders   map[string]string

	// Advanced configuration
	PromptTemplateDir string

	// Custom prompt overrides keyed by template type name (e.g., "api", "schema")
	CustomPrompts map[string]CustomPromptConfig
}

// DefaultConfig provides sensible defaults
var DefaultConfig = Config{
	Language:    "en",
	Concurrency: 3,
	MaxRetries:  2,
	Timeout:     30 * time.Second,
}

// Validate validates the configuration
// Note: This validates after merging with defaults, so zero values for optional fields are allowed
func (c *Config) Validate() error {
	if c.Provider == "" {
		return errors.New("provider is required")
	}
	if c.Model == "" {
		return errors.New("model is required")
	}
	if c.Provider == "custom" && c.CustomBaseURL == "" {
		return errors.New("customBaseURL is required for custom provider")
	}
	// Concurrency and Timeout zero values are handled by MergeWithDefaults
	// Only reject explicitly invalid values (negative)
	if c.Concurrency < 0 {
		return errors.New("concurrency cannot be negative")
	}
	if c.Timeout < 0 {
		return errors.New("timeout cannot be negative")
	}
	return nil
}

// MergeWithDefaults merges the config with defaults for zero values
func (c Config) MergeWithDefaults() Config { //nolint:gocritic // receiver by design
	if c.Language == "" {
		c.Language = DefaultConfig.Language
	}
	if c.Concurrency == 0 {
		c.Concurrency = DefaultConfig.Concurrency
	}
	if c.MaxRetries == 0 {
		c.MaxRetries = DefaultConfig.MaxRetries
	}
	if c.Timeout == 0 {
		c.Timeout = DefaultConfig.Timeout
	}
	return c
}

// GetAPIKeyEnv returns the environment variable name for the API key
func (c *Config) GetAPIKeyEnv() string {
	switch c.Provider {
	case "openai":
		return "OPENAI_API_KEY"
	case "anthropic":
		return "ANTHROPIC_API_KEY"
	case "custom":
		if c.CustomAPIKeyEnv != "" {
			return c.CustomAPIKeyEnv
		}
		return "LLM_API_KEY"
	case "ollama":
		return "" // Ollama doesn't require an API key
	default:
		return "LLM_API_KEY"
	}
}
