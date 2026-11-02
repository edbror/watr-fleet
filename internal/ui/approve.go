package ui

import (
	"errors"
	"os/exec"
	"strings"

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

// ErrPuertaDeConfianza says the pane is on a screen this feature must not
// answer. Measured against Claude Code v2.1.316 (WE-553).
var ErrPuertaDeConfianza = errors.New("folder trust screen: answer it inside the pane")

// enLaPuertaDeConfianza reports whether the pane is showing Claude Code's
// first-run folder prompt.
//
// It is not a permission prompt and it does not behave like one. Its options
// are arrow-driven with no numbers, so "1" lands on nothing and the user sees
// no effect at all. Worse, its Escape is wired to "Esc to cancel": it does not
// decline one action, it quits Claude Code and drops the pane back to the
// shell. Denying from the dashboard would kill the agent.
//
// On top of that the highlighted default is "No, exit", so there is no safe
// keystroke to send blind. The honest answer is to refuse and say so.
func enLaPuertaDeConfianza(paneTail string) bool {
	return strings.Contains(paneTail, "I trust this folder")
}

// capturarPane reads the tail of a pane. The adapter has its own capture for
// status detection; this one is three lines and lives in the package that
// needs it, which beats exporting a helper across the boundary for one call.
func capturarPane(target string) (string, error) {
	out, err := exec.Command("tmux", "capture-pane", "-p", "-t", target, "-S", "-20").Output()
	return string(out), err
}

// sendAnswer delivers the keys to the session's pane. Only sessions with
// a tmux target (and a live prompt) can be answered remotely.
//
// It looks before it types: a pane can be blocked on something that is not a
// permission prompt, and the keystrokes that answer one can destroy the other.
func sendAnswer(s fleet.Session, approve bool) error {
	if s.Agent == fleet.AgentClaude {
		// A capture that fails is not a reason to refuse: the send below will
		// fail too, and with a better message.
		if tail, err := capturarPane(s.Target); err == nil && enLaPuertaDeConfianza(tail) {
			return ErrPuertaDeConfianza
		}
	}
	args := append([]string{"send-keys", "-t", s.Target}, approvalKeys(s.Agent, approve)...)
	return exec.Command("tmux", args...).Run()
}
