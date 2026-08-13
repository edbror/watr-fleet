package adapter

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/edbror/watr-fleet/internal/fleet"
)

const fixtureTranscript = `{"type":"user","sessionId":"abc-123","cwd":"/home/ed/watr-launchpad","message":{"content":[{"type":"text","text":"fix the auth bug"}]}}
{"type":"assistant","sessionId":"abc-123","cwd":"/home/ed/watr-launchpad","message":{"model":"claude-sonnet-4-5","usage":{"input_tokens":1000,"output_tokens":500,"cache_read_input_tokens":90000,"cache_creation_input_tokens":2000},"content":[{"type":"text","text":"Looking at the middleware now."},{"type":"tool_use","name":"Bash"}]}}
`

func writeFixture(t *testing.T, content string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "-home-ed-watr-launchpad")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "abc-123.jsonl")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestClaudeSourceParsesTelemetry(t *testing.T) {
	root := writeFixture(t, fixtureTranscript)
	src := NewClaudeSource(root, nil)

	sessions, err := src.Snapshot()
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("want 1 session, got %d", len(sessions))
	}
	s := sessions[0]
	if s.Agent != fleet.AgentClaude {
		t.Errorf("agent = %s, want claude", s.Agent)
	}
	if want := 1000 + 500 + 90000 + 2000; s.Tokens != want {
		t.Errorf("tokens = %d, want %d", s.Tokens, want)
	}
	if s.Dir != "/home/ed/watr-launchpad" {
		t.Errorf("dir = %q", s.Dir)
	}
	if s.Project != "watr-launchpad" {
		t.Errorf("project = %q", s.Project)
	}
	// sonnet pricing: 1000*3 + 500*15 + 90000*0.30 + 2000*3.75 per MTok
	want := 1000*3.0/1e6 + 500*15.0/1e6 + 90000*0.30/1e6 + 2000*3.75/1e6
	if diff := s.CostUSD - want; diff > 1e-9 || diff < -1e-9 {
		t.Errorf("cost = %f, want %f", s.CostUSD, want)
	}
	if s.ContextPct <= 0.4 || s.ContextPct >= 0.5 {
		t.Errorf("contextPct = %f, want ~0.4675", s.ContextPct)
	}
}

func TestClaudeSourceFlagsPendingToolAsNeedsYou(t *testing.T) {
	root := writeFixture(t, fixtureTranscript)
	// Age the file past the pending-tool grace period.
	path := filepath.Join(root, "-home-ed-watr-launchpad", "abc-123.jsonl")
	old := time.Now().Add(-2 * time.Minute)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	src := NewClaudeSource(root, nil)

	sessions, err := src.Snapshot()
	if err != nil || len(sessions) != 1 {
		t.Fatalf("snapshot: %v, n=%d", err, len(sessions))
	}
	if sessions[0].Status != fleet.StatusNeedsYou {
		t.Errorf("status = %s, want needs you (unresolved tool_use)", sessions[0].Status)
	}
}

func TestHookLogBlockedSince(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")
	events := `{"session_id":"abc-123","event":"Stop","ts":1000}
{"session_id":"abc-123","event":"Notification","ts":2000}
{"session_id":"other","event":"Stop","ts":3000}
`
	if err := os.WriteFile(path, []byte(events), 0o644); err != nil {
		t.Fatal(err)
	}
	log := NewHookLog(path)
	blocked := log.blockedMap()

	if _, isBlocked := blocked["abc-123"]; !isBlocked {
		t.Error("abc-123 should be blocked (Notification after Stop)")
	}
	if _, isBlocked := blocked["other"]; isBlocked {
		t.Error("other should not be blocked (last event is Stop)")
	}
}

func TestTranscriptCacheReusesUnchangedFiles(t *testing.T) {
	root := writeFixture(t, fixtureTranscript)
	src := NewClaudeSource(root, nil).(*claudeSource)

	if _, err := src.Snapshot(); err != nil {
		t.Fatal(err)
	}
	if len(src.cache) != 1 {
		t.Fatalf("cache entries = %d, want 1", len(src.cache))
	}
	// Second snapshot with unchanged file must serve from cache and
	// produce identical telemetry.
	first, _ := src.Snapshot()
	second, _ := src.Snapshot()
	if first[0].Tokens != second[0].Tokens || first[0].CostUSD != second[0].CostUSD {
		t.Error("cached pass diverged from fresh pass")
	}
}
