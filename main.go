// fleet — a beautiful terminal dashboard for your fleet of AI coding agents.
//
//	fleet          monitor agents running in tmux
//	fleet --demo   simulated fleet, demoable anywhere
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/edbror/watr-fleet/internal/adapter"
	"github.com/edbror/watr-fleet/internal/config"
	"github.com/edbror/watr-fleet/internal/fleet"
	"github.com/edbror/watr-fleet/internal/notify"
	"github.com/edbror/watr-fleet/internal/ui"
)

// version is stamped at release time with -ldflags "-X main.version=<tag>".
var version string

// resolveVersion prefers the release stamp, then the module version the Go
// toolchain records for `go install ...@latest`, so both install paths report
// something real instead of an empty string.
func resolveVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return info.Main.Version
	}
	return "dev"
}

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	demo := flag.Bool("demo", false, "run with a simulated fleet")
	refresh := flag.Duration("refresh", 800*time.Millisecond, "dashboard refresh interval")
	seed := flag.Int64("seed", 7, "seed for the demo fleet")
	claudeRoot := flag.String("claude-root", adapter.DefaultClaudeRoot(), "Claude Code transcripts directory")
	hookLogPath := flag.String("hook-log", adapter.DefaultHookLogPath(), "fleet hook events file")
	configPath := flag.String("config", config.DefaultPath(), "fleet.toml path")
	flag.Parse()

	// Answer before touching config or tmux: `fleet --version` must work on a
	// machine where neither is set up yet.
	if *showVersion {
		fmt.Println("fleet", resolveVersion())
		return
	}

	if *refresh < 200*time.Millisecond {
		*refresh = 200 * time.Millisecond // floor: protect tmux and CPU from a zero-interval poll loop
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fleet: bad config:", err)
		os.Exit(1)
	}
	applyConfig(cfg)

	source, err := buildSource(*demo, *seed, *claudeRoot, *hookLogPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fleet:", err)
		os.Exit(1)
	}

	notifier := &notify.Notifier{
		Threshold: time.Duration(cfg.Notify.ThresholdSeconds) * time.Second,
		NtfyTopic: cfg.Notify.NtfyTopic,
		Command:   cfg.Notify.Command,
	}
	model := ui.NewModel(source, *refresh).WithNotifier(notifier)
	program := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "fleet:", err)
		os.Exit(1)
	}
}

// applyConfig registers custom agents and pricing overrides.
func applyConfig(cfg config.Config) {
	for name, agent := range cfg.Agents {
		fleet.RegisterAgent(name, agent.Hex, agent.Aliases...)
	}
	for family, p := range cfg.Pricing {
		adapter.OverridePricing(family, p.Input, p.Output, p.CacheRead, p.CacheWrite)
	}
}

// buildSource composes the fleet view: tmux discovers panes, and the
// Claude Code source enriches matching sessions with real telemetry.
func buildSource(demo bool, seed int64, claudeRoot, hookLogPath string) (adapter.Source, error) {
	if demo {
		return adapter.NewDemoFleet(seed), nil
	}
	tmux, err := adapter.NewTmuxSource()
	if err != nil {
		return nil, err
	}
	claude := adapter.NewClaudeSource(claudeRoot, adapter.NewHookLog(hookLogPath))
	return adapter.NewMultiSource(tmux, claude), nil
}
