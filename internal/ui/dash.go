package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/edbror/watr-fleet/internal/fleet"
)

// Dashboard mode: the movie-computer view. Big token-flow area chart,
// stat tiles, per-agent distribution bars, and context pressure gauges.

const flowChartHeight = 6

func (m Model) viewDashboard(l layout) string {
	var b strings.Builder
	b.WriteString(m.viewStatTiles(l) + "\n")
	b.WriteString(sectionTitle("TOKEN FLOW", l.width) + "\n")
	b.WriteString(m.viewFlowChart(l) + "\n")
	b.WriteString(sectionTitle("FLEET DISTRIBUTION", l.width) + "\n")
	b.WriteString(m.viewAgentBars(l))
	b.WriteString(sectionTitle("CONTEXT PRESSURE", l.width) + "\n")
	b.WriteString(m.viewContextGauges(l))
	return b.String()
}

func sectionTitle(title string, width int) string {
	label := " " + styleFaint.Render("── ") +
		styleInk.Bold(true).Render(title) + " " + styleFaint.Render(strings.Repeat("─", maxInt(width-len(title)-7, 0)))
	return label
}

// viewStatTiles is the big-numbers row: fleet totals at a glance.
func (m Model) viewStatTiles(l layout) string {
	sum := fleet.Summarize(m.sessions)
	burn := m.burnPerMinute()
	tiles := []struct {
		label, value, hex string
	}{
		{"TOKENS", compactInt(sum.Tokens), colCyan},
		{"COST", fmt.Sprintf("$%.2f", sum.CostUSD), colViolet},
		{"BURN", compactInt(burn) + "/min", colDeep},
		{"THROUGHPUT", throughputLabel(burn), colGreen},
		{"BLOCKED", fmt.Sprint(sum.NeedsYou), colAmber},
	}
	var parts []string
	for _, t := range tiles {
		tile := lipgloss.NewStyle().
			Background(lipgloss.Color(darken(t.hex, 0.82))).
			Padding(0, 2).
			Render(
				lipgloss.NewStyle().Foreground(lipgloss.Color(t.hex)).Bold(true).Render(t.value) + "\n" +
					styleFaint.Render(t.label))
		parts = append(parts, tile)
	}
	return " " + lipgloss.JoinHorizontal(lipgloss.Top, joinWithGap(parts, " ")...)
}

func joinWithGap(parts []string, gap string) []string {
	out := make([]string, 0, len(parts)*2)
	for i, p := range parts {
		if i > 0 {
			out = append(out, gap)
		}
		out = append(out, p)
	}
	return out
}

// burnPerMinute averages recent token deltas into a rate.
func (m Model) burnPerMinute() int {
	if len(m.burn) == 0 {
		return 0
	}
	window := m.burn
	if len(window) > 8 {
		window = window[len(window)-8:]
	}
	total := 0
	for _, d := range window {
		total += d
	}
	perTick := float64(total) / float64(len(window))
	ticksPerMinute := float64(60) / m.refresh.Seconds()
	return int(perTick * ticksPerMinute)
}

// throughputLabel converts token burn into an estimated line rate —
// the "bits transmitted" of the fleet (≈4 bytes per token).
func throughputLabel(tokensPerMinute int) string {
	bitsPerSecond := float64(tokensPerMinute) / 60.0 * 4 * 8
	switch {
	case bitsPerSecond >= 1_000_000:
		return fmt.Sprintf("%.1f Mbit/s", bitsPerSecond/1_000_000)
	case bitsPerSecond >= 1_000:
		return fmt.Sprintf("%.1f kbit/s", bitsPerSecond/1_000)
	}
	return fmt.Sprintf("%.0f bit/s", bitsPerSecond)
}

// viewFlowChart renders the token-burn history as a gradient area chart.
func (m Model) viewFlowChart(l layout) string {
	width := l.width - 4
	samples := m.burn
	if len(samples) > width {
		samples = samples[len(samples)-width:]
	}
	peak := 1
	for _, s := range samples {
		if s > peak {
			peak = s
		}
	}
	partials := []rune(" ▁▂▃▄▅▆▇█")
	rows := make([]strings.Builder, flowChartHeight)
	pad := width - len(samples)
	for row := range rows {
		rows[row].WriteString(strings.Repeat(" ", pad))
	}
	for i, s := range samples {
		level := float64(s) / float64(peak) * float64(flowChartHeight)
		color := columnColor(i, len(samples), m.frame)
		for row := 0; row < flowChartHeight; row++ {
			cellFloor := float64(flowChartHeight - 1 - row)
			var glyph string
			switch {
			case level >= cellFloor+1:
				glyph = "█"
			case level > cellFloor:
				fraction := level - cellFloor
				glyph = string(partials[int(fraction*8)])
			default:
				glyph = " "
			}
			rows[row].WriteString(lipgloss.NewStyle().
				Foreground(lipgloss.Color(color)).Render(glyph))
		}
	}
	var b strings.Builder
	for row := range rows {
		b.WriteString("  " + rows[row].String() + "\n")
	}
	b.WriteString("  " + styleFaint.Render(fmt.Sprintf("peak %s tok/tick · window %d ticks", compactInt(peak), len(samples))))
	return b.String()
}

// columnColor drifts ocean→violet across the chart and shimmers with time.
func columnColor(i, n, frame int) string {
	t := fold(floatMod(float64(i)/float64(maxInt(n-1, 1))+float64(frame)/80.0, 1))
	return lerpHex(colDeep, colViolet, t)
}

// viewAgentBars shows where the tokens are going, by agent.
func (m Model) viewAgentBars(l layout) string {
	totals := map[fleet.Agent]int{}
	for _, s := range m.sessions {
		totals[s.Agent] += s.Tokens
	}
	type share struct {
		agent  fleet.Agent
		tokens int
	}
	var shares []share
	peak := 1
	for a, t := range totals {
		shares = append(shares, share{a, t})
		if t > peak {
			peak = t
		}
	}
	sort.Slice(shares, func(i, j int) bool { return shares[i].tokens > shares[j].tokens })
	barMax := maxInt(l.width-38, 20)
	var b strings.Builder
	for _, sh := range shares {
		barLen := int(float64(sh.tokens) / float64(peak) * float64(barMax))
		bar := lipgloss.NewStyle().
			Foreground(lipgloss.Color(darken(fleet.HexFor(sh.agent), 0.2))).
			Render(strings.Repeat("█", maxInt(barLen, 1)))
		b.WriteString(" " + agentPill(sh.agent) + " " + bar + " " +
			styleInk.Render(padLeft(tokensLabel(sh.tokens), 7)) + "\n")
	}
	b.WriteString("\n")
	return b.String()
}

// viewContextGauges ranks sessions by context-window pressure — who is
// closest to running out of room.
func (m Model) viewContextGauges(l layout) string {
	sessions := make([]fleet.Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		if s.ContextPct >= 0 {
			sessions = append(sessions, s)
		}
	}
	sort.Slice(sessions, func(i, j int) bool { return sessions[i].ContextPct > sessions[j].ContextPct })
	if len(sessions) > 4 {
		sessions = sessions[:4]
	}
	if len(sessions) == 0 {
		return " " + styleFaint.Render("no context telemetry yet") + "\n"
	}
	var b strings.Builder
	for _, s := range sessions {
		b.WriteString(" " + styleBright.Render(padRight(s.Project, 16)) + " " +
			contextBar(s.ContextPct, maxInt(l.width-32, 20)) + " " +
			contextPctLabel(s.ContextPct) + "\n")
	}
	return b.String()
}
