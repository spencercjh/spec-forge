package config

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := Default()
	// Empty Dir means CLI defaults to project root unless overridden by config or flags
	if cfg.Output.Dir != "" {
		t.Errorf("expected default output dir to be empty (defer to CLI/project-root default), got %s", cfg.Output.Dir)
	}
	if cfg.Output.Format != "yaml" {
		t.Errorf("expected default format yaml, got %s", cfg.Output.Format)
	}
}
