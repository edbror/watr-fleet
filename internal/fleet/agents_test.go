package fleet

import "testing"

func TestAgentFromCommandKnownAgents(t *testing.T) {
	cases := map[string]Agent{
		"claude":       AgentClaude,
		"Claude":       AgentClaude,
		"opencode":     AgentOpenCode,
		"pi":           AgentPi,
		"grok":         AgentGrok,
		"kimi":         "kimi",
		"hermes":       "hermes",
		"antigravity":  "antigravity",
		"gemini-cli":   "gemini",
		"cursor-agent": "cursor",
		"bash":         AgentUnknown,
		"vim":          AgentUnknown,
		"node":         AgentUnknown,
	}
	for command, want := range cases {
		if got := AgentFromCommand(command); got != want {
			t.Errorf("AgentFromCommand(%q) = %q, want %q", command, got, want)
		}
	}
}

func TestAgentFromArgsResolvesInterpreterCLIs(t *testing.T) {
	cases := map[string]Agent{
		"node /usr/local/bin/qwen":                     "qwen",
		"node /opt/homebrew/bin/gemini chat":           "gemini",
		"/usr/local/bin/crush":                         "crush",
		"python3 /home/ed/.local/bin/aider --model o3": "aider",
		"bun /Users/ed/.bun/bin/kimi":                  "kimi",
		"/opt/kiro/bin/kiro chat":                      "kiro",
		"node server.js":                               AgentUnknown,
		"vim main.go":                                  AgentUnknown,
	}
	for args, want := range cases {
		if got := AgentFromArgs(args); got != want {
			t.Errorf("AgentFromArgs(%q) = %q, want %q", args, got, want)
		}
	}
}

func TestHexForFallsBackForUnknown(t *testing.T) {
	if HexFor("mystery") == "" {
		t.Error("unknown agent must still get a color")
	}
	if HexFor(AgentClaude) == HexFor("mystery") {
		t.Error("known agents should have identity colors, not the fallback")
	}
}
