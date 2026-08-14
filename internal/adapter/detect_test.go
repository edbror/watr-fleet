package adapter

import (
	"testing"
	"time"

	"github.com/edbror/watr-fleet/internal/fleet"
)

func TestClassifyClaudePermissionPrompt(t *testing.T) {
	pane := `
● Bash(rm -rf ./cache)

Do you want to proceed?
❯ 1. Yes
  2. Yes, and don't ask again
  3. No, and tell Claude what to do differently
`
	status, _ := classify(pane, fleet.AgentClaude)
	if status != fleet.StatusNeedsYou {
		t.Errorf("status = %s, want needs you", status)
	}
}

func TestClassifyClaudeWorking(t *testing.T) {
	pane := `
✻ Pondering… (12s · esc to interrupt)
`
	status, _ := classify(pane, fleet.AgentClaude)
	if status != fleet.StatusWorking {
		t.Errorf("status = %s, want working", status)
	}
}

func TestClassifyOpenCodePermission(t *testing.T) {
	pane := `opencode wants to run: npm install left-pad [y/n]`
	status, _ := classify(pane, fleet.AgentOpenCode)
	if status != fleet.StatusNeedsYou {
		t.Errorf("status = %s, want needs you", status)
	}
}

func TestClassifyIdleFallback(t *testing.T) {
	pane := `$ `
	status, _ := classify(pane, fleet.AgentGrok)
	if status != fleet.StatusIdle {
		t.Errorf("status = %s, want idle", status)
	}
}

func TestClassifyIgnoresScrollbackError(t *testing.T) {
	// "error:" high in the scrollback is a coding session doing its job —
	// only the bottom of the pane can declare the CLI itself dead.
	pane := `
error: cannot find module 'left-pad'
  at require (node:internal/modules)

fixed! rebuilding now
build ok
tests passing
all green
$ `
	status, _ := classify(pane, fleet.Agent("codex"))
	if status == fleet.StatusError {
		t.Errorf("scrollback error text classified as errored")
	}
}

func TestClassifyBottomErrorStillCounts(t *testing.T) {
	pane := `
doing things
rate limit reached, try again later`
	status, _ := classify(pane, fleet.AgentGrok)
	if status != fleet.StatusError {
		t.Errorf("status = %s, want error for bottom-line failure", status)
	}
}

func TestReconcileDowngradesStaleWorking(t *testing.T) {
	// Old "✻ Pondering…" in scrollback, but the pane hasn't changed in a
	// while: the session is idle, not working.
	got := reconcileWithActivity(fleet.StatusWorking, 5*time.Second, true)
	if got != fleet.StatusIdle {
		t.Errorf("static working pane = %s, want idle", got)
	}
	// A genuinely animating pane keeps its working status.
	if got := reconcileWithActivity(fleet.StatusWorking, 0, true); got != fleet.StatusWorking {
		t.Errorf("changing working pane = %s, want working", got)
	}
	// First poll has no baseline: trust the markers, don't downgrade.
	if got := reconcileWithActivity(fleet.StatusWorking, 0, false); got != fleet.StatusWorking {
		t.Errorf("first-poll working pane = %s, want working", got)
	}
}

func TestReconcileUpgradesActiveIdle(t *testing.T) {
	// No known markers but the pane is streaming output: that is work.
	if got := reconcileWithActivity(fleet.StatusIdle, 0, true); got != fleet.StatusWorking {
		t.Errorf("changing idle pane = %s, want working", got)
	}
	// Blocked screens are static by nature — never touched.
	if got := reconcileWithActivity(fleet.StatusNeedsYou, 10*time.Second, true); got != fleet.StatusNeedsYou {
		t.Errorf("blocked pane = %s, want needs you", got)
	}
}

func TestReconcileErrorOnStreamingPaneIsWorking(t *testing.T) {
	// Error text flowing through an actively streaming pane is a healthy
	// session at work; a genuinely dead CLI's screen is static.
	if got := reconcileWithActivity(fleet.StatusError, 0, true); got != fleet.StatusWorking {
		t.Errorf("streaming errored pane = %s, want working", got)
	}
	if got := reconcileWithActivity(fleet.StatusError, 10*time.Second, true); got != fleet.StatusError {
		t.Errorf("static errored pane = %s, want error", got)
	}
}

func TestMergePrefersStrongerStatusAndTelemetry(t *testing.T) {
	base := fleet.Session{Dir: "/p", Status: fleet.StatusWorking, Target: "s:0.0"}
	extra := fleet.Session{Dir: "/p", Status: fleet.StatusNeedsYou, Tokens: 1234, CostUSD: 5.5, ContextPct: 0.4}

	merged := enrich(base, extra)
	if merged.Status != fleet.StatusNeedsYou {
		t.Errorf("status = %s, want needs you", merged.Status)
	}
	if merged.Tokens != 1234 || merged.CostUSD != 5.5 {
		t.Errorf("telemetry not merged: %+v", merged)
	}
	if merged.Target != "s:0.0" {
		t.Errorf("target lost in merge: %q", merged.Target)
	}
}

func TestPricingFamilies(t *testing.T) {
	u := tokenUsage{InputTokens: 1_000_000}
	if got := u.costUSD("claude-opus-4-6"); got != 15 {
		t.Errorf("opus input MTok = $%f, want $15", got)
	}
	if got := u.costUSD("claude-haiku-4-5"); got != 1 {
		t.Errorf("haiku input MTok = $%f, want $1", got)
	}
	if got := u.costUSD("mystery-model"); got != 3 {
		t.Errorf("default input MTok = $%f, want $3 (sonnet)", got)
	}
}
