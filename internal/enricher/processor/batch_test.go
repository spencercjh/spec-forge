package processor

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/spencercjh/spec-forge/internal/enricher/prompt"
	"github.com/spencercjh/spec-forge/internal/enricher/provider"
)

// mockProvider for testing
type mockProvider struct {
	response     string
	responseFunc func() (string, *TokenUsage, error)
	err          error
	called       atomic.Int32
}

func (m *mockProvider) Generate(ctx context.Context, p string, opts ...provider.Option) (string, *TokenUsage, error) {
	m.called.Add(1)
	if m.responseFunc != nil {
		return m.responseFunc()
	}
	if m.err != nil {
		return "", nil, m.err
	}
	return m.response, nil, nil
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

	if mock.called.Load() != 1 {
		t.Errorf("provider called %d times, want 1", mock.called.Load())
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

	if mock.called.Load() != 2 {
		t.Errorf("provider called %d times, want 2", mock.called.Load())
	}
}

func TestConcurrentProcessor_ProcessAll_PartialFailure(t *testing.T) {
	callCount := 0
	mock := &mockProvider{
		responseFunc: func() (string, *TokenUsage, error) {
			callCount++
			if callCount == 1 {
				return "", nil, errors.New("first call fails")
			}
			return `{"description": "test"}`, nil, nil
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

func TestParseSchemaResponse(t *testing.T) {
	tests := []struct {
		name         string
		response     string
		fieldCount   int
		expectFields map[string]string
	}{
		{
			name: "simple JSON",
			response: `{
				"id": "The unique identifier of the user",
				"name": "The display name of the user",
				"email": "The user's email address"
			}`,
			fieldCount: 3,
			expectFields: map[string]string{
				"id":    "The unique identifier of the user",
				"name":  "The display name of the user",
				"email": "The user's email address",
			},
		},
		{
			name:       "JSON in markdown code block",
			response:   "```json\n{\"userId\": \"The user ID\", \"status\": \"The account status\"}\n```",
			fieldCount: 2,
			expectFields: map[string]string{
				"userId": "The user ID",
				"status": "The account status",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseSchemaResponse(tt.response)
			if len(result) != tt.fieldCount {
				t.Errorf("expected %d fields, got %d", tt.fieldCount, len(result))
			}
			for name, expectedDesc := range tt.expectFields {
				if result[name] != expectedDesc {
					t.Errorf("field %s: expected %q, got %q", name, expectedDesc, result[name])
				}
			}
		})
	}
}

// MockProvider for schema batch test
type MockProvider struct {
	GenerateFunc func(ctx context.Context, p string, opts ...provider.Option) (string, *TokenUsage, error)
}

func (m *MockProvider) Generate(ctx context.Context, p string, opts ...provider.Option) (string, *TokenUsage, error) {
	if m.GenerateFunc != nil {
		return m.GenerateFunc(ctx, p, opts...)
	}
	return "", nil, nil
}

func (m *MockProvider) Name() string {
	return "mock"
}

func TestBatchProcessor_ProcessSchemaBatch(t *testing.T) {
	mockProvider := &MockProvider{
		GenerateFunc: func(_ context.Context, prompt string, opts ...provider.Option) (string, error) {
			// Return mock schema field descriptions
			if strings.Contains(prompt, "User") {
				return `{"id": "User ID", "name": "User name"}`, nil
			}
			return "{}", nil
		},
	}

	tmplMgr := prompt.NewTemplateManager()
	bp := NewBatchProcessor(mockProvider, tmplMgr)

	fields := []FieldElement{
		{FieldName: "id", FieldType: "integer", Required: true},
		{FieldName: "name", FieldType: "string", Required: false},
	}

	var setValues []string
	for i := range fields {
		f := &fields[i]
		f.SetValue = func(desc string) {
			setValues = append(setValues, f.FieldName+":"+desc)
		}
	}

	batch := &Batch{
		Type: prompt.TemplateTypeSchema,
		Elements: []EnrichmentElement{
			{
				Type: prompt.TemplateTypeSchema,
				Path: "#/components/schemas/User",
				Context: prompt.TemplateContext{
					Type:       prompt.TemplateTypeSchema,
					Language:   "en",
					SchemaName: "User",
					Fields:     convertFieldElements(fields),
				},
				SchemaFields: fields,
			},
		},
	}

	ctx := context.Background()
	err := bp.ProcessBatch(ctx, batch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(setValues) != 2 {
		t.Errorf("expected 2 SetValue calls, got %d", len(setValues))
	}
}

// mockStreamingAwareProvider tracks if streaming options are passed
type mockStreamingAwareProvider struct {
	response        string
	streamingCalled bool
	streamingChunks [][]byte
}

func (m *mockStreamingAwareProvider) Generate(ctx context.Context, p string, opts ...provider.Option) (string, *TokenUsage, error) {
	cfg := &provider.GenerateOptions{}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.StreamingFunc != nil {
		m.streamingCalled = true
		// Simulate streaming chunks
		chunks := [][]byte{[]byte("chunk1"), []byte("chunk2")}
		for _, chunk := range chunks {
			m.streamingChunks = append(m.streamingChunks, chunk)
			if err := cfg.StreamingFunc(ctx, chunk); err != nil {
				return "", nil, err
			}
		}
	}
	return m.response, nil, nil
}

func (m *mockStreamingAwareProvider) Name() string {
	return "mock-streaming-aware"
}

func TestBatchProcessor_ProcessBatch_WithStreaming(t *testing.T) {
	mockProvider := &mockStreamingAwareProvider{
		response: `{"summary": "Get users", "description": "Retrieves user list"}`,
	}

	tmplMgr := prompt.NewTemplateManager()
	var buf strings.Builder
	sw := NewStreamWriter(&buf)

	bp := NewBatchProcessor(mockProvider, tmplMgr, WithStreamWriter(sw))

	batch := &Batch{
		Type: prompt.TemplateTypeAPI,
		Elements: []EnrichmentElement{
			{
				Type: prompt.TemplateTypeAPI,
				Context: prompt.TemplateContext{
					Method: "GET",
					Path:   "/users",
				},
				SetValue: func(desc string) {},
			},
		},
	}

	err := bp.ProcessBatch(context.Background(), batch)
	if err != nil {
		t.Fatalf("ProcessBatch() error = %v", err)
	}

	// Verify streaming was called
	if !mockProvider.streamingCalled {
		t.Error("Expected streaming function to be called when StreamWriter is configured")
	}

	// Verify chunks were streamed
	if len(mockProvider.streamingChunks) != 2 {
		t.Errorf("Expected 2 streaming chunks, got %d", len(mockProvider.streamingChunks))
	}

	// Flush the buffer before checking output (StreamWriter uses buffering)
	if err := sw.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	// Verify output contains the prefix
	output := buf.String()
	if !strings.Contains(output, "[api]") {
		t.Errorf("Expected output to contain '[api]' prefix, got: %s", output)
	}
}
