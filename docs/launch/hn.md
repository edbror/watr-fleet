# Launch kit — Fleet (Show HN + Reddit)

**What ships:** an open-source Go TUI that watches your fleet of AI coding agents — who's working, who's blocked waiting on you, what it's costing.
**Launch vehicle:** the GitHub repo. For an OSS tool the repo *is* the landing page — README, GIF, and a one-line install are the whole funnel.
**Destination:** `github.com/edbror/watr-fleet`
**The asset:** a short loop of `fleet --demo` running — the attention queue reordering, the dashboard (`d`) with token flow and context pressure. That GIF is what makes people run one command.

> Honesty line, lead with it everywhere: **v0.10.4, pre-release.** Deep telemetry (real tokens/cost/context pressure) is richest for Claude Code today via its on-disk transcripts; the other 24+ CLIs are detected via tmux (presence + activity) with richer per-agent telemetry on the roadmap. Say that plainly — this crowd rewards it and punishes the opposite.

---

## 0. Pre-flight — MUST be done before any post

This is the fire order. Nothing below goes out until these are true.

- [x] **Repo público: `github.com/edbror/watr-fleet`.** Verificado vía API: `private: false`. README renderiza en la portada.
- [x] **Homebrew tap: `edbror/homebrew-tap`.** La fórmula existe (`watr-fleet.rb`, `version "0.10.4"`) con binarios para darwin/linux × amd64/arm64. **Ojo: el comando es `brew install edbror/tap/watr-fleet`, no `.../fleet`** — la fórmula se llama por su archivo. Falta confirmarlo en una máquina limpia.
- [x] **GIF grabado.** `docs/tour.gif` y `docs/opensea.gif` commiteados.
- [x] **LICENSE + go.mod.** MIT en el repo; module path `github.com/edbror/watr-fleet`.
- [x] **CI verde.** El workflow de release pasó en v0.10.4 (2026-08-15, 1m11s).
- [ ] **Cuenta de Reddit con historial.** Tuyo — una cuenta nueva soltando un link de GitHub se auto-remueve en la mayoría de los subs.
- [ ] **Estar al teclado 2 h después de cada post.** La primera hora de comentarios decide todo.

> Estado real a hoy: **v0.10.4 publicada desde el 15-ago**, repo público, tap funcionando, GIFs listos, CI verde. Lo único que falta del pre-flight es tuyo (cuenta de Reddit y disponibilidad). El post nunca se disparó.

---

## 1. Cadence — stagger, don't blast

Same rule as any launch: posting the same link everywhere in one hour gets you flagged. Space it, and write each one fresh — never paste identical bodies.

| Day | Channel | Why this order |
|-----|---------|----------------|
| **Day 1** | **Show HN** | The main event. Do it on a weekday morning US-Eastern. Front-load your best comment (the "how it detects agents" detail) as the first reply. |
| **Day 2** | **r/commandline** | Loves a good TUI and a GIF. Lowest-friction win; also sharpens your FAQ before the tougher crowd. |
| **Day 3** | **r/golang** | Technical and skeptical about the *code*, not the idea. Post only after Day 1-2 comments are answered and CI is visibly green. Lead with the stack and the honest limitations. |

Skip r/programming (removes almost all self-promo) and the AI hype subs (they'll argue about agents instead of running the tool). Revisit r/selfhosted or r/tmux later if Days 1-3 land.

**One post per channel. Never crosspost identical text.**

---

## 2. The posts

### Show HN — Day 1

**Title:** Show HN: Fleet – a terminal dashboard for your fleet of AI coding agents

**URL:** `https://github.com/edbror/watr-fleet`

**Text (first comment, posted by you immediately):**
> I run several coding agents at once — Claude Code refactoring in one tmux pane, Codex writing tests in another, Aider mid-migration in a third — and I kept losing time to the same thing: alt-tabbing into a pane that had been sitting on a y/n prompt for minutes, blocked, while the others quietly burned tokens. Session managers show you the panes; they don't tell you *which* agent needs you or what the fleet is costing.
>
> Fleet is a Go TUI (Bubble Tea + Lip Gloss, MIT) that does three things:
> - **Attention router:** every blocked session in one queue, longest-waiting first; you answer y/n from the dashboard without switching panes.
> - **Telemetry:** real tokens, cost, burn rate, throughput, context-window pressure — htop for agents.
> - **Dashboard mode** (`d`): token flow, cost distribution, context pressure across the fleet.
>
> How it detects agents, honestly: it reads tmux and identifies CLIs by process name *and* by walking the process tree (so a node/python-launched agent is identified by its script, not hidden behind the interpreter) — that's 24+ CLIs. The *deep* telemetry (exact tokens/cost/context) comes today from Claude Code, because its transcripts are on disk and parseable, plus an optional hook for an exact "needs you" signal. For the rest, tmux gives presence + activity now; richer per-agent telemetry (Grok Build headless JSON, OpenCode's server API, ACP as a universal layer) is the roadmap. I'd rather draw that line than blur it.
>
> Try it with nothing running:
> `go install github.com/edbror/watr-fleet@latest` (or `brew install edbror/tap/watr-fleet`), then `fleet --demo`. Plain `fleet` discovers your real tmux + Claude Code sessions.
>
> It's v0.10.4, pre-release — the detection heuristics are young, and every agent signals "blocked" differently. If you run one Fleet reads wrong, an issue with your agent's actual prompt patterns is the most useful thing you can send. Happy to answer anything about the tmux process-tree walk, the transcript parsing, or the cost model.

---

### r/commandline — Day 2 (lead with the GIF)

**Title:** Fleet — a TUI that shows your whole fleet of AI coding agents: who's working, who's blocked waiting on you, what it costs

**Body:** *(this sub wants the thing itself — GIF first, minimal preamble)*
> If you run more than one coding agent at a time (Claude Code, Codex, Gemini, Aider…), the annoying part isn't seeing the panes — it's not knowing *which* one is stuck waiting on a y/n prompt while the others run up tokens. Fleet puts every blocked session in one queue (longest-waiting first), lets you answer y/n without leaving the dashboard, and shows real tokens/cost/context pressure. `d` toggles a full dashboard view.
>
> Go + Bubble Tea + Lip Gloss, MIT. Runs with nothing installed:
> `go install github.com/edbror/watr-fleet@latest && fleet --demo`
>
> **github.com/edbror/watr-fleet**
>
> v0.10.4, pre-release — detection heuristics are young, issues with your agent's prompt patterns are gold. GIF is the demo mode; keys are j/k, enter, y/n, d, s, q.

---

### r/golang — Day 3 (technical, about the code)

**Title:** Fleet: a Bubble Tea TUI for monitoring a fleet of AI coding agents (tmux process-tree detection + transcript parsing)

**Body:**
> Sharing a TUI I built on the Charm stack (Bubble Tea + Lip Gloss), MIT. It monitors multiple AI coding agents running in parallel and surfaces which one is blocked waiting on you, plus live cost/token/context telemetry.
>
> The Go bits this sub might care about:
> - **Agent detection** walks the process tree from tmux panes, so a `node`- or `python`-launched agent is identified by the script it's actually running rather than the interpreter. ~24 CLIs mapped so far.
> - **Telemetry** for Claude Code comes from parsing its on-disk transcripts (`~/.claude/projects`) for real token/cost/context numbers, with an optional hook file for an exact "needs-you" event. Other agents are presence/activity via tmux today; a source interface (`adapter.Source`, composed via a multi-source) is there so new backends — Grok Build headless JSON, OpenCode's server API, eventually ACP — plug in without touching the UI.
> - Single static binary, `CGO_ENABLED=0`, GoReleaser + Homebrew tap.
>
> `go install github.com/edbror/watr-fleet@latest`, then `fleet --demo` for a simulated fleet (deterministic seed, so it's demoable anywhere). Code: **github.com/edbror/watr-fleet**
>
> It's v0.10.4 and the detection heuristics are the weakest part — genuinely looking for critique of the process-tree approach and the transcript parsing, and PRs/issues for agents I've mapped wrong. What would you do differently for the "is this session blocked?" signal across heterogeneous CLIs?

---

## 3. Comment kit — anticipate & answer fast

**"Doesn't [session manager / tmux / Zellij] already do this?"** → Those show you the panes — visibility. Fleet is about *triage*: one queue of what's blocked waiting on you, ranked, answerable in place, plus cost/context telemetry you don't get from a pane list. Complementary, not a replacement — it reads your existing tmux.

**"Does it really work with all those agents, or just Claude?"** → Two layers, and I try to be clear about it. Detection (is this agent running, is it active) works across 24+ CLIs via tmux + process-tree walk. Deep telemetry (exact tokens/cost/context) is Claude Code today because its transcripts are on disk. Broader per-agent telemetry is the roadmap (Grok Build, OpenCode API, ACP). I won't claim more than that.

**"How does the y/n approval actually reach the agent?"** → The selected session's prompt is answered through its tmux pane — Fleet sends the keystroke to the right pane so you don't have to switch to it. Happy to go into the plumbing.

**"Is it sending my code/tokens anywhere?"** → No. It reads local tmux and local Claude Code transcripts on your machine. The only outbound thing is optional: an ntfy.sh push *you* configure for phone notifications. It's open source — the network surface is auditable in the repo.

**"Why Go / why a TUI and not a web dashboard?"** → Single static binary, no runtime, lives where the agents already live (the terminal). Bubble Tea makes the animated stuff cheap. A web UI means a server and a browser tab for something that should be one `htop`-shaped command.

**"How's cost calculated?"** → Token counts from the transcript × per-model pricing, overridable in `fleet.toml` (`[pricing.<family>]`). It's an estimate, and I say so — good enough to catch a runaway, not an invoice.

**"Is it open source / can I add my agent?"** → Yes, MIT. Adding or recoloring an agent is a few lines of config (`fleet.example.toml`); teaching Fleet a new *blocked-prompt* pattern is the high-value PR. Issues with your agent's real prompt text are exactly what I need.

**Don't:** oversell the multi-agent telemetry, call it finished, or argue that it replaces someone's setup. "Early, here's exactly what works today, here's the roadmap, tell me what breaks" wins this crowd.

---

## 4. After posting

- Watch GitHub stars + issues, not vanity metrics. A single well-written "here's how my agent signals blocked" issue is worth more than 50 stars.
- Pin the honest-status paragraph so late arrivals don't relitigate "does it work with X."
- Save every "it read my agent wrong" report — that's the detection-heuristics backlog and the raw material for v0.11.
- If Show HN or one sub pops, a follow-up "what I learned building a TUI for AI agents" post a week later usually outperforms the launch.

---

*Pre-flight técnico: hecho (repo público, tap v0.10.4, GIFs, CI verde). Falta solo: verificar `brew install edbror/tap/watr-fleet` en una máquina limpia, elegir la mañana entre semana, y estar al teclado. Luego postear.*
