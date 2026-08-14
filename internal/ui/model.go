// Package ui implements the fleet dashboard as a Bubble Tea program.
package ui

import (
	"os"
	"os/exec"
	"sort"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/edbror/watr-fleet/internal/adapter"
	"github.com/edbror/watr-fleet/internal/fleet"
	"github.com/edbror/watr-fleet/internal/notify"
)

const (
	animInterval = 130 * time.Millisecond // UI motion cadence
	sparkWindow  = 12                     // activity samples kept per session
)

type sortMode int

const (
	sortByAttention sortMode = iota
	sortByCost
	sortByProject
)

func (m sortMode) String() string {
	switch m {
	case sortByCost:
		return "cost"
	case sortByProject:
		return "project"
	default:
		return "attention"
	}
}

// Model is the Bubble Tea model for the dashboard.
type Model struct {
	source   adapter.Source
	refresh  time.Duration
	sessions []fleet.Session
	err      error
	cursor   int
	frame    int
	sort     sortMode
	width    int
	height   int

	prevTokens map[string]int     // session ID -> tokens at last snapshot
	activity   map[string][]int   // session ID -> recent activity samples
	voyage     map[string]float64 // session ID -> drift phase on the open sea

	dashboard  bool  // 'd' toggles the htop-style dashboard view
	fleetTotal int   // total tokens at last snapshot
	burn       []int // fleet-wide token deltas per data tick

	notifier    *notify.Notifier
	notified    map[string]bool // session ID -> already escalated
	flash       string          // transient status-bar message
	flashFrames int             // remaining animation frames for flash

	launcher    *launcher // non-nil while the launch card is open
	confirmKill string    // target awaiting a second 'x' press
	monitor     bool      // 'm' toggles the ambient sci-fi monitor view
}

// WithNotifier enables out-of-terminal escalation for blocked sessions.
func (m Model) WithNotifier(n *notify.Notifier) Model {
	m.notifier = n
	return m
}

// NewModel wires a data source into the dashboard.
func NewModel(source adapter.Source, refresh time.Duration) Model {
	return Model{
		source:     source,
		refresh:    refresh,
		prevTokens: map[string]int{},
		activity:   map[string][]int{},
		voyage:     map[string]float64{},
		notified:   map[string]bool{},
	}
}

type animTickMsg time.Time
type dataTickMsg time.Time
type snapshotMsg struct {
	sessions []fleet.Session
	err      error
}
type answerResultMsg struct {
	project  string
	approved bool
	err      error
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.takeSnapshot, m.animTick(), m.dataTick())
}

// Motion runs faster than data: the UI animates at animInterval while
// sources are polled only every refresh interval.
func (m Model) animTick() tea.Cmd {
	return tea.Tick(animInterval, func(t time.Time) tea.Msg { return animTickMsg(t) })
}

func (m Model) dataTick() tea.Cmd {
	return tea.Tick(m.refresh, func(t time.Time) tea.Msg { return dataTickMsg(t) })
}

func (m Model) takeSnapshot() tea.Msg {
	sessions, err := m.source.Snapshot()
	return snapshotMsg{sessions: sessions, err: err}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case animTickMsg:
		m.frame++
		m.advanceVoyage()
		if m.flashFrames > 0 {
			m.flashFrames--
			if m.flashFrames == 0 {
				m.flash = ""
			}
		}
		return m, m.animTick()
	case dataTickMsg:
		return m, tea.Batch(m.takeSnapshot, m.dataTick())
	case snapshotMsg:
		m.err = msg.err
		if msg.err == nil {
			m.sessions = msg.sessions
			m.recordActivity()
			m.escalateBlocked()
			m.applySort()
			m.clampCursor()
		}
	case answerResultMsg:
		m.flash = answerFlash(msg)
		m.flashFrames = 30
	case launchResultMsg:
		if msg.err != nil {
			m.flash = "⚠ " + msg.err.Error()
		} else {
			m.flash = "⚓ launched " + string(msg.agent)
		}
		m.flashFrames = 40
		return m, m.takeSnapshot
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

// answerFlash reports what actually happened — never a fake success.
func answerFlash(r answerResultMsg) string {
	if r.err != nil {
		return "⚠ couldn't reach " + r.project
	}
	if r.approved {
		return "✓ approved " + r.project
	}
	return "✗ denied " + r.project
}

const burnWindow = 240 // ~3 minutes of history at the default refresh

// recordActivity samples per-session work (token deltas, or liveness for
// sessions without telemetry) to feed the sparklines and the flow chart.
// It also prunes state for vanished sessions: fleet runs for hours, and
// per-session maps must not grow with every session that ever existed.
func (m *Model) recordActivity() {
	m.pruneVanished()
	total := 0
	for _, s := range m.sessions {
		total += s.Tokens
	}
	if delta := total - m.fleetTotal; delta > 0 && m.fleetTotal > 0 {
		m.burn = append(m.burn, delta)
	} else if m.fleetTotal > 0 {
		m.burn = append(m.burn, 0)
	}
	if len(m.burn) > burnWindow {
		m.burn = m.burn[len(m.burn)-burnWindow:]
	}
	m.fleetTotal = total
	for _, s := range m.sessions {
		sample := 0
		if prev, seen := m.prevTokens[s.ID]; seen && s.Tokens > prev {
			sample = s.Tokens - prev
		} else if s.Tokens == 0 && s.Status == fleet.StatusWorking {
			sample = 1
		}
		m.prevTokens[s.ID] = s.Tokens
		window := append(m.activity[s.ID], sample)
		if len(window) > sparkWindow {
			window = window[len(window)-sparkWindow:]
		}
		m.activity[s.ID] = window
	}
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.launcher != nil {
		next, cmd, done := m.handleLauncherKey(msg)
		if done {
			next.launcher = nil
		}
		return next, cmd
	}
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "n":
		m.launcher = newLauncher()
		return m, nil
	case "x":
		return m.killSelected()
	case "j", "down":
		m.cursor++
		m.clampCursor()
	case "k", "up":
		m.cursor--
		m.clampCursor()
	case "s":
		m.sort = (m.sort + 1) % 3
		m.applySort()
	case "d", "tab":
		m.dashboard = !m.dashboard
	case "m":
		m.monitor = !m.monitor
	case "y":
		return m.answerSelected(true)
	case "N":
		return m.answerSelected(false)
	case "enter":
		return m, m.jumpToSelected()
	}
	return m, nil
}

// pruneVanished drops per-session state for sessions no longer present.
func (m *Model) pruneVanished() {
	alive := make(map[string]bool, len(m.sessions))
	for _, s := range m.sessions {
		alive[s.ID] = true
	}
	for id := range m.prevTokens {
		if !alive[id] {
			delete(m.prevTokens, id)
			delete(m.activity, id)
			delete(m.voyage, id)
			delete(m.notified, id)
		}
	}
}

// escalateBlocked fires one notification per session once its wait
// crosses the notifier threshold; the flag clears when it unblocks.
func (m *Model) escalateBlocked() {
	if m.notifier == nil {
		return
	}
	now := time.Now()
	for _, s := range m.sessions {
		if s.Status != fleet.StatusNeedsYou {
			delete(m.notified, s.ID)
			continue
		}
		if !m.notified[s.ID] && s.WaitingFor(now) >= m.notifier.Threshold {
			m.notifier.Notify(s.Project, s.LastEvent)
			m.notified[s.ID] = true
		}
	}
}

// answerSelected approves or denies the selected session's prompt from
// the dashboard — no jump required.
func (m Model) answerSelected(approve bool) (tea.Model, tea.Cmd) {
	if m.cursor >= len(m.sessions) {
		return m, nil
	}
	s := m.sessions[m.cursor]
	if s.Status != fleet.StatusNeedsYou || s.Target == "" {
		m.flash = "nothing to answer here"
		m.flashFrames = 20
		return m, nil
	}
	return m, func() tea.Msg {
		err := sendAnswer(s, approve)
		return answerResultMsg{project: s.Project, approved: approve, err: err}
	}
}

// killSelected terminates a fleet-managed session; a second press
// confirms, so a stray 'x' never sinks a vessel.
func (m Model) killSelected() (tea.Model, tea.Cmd) {
	if m.cursor >= len(m.sessions) {
		return m, nil
	}
	s := m.sessions[m.cursor]
	if s.Target == "" {
		m.flash = "nothing to kill here"
		m.flashFrames = 20
		return m, nil
	}
	if m.confirmKill != s.Target {
		m.confirmKill = s.Target
		m.flash = "press x again to sink " + s.Project
		m.flashFrames = 40
		return m, nil
	}
	m.confirmKill = ""
	if err := adapter.KillManagedSession(s.Target); err != nil {
		m.flash = "⚠ " + err.Error()
	} else {
		m.flash = "⚓ sank " + s.Project
	}
	m.flashFrames = 40
	return m, m.takeSnapshot
}

// jumpToSelected brings the user to the session: inside tmux it switches
// clients; outside, it attaches — and returns to fleet on detach, so
// tmux never has to be driven by hand.
func (m Model) jumpToSelected() tea.Cmd {
	if m.cursor >= len(m.sessions) {
		return nil
	}
	target := m.sessions[m.cursor].Target
	if target == "" {
		return nil
	}
	if os.Getenv("TMUX") != "" {
		return func() tea.Msg {
			_ = exec.Command("tmux", "switch-client", "-t", target).Run()
			return nil
		}
	}
	attach := exec.Command("tmux", "attach-session", "-t", target)
	return tea.ExecProcess(attach, func(error) tea.Msg { return dataTickMsg(time.Time{}) })
}

func (m *Model) clampCursor() {
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.sessions) && len(m.sessions) > 0 {
		m.cursor = len(m.sessions) - 1
	}
}

// applySort orders the fleet by the active mode. Attention sorting puts
// blocked sessions first, longest-waiting on top — the interrupt queue.
func (m *Model) applySort() {
	now := time.Now()
	sort.SliceStable(m.sessions, func(i, j int) bool {
		a, b := m.sessions[i], m.sessions[j]
		switch m.sort {
		case sortByCost:
			return a.CostUSD > b.CostUSD
		case sortByProject:
			return a.Project < b.Project
		default:
			return attentionRank(a, now) > attentionRank(b, now)
		}
	})
}

// attentionRank scores urgency: needs-you (by wait time) > error > working > idle.
func attentionRank(s fleet.Session, now time.Time) float64 {
	switch s.Status {
	case fleet.StatusNeedsYou:
		return 1e9 + s.WaitingFor(now).Seconds()
	case fleet.StatusError:
		return 1e6
	case fleet.StatusWorking:
		return 1e3
	default:
		return 0
	}
}
