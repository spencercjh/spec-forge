// Package config handles configuration loading and management.
package config

import (
	"log/slog"

	"github.com/spf13/viper"
)

// Config represents the complete configuration for spec-forge.
type Config struct {
	Enrich  EnrichConfig  `mapstructure:"enrich"`
	Output  OutputConfig  `mapstructure:"output"`
	Extract ExtractConfig `mapstructure:"extract"`
	ReadMe  ReadMeConfig  `mapstructure:"readme"`
}

// ReadMeConfig contains ReadMe publishing settings.
type ReadMeConfig struct {
	APIKey         string `mapstructure:"apiKey"`
	APIKeyEnv      string `mapstructure:"apiKeyEnv"`
	Branch         string `mapstructure:"branch"`
	Slug           string `mapstructure:"slug"`
	UseSpecVersion bool   `mapstructure:"useSpecVersion"`
}

// EnrichConfig contains LLM enrichment settings.
type EnrichConfig struct {
	Enabled       bool                       `mapstructure:"enabled"`
	Provider      string                     `mapstructure:"provider"`
	Model         string                     `mapstructure:"model"`
	Language      string                     `mapstructure:"language"`
	APIKey        string                     `mapstructure:"apiKey"`
	Headers       map[string]string          `mapstructure:"headers"`
	BaseURL       string                     `mapstructure:"baseUrl"`
	APIKeyEnv     string                     `mapstructure:"apiKeyEnv"`
	Timeout       string                     `mapstructure:"timeout"`
	SkipEnrich    bool                       `mapstructure:"skipEnrich"`
	CustomPrompts map[string]CustomPromptCfg `mapstructure:"customPrompts"`
}

// CustomPromptCfg holds custom system/user prompt overrides for a template type.
type CustomPromptCfg struct {
	System string `mapstructure:"system"`
	User   string `mapstructure:"user"`
}

// OutputConfig contains output settings.
type OutputConfig struct {
	Dir    string `mapstructure:"dir"`
	Format string `mapstructure:"format"`
}

// ExtractConfig contains extraction settings.
type ExtractConfig struct {
	Strict bool `mapstructure:"strict"`
}

// global is the global configuration instance.
var global *Config

// Default returns the default configuration.
func Default() *Config {
	return &Config{
		Enrich: EnrichConfig{
			Enabled: true,
		},
		Output: OutputConfig{
			Dir:    "", // Empty means CLI defaults to project root unless overridden by config output.dir or --output-dir flag
			Format: "yaml",
		},
		Extract: ExtractConfig{
			Strict: false,
		},
	}
}

// Load loads configuration from viper and returns the global config.
func Load() *Config {
	cfg := Default()

	// Unmarshal from viper
	if err := viper.Unmarshal(cfg); err != nil {
		slog.Warn("failed to unmarshal config, using defaults", "error", err)
	}

	// Override with environment variables
	if apiKey := viper.GetString("llm_api_key"); apiKey != "" {
		cfg.Enrich.APIKey = apiKey
	}

	// Override with flags
	if dir := viper.GetString("output"); dir != "" {
		cfg.Output.Dir = dir
	}
	if format := viper.GetString("format"); format != "" {
		cfg.Output.Format = format
	}

	global = cfg
	return cfg
}

// Get returns the global configuration.
func Get() *Config {
	if global == nil {
		return Load()
	}
	return global
}
