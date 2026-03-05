package enricher

import (
	"testing"
	"time"
)

func TestConfig_Defaults(t *testing.T) {
	cfg := DefaultConfig

	if cfg.Language != "en" {
		t.Errorf("Language = %q, want %q", cfg.Language, "en")
	}
	if cfg.Concurrency != 3 {
		t.Errorf("Concurrency = %d, want %d", cfg.Concurrency, 3)
	}
	if cfg.MaxRetries != 2 {
		t.Errorf("MaxRetries = %d, want %d", cfg.MaxRetries, 2)
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want %v", cfg.Timeout, 30*time.Second)
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name:    "valid openai config",
			config:  Config{Provider: "openai", Model: "gpt-4o"},
			wantErr: false,
		},
		{
			name:    "valid anthropic config",
			config:  Config{Provider: "anthropic", Model: "claude-3-opus"},
			wantErr: false,
		},
		{
			name:    "valid ollama config",
			config:  Config{Provider: "ollama", Model: "llama3", CustomBaseURL: "http://localhost:11434/v1"},
			wantErr: false,
		},
		{
			name:    "valid custom config",
			config:  Config{Provider: "custom", Model: "internal-model", CustomBaseURL: "https://ai.company.com/v1"},
			wantErr: false,
		},
		{
			name:    "missing provider",
			config:  Config{Model: "gpt-4o"},
			wantErr: true,
		},
		{
			name:    "missing model",
			config:  Config{Provider: "openai"},
			wantErr: true,
		},
		{
			name:    "custom without base url",
			config:  Config{Provider: "custom", Model: "model"},
			wantErr: true,
		},
		{
			name:    "invalid concurrency negative",
			config:  Config{Provider: "openai", Model: "gpt-4o", Concurrency: -1},
			wantErr: true,
		},
		{
			name:    "invalid timeout negative",
			config:  Config{Provider: "openai", Model: "gpt-4o", Timeout: -1 * time.Second},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_MergeWithDefaults(t *testing.T) {
	cfg := Config{
		Provider: "openai",
		Model:    "gpt-4o",
	}

	merged := cfg.MergeWithDefaults()

	if merged.Language != DefaultConfig.Language {
		t.Errorf("Language should use default")
	}
	if merged.Concurrency != DefaultConfig.Concurrency {
		t.Errorf("Concurrency should use default")
	}
	if merged.Timeout != DefaultConfig.Timeout {
		t.Errorf("Timeout should use default")
	}
	// Original values should be preserved
	if merged.Provider != "openai" {
		t.Errorf("Provider should be preserved")
	}
	if merged.Model != "gpt-4o" {
		t.Errorf("Model should be preserved")
	}
}

func TestConfig_APITKeyEnv(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected string
	}{
		{
			name:     "openai uses OPENAI_API_KEY",
			config:   Config{Provider: "openai"},
			expected: "OPENAI_API_KEY",
		},
		{
			name:     "anthropic uses ANTHROPIC_API_KEY",
			config:   Config{Provider: "anthropic"},
			expected: "ANTHROPIC_API_KEY",
		},
		{
			name:     "custom uses CustomAPIKeyEnv",
			config:   Config{Provider: "custom", CustomAPIKeyEnv: "MY_API_KEY"},
			expected: "MY_API_KEY",
		},
		{
			name:     "custom default to LLM_API_KEY",
			config:   Config{Provider: "custom"},
			expected: "LLM_API_KEY",
		},
		{
			name:     "ollama has no api key",
			config:   Config{Provider: "ollama"},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.GetAPIKeyEnv(); got != tt.expected {
				t.Errorf("GetAPIKeyEnv() = %q, want %q", got, tt.expected)
			}
		})
	}
}
