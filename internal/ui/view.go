package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/edbror/watr-fleet/internal/fleet"
)

// Crush-inflected layout: color chips, tinted attention band, a rounded
// detail card, and a full-width segmented status bar. Hierarchy from
// color fields and spacing, not from boxing every section.

const version = "v0.10.2"

const (
	colAgent = 10 // pill: space + 8 name + space
	colState = 10
	colTok   = 7
	colCost  = 8
	colBar   = 8
	colSpark = 8
)

// cell is one styled fragment of a composed line; a background can be
// applied to every cell at render time (full-row highlights, tints).
type cell struct {
	style lipgloss.Style
	text  string
}

func renderCells(cells []cell, bg string) string {
	var b strings.Builder
	for _, c := range cells {
		st := c.style
		if bg != "" {
			st = st.Background(lipgloss.Color(bg))
		}
		b.WriteString(st.Render(c.text))
	}
	return b.String()
}

// layout carries the responsive column plan for the current width.
type layout struct {
	width      int
	project    int
	event      int
	showSpark  bool
	showCtx    bool
	showMoney  bool
	showDetail bool
}

func (m Model) layout() layout {
	w := m.width
	if w <= 0 {
		w = 110
	}
	l := layout{
		width:      w,
		project:    16,
		showSpark:  w >= 112,
		showCtx:    w >= 96,
		showMoney:  w >= 78,
		showDetail: m.height == 0 || m.height >= 22,
	}
	fixed := 4 + colAgent + 1 + colState + 1
	if l.showMoney {
		fixed += colTok + 1 + colCost + 1
	}
	if l.showCtx {
		fixed += colBar + 2
	}
	if l.showSpark {
		fixed += colSpark + 2
	}
	l.event = maxInt(w-fixed-l.project-3, 16)
	return l
}

func (m Model) View() string {
	if m.err != nil {
		return "\n " + styleAmber.Render("⚠ "+m.err.Error()) + "\n " +
			styleFaint.Render("is tmux running? try --demo for the simulated fleet") + "\n"
	}
	l := m.layout()
	if m.monitor {
		return m.viewMonitor(l)
	}
	var b strings.Builder
	if m.height == 0 || m.height >= 34 {
		b.WriteString("\n" + m.viewLogo(l) + "\n")
	} else {
		b.WriteString("\n" + m.viewHeader(l) + "\n")
	}
	b.WriteString(" " + wave(l.width-2, m.frame) + "\n")
	b.WriteString(m.viewFleetBadges(l) + "\n\n")
	b.WriteString(m.viewAttentionQueue(l))
	if m.dashboard {
		b.WriteString(m.viewDashboard(l))
	} else {
		b.WriteString(m.viewFleetTable(l))
		switch {
		case m.launcher != nil:
			b.WriteString(m.viewLauncher(l))
		case l.showDetail:
			b.WriteString(m.viewDetail(l))
		}
	}
	b.WriteString(m.viewStatusBar(l))
	return b.String()
}

func (m Model) viewHeader(l layout) string {
	title := gradientPhase("≈ FLEET", colDeep, colViolet, float64(m.frame)/60.0)
	ver := chip(" "+version+" ", darken(colViolet, 0.7), colViolet)
	sum := fleet.Summarize(m.sessions)
	stats := fmt.Sprintf("  %d vessels · %s working · %s need you · %s idle",
		sum.Total,
		statusStyle(fleet.StatusWorking).Render(fmt.Sprint(sum.Working)),
		statusStyle(fleet.StatusNeedsYou).Render(fmt.Sprint(sum.NeedsYou)),
		statusStyle(fleet.StatusIdle).Render(fmt.Sprint(sum.Idle)),
	)
	return " " + title + " " + ver + styleInk.Render(stats)
}

// logoRows is FLEET in half-block letters — the maximalist wordmark.
var logoRows = []string{
	"█▀▀▀ █    █▀▀▀ █▀▀▀ ▀▀█▀▀",
	"█▀▀  █    █▀▀  █▀▀    █",
	"▀    ▀▀▀▀ ▀▀▀▀ ▀▀▀▀   ▀",
}

// viewLogo renders the block wordmark with an animated gradient, plus
// the fleet stats to its right.
func (m Model) viewLogo(l layout) string {
	sum := fleet.Summarize(m.sessions)
	side := []string{
		" " + chip(" "+version+" ", darken(colViolet, 0.7), colViolet) +
			styleFaint.Render("  every agent, one horizon"),
		" " + styleInk.Render(fmt.Sprintf("%d vessels · ", sum.Total)) +
			statusStyle(fleet.StatusWorking).Render(fmt.Sprintf("%d working", sum.Working)) +
			styleInk.Render(" · ") +
			statusStyle(fleet.StatusNeedsYou).Render(fmt.Sprintf("%d need you", sum.NeedsYou)) +
			styleInk.Render(" · ") +
			statusStyle(fleet.StatusIdle).Render(fmt.Sprintf("%d idle", sum.Idle)),
		" " + styleFaint.Render("["+m.source.Name()+"]"),
	}
	var b strings.Builder
	for i, row := range logoRows {
		phase := float64(m.frame)/50.0 + float64(i)*0.12
		b.WriteString(" " + gradientPhase(row, colDeep, colViolet, phase) + side[i])
		if i < len(logoRows)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// viewFleetBadges is the maximalist roster line: one pill per agent type
// with its session count.
func (m Model) viewFleetBadges(l layout) string {
	counts := map[fleet.Agent]int{}
	var order []fleet.Agent
	for _, s := range m.sessions {
		if counts[s.Agent] == 0 {
			order = append(order, s.Agent)
		}
		counts[s.Agent]++
	}
	var b strings.Builder
	b.WriteString(" ")
	for _, a := range order {
		b.WriteString(agentPill(a) + styleFaint.Render(fmt.Sprintf("×%d ", counts[a])))
	}
	return b.String()
}

// chip renders a small color pill: dark text on a colored field.
func chip(text, bg, fg string) string {
	return lipgloss.NewStyle().
		Background(lipgloss.Color(bg)).
		Foreground(lipgloss.Color(fg)).
		Bold(true).
		Render(text)
}

// agentPill is the agent badge: abyss text on the agent's color.
func agentPill(a fleet.Agent) string {
	name := string(a)
	if len(name) > 8 {
		name = name[:8]
	}
	bg := fleet.HexFor(a)
	return lipgloss.NewStyle().
		Background(lipgloss.Color(darken(bg, 0.15))).
		Foreground(lipgloss.Color(colAbyss)).
		Bold(true).
		Render(" " + padRight(name, 8) + " ")
}

// viewAttentionQueue is the interrupt manager rendered as a glowing amber
// band: tinted background, pulsing accent, longest wait first.
func (m Model) viewAttentionQueue(l layout) string {
	now := time.Now()
	var blocked []fleet.Session
	for _, s := range m.sessions {
		if s.Status == fleet.StatusNeedsYou {
			blocked = append(blocked, s)
		}
	}
	if len(blocked) == 0 {
		return ""
	}
	tint := darken(colAmber, 0.88)
	pulse := pulseAmber(m.frame)
	var b strings.Builder
	header := []cell{
		{pulse, " ▌ NEEDS YOU"},
		{styleFaint, fmt.Sprintf(" · %d waiting", len(blocked))},
	}
	b.WriteString(padBand(header, l.width, tint) + "\n")
	for _, s := range blocked {
		row := []cell{
			{pulse, " ▌ "},
			{styleBright.Bold(true), padRight(s.Project, l.project)},
			{styleAmber, " " + padLeft(humanDuration(s.WaitingFor(now)), 7)},
			{styleInk, "  " + truncate(s.LastEvent, l.event+24)},
		}
		b.WriteString(padBand(row, l.width, tint) + "\n")
	}
	b.WriteString("\n")
	return b.String()
}

// padBand renders cells over a full-width tinted background.
func padBand(cells []cell, width int, bg string) string {
	line := renderCells(cells, bg)
	gap := width - lipgloss.Width(line)
	if gap > 0 {
		line += lipgloss.NewStyle().Background(lipgloss.Color(bg)).Render(strings.Repeat(" ", gap))
	}
	return line
}

func (m Model) viewFleetTable(l layout) string {
	if len(m.sessions) == 0 {
		return " " + styleFaint.Render("no vessels at sea — press ") +
			styleCyan.Bold(true).Render("n") +
			styleFaint.Render(" to launch your first agent, or run with --demo") + "\n\n"
	}
	var b strings.Builder
	b.WriteString(m.viewTableHeader(l) + "\n")
	b.WriteString(" " + styleFaint.Render(strings.Repeat("─", l.width-2)) + "\n")
	for i, s := range m.sessions {
		b.WriteString(m.viewRow(i, s, l) + "\n")
	}
	b.WriteString("\n")
	return b.String()
}

func (m Model) viewTableHeader(l layout) string {
	h := "    " + padRight("PROJECT", l.project) + " " +
		padRight("AGENT", colAgent) + " " +
		padRight("STATUS", colState) + " "
	if l.showMoney {
		h += padLeft("TOKENS", colTok) + " " + padLeft("COST", colCost) + " "
	}
	if l.showCtx {
		h += padRight("CONTEXT", colBar) + "  "
	}
	if l.showSpark {
		h += padRight("ACTIVITY", colSpark) + "  "
	}
	h += "LAST EVENT"
	return styleFaint.Render(h)
}

func (m Model) viewRow(i int, s fleet.Session, l layout) string {
	selected := i == m.cursor
	bg := backgroundIf(selected)
	cursor := " "
	if selected {
		cursor = "▍"
	}
	lead := []cell{
		{styleCyan, " " + cursor},
		{statusStyle(s.Status), statusGlyph(s.Status, m.frame) + " "},
		{styleBright, padRight(s.Project, l.project) + " "},
	}
	mid := []cell{
		{lipgloss.NewStyle(), " "},
		{statusStyle(s.Status), padRight(s.Status.String(), colState) + " "},
	}
	if l.showMoney {
		mid = append(mid,
			cell{styleInk, padLeft(tokensLabel(s.Tokens), colTok) + " "},
			cell{styleInk, padLeft(costLabel(s.CostUSD), colCost) + " "},
		)
	}
	// The pill, context bar, and sparkline carry their own colors and stay
	// unhighlighted; selection reads from the surrounding field.
	row := renderCells(lead, bg) + agentPill(s.Agent) + renderCells(mid, bg)
	if l.showCtx {
		row += contextBar(s.ContextPct, colBar) + "  "
	}
	if l.showSpark {
		row += sparkline(m.activity[s.ID], colSpark) + "  "
	}
	row += renderCells([]cell{{styleFaint, truncate(s.LastEvent, l.event)}}, bg)
	return row
}

func backgroundIf(selected bool) string {
	if selected {
		return colSurface
	}
	return ""
}

// viewDetail is the selected session as a rounded card — the one boxed
// element on screen, so focus reads instantly.
func (m Model) viewDetail(l layout) string {
	if m.cursor >= len(m.sessions) {
		return ""
	}
	s := m.sessions[m.cursor]
	label := func(k string) string { return styleFaint.Render(k + " ") }
	title := agentPill(s.Agent) + styleBright.Bold(true).Render(" "+s.Project) +
		"  " + statusStyle(s.Status).Render(statusGlyph(s.Status, m.frame)+" "+s.Status.String())
	line1 := label("dir") + styleInk.Render(valueOrDash(s.Dir)) +
		"   " + label("target") + styleInk.Render(valueOrDash(s.Target))
	line2 := label("tokens") + styleBright.Render(tokensLabel(s.Tokens)) +
		"   " + label("cost") + styleBright.Render(costLabel(s.CostUSD)) +
		"   " + label("context") + contextPctLabel(s.ContextPct) + " " + contextBar(s.ContextPct, 10)
	line3 := label("event") + styleInk.Render(truncate(s.LastEvent, l.width-16))
	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(darken(colDeep, 0.35))).
		Padding(0, 1).
		Width(l.width - 3)
	return " " + strings.ReplaceAll(
		card.Render(title+"\n"+line1+"\n"+line2+"\n"+line3), "\n", "\n ") + "\n\n"
}

// viewStatusBar is the Crush signature: a full-width bar of color segments.
func (m Model) viewStatusBar(l layout) string {
	sum := fleet.Summarize(m.sessions)
	fillBg := darken(colSurface, 0.45)
	left := chip(" ≈ fleet ", darken(colDeep, 0.55), colCyan) +
		chip(" "+m.source.Name()+" ", fillBg, colSlate) +
		chip(" sort:"+m.sort.String()+" ", fillBg, colSlate)
	money := ""
	if sum.Tokens > 0 {
		money = chip(fmt.Sprintf(" %s tok · $%.2f ", compactInt(sum.Tokens), sum.CostUSD),
			darken(colViolet, 0.65), colViolet)
	}
	if m.flash != "" {
		money += chip(" "+m.flash+" ", darken(colGreen, 0.6), colGreen)
	}
	right := chip(" n new ", fillBg, colCyan) + chip(" y/N answer ", fillBg, colInk) +
		chip(" ⏎ jump ", fillBg, colInk) + chip(" d dash ", fillBg, colInk) +
		chip(" m sea ", fillBg, colViolet) + chip(" q quit ", fillBg, colInk)
	used := lipgloss.Width(left) + lipgloss.Width(money) + lipgloss.Width(right)
	if used > l.width && money != "" {
		money = "" // narrow terminal: drop telemetry chip before wrapping the bar
		used = lipgloss.Width(left) + lipgloss.Width(right)
	}
	gap := maxInt(l.width-used, 0)
	filler := lipgloss.NewStyle().Background(lipgloss.Color(fillBg)).
		Render(strings.Repeat(" ", gap))
	return left + money + filler + right
}

func contextPctLabel(pct float64) string {
	if pct < 0 {
		return styleInk.Render("—")
	}
	color := lerpHex(colDeep, colViolet, pct)
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).
		Render(fmt.Sprintf("%.0f%%", pct*100))
}

func valueOrDash(v string) string {
	if v == "" {
		return "—"
	}
	return v
}

func tokensLabel(tokens int) string {
	if tokens == 0 {
		return "—"
	}
	return compactInt(tokens)
}

func costLabel(cost float64) string {
	switch {
	case cost == 0:
		return "—"
	case cost >= 1_000:
		// Past a grand, cents are noise: $31,834 reads, $31834.22 alarms.
		return "$" + groupThousands(int(cost+0.5))
	default:
		return fmt.Sprintf("$%.2f", cost)
	}
}

func compactInt(n int) string {
	switch {
	case n >= 1_000_000_000:
		return fmt.Sprintf("%.2fB", float64(n)/1_000_000_000)
	case n >= 100_000_000:
		return fmt.Sprintf("%.0fM", float64(n)/1_000_000)
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.0fk", float64(n)/1_000)
	}
	return fmt.Sprint(n)
}

// groupThousands renders 31834 as "31,834".
func groupThousands(n int) string {
	s := fmt.Sprint(n)
	if n < 0 || len(s) <= 3 {
		return s
	}
	var b strings.Builder
	pre := len(s) % 3
	if pre > 0 {
		b.WriteString(s[:pre])
	}
	for i := pre; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

func humanDuration(d time.Duration) string {
	switch {
	case d >= time.Hour:
		return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
	case d >= time.Minute:
		return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
	default:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
}

// padRight, padLeft and truncate are rune-safe: project names and events
// with accents or CJK must not break column alignment.
func padRight(s string, width int) string {
	runes := []rune(s)
	if len(runes) >= width {
		return string(runes[:width])
	}
	return s + strings.Repeat(" ", width-len(runes))
}

func padLeft(s string, width int) string {
	runes := []rune(s)
	if len(runes) >= width {
		return s
	}
	return strings.Repeat(" ", width-len(runes)) + s
}

func truncate(s string, width int) string {
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	if width < 2 {
		return string(runes[:width])
	}
	return string(runes[:width-1]) + "…"
}
