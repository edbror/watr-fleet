package ui

import (
	"os/exec"

	"github.com/edbror/watr-fleet/internal/fleet"
)

// Approve-without-jumping: answer an agent's permission prompt from the
// dashboard by sending the right keystrokes to its tmux pane. Each agent
// CLI has its own prompt idiom.

// approvalKeys returns the tmux send-keys arguments that answer the
// selected agent's prompt.
func approvalKeys(agent fleet.Agent, approve bool) []string {
	switch agent {
	case fleet.AgentClaude:
		// Claude Code renders a numbered menu: 1 = yes. For deny, Escape
		// dismisses the prompt cleanly; option 3 would trap the session
		// waiting for typed feedback nobody is there to give.
		if approve {
			return []string{"1"}
		}
		return []string{"Escape"}
	case fleet.AgentOpenCode:
		if approve {
			return []string{"y"}
		}
		return []string{"n"}
	default:
		if approve {
			return []string{"y", "Enter"}
		}
		return []string{"n", "Enter"}
	}
}

// sendAnswer delivers the keys to the session's pane. Only sessions with
// a tmux target (and a live prompt) can be answered remotely.
func sendAnswer(s fleet.Session, approve bool) error {
	args := append([]string{"send-keys", "-t", s.Target}, approvalKeys(s.Agent, approve)...)
	return exec.Command("tmux", args...).Run()
}
