package adapter

import (
	"github.com/edbror/watr-fleet/internal/fleet"
)

// multiSource merges several sources into one fleet view. Sessions from
// different sources describing the same work (matched by working directory)
// collapse into one row: tmux contributes the jump target and live pane
// signals; the Claude Code source contributes real telemetry.
type multiSource struct {
	sources []Source
}

// NewMultiSource composes sources in priority order: later sources enrich
// (and their non-idle status overrides) earlier ones on a directory match.
func NewMultiSource(sources ...Source) Source {
	return &multiSource{sources: sources}
}

func (m *multiSource) Name() string {
	name := ""
	for i, s := range m.sources {
		if i > 0 {
			name += "+"
		}
		name += s.Name()
	}
	return name
}

func (m *multiSource) Snapshot() ([]fleet.Session, error) {
	var merged []fleet.Session
	sourceOf := []int{}       // parallel to merged: which source produced it
	byDir := map[string]int{} // Dir -> index in merged
	var firstErr error
	for srcIdx, source := range m.sources {
		sessions, err := source.Snapshot()
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		for _, s := range sessions {
			// Merge only ACROSS sources: two panes in the same directory
			// from the SAME source are distinct sessions, not one.
			if i, seen := byDir[s.Dir]; seen && s.Dir != "" && sourceOf[i] != srcIdx {
				merged[i] = enrich(merged[i], s)
				continue
			}
			merged = append(merged, s)
			sourceOf = append(sourceOf, srcIdx)
			if s.Dir != "" {
				byDir[s.Dir] = len(merged) - 1
			}
		}
	}
	// All sources failing is an error; partial data is a working fleet.
	if len(merged) == 0 && firstErr != nil {
		return nil, firstErr
	}
	return merged, nil
}

// enrich merges a richer reading of the same session into the base row.
// Telemetry always flows in; status upgrades only on a stronger signal.
func enrich(base, extra fleet.Session) fleet.Session {
	if extra.Tokens > 0 {
		base.Tokens = extra.Tokens
	}
	if extra.CostUSD > 0 {
		base.CostUSD = extra.CostUSD
	}
	if extra.ContextPct >= 0 {
		base.ContextPct = extra.ContextPct
	}
	if extra.LastEvent != "" {
		base.LastEvent = extra.LastEvent
	}
	if statusRank(extra.Status) > statusRank(base.Status) {
		base.Status = extra.Status
		base.NeedsYouSince = extra.NeedsYouSince
	}
	if base.Target == "" {
		base.Target = extra.Target
	}
	return base
}

// statusRank orders signals by strength: a blocked or errored reading from
// any source beats working, which beats idle.
func statusRank(s fleet.Status) int {
	switch s {
	case fleet.StatusNeedsYou:
		return 3
	case fleet.StatusError:
		return 2
	case fleet.StatusWorking:
		return 1
	default:
		return 0
	}
}
