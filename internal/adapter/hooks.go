package adapter

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// hookLog reads precise session events emitted by Claude Code hooks into
// ~/.fleet/events.jsonl. Configure in ~/.claude/settings.json:
//
//	"hooks": {
//	  "Notification": [{"hooks": [{"type": "command",
//	    "command": "jq -c '{session_id, event: \"Notification\", ts: now}' >> ~/.fleet/events.jsonl"}]}],
//	  "Stop": [{"hooks": [{"type": "command",
//	    "command": "jq -c '{session_id, event: \"Stop\", ts: now}' >> ~/.fleet/events.jsonl"}]}]
//	}
//
// Notification fires when Claude needs your input; Stop when a turn ends.
// Hooks are optional — without them fleet falls back to heuristics.
type hookLog struct {
	path string
}

// NewHookLog points at an events file; missing files are fine.
func NewHookLog(path string) *hookLog {
	return &hookLog{path: path}
}

// DefaultHookLogPath resolves ~/.fleet/events.jsonl.
func DefaultHookLogPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".fleet", "events.jsonl")
}

type hookEvent struct {
	SessionID string  `json:"session_id"`
	Event     string  `json:"event"`
	TS        float64 `json:"ts"` // unix seconds (jq's now)
}

// blockedMap reads the events file ONCE and returns every session whose
// latest event is a Notification (needs input) not yet cleared by a Stop.
// One pass per poll, regardless of how many sessions exist.
func (h *hookLog) blockedMap() map[string]time.Time {
	blocked := map[string]time.Time{}
	if h == nil || h.path == "" {
		return blocked
	}
	file, err := os.Open(h.path)
	if err != nil {
		return blocked
	}
	defer file.Close()

	lastStop := map[string]time.Time{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var ev hookEvent
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil || ev.SessionID == "" {
			continue
		}
		at := time.Unix(int64(ev.TS), 0)
		switch ev.Event {
		case "Notification":
			blocked[ev.SessionID] = at
		case "Stop":
			lastStop[ev.SessionID] = at
		}
	}
	for id, notifiedAt := range blocked {
		if stop, ok := lastStop[id]; ok && stop.After(notifiedAt) {
			delete(blocked, id)
		}
	}
	return blocked
}
