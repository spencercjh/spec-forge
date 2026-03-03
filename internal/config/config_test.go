package config

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := Default()
	if cfg.Output.Dir != "./openapi" {
		t.Errorf("expected default output dir ./openapi, got %s", cfg.Output.Dir)
	}
	if cfg.Output.Format != "yaml" {
		t.Errorf("expected default format yaml, got %s", cfg.Output.Format)
	}
}
