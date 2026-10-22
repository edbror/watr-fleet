package adapter

import "testing"

func TestContextWindowDetectsLongContext(t *testing.T) {
	c := &claudeSource{contextMax: 200_000}
	// A full-but-plausible 200k session stays graded against 200k.
	if got := c.contextWindow(210_000); got != 200_000 {
		t.Errorf("window(210k) = %d, want 200k", got)
	}
	// Anything clearly past the window (25%% margin) is a 1M session:
	// grading it against 200k would peg the gauge at a bogus 100%%.
	if got := c.contextWindow(300_000); got != 1_000_000 {
		t.Errorf("window(300k) = %d, want 1M", got)
	}
	if got := c.contextWindow(120_000); got != 200_000 {
		t.Errorf("window(120k) = %d, want 200k", got)
	}
}

// Un <synthetic> ("API Error…", "Prompt is too long") llega con el usage en
// ceros. Si cuenta como la última lectura, el medidor se vacía justo cuando
// el contexto está desbordado, que es cuando más importa verlo.
func TestSyntheticNoVaciaElContexto(t *testing.T) {
	var sum transcriptSummary
	real := transcriptEntry{Type: "assistant"}
	real.Message.Model = "claude-opus-4-8"
	real.Message.Usage = &tokenUsage{InputTokens: 1_200, CacheReadTokens: 286_343}
	absorb(&sum, real)

	sintetico := transcriptEntry{Type: "assistant"}
	sintetico.Message.Model = "<synthetic>"
	sintetico.Message.Usage = &tokenUsage{}
	absorb(&sum, sintetico)

	if got := sum.lastUsage.contextSize(); got != 287_543 {
		t.Errorf("contexto tras el sintético = %d, se esperaba 287543", got)
	}
	if got := sum.totals.total(); got != 287_543 {
		t.Errorf("totales = %d, se esperaba 287543", got)
	}
}
