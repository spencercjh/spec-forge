package provider

import "context"

// Provider defines the interface for LLM providers
type Provider interface {
	// Generate generates a response for the given prompt
	Generate(ctx context.Context, prompt string) (string, error)
	// Name returns the provider name
	Name() string
}
