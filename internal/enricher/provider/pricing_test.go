package provider

import "testing"

func TestEstimateCost_KnownModel(t *testing.T) {
	usage := &TokenUsage{InputTokens: 1000, OutputTokens: 500}

	cost, ok := EstimateCost("openai", "gpt-4o", usage)
	if !ok {
		t.Fatal("expected ok=true for known model")
	}

	// gpt-4o: $2.50/1M input, $10.00/1M output
	// 1000 * 2.50/1M + 500 * 10.00/1M = 0.0025 + 0.005 = 0.0075
	if cost < 0.007 || cost > 0.008 {
		t.Errorf("cost = %f, want ~0.0075", cost)
	}
}

func TestEstimateCost_UnknownModel(t *testing.T) {
	usage := &TokenUsage{InputTokens: 1000, OutputTokens: 500}

	_, ok := EstimateCost("unknown-provider", "unknown-model", usage)
	if ok {
		t.Fatal("expected ok=false for unknown model")
	}
}

func TestEstimateCost_NilUsage(t *testing.T) {
	cost, ok := EstimateCost("openai", "gpt-4o", nil)
	if ok {
		t.Fatal("expected ok=false for nil usage")
	}
	if cost != 0 {
		t.Errorf("cost = %f, want 0 for nil usage", cost)
	}
}

func TestEstimateCost_DeepSeek(t *testing.T) {
	usage := &TokenUsage{InputTokens: 1000000, OutputTokens: 1000000}

	cost, ok := EstimateCost("custom", "deepseek-chat", usage)
	if !ok {
		t.Fatal("expected ok=true for deepseek-chat")
	}

	// deepseek-chat: $0.14/1M input, $0.28/1M output
	// 0.14 + 0.28 = 0.42
	if cost < 0.41 || cost > 0.43 {
		t.Errorf("cost = %f, want ~0.42", cost)
	}
}

func TestEstimateCost_Anthropic(t *testing.T) {
	usage := &TokenUsage{InputTokens: 1000000, OutputTokens: 1000000}

	cost, ok := EstimateCost("anthropic", "claude-sonnet-4-20250514", usage)
	if !ok {
		t.Fatal("expected ok=true for known anthropic model")
	}

	// $3.00/1M input + $15.00/1M output = 18.00
	if cost < 17.00 || cost > 19.00 {
		t.Errorf("cost = %f, want ~18.00", cost)
	}
}
