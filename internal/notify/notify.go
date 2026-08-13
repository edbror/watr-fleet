// Package notify escalates blocked sessions beyond the terminal:
// system notification, optional custom command, optional ntfy push.
package notify

import (
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Notifier fires once per blocked session once it crosses the threshold.
type Notifier struct {
	Threshold time.Duration
	NtfyTopic string // ntfy.sh topic for phone push; empty disables
	Command   string // custom shell command; empty disables
}

// Notify announces a blocked session on every configured channel.
// Best-effort by design: a failed channel never breaks the dashboard.
func (n Notifier) Notify(project, event string) {
	title := "fleet: " + project + " needs you"
	go n.systemNotification(title, event)
	if n.Command != "" {
		go func() { _ = exec.Command("sh", "-c", n.Command).Run() }()
	}
	if n.NtfyTopic != "" {
		go n.push(title, event)
	}
}

func (n Notifier) systemNotification(title, body string) {
	switch runtime.GOOS {
	case "darwin":
		script := fmt.Sprintf("display notification %q with title %q sound name \"Ping\"", body, title)
		_ = exec.Command("osascript", "-e", script).Run()
	default:
		if _, err := exec.LookPath("notify-send"); err == nil {
			_ = exec.Command("notify-send", "-u", "critical", title, body).Run()
		}
	}
}

func (n Notifier) push(title, body string) {
	req, err := http.NewRequest("POST", "https://ntfy.sh/"+n.NtfyTopic, strings.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Title", title)
	req.Header.Set("Priority", "high")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err == nil {
		resp.Body.Close()
	}
}
