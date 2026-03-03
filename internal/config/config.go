// Package config handles configuration loading and management.
package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config represents the complete configuration for spec-forge.
type Config struct {
	Enrich  EnrichConfig  `mapstructure:"enrich"`
	Output  OutputConfig  `mapstructure:"output"`
	Extract ExtractConfig `mapstructure:"extract"`
}

// EnrichConfig contains LLM enrichment settings.
type EnrichConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Provider string `mapstructure:"provider"`
	Model    string `mapstructure:"model"`
	APIKey   string `mapstructure:"apiKey"`
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
			Dir:    "./openapi",
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
		fmt.Printf("warning: failed to unmarshal config: %v\n", err)
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
