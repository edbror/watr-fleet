// Package fleet holds the domain model: sessions, status, and fleet-level telemetry.
package fleet

import "time"

// Agent identifies which CLI coding agent runs inside a session.
type Agent string

const (
	AgentClaude   Agent = "claude"
	AgentOpenCode Agent = "opencode"
	AgentPi       Agent = "pi"
	AgentGrok     Agent = "grok"
	AgentUnknown  Agent = "agent"
)

// Status is the lifecycle state of an agent session.
type Status int

const (
	StatusWorking Status = iota
	StatusNeedsYou
	StatusIdle
	StatusError
)

func (s Status) String() string {
	switch s {
	case StatusWorking:
		return "working"
	case StatusNeedsYou:
		return "needs you"
	case StatusIdle:
		return "idle"
	case StatusError:
		return "error"
	}
	return "unknown"
}

// Session is one live agent session and its telemetry.
type Session struct {
	ID            string
	Project       string
	Agent         Agent
	Status        Status
	LastEvent     string
	Tokens        int
	CostUSD       float64
	ContextPct    float64 // context-window pressure, 0..1; negative means unknown
	NeedsYouSince time.Time
	Target        string // tmux target ("session:window.pane") for jumping
	Dir           string // working directory; correlation key across sources
}

// WaitingFor reports how long the session has been blocked on a human.
func (s Session) WaitingFor(now time.Time) time.Duration {
	if s.Status != StatusNeedsYou || s.NeedsYouSince.IsZero() {
		return 0
	}
	return now.Sub(s.NeedsYouSince)
}

// Summary aggregates fleet-level telemetry for the header.
type Summary struct {
	Total    int
	Working  int
	NeedsYou int
	Idle     int
	Errored  int
	Tokens   int
	CostUSD  float64
}

// Summarize folds a slice of sessions into fleet totals.
func Summarize(sessions []Session) Summary {
	var sum Summary
	sum.Total = len(sessions)
	for _, s := range sessions {
		switch s.Status {
		case StatusWorking:
			sum.Working++
		case StatusNeedsYou:
			sum.NeedsYou++
		case StatusIdle:
			sum.Idle++
		case StatusError:
			sum.Errored++
		}
		sum.Tokens += s.Tokens
		sum.CostUSD += s.CostUSD
	}
	return sum
}
