package fleet

import "strings"

// AgentSpec is the registry entry for a known CLI coding agent.
type AgentSpec struct {
	Hex string // identity color for pills and accents
}

// Agents is the built-in registry. Broad on purpose: discovery should
// recognize whatever the user runs, not a curated shortlist.
var Agents = map[Agent]AgentSpec{
	AgentClaude:   {Hex: "#D97757"}, // anthropic clay
	AgentOpenCode: {Hex: "#4ADE80"},
	AgentPi:       {Hex: "#A78BFA"},
	AgentGrok:     {Hex: "#E2E8F0"},
	"kimi":        {Hex: "#F472B6"},
	"hermes":      {Hex: "#FB923C"},
	"antigravity": {Hex: "#38BDF8"},
	"codex":       {Hex: "#2DD4BF"},
	"gemini":      {Hex: "#60A5FA"},
	"crush":       {Hex: "#E879F9"},
	"aider":       {Hex: "#A3E635"},
	"amp":         {Hex: "#FB7185"},
	"goose":       {Hex: "#93C5FD"},
	"cursor":      {Hex: "#CBD5E1"},
	"qwen":        {Hex: "#C084FC"},
	"droid":       {Hex: "#5EEAD4"},
	"kiro":        {Hex: "#FF9900"}, // aws orange
	"copilot":     {Hex: "#A5B4FC"},
	"auggie":      {Hex: "#4D7C0F"},
	"openhands":   {Hex: "#FDBA74"},
	"cline":       {Hex: "#86EFAC"},
	"codebuff":    {Hex: "#FCD34D"},
	"devin":       {Hex: "#7DD3FC"},
	"amazonq":     {Hex: "#F59E0B"},
}

// commandAliases maps process names (tmux pane_current_command) to agents
// whose binary name differs from their display name.
var commandAliases = map[string]Agent{
	"gemini-cli":   "gemini",
	"cursor-agent": "cursor",
	"claude-code":  AgentClaude,
	"grok-cli":     AgentGrok,
	"kiro-cli":     "kiro",
	"gh-copilot":   "copilot",
	"amazon-q":     "amazonq",
	"qwen-code":    "qwen",
}

// RegisterAgent adds or overrides an agent at runtime (fleet.toml).
func RegisterAgent(name, hex string, aliases ...string) {
	agent := Agent(strings.ToLower(name))
	if hex == "" {
		hex = "#64748B"
	}
	Agents[agent] = AgentSpec{Hex: hex}
	for _, alias := range aliases {
		commandAliases[strings.ToLower(alias)] = agent
	}
}

// AgentFromCommand resolves a pane's process name to a known agent,
// or AgentUnknown when the process is not an agent CLI.
func AgentFromCommand(command string) Agent {
	name := strings.ToLower(strings.TrimSpace(command))
	if alias, ok := commandAliases[name]; ok {
		return alias
	}
	if _, ok := Agents[Agent(name)]; ok {
		return Agent(name)
	}
	return AgentUnknown
}

// HexFor returns the agent's identity color, with a neutral fallback.
func HexFor(a Agent) string {
	if spec, ok := Agents[a]; ok {
		return spec.Hex
	}
	return "#64748B"
}

// AgentFromArgs resolves a full command line to a known agent. It covers
// interpreter-launched CLIs where the process name is just "node",
// "python", or "bun": the script path holds the real identity
// (e.g. "node /usr/local/bin/qwen").
func AgentFromArgs(args string) Agent {
	fields := strings.Fields(args)
	limit := len(fields)
	if limit > 3 {
		limit = 3 // binary, script path, first subcommand — identity lives early
	}
	for _, field := range fields[:limit] {
		name := strings.TrimSuffix(baseName(field), ".js")
		if agent := AgentFromCommand(name); agent != AgentUnknown {
			return agent
		}
	}
	return AgentUnknown
}

func baseName(path string) string {
	if i := strings.LastIndexByte(path, '/'); i >= 0 {
		return path[i+1:]
	}
	return path
}
