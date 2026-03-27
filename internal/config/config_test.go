package config

import (
	"testing"

	"github.com/spf13/viper"
)

// Note: These tests intentionally do NOT use t.Parallel() because viper is a
// global singleton. Parallel tests would race on viper.Set()/viper.Reset()
// calls, causing intermittent failures. Cleanup functions ensure tests leave
// state as they found it.

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

func TestGet_ReturnsGlobalWhenSet(t *testing.T) {
	// Save and restore global state
	prev := global
	t.Cleanup(func() { global = prev })

	// Set global directly
	expected := &Config{
		Output: OutputConfig{
			Dir:    "/custom/dir",
			Format: "json",
		},
	}
	global = expected

	got := Get()
	if got != expected {
		t.Errorf("Get() = %v, want %v", got, expected)
	}
	if got.Output.Dir != "/custom/dir" {
		t.Errorf("Get().Output.Dir = %q, want %q", got.Output.Dir, "/custom/dir")
	}
}

func TestGet_CallsLoadWhenGlobalNil(t *testing.T) {
	// Save and restore global and viper state
	prevGlobal := global
	t.Cleanup(func() {
		global = prevGlobal
		viper.Reset()
	})

	// Reset global to nil to trigger Load()
	global = nil
	viper.Reset()

	// Set viper value that Load() should pick up
	viper.Set("format", "json")

	got := Get()
	// Load() should have been called and picked up the format override
	if got.Output.Format != "json" {
		t.Errorf("Get().Output.Format = %q, want %q", got.Output.Format, "json")
	}
	// global should now be set
	if global == nil {
		t.Error("global should be set after Get() called Load()")
	}
}

func TestLoad_OverridesWithViperSettings(t *testing.T) {
	// Save and restore global and viper state
	prevGlobal := global
	t.Cleanup(func() {
		global = prevGlobal
		viper.Reset()
	})

	// Reset state
	global = nil
	viper.Reset()

	// Set viper values that Load() should override
	viper.Set("llm_api_key", "test-api-key-123")
	viper.Set("output", "/output/path")
	viper.Set("format", "json")

	cfg := Load()

	// Verify viper overrides were applied
	if cfg.Enrich.APIKey != "test-api-key-123" {
		t.Errorf("Load().Enrich.APIKey = %q, want %q", cfg.Enrich.APIKey, "test-api-key-123")
	}
	if cfg.Output.Dir != "/output/path" {
		t.Errorf("Load().Output.Dir = %q, want %q", cfg.Output.Dir, "/output/path")
	}
	if cfg.Output.Format != "json" {
		t.Errorf("Load().Output.Format = %q, want %q", cfg.Output.Format, "json")
	}

	// Verify global was set
	if global != cfg {
		t.Error("Load() should set global to returned config")
	}
}

func TestLoad_ReturnsDefaultsWithoutViperSettings(t *testing.T) {
	// Save and restore global and viper state
	prevGlobal := global
	t.Cleanup(func() {
		global = prevGlobal
		viper.Reset()
	})

	// Reset state
	global = nil
	viper.Reset()

	cfg := Load()

	// Should return defaults (no viper overrides)
	if cfg.Enrich.APIKey != "" {
		t.Errorf("Load().Enrich.APIKey = %q, want empty", cfg.Enrich.APIKey)
	}
	if cfg.Output.Dir != "" {
		t.Errorf("Load().Output.Dir = %q, want empty", cfg.Output.Dir)
	}
	if cfg.Output.Format != "yaml" {
		t.Errorf("Load().Output.Format = %q, want %q", cfg.Output.Format, "yaml")
	}
}
