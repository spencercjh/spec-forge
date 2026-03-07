//go:build e2e

package e2e_test

import (
	"context"

	"github.com/spencercjh/spec-forge/internal/enricher/provider"
)

// countingMockProvider tracks call counts for verification.
type countingMockProvider struct {
	callCount int
	responses map[string]string
}

func (m *countingMockProvider) Generate(ctx context.Context, prompt string) (string, error) {
	m.callCount++
	if resp, ok := m.responses["default"]; ok {
		return resp, nil
	}
	return `{"description": "Mock response"}`, nil
}

func (m *countingMockProvider) Name() string {
	return "mock"
}

var _ provider.Provider = (*countingMockProvider)(nil)
