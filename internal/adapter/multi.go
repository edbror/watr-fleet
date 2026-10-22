package adapter

import (
	"path/filepath"
	"strings"

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
	sourceOf := []int{}  // parallel to merged: which source produced it
	enriched := []bool{} // parallel to merged: already took another source's reading
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
			if i := rowFor(merged, sourceOf, enriched, srcIdx, s.Dir); i >= 0 {
				merged[i] = enrich(merged[i], s)
				enriched[i] = true
				continue
			}
			merged = append(merged, s)
			sourceOf = append(sourceOf, srcIdx)
			enriched = append(enriched, false)
		}
	}
	// All sources failing is an error; partial data is a working fleet.
	if len(merged) == 0 && firstErr != nil {
		return nil, firstErr
	}
	return merged, nil
}

// rowFor finds the row a session belongs to, or -1 when it opens its own.
//
// The two sources do not report the same directory: a tmux pane reports the
// shell's cwd, fixed since launch, while a transcript reports the agent's,
// which follows every cd the agent makes. Keying on equality alone split one
// session into two rows the moment an agent stepped into a subdirectory: the
// pane kept the jump target with its telemetry at zero, and the transcript
// kept the numbers with nowhere to jump. So an exact match comes first, and
// failing that the deepest row whose directory contains this one.
//
// Two rules keep that from over-merging. Rows from the same source never
// merge, because two panes in one tree are two sessions. And a row that
// already took a reading is skipped, so a second transcript under the same
// pane opens its own row instead of overwriting the first one's numbers.
//
// ponytail: linear scan, a fleet is tens of rows; index by directory if one
// ever reaches thousands.
func rowFor(merged []fleet.Session, sourceOf []int, enriched []bool, srcIdx int, dir string) int {
	if dir == "" {
		return -1
	}
	best, bestLen := -1, -1
	for i, s := range merged {
		if sourceOf[i] == srcIdx || enriched[i] || s.Dir == "" {
			continue
		}
		if s.Dir == dir {
			return i // nothing beats an exact match
		}
		if len(s.Dir) > bestLen && under(dir, s.Dir) {
			best, bestLen = i, len(s.Dir)
		}
	}
	return best
}

// under reports whether dir sits inside parent.
func under(dir, parent string) bool {
	rel, err := filepath.Rel(parent, dir)
	return err == nil && rel != "." && !strings.HasPrefix(rel, "..")
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
	if extra.LastActive.After(base.LastActive) {
		base.LastActive = extra.LastActive
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
