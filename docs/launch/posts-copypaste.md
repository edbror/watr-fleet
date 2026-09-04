# Fleet — copy-paste (Show HN + Reddit)

Texto plano, listo para pegar (sin `>`). Estrategia/comment-kit completo: ver `hn.md`.
Orden: Show HN (día 1, mañana entre semana US-Eastern) → r/commandline (día 2) → r/golang (día 3).
Repo = la landing: github.com/edbror/watr-fleet. Honestidad: v0.10.5 pre-release; telemetría profunda hoy = Claude Code, resto vía tmux (dilo de frente).

═══════════════════════════════════════════
## 1 · Show HN  (día 1)

TÍTULO
Show HN: Fleet – a terminal dashboard for your fleet of AI coding agents

URL
https://github.com/edbror/watr-fleet

PRIMER COMENTARIO (pégalo tú mismo, apenas postees)
I run several coding agents at once — Claude Code refactoring in one tmux pane, Codex writing tests in another, Aider mid-migration in a third — and I kept losing time to the same thing: alt-tabbing into a pane that had been sitting on a y/n prompt for minutes, blocked, while the others quietly burned tokens. Session managers show you the panes; they don't tell you which agent needs you or what the fleet is costing.

Fleet is a Go TUI (Bubble Tea + Lip Gloss, MIT) that does three things:
- Attention router: every blocked session in one queue, longest-waiting first; you answer y/n from the dashboard without switching panes.
- Telemetry: real tokens, cost, burn rate, throughput, context-window pressure — htop for agents.
- Dashboard mode (d): token flow, cost distribution, context pressure across the fleet.

How it detects agents, honestly: it reads tmux and identifies CLIs by process name and by walking the process tree (so a node/python-launched agent is identified by its script, not hidden behind the interpreter) — that's 24+ CLIs. The deep telemetry (exact tokens/cost/context) comes today from Claude Code, because its transcripts are on disk and parseable, plus an optional hook for an exact "needs you" signal. For the rest, tmux gives presence + activity now; richer per-agent telemetry (Grok Build headless JSON, OpenCode's server API, ACP as a universal layer) is the roadmap. I'd rather draw that line than blur it.

Try it with nothing running:
go install github.com/edbror/watr-fleet@latest (or brew install edbror/tap/watr-fleet), then fleet --demo. Plain fleet discovers your real tmux + Claude Code sessions.

It's v0.10.5, pre-release — the detection heuristics are young, and every agent signals "blocked" differently. If you run one Fleet reads wrong, an issue with your agent's actual prompt patterns is the most useful thing you can send. Happy to answer anything about the tmux process-tree walk, the transcript parsing, or the cost model.

═══════════════════════════════════════════
## 2 · r/commandline  (día 2 — lidera con el GIF)

TÍTULO
Fleet — a TUI that shows your whole fleet of AI coding agents: who's working, who's blocked waiting on you, what it costs

CUERPO
If you run more than one coding agent at a time (Claude Code, Codex, Gemini, Aider…), the annoying part isn't seeing the panes — it's not knowing which one is stuck waiting on a y/n prompt while the others run up tokens. Fleet puts every blocked session in one queue (longest-waiting first), lets you answer y/n without leaving the dashboard, and shows real tokens/cost/context pressure. d toggles a full dashboard view.

Go + Bubble Tea + Lip Gloss, MIT. Runs with nothing installed:
go install github.com/edbror/watr-fleet@latest && fleet --demo

github.com/edbror/watr-fleet

v0.10.5, pre-release — detection heuristics are young, issues with your agent's prompt patterns are gold. GIF is the demo mode; keys are j/k, enter, y/n, d, s, q.

═══════════════════════════════════════════
## 3 · r/golang  (día 3 — técnico, sobre el código)

TÍTULO
Fleet: a Bubble Tea TUI for monitoring a fleet of AI coding agents (tmux process-tree detection + transcript parsing)

CUERPO
Sharing a TUI I built on the Charm stack (Bubble Tea + Lip Gloss), MIT. It monitors multiple AI coding agents running in parallel and surfaces which one is blocked waiting on you, plus live cost/token/context telemetry.

The Go bits this sub might care about:
- Agent detection walks the process tree from tmux panes, so a node- or python-launched agent is identified by the script it's actually running rather than the interpreter. ~24 CLIs mapped so far.
- Telemetry for Claude Code comes from parsing its on-disk transcripts (~/.claude/projects) for real token/cost/context numbers, with an optional hook file for an exact "needs-you" event. Other agents are presence/activity via tmux today; a source interface (adapter.Source, composed via a multi-source) is there so new backends — Grok Build headless JSON, OpenCode's server API, eventually ACP — plug in without touching the UI.
- Single static binary, CGO_ENABLED=0, GoReleaser + Homebrew tap.

go install github.com/edbror/watr-fleet@latest, then fleet --demo for a simulated fleet (deterministic seed, so it's demoable anywhere). Code: github.com/edbror/watr-fleet

It's v0.10.5 and the detection heuristics are the weakest part — genuinely looking for critique of the process-tree approach and the transcript parsing, and PRs/issues for agents I've mapped wrong. What would you do differently for the "is this session blocked?" signal across heterogeneous CLIs?
