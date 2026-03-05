package enricher

import (
	"errors"
	"fmt"
	"testing"
)

func TestEnrichmentError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *EnrichmentError
		expected string
	}{
		{
			name: "with cause",
			err: &EnrichmentError{
				Type:    "llm_call",
				Message: "failed to call LLM",
				Cause:   errors.New("connection refused"),
			},
			expected: "enrichment error [llm_call]: failed to call LLM: connection refused",
		},
		{
			name: "without cause",
			err: &EnrichmentError{
				Type:    "config",
				Message: "API key not found",
				Cause:   nil,
			},
			expected: "enrichment error [config]: API key not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestEnrichmentError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := &EnrichmentError{
		Type:    "test",
		Message: "test message",
		Cause:   cause,
	}

	if !errors.Is(err, cause) {
		t.Error("errors.Is should match the cause")
	}
}

func TestNewEnrichmentError(t *testing.T) {
	cause := errors.New("cause")
	err := NewEnrichmentError("llm_call", "test message", cause)

	if err.Type != "llm_call" {
		t.Errorf("Type = %q, want %q", err.Type, "llm_call")
	}
	if err.Message != "test message" {
		t.Errorf("Message = %q, want %q", err.Message, "test message")
	}
	if !errors.Is(err, cause) {
		t.Error("Cause should match")
	}
}

func TestIsConfigError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "config error",
			err:      NewEnrichmentError("config", "no api key", nil),
			expected: true,
		},
		{
			name:     "llm_call error",
			err:      NewEnrichmentError("llm_call", "timeout", nil),
			expected: false,
		},
		{
			name:     "other error",
			err:      errors.New("random error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsConfigError(tt.err); got != tt.expected {
				t.Errorf("IsConfigError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestErrors_ErrorTypes(t *testing.T) {
	// Verify error type constants
	if ErrorTypeConfig != "config" {
		t.Errorf("ErrorTypeConfig = %q, want %q", ErrorTypeConfig, "config")
	}
	if ErrorTypeLLMCall != "llm_call" {
		t.Errorf("ErrorTypeLLMCall = %q, want %q", ErrorTypeLLMCall, "llm_call")
	}
	if ErrorTypeParse != "parse" {
		t.Errorf("ErrorTypeParse = %q, want %q", ErrorTypeParse, "parse")
	}
	if ErrorTypeTemplate != "template" {
		t.Errorf("ErrorTypeTemplate = %q, want %q", ErrorTypeTemplate, "template")
	}
}

// Example demonstrates creating and checking errors
func ExampleEnrichmentError() {
	err := NewEnrichmentError("config", "API key not configured", nil)
	fmt.Println(err.Error())
	// Output: enrichment error [config]: API key not configured
}
