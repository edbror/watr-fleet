package adapter

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/edbror/watr-fleet/internal/fleet"
)

// Session lifecycle: fleet launches and manages agent sessions so the
// user never touches tmux directly. Managed sessions carry the fleet-
// prefix; fleet only ever kills what it launched.

// ManagedPrefix marks tmux sessions created by fleet.
const ManagedPrefix = "fleet-"

var unsafeSessionChars = regexp.MustCompile(`[^a-zA-Z0-9_-]`)

// LaunchSession starts an agent in a new detached tmux session rooted at
// dir and returns the session name. The session shows a hint on how to
// come back, so tmux stays invisible to the user.
func LaunchSession(agent fleet.Agent, dir string) (string, error) {
	name, err := availableSessionName(agent, dir)
	if err != nil {
		return "", err
	}
	create := exec.Command("tmux", "new-session", "-d", "-s", name, "-c", dir, string(agent))
	if out, err := create.CombinedOutput(); err != nil {
		return "", fmt.Errorf("launching %s: %s", agent, strings.TrimSpace(string(out)))
	}
	// Best-effort quality of life: a visible way home.
	_ = exec.Command("tmux", "set-option", "-t", name,
		"status-right", " ctrl-b d → back to fleet ").Run()
	return name, nil
}

// availableSessionName derives fleet-<project>-<agent>, suffixing a
// counter when the name is taken.
func availableSessionName(agent fleet.Agent, dir string) (string, error) {
	base := ManagedPrefix + sanitize(filepath.Base(dir)) + "-" + sanitize(string(agent))
	existing, err := sessionNames()
	if err != nil {
		return "", err
	}
	if !existing[base] {
		return base, nil
	}
	for i := 2; i < 100; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if !existing[candidate] {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("too many sessions named %s", base)
}

func sessionNames() (map[string]bool, error) {
	names := map[string]bool{}
	out, err := exec.Command("tmux", "list-sessions", "-F", "#{session_name}").Output()
	if err != nil {
		// No server running yet is not an error: the first launch starts it.
		return names, nil
	}
	for _, name := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if name != "" {
			names[name] = true
		}
	}
	return names, nil
}

func sanitize(s string) string {
	clean := unsafeSessionChars.ReplaceAllString(s, "-")
	if clean == "" {
		return "session"
	}
	return clean
}

// KillManagedSession terminates a session fleet launched. Refuses to
// touch sessions the user created themselves.
func KillManagedSession(target string) error {
	session := strings.SplitN(target, ":", 2)[0]
	if !strings.HasPrefix(session, ManagedPrefix) {
		return fmt.Errorf("%s was not launched by fleet", session)
	}
	return exec.Command("tmux", "kill-session", "-t", session).Run()
}
