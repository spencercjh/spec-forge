package config

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := Default()
	// Empty Dir means use framework-specific default
	if cfg.Output.Dir != "" {
		t.Errorf("expected default output dir to be empty (framework default), got %s", cfg.Output.Dir)
	}
	if cfg.Output.Format != "yaml" {
		t.Errorf("expected default format yaml, got %s", cfg.Output.Format)
	}
}
