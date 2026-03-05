package processor

import (
	"context"
	"errors"
	"testing"

	"github.com/spencercjh/spec-forge/internal/enricher/prompt"
	"github.com/spencercjh/spec-forge/internal/enricher/provider"
)

// mockProvider for testing
type mockProvider struct {
	response     string
	responseFunc func() (string, error)
	err          error
	called       int
}

func (m *mockProvider) Generate(ctx context.Context, p string) (string, error) {
	m.called++
	if m.responseFunc != nil {
		return m.responseFunc()
	}
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

func (m *mockProvider) Name() string {
	return "mock"
}

var _ provider.Provider = (*mockProvider)(nil)

func TestBatchProcessor_ProcessBatch(t *testing.T) {
	mock := &mockProvider{
		response: `{"summary": "Get user by ID", "description": "Retrieves a user by their unique identifier"}`,
	}

	processor := NewBatchProcessor(mock, prompt.NewTemplateManager())

	var result string
	batch := &Batch{
		Type: prompt.TemplateTypeAPI,
		Elements: []EnrichmentElement{
			{
				Type: prompt.TemplateTypeAPI,
				Context: prompt.TemplateContext{
					Method: "GET",
					Path:   "/users/{id}",
				},
				SetValue: func(desc string) { result = desc },
			},
		},
	}

	err := processor.ProcessBatch(context.Background(), batch)
	if err != nil {
		t.Fatalf("ProcessBatch() error = %v", err)
	}

	if mock.called != 1 {
		t.Errorf("provider called %d times, want 1", mock.called)
	}

	// The SetValue should have been called with the full response
	if result == "" {
		t.Error("result should not be empty")
	}
}

func TestBatchProcessor_ProcessBatch_ProviderError(t *testing.T) {
	mock := &mockProvider{
		err: errors.New("API error"),
	}

	processor := NewBatchProcessor(mock, prompt.NewTemplateManager())

	batch := &Batch{
		Type: prompt.TemplateTypeAPI,
		Elements: []EnrichmentElement{
			{
				Type:     prompt.TemplateTypeAPI,
				SetValue: func(desc string) {},
			},
		},
	}

	err := processor.ProcessBatch(context.Background(), batch)
	if err == nil {
		t.Fatal("expected error for provider failure")
	}
}

func TestBatchProcessor_ProcessBatch_InvalidJSON(t *testing.T) {
	mock := &mockProvider{
		response: "not valid json",
	}

	processor := NewBatchProcessor(mock, prompt.NewTemplateManager())

	batch := &Batch{
		Type: prompt.TemplateTypeAPI,
		Elements: []EnrichmentElement{
			{
				Type:     prompt.TemplateTypeAPI,
				SetValue: func(desc string) {},
			},
		},
	}

	// Should not fail - invalid JSON means we use the raw response
	err := processor.ProcessBatch(context.Background(), batch)
	if err != nil {
		t.Fatalf("ProcessBatch() should handle invalid JSON, got error: %v", err)
	}
}

func TestConcurrentProcessor_ProcessAll(t *testing.T) {
	mock := &mockProvider{
		response: `{"description": "test"}`,
	}

	batchProcessor := NewBatchProcessor(mock, prompt.NewTemplateManager())
	concurrent := NewConcurrentProcessor(batchProcessor, 2)

	batches := []*Batch{
		{Type: prompt.TemplateTypeAPI, Elements: []EnrichmentElement{
			{Type: prompt.TemplateTypeAPI, SetValue: func(desc string) {}},
		}},
		{Type: prompt.TemplateTypeSchema, Elements: []EnrichmentElement{
			{Type: prompt.TemplateTypeSchema, SetValue: func(desc string) {}},
		}},
	}

	err := concurrent.ProcessAll(context.Background(), batches)
	if err != nil {
		t.Fatalf("ProcessAll() error = %v", err)
	}

	if mock.called != 2 {
		t.Errorf("provider called %d times, want 2", mock.called)
	}
}

func TestConcurrentProcessor_ProcessAll_PartialFailure(t *testing.T) {
	callCount := 0
	mock := &mockProvider{
		responseFunc: func() (string, error) {
			callCount++
			if callCount == 1 {
				return "", errors.New("first call fails")
			}
			return `{"description": "test"}`, nil
		},
	}

	batchProcessor := NewBatchProcessor(mock, prompt.NewTemplateManager())
	concurrent := NewConcurrentProcessor(batchProcessor, 1)

	batches := []*Batch{
		{Type: prompt.TemplateTypeAPI, Elements: []EnrichmentElement{
			{Type: prompt.TemplateTypeAPI, SetValue: func(desc string) {}},
		}},
		{Type: prompt.TemplateTypeSchema, Elements: []EnrichmentElement{
			{Type: prompt.TemplateTypeSchema, SetValue: func(desc string) {}},
		}},
	}

	err := concurrent.ProcessAll(context.Background(), batches)
	if err == nil {
		t.Fatal("expected PartialEnrichmentError")
	}

	// Should continue processing despite failure
	if callCount < 2 {
		t.Errorf("should have continued after first failure, callCount = %d", callCount)
	}
}

func TestParseResponse_JSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantDesc string
	}{
		{
			name:     "with description",
			input:    `{"description": "test description"}`,
			wantDesc: "test description",
		},
		{
			name:     "with summary and description",
			input:    `{"summary": "test summary", "description": "full description"}`,
			wantDesc: "test summary\n\nfull description",
		},
		{
			name:     "plain text",
			input:    "plain text response",
			wantDesc: "plain text response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseDescriptionResponse(tt.input)
			if result != tt.wantDesc {
				t.Errorf("parseDescriptionResponse() = %q, want %q", result, tt.wantDesc)
			}
		})
	}
}
