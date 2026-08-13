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
