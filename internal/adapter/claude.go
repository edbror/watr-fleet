package adapter

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	cache      map[string]cachedTranscript
}

// cachedTranscript avoids reparsing unchanged transcripts every poll —
// large sessions would otherwise burn CPU on every refresh tick.
type cachedTranscript struct {
	mtime time.Time
	size  int64
	sum   transcriptSummary
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
	transcripts, err := c.recentTranscripts()
	if err != nil {
		return nil, err
	}
	c.pruneCache(transcripts)
	blocked := c.hooks.blockedMap() // one pass over the events file per poll
	var sessions []fleet.Session
	for _, path := range transcripts {
		if s, ok := c.sessionFromTranscript(path, blocked); ok {
			sessions = append(sessions, s)
		}
	}
	return sessions, nil
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
		ContextPct:    clamp01(float64(sum.lastUsage.contextSize()) / float64(c.contextMax)),
		NeedsYouSince: since,
		Dir:           sum.cwd,
	}, true
}

// summarize parses a transcript, reusing the cached pass when the file
// has not changed since the last poll.
func (c *claudeSource) summarize(path string, info os.FileInfo) (transcriptSummary, bool) {
	if hit, ok := c.cache[path]; ok && hit.mtime.Equal(info.ModTime()) && hit.size == info.Size() {
		return hit.sum, true
	}
	file, err := os.Open(path)
	if err != nil {
		return transcriptSummary{}, false
	}
	defer file.Close()
	sum := summarizeTranscript(file)
	c.cache[path] = cachedTranscript{mtime: info.ModTime(), size: info.Size(), sum: sum}
	return sum, true
}

func summarizeTranscript(file *os.File) transcriptSummary {
	var sum transcriptSummary
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 8*1024*1024)
	for scanner.Scan() {
		var entry transcriptEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}
		absorb(&sum, entry)
	}
	return sum
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
			sum.lastUsage = *u
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
