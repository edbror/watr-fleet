package adapter

import "strings"

// modelPricing is USD per million tokens. Estimates for cost display —
// fleet is a dashboard, not a billing system. Override in config later.
type modelPricing struct {
	Input      float64
	Output     float64
	CacheRead  float64
	CacheWrite float64
}

var pricingTable = map[string]modelPricing{
	"opus":   {Input: 15, Output: 75, CacheRead: 1.50, CacheWrite: 18.75},
	"sonnet": {Input: 3, Output: 15, CacheRead: 0.30, CacheWrite: 3.75},
	"haiku":  {Input: 1, Output: 5, CacheRead: 0.10, CacheWrite: 1.25},
}

var defaultPricing = pricingTable["sonnet"]

// OverridePricing replaces one model family's rates (fleet.toml).
func OverridePricing(family string, input, output, cacheRead, cacheWrite float64) {
	pricingTable[strings.ToLower(family)] = modelPricing{
		Input: input, Output: output, CacheRead: cacheRead, CacheWrite: cacheWrite,
	}
}

func pricingFor(model string) modelPricing {
	lower := strings.ToLower(model)
	for family, p := range pricingTable {
		if strings.Contains(lower, family) {
			return p
		}
	}
	return defaultPricing
}

// tokenUsage mirrors the usage block of a Claude Code transcript entry.
type tokenUsage struct {
	InputTokens         int `json:"input_tokens"`
	OutputTokens        int `json:"output_tokens"`
	CacheReadTokens     int `json:"cache_read_input_tokens"`
	CacheCreationTokens int `json:"cache_creation_input_tokens"`
}

func (u tokenUsage) total() int {
	return u.InputTokens + u.OutputTokens + u.CacheReadTokens + u.CacheCreationTokens
}

// contextSize approximates tokens currently in the context window.
func (u tokenUsage) contextSize() int {
	return u.InputTokens + u.CacheReadTokens + u.CacheCreationTokens + u.OutputTokens
}

func (u tokenUsage) costUSD(model string) float64 {
	p := pricingFor(model)
	return float64(u.InputTokens)*p.Input/1e6 +
		float64(u.OutputTokens)*p.Output/1e6 +
		float64(u.CacheReadTokens)*p.CacheRead/1e6 +
		float64(u.CacheCreationTokens)*p.CacheWrite/1e6
}
