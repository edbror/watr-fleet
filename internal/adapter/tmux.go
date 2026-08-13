package adapter

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/edbror/watr-fleet/internal/fleet"
)

// tmuxSource discovers agent sessions by walking tmux panes and inferring
// status from each pane's recent output with per-agent detectors. It is
// the discovery layer; richer sources (Claude Code transcripts, hooks)
// enrich its sessions through multiSource.
type tmuxSource struct {
	needsYouSince map[string]time.Time    // target -> first time we saw it blocked
	activity      map[string]paneActivity // target -> content-change tracking
}

// paneActivity remembers what a pane looked like last poll. A pane whose
// content stops changing is not working, no matter what old scrollback
// says — spinners animate, idle sessions are static.
type paneActivity struct {
	hash       uint64
	lastChange time.Time
	seen       bool
}

// stableAfter is how long pane content must stay identical before a
// marker-based "working" verdict is downgraded to idle.
const stableAfter = 3 * time.Second

// NewTmuxSource returns a Source backed by the local tmux server.
func NewTmuxSource() (Source, error) {
	if _, err := exec.LookPath("tmux"); err != nil {
		return nil, fmt.Errorf("tmux not found in PATH: %w", err)
	}
	return &tmuxSource{
		needsYouSince: map[string]time.Time{},
		activity:      map[string]paneActivity{},
	}, nil
}

func (t *tmuxSource) Name() string { return "tmux" }

const paneFormat = "#{session_name}\t#{window_index}\t#{pane_index}\t#{pane_current_command}\t#{pane_current_path}\t#{pane_pid}"

func (t *tmuxSource) Snapshot() ([]fleet.Session, error) {
	out, err := exec.Command("tmux", "list-panes", "-a", "-F", paneFormat).Output()
	if err != nil {
		return nil, fmt.Errorf("listing tmux panes: %w", err)
	}
	tree := loadProcessTree()
	var sessions []fleet.Session
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		if s, ok := t.paneToSession(line, tree); ok {
			sessions = append(sessions, s)
		}
	}
	return sessions, nil
}

func (t *tmuxSource) paneToSession(line string, tree processTree) (fleet.Session, bool) {
	parts := strings.Split(line, "\t")
	if len(parts) < 6 {
		return fleet.Session{}, false
	}
	agent := agentFromCommand(parts[3])
	if agent == fleet.AgentUnknown {
		// Interpreter-launched CLIs (node/python/bun) hide the agent name;
		// the pane's process subtree still knows who is really running.
		agent = tree.agentUnder(parts[5])
	}
	if agent == fleet.AgentUnknown {
		return fleet.Session{}, false
	}
	target := fmt.Sprintf("%s:%s.%s", parts[0], parts[1], parts[2])
	status, lastEvent := t.statusOf(target, agent)
	session := fleet.Session{
		ID:         target,
		Project:    parts[0],
		Agent:      agent,
		Status:     status,
		LastEvent:  lastEvent,
		ContextPct: -1, // unknown until a telemetry source enriches it
		Target:     target,
		Dir:        parts[4],
	}
	t.trackBlockedTime(&session)
	return session, true
}

// trackBlockedTime remembers when a pane first showed a blocked prompt so
// the attention queue can sort by real waiting time.
func (t *tmuxSource) trackBlockedTime(s *fleet.Session) {
	if s.Status == fleet.StatusNeedsYou {
		if _, seen := t.needsYouSince[s.Target]; !seen {
			t.needsYouSince[s.Target] = time.Now()
		}
		s.NeedsYouSince = t.needsYouSince[s.Target]
		return
	}
	delete(t.needsYouSince, s.Target)
}

func agentFromCommand(command string) fleet.Agent {
	return fleet.AgentFromCommand(command)
}

// detector holds the pane-output patterns that identify each state for
// one agent. Evaluation order: needsYou, errored, working, else idle.
type detector struct {
	needsYou *regexp.Regexp
	working  *regexp.Regexp
	errored  *regexp.Regexp
}

// Per-agent detectors, tuned to each CLI's real prompts. The generic
// detector is the fallback for agents without a tuned table yet.
var (
	genericDetector = detector{
		needsYou: regexp.MustCompile(`(?i)(\by/n\b|\(y\)es|do you want|proceed\?|approve|allow|permission|continue\?|choose an option)`),
		working:  regexp.MustCompile(`(?i)(thinking|working|running|generating|executing|\.\.\.$)`),
		errored:  regexp.MustCompile(`(?i)(error:|fatal:|panic:|traceback|rate limit)`),
	}
	detectors = map[fleet.Agent]detector{
		fleet.AgentClaude: {
			needsYou: regexp.MustCompile(`(?i)(do you want|❯\s*1\.|\b1\.\s*yes\b|allow .+\?|permission|waiting for your|proceed\?)`),
			working:  regexp.MustCompile(`(esc to interrupt|✻|✳|(?i:thinking|pondering|crafting|wrangling))`),
			errored:  regexp.MustCompile(`(?i)(api error|rate limit reached|context left until)`),
		},
		fleet.AgentOpenCode: {
			needsYou: regexp.MustCompile(`(?i)(\[y/n\]|permission|approve|allow this)`),
			working:  regexp.MustCompile(`(?i)(working|thinking|generating|⣾|⣽|⣻|◍)`),
			errored:  regexp.MustCompile(`(?i)(error:|failed)`),
		},
	}
)

func detectorFor(agent fleet.Agent) detector {
	if d, ok := detectors[agent]; ok {
		return d
	}
	return genericDetector
}

// statusOf infers status from pane output, cross-checked against real
// activity: markers say what the agent CLAIMS, content change says what
// it is actually DOING. Stale scrollback can no longer fake "working".
func (t *tmuxSource) statusOf(target string, agent fleet.Agent) (fleet.Status, string) {
	out, err := exec.Command("tmux", "capture-pane", "-p", "-t", target, "-S", "-25").Output()
	if err != nil {
		return fleet.StatusError, "cannot read pane"
	}
	status, lastLine := classify(string(out), agent)
	stableFor, baseline := t.trackContent(target, string(out))
	return reconcileWithActivity(status, stableFor, baseline), lastLine
}

// trackContent hashes the pane tail and reports how long it has been
// unchanged, plus whether we have a baseline from a previous poll.
func (t *tmuxSource) trackContent(target, tail string) (time.Duration, bool) {
	h := fnv1a(tail)
	act, seen := t.activity[target]
	now := time.Now()
	if !seen || h != act.hash {
		t.activity[target] = paneActivity{hash: h, lastChange: now, seen: true}
		return 0, seen
	}
	return now.Sub(act.lastChange), true
}

// reconcileWithActivity applies the ground truth of content change:
//   - a "working" verdict on a pane that has been static beyond the
//     threshold is stale scrollback → idle
//   - an "idle" verdict on a pane whose content is actively changing is
//     an agent streaming output without known markers → working
//
// Blocked and errored verdicts always stand: their screens ARE static.
func reconcileWithActivity(status fleet.Status, stableFor time.Duration, baseline bool) fleet.Status {
	switch status {
	case fleet.StatusWorking:
		if baseline && stableFor > stableAfter {
			return fleet.StatusIdle
		}
	case fleet.StatusIdle:
		if baseline && stableFor == 0 {
			return fleet.StatusWorking
		}
	}
	return status
}

// fnv1a hashes pane content for change detection.
func fnv1a(s string) uint64 {
	const offset, prime = 14695981039346656037, 1099511628211
	h := uint64(offset)
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= prime
	}
	return h
}

// classify applies the agent's detector to captured pane output.
func classify(paneTail string, agent fleet.Agent) (fleet.Status, string) {
	tail := strings.TrimSpace(paneTail)
	lastLine := lastNonEmptyLine(tail)
	d := detectorFor(agent)
	switch {
	case d.needsYou.MatchString(tail):
		return fleet.StatusNeedsYou, lastLine
	case d.errored.MatchString(tail):
		return fleet.StatusError, lastLine
	case d.working.MatchString(tail):
		return fleet.StatusWorking, lastLine
	default:
		return fleet.StatusIdle, lastLine
	}
}

func lastNonEmptyLine(text string) string {
	lines := strings.Split(text, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if line := strings.TrimSpace(lines[i]); line != "" {
			if len(line) > 60 {
				return line[:57] + "..."
			}
			return line
		}
	}
	return ""
}
