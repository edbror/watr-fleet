package adapter

import (
	"testing"

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
