package provider

// modelPricing holds per-million-token pricing for a model.
type modelPricing struct {
	provider         string
	model            string
	inputPerMillion  float64
	outputPerMillion float64
}

// pricingTable contains known model pricing.
// Prices are in USD per million tokens.
var pricingTable = []modelPricing{
	// OpenAI
	{"openai", "gpt-4o", 2.50, 10.00},
	{"openai", "gpt-4o-mini", 0.15, 0.60},
	// Anthropic
	{"anthropic", "claude-sonnet-4-20250514", 3.00, 15.00},
	// DeepSeek (via custom provider)
	{"custom", "deepseek-chat", 0.14, 0.28},
}

// EstimateCost estimates the USD cost for given token usage.
// Returns (cost, true) if the provider+model is found in the pricing table,
// or (0, false) if unknown.
func EstimateCost(providerName, model string, usage *TokenUsage) (float64, bool) {
	if usage == nil {
		return 0, false
	}

	for _, p := range pricingTable {
		if p.provider == providerName && p.model == model {
			inputCost := float64(usage.InputTokens) / 1_000_000 * p.inputPerMillion
			outputCost := float64(usage.OutputTokens) / 1_000_000 * p.outputPerMillion
			return inputCost + outputCost, true
		}
	}
	return 0, false
}