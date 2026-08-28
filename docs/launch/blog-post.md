# You're running five coding agents. Which one is waiting on you right now?

*We built a terminal dashboard to answer that. It's called Fleet, it's open source, and it works with whatever agents you already run.*

If you run more than one AI coding agent at a time, you already know the failure mode. Claude Code is refactoring in one tmux pane. Codex is writing tests in another. Something called Aider is halfway through a migration in a third. You alt-tab between them like a short-order cook, and half the time you land on a pane that's been sitting on a `y/n` prompt for four minutes, blocked, waiting on you — while the other two burned tokens you weren't watching.

Session managers fix the visibility half of this: they show you the panes. Fleet goes at the part that actually costs you — **which agent needs you, and what the whole fleet is costing while you look away.**

## What it is

Fleet is a terminal dashboard (a TUI) for a fleet of AI coding agents. It's written in Go on the [Charm](https://charm.sh) stack — Bubble Tea and Lip Gloss — and it's MIT-licensed. It watches the agents you're already running and gives you three things:

**An attention router.** Every blocked session lands in one queue, longest-waiting first. When an agent is stuck on a prompt, you answer `y`/`n` from the dashboard without switching to its pane. The thing that's been quietly wasting your afternoon — an agent parked on a confirmation you never saw — becomes the first thing you see.

**Fleet telemetry.** Real tokens, cost, burn rate, throughput, and context-window pressure. It's `htop` for agents, not another window switcher. When something is 90% through its context window or quietly running up a bill, that shows up as a number, not a surprise.

**A dashboard mode.** Press `d` and you get token flow, cost distribution, and context pressure across the whole fleet — the movie-computer view. We think aesthetics is distribution for an open-source tool, so yes, there are gradients and it's meant to look good on a second monitor.

## How it actually knows

This is the part worth being honest about, because "monitors all your agents" can mean a lot of things.

Fleet finds agents two ways. It reads **tmux** and identifies agent CLIs by process name *and* by walking the process tree — so a Python- or Node-launched agent gets identified by the script it's running, not hidden behind `node`. That's how it detects 24+ agent CLIs: Claude Code, Codex, Gemini, Crush, Aider, Cursor Agent, Goose, OpenCode, Grok, Kimi, Antigravity, Kiro, and more.

The **deep** telemetry — exact tokens, real cost, context pressure — comes today from Claude Code, because its transcripts (`~/.claude/projects`) are on disk and parseable, and an optional hook gives an exact "this session needs you" signal. For the rest of the fleet, tmux detection gives you presence and activity now; richer per-agent telemetry is the roadmap, not a claim about today. We'd rather tell you where the line is than pretend it isn't there.

That roadmap is Grok Build (headless JSON + ACP), the OpenCode server API, and eventually [ACP](https://agentclientprotocol.com) as a universal layer so every agent reports the same way.

## Try it in five seconds

There's a demo mode that needs nothing running — a simulated fleet you can drive on any machine:

```bash
brew install edbror/tap/watr-fleet   # or: go install github.com/edbror/watr-fleet@latest
fleet --demo
```

Then `fleet` on its own discovers the real agents in your tmux and Claude Code sessions.

The keys are boring on purpose: `j`/`k` to move, `enter` to jump to that pane, `y`/`n` to approve or deny the selected prompt, `d` for the dashboard, `s` to cycle the sort (attention / cost / project), `q` to quit.

There's also escalation for when you walk away: past a threshold, a blocked session fires a system notification, an optional shell command (a sound, if you want one), or an [ntfy.sh](https://ntfy.sh) push to your phone.

## Honest status

Fleet is **v0.10.4, pre-release.** The core loop works and the demo is real, but the detection heuristics are young — every agent CLI has its own way of signalling "I'm blocked," and we've only mapped some of them. That's exactly where you come in: if you run an agent Fleet reads wrong, an issue with your agent's actual prompt patterns is the single most valuable thing you can send us. Same for a PR.

It's open source because a tool that watches *your* fleet should be one you can read, fork, and correct.

**Repo, issues, and install:** [github.com/edbror/watr-fleet](https://github.com/edbror/watr-fleet)

*Built by [WATR](https://watr.mx).*
