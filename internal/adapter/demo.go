package adapter

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/edbror/watr-fleet/internal/fleet"
)

// demoFleet simulates a realistic multi-agent fleet so the dashboard is
// demoable on any machine, with no agents running.
type demoFleet struct {
	rng      *rand.Rand
	sessions []fleet.Session
	started  time.Time
}

// NewDemoFleet builds a simulated fleet with deterministic seeding.
func NewDemoFleet(seed int64) Source {
	rng := rand.New(rand.NewSource(seed))
	d := &demoFleet{rng: rng, started: time.Now()}
	d.sessions = d.initialSessions()
	return d
}

func (d *demoFleet) Name() string { return "demo" }

func (d *demoFleet) Snapshot() ([]fleet.Session, error) {
	for i := range d.sessions {
		d.evolve(&d.sessions[i])
	}
	out := make([]fleet.Session, len(d.sessions))
	copy(out, d.sessions)
	return out, nil
}

type demoSeed struct {
	project string
	agent   fleet.Agent
	event   string
}

func (d *demoFleet) initialSessions() []fleet.Session {
	seeds := []demoSeed{
		{"watr-launchpad", fleet.AgentClaude, "refactoring auth middleware"},
		{"docbrain", fleet.AgentOpenCode, "indexing document corpus"},
		{"pulse-mx", fleet.AgentClaude, "writing integration tests"},
		{"edi-pipeline", fleet.AgentPi, "mapping X12 segments"},
		{"content-weaver", fleet.AgentGrok, "drafting campaign copy"},
		{"finpol-api", fleet.AgentOpenCode, "migrating DB schema"},
		{"kouchea-web", fleet.AgentClaude, "fixing responsive layout"},
		{"competentia", "kimi", "summarizing course catalog"},
		{"ensoluciones", "hermes", "scraping tariff tables"},
		{"efi-gra-api", "antigravity", "profiling slow queries"},
		{"presentia", "crush", "generating slide outlines"},
		{"watr-infra", "kiro", "reviewing IaC templates"},
	}
	sessions := make([]fleet.Session, 0, len(seeds))
	for i, s := range seeds {
		sessions = append(sessions, fleet.Session{
			ID:         fmt.Sprintf("s%02d", i+1),
			Project:    s.project,
			Agent:      s.agent,
			Status:     fleet.StatusWorking,
			LastEvent:  s.event,
			Tokens:     20_000 + d.rng.Intn(400_000),
			CostUSD:    0.4 + d.rng.Float64()*7,
			ContextPct: 0.1 + d.rng.Float64()*0.5,
			Target:     fmt.Sprintf("demo:%d.0", i),
		})
	}
	// Guarantee an interesting opening frame: someone always needs you.
	sessions[3].Status = fleet.StatusNeedsYou
	sessions[3].NeedsYouSince = time.Now().Add(-4 * time.Minute)
	sessions[3].LastEvent = "approve: run db migration?"
	sessions[4].Status = fleet.StatusNeedsYou
	sessions[4].NeedsYouSince = time.Now().Add(-40 * time.Second)
	sessions[4].LastEvent = "choose tone: formal or playful?"
	return sessions
}

// evolve advances one session a single simulation step.
func (d *demoFleet) evolve(s *fleet.Session) {
	switch s.Status {
	case fleet.StatusWorking:
		s.Tokens += d.rng.Intn(2_500)
		s.CostUSD += d.rng.Float64() * 0.03
		s.ContextPct = clamp01(s.ContextPct + d.rng.Float64()*0.01)
		if d.rng.Float64() < 0.04 {
			s.Status = fleet.StatusNeedsYou
			s.NeedsYouSince = time.Now()
			s.LastEvent = pick(d.rng, needsYouEvents)
		} else if d.rng.Float64() < 0.01 {
			s.Status = fleet.StatusError
			s.LastEvent = pick(d.rng, errorEvents)
		} else if d.rng.Float64() < 0.15 {
			s.LastEvent = pick(d.rng, workingEvents)
		}
	case fleet.StatusNeedsYou:
		if d.rng.Float64() < 0.06 {
			s.Status = fleet.StatusWorking
			s.NeedsYouSince = time.Time{}
			s.LastEvent = pick(d.rng, workingEvents)
		}
	case fleet.StatusIdle:
		if d.rng.Float64() < 0.05 {
			s.Status = fleet.StatusWorking
			s.LastEvent = pick(d.rng, workingEvents)
		}
	case fleet.StatusError:
		if d.rng.Float64() < 0.10 {
			s.Status = fleet.StatusIdle
			s.LastEvent = "recovered; awaiting instructions"
		}
	}
}

var workingEvents = []string{
	"running test suite",
	"editing 4 files",
	"reading repo context",
	"executing build",
	"searching codebase",
	"writing migration",
	"reviewing diff",
}

var needsYouEvents = []string{
	"approve: push to main?",
	"approve: install dependency?",
	"question: keep legacy endpoint?",
	"approve: rm -rf ./cache?",
	"plan ready for review",
}

var errorEvents = []string{
	"build failed: exit 1",
	"API rate limit hit",
	"context window exhausted",
}

func pick(rng *rand.Rand, options []string) string {
	return options[rng.Intn(len(options))]
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
