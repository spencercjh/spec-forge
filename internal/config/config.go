// Package config handles configuration loading for spec-forge.
package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config holds the application configuration.
type Config struct {
	Verbose bool `mapstructure:"verbose"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Verbose: false,
	}
}

// Load reads configuration from viper and returns a Config struct.
func Load() *Config {
	cfg := DefaultConfig()

	// Bind verbose flag
	if viper.IsSet("verbose") {
		cfg.Verbose = viper.GetBool("verbose")
	}

	if cfg.Verbose {
		fmt.Println("Configuration loaded successfully")
	}

	return cfg
}
