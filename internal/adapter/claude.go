package adapter

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/edbror/watr-fleet/internal/fleet"
)

// claudeSource reads Claude Code session transcripts (~/.claude/projects)
// and produces sessions with REAL telemetry: tokens, cost, and context
// pressure, plus a pending-approval heuristic from unresolved tool calls.
type claudeSource struct {
	root       string        // transcripts root, e.g. ~/.claude/projects
	hooks      *hookLog      // optional precise signals from Claude Code hooks
	lookback   time.Duration // ignore transcripts older than this
	contextMax int           // assumed context window size in tokens
	mu         sync.Mutex    // the cache is shared by whoever calls Snapshot
	cache      map[string]cachedTranscript
}

// cachedTranscript avoids reparsing unchanged transcripts every poll, and
// remembers how far the last pass got: transcripts are append-only JSONL,
// so a changed file usually means a few new lines on the end, not a new
// file. Rereading a 200MB session on every tick is what melts a laptop.
type cachedTranscript struct {
	mtime  time.Time
	size   int64
	offset int64 // bytes of complete lines already folded into sum
	sum    transcriptSummary
}

// NewClaudeSource builds the Claude Code transcript adapter.
func NewClaudeSource(root string, hooks *hookLog) Source {
	return &claudeSource{
		root:       root,
		hooks:      hooks,
		lookback:   12 * time.Hour,
		contextMax: 200_000,
		cache:      map[string]cachedTranscript{},
	}
}

// DefaultClaudeRoot resolves ~/.claude/projects for the current user.
func DefaultClaudeRoot() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude", "projects")
}

func (c *claudeSource) Name() string { return "claude-code" }

func (c *claudeSource) Snapshot() ([]fleet.Session, error) {
	// One caller at a time: the parsed-transcript cache is plain maps, and two
	// overlapping polls writing them is a runtime fatal error, not a race you
	// get to recover from.
	c.mu.Lock()
	defer c.mu.Unlock()
	transcripts, err := c.recentTranscripts()
	if err != nil {
		return nil, err
	}
	c.pruneCache(transcripts)
	blocked := c.hooks.blockedMap() // one pass over the events file per poll
	delegated := c.subagentTotals(transcripts)
	var sessions []fleet.Session
	for _, path := range transcripts {
		if subagentTranscript(path) {
			continue // already folded into the session that launched it
		}
		s, ok := c.sessionFromTranscript(path, blocked)
		if !ok {
			continue
		}
		if d, ok := delegated[path]; ok {
			s.Tokens += d.tokens
			s.CostUSD += d.cost
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

// delegatedSpend is what a session spent inside the subagents it launched.
type delegatedSpend struct {
	tokens int
	cost   float64
}

// subagentTranscript reports whether a path is a Task's own transcript.
// Claude Code writes those to <project>/<session-id>/subagents/agent-*.jsonl.
func subagentTranscript(path string) bool {
	return filepath.Base(filepath.Dir(path)) == "subagents"
}

// parentTranscript maps a Task transcript back to the session that launched
// it: <...>/<session-id>/subagents/agent-x.jsonl -> <...>/<session-id>.jsonl
func parentTranscript(path string) string {
	return filepath.Dir(filepath.Dir(path)) + ".jsonl"
}

// subagentTotals adds up each session's Task transcripts.
//
// Those files carry the PARENT's sessionId, so as rows of their own they show
// one session several times over: a session with four Tasks in flight filled
// five rows, all with the same id. Their tokens are not in the parent's usage
// either (no sidechain entry appears in the parent file, and no subagent entry
// is repeated there), so dropping them would understate what a session that
// delegates really spent. A Task whose parent aged out of the lookback window
// has no row to land on and its spend is left out.
//
// A file that cannot be read is skipped rather than returned as an error, the
// same stance as sessionFromTranscript and the walk in recentTranscripts: this
// source polls, and one unreadable transcript must not blank the whole roster.
func (c *claudeSource) subagentTotals(paths []string) map[string]delegatedSpend {
	totals := map[string]delegatedSpend{}
	for _, path := range paths {
		if !subagentTranscript(path) {
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		sum, ok := c.summarize(path, info)
		if !ok {
			continue
		}
		parent := parentTranscript(path)
		t := totals[parent]
		t.tokens += sum.totals.total()
		t.cost += sum.cost
		totals[parent] = t
	}
	return totals
}

// pruneCache drops cached parses for transcripts that aged out of the
// lookback window, keeping memory flat across long runs.
func (c *claudeSource) pruneCache(current []string) {
	keep := make(map[string]bool, len(current))
	for _, path := range current {
		keep[path] = true
	}
	for path := range c.cache {
		if !keep[path] {
			delete(c.cache, path)
		}
	}
}

func (c *claudeSource) recentTranscripts() ([]string, error) {
	if c.root == "" {
		return nil, nil
	}
	var paths []string
	cutoff := time.Now().Add(-c.lookback)
	err := filepath.WalkDir(c.root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".jsonl") {
			return nil //nolint:nilerr // unreadable entries are skipped, not fatal
		}
		if info, err := d.Info(); err == nil && info.ModTime().After(cutoff) {
			paths = append(paths, path)
		}
		return nil
	})
	if os.IsNotExist(err) {
		return nil, nil
	}
	return paths, err
}

// transcriptEntry is the subset of a Claude Code JSONL line fleet needs.
type transcriptEntry struct {
	Type      string `json:"type"`
	SessionID string `json:"sessionId"`
	Cwd       string `json:"cwd"`
	Message   struct {
		Model   string          `json:"model"`
		Usage   *tokenUsage     `json:"usage"`
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
	Name string `json:"name"`
}

// transcriptSummary accumulates what one pass over a transcript learns.
type transcriptSummary struct {
	sessionID   string
	cwd         string
	model       string
	totals      tokenUsage
	lastUsage   tokenUsage
	lastEvent   string
	pendingTool string // set when an assistant tool_use has no later reply
	cost        float64
}

func (c *claudeSource) sessionFromTranscript(path string, blocked map[string]time.Time) (fleet.Session, bool) {
	info, err := os.Stat(path)
	if err != nil {
		return fleet.Session{}, false
	}
	sum, ok := c.summarize(path, info)
	if !ok {
		return fleet.Session{}, false
	}
	if sum.sessionID == "" && sum.totals.total() == 0 {
		return fleet.Session{}, false
	}
	status, event, since := c.resolveStatus(sum, info.ModTime(), blocked)
	return fleet.Session{
		ID:            "cc:" + sum.sessionID,
		Project:       filepath.Base(sum.cwd),
		Agent:         fleet.AgentClaude,
		Status:        status,
		LastEvent:     event,
		Tokens:        sum.totals.total(),
		CostUSD:       sum.cost,
		ContextPct:    clamp01(float64(sum.lastUsage.contextSize()) / float64(c.contextWindow(sum.lastUsage.contextSize()))),
		NeedsYouSince: since,
		LastActive:    info.ModTime(),
		Dir:           sum.cwd,
	}, true
}

// contextWindow picks the assumed window for a session. A context clearly
// past the configured window (25% margin absorbs output overshoot on a
// genuinely full window) can only belong to a long-context model (Sonnet
// 1M): grading those against 200k pegs every gauge at 100%, which turns
// the roster into a wall of identical readings.
func (c *claudeSource) contextWindow(size int) int {
	const longContext = 1_000_000
	if size > c.contextMax+c.contextMax/4 {
		return longContext
	}
	return c.contextMax
}

// summarize parses a transcript, reusing the cached pass when the file
// has not changed — and continuing from the cached offset when it only
// grew. The summary is a left fold over lines, so absorbing just the new
// tail is exactly equivalent to reparsing the whole file. A file that
// shrank was rewritten: start over from byte zero.
func (c *claudeSource) summarize(path string, info os.FileInfo) (transcriptSummary, bool) {
	hit, cached := c.cache[path]
	if cached && hit.mtime.Equal(info.ModTime()) && hit.size == info.Size() {
		return hit.sum, true
	}
	file, err := os.Open(path)
	if err != nil {
		return transcriptSummary{}, false
	}
	defer file.Close()

	var sum transcriptSummary
	var offset int64
	if cached && hit.offset > 0 && info.Size() >= hit.size {
		if _, err := file.Seek(hit.offset, io.SeekStart); err == nil {
			sum, offset = hit.sum, hit.offset
		}
	}
	offset += absorbLines(file, &sum)
	c.cache[path] = cachedTranscript{mtime: info.ModTime(), size: info.Size(), offset: offset, sum: sum}
	return sum, true
}

func summarizeTranscript(file *os.File) transcriptSummary {
	var sum transcriptSummary
	absorbLines(file, &sum)
	return sum
}

// absorbLines folds every complete line from r into sum and returns how
// many bytes it consumed. A trailing line without a newline is a write
// caught mid-flight: it is absorbed only if it already parses as JSON
// (then it was whole, just unterminated); otherwise its bytes are left
// uncounted so the next poll rereads it once it is finished.
func absorbLines(r io.Reader, sum *transcriptSummary) int64 {
	reader := bufio.NewReaderSize(r, 1024*1024)
	var consumed int64
	for {
		line, err := reader.ReadBytes('\n')
		var entry transcriptEntry
		parsed := json.Unmarshal(line, &entry) == nil
		if err == nil {
			consumed += int64(len(line))
			if parsed {
				absorb(sum, entry)
			}
			continue
		}
		if len(line) > 0 && parsed { // whole JSON, missing only its newline
			consumed += int64(len(line))
			absorb(sum, entry)
		}
		return consumed
	}
}

// absorb folds one transcript entry into the running summary.
func absorb(sum *transcriptSummary, entry transcriptEntry) {
	if entry.SessionID != "" {
		sum.sessionID = entry.SessionID
	}
	if entry.Cwd != "" {
		sum.cwd = entry.Cwd
	}
	switch entry.Type {
	case "assistant":
		if entry.Message.Model != "" {
			sum.model = entry.Message.Model
		}
		if u := entry.Message.Usage; u != nil {
			sum.totals.InputTokens += u.InputTokens
			sum.totals.OutputTokens += u.OutputTokens
			sum.totals.CacheReadTokens += u.CacheReadTokens
			sum.totals.CacheCreationTokens += u.CacheCreationTokens
			// Una entrada <synthetic> (error de API, "Prompt is too long")
			// trae el usage en ceros. Tomarla como la última lectura vacía el
			// medidor de contexto justo cuando el contexto está más lleno, así
			// que sólo una medición real reemplaza a la anterior.
			if u.contextSize() > 0 {
				sum.lastUsage = *u
			}
			sum.cost += u.costUSD(sum.model)
		}
		sum.lastEvent, sum.pendingTool = describeAssistant(entry.Message.Content)
	case "user":
		sum.pendingTool = "" // a reply arrived; nothing is pending
	}
}

// describeAssistant extracts a display line and any trailing tool call.
func describeAssistant(raw json.RawMessage) (event, pendingTool string) {
	var blocks []contentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return "", ""
	}
	for _, b := range blocks {
		switch b.Type {
		case "tool_use":
			event = "tool: " + b.Name
			pendingTool = b.Name
		case "text":
			if text := strings.TrimSpace(b.Text); text != "" {
				event = firstLine(text)
				pendingTool = ""
			}
		}
	}
	return event, pendingTool
}

// resolveStatus turns transcript facts into a status, letting hook events
// (precise) outrank heuristics (mtime freshness, unresolved tool calls).
func (c *claudeSource) resolveStatus(sum transcriptSummary, mtime time.Time, blocked map[string]time.Time) (fleet.Status, string, time.Time) {
	if blockedAt, isBlocked := blocked[sum.sessionID]; isBlocked {
		return fleet.StatusNeedsYou, sum.lastEvent, blockedAt
	}
	age := time.Since(mtime)
	switch {
	case sum.pendingTool != "" && age > 10*time.Second:
		return fleet.StatusNeedsYou, fmt.Sprintf("approve: %s?", sum.pendingTool), mtime
	case age < 20*time.Second:
		return fleet.StatusWorking, sum.lastEvent, time.Time{}
	default:
		return fleet.StatusIdle, sum.lastEvent, time.Time{}
	}
}

func firstLine(text string) string {
	if i := strings.IndexByte(text, '\n'); i >= 0 {
		text = text[:i]
	}
	if len(text) > 60 {
		return text[:57] + "..."
	}
	return text
}
