// Package adapter connects flotilla to the systems where agent sessions live.
// Each Source normalizes one substrate (tmux, demo simulator, later: ACP,
// Claude Code hooks, OpenCode server API) into fleet.Session values.
package adapter

import "github.com/edbror/watr-fleet/internal/fleet"

// Source produces a point-in-time snapshot of agent sessions.
type Source interface {
	Name() string
	Snapshot() ([]fleet.Session, error)
}
