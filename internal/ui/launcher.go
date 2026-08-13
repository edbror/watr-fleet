package ui

import (
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/edbror/watr-fleet/internal/adapter"
	"github.com/edbror/watr-fleet/internal/fleet"
)

// The launcher makes tmux invisible: press n, pick an agent, type a
// directory, and fleet creates and manages the session itself.

// launchableAgents is the picker order: the user's daily drivers first.
var launchableAgents = []fleet.Agent{
	fleet.AgentClaude, fleet.AgentOpenCode, fleet.AgentGrok, fleet.AgentPi,
	"kimi", "hermes", "antigravity", "codex", "gemini", "crush", "aider", "kiro",
}

const (
	launchStepAgent = iota
	launchStepDir
)

type launcher struct {
	step     int
	agentIdx int
	dir      string
}

func newLauncher() *launcher {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "~"
	}
	return &launcher{dir: home + "/"}
}

func (l *launcher) agent() fleet.Agent {
	return launchableAgents[l.agentIdx]
}

type launchResultMsg struct {
	agent   fleet.Agent
	session string
	err     error
}

// handleLauncherKey drives the two-step flow; returns done=true when the
// launcher should close.
func (m Model) handleLauncherKey(msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	l := m.launcher
	switch msg.String() {
	case "esc":
		return m, nil, true
	case "enter":
		if l.step == launchStepAgent {
			l.step = launchStepDir
			return m, nil, false
		}
		agent, dir := l.agent(), strings.TrimSpace(l.dir)
		return m, func() tea.Msg {
			session, err := adapter.LaunchSession(agent, dir)
			return launchResultMsg{agent: agent, session: session, err: err}
		}, true
	}
	if l.step == launchStepAgent {
		switch msg.String() {
		case "j", "down", "l", "right":
			l.agentIdx = (l.agentIdx + 1) % len(launchableAgents)
		case "k", "up", "h", "left":
			l.agentIdx = (l.agentIdx + len(launchableAgents) - 1) % len(launchableAgents)
		}
		return m, nil, false
	}
	// Directory step: a minimal line editor is all this needs.
	switch msg.Type {
	case tea.KeyBackspace:
		if len(l.dir) > 0 {
			runes := []rune(l.dir)
			l.dir = string(runes[:len(runes)-1])
		}
	case tea.KeyRunes, tea.KeySpace:
		l.dir += string(msg.Runes)
	}
	return m, nil, false
}

// viewLauncher renders the launch card.
func (m Model) viewLauncher(l layout) string {
	lc := m.launcher
	title := styleCyan.Bold(true).Render("⚓ LAUNCH VESSEL")
	var agents strings.Builder
	for i, a := range launchableAgents {
		marker := " "
		if i == lc.agentIdx {
			marker = styleCyan.Bold(true).Render("◄") // never re-style a rendered pill: nested ANSI mangles
		}
		agents.WriteString(agentPill(a) + marker + " ")
		if (i+1)%6 == 0 {
			agents.WriteString("\n")
		}
	}
	dirLabel := styleFaint.Render("directory ")
	dirValue := styleBright.Render(lc.dir)
	if lc.step == launchStepDir {
		dirValue += styleCyan.Render(cursorGlyph(m.frame))
	}
	hint := styleFaint.Render("enter next · esc cancel")
	if lc.step == launchStepDir {
		hint = styleFaint.Render("enter launch · esc cancel")
	}
	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colCyan)).
		Padding(0, 1).
		Width(l.width - 3)
	body := title + "\n" + strings.TrimRight(agents.String(), "\n") + "\n" +
		dirLabel + dirValue + "\n" + hint
	return " " + strings.ReplaceAll(card.Render(body), "\n", "\n ") + "\n\n"
}

func cursorGlyph(frame int) string {
	if frame%8 < 4 {
		return "▏"
	}
	return " "
}
