package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/edbror/watr-fleet/internal/fleet"
)

// Palette: deep ocean → cyan → violet. Water identity, aurora accents.
const (
	colDeep    = "#0EA5E9"
	colCyan    = "#22D3EE"
	colViolet  = "#A78BFA"
	colAmber   = "#FBBF24"
	colAmberHi = "#FDE68A"
	colRed     = "#F87171"
	colSlate   = "#64748B"
	colInk     = "#94A3B8"
	colBright  = "#E2E8F0"
	colClay    = "#D97757"
	colGreen   = "#4ADE80"
	colSurface = "#1E293B"
	colAbyss   = "#0B1120"
)

var (
	styleFaint  = lipgloss.NewStyle().Foreground(lipgloss.Color(colSlate))
	styleInk    = lipgloss.NewStyle().Foreground(lipgloss.Color(colInk))
	styleBright = lipgloss.NewStyle().Foreground(lipgloss.Color(colBright))
	styleAmber  = lipgloss.NewStyle().Foreground(lipgloss.Color(colAmber)).Bold(true)
	styleCyan   = lipgloss.NewStyle().Foreground(lipgloss.Color(colCyan))

	stylePanel = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colSurface)).
			Padding(0, 1)

	styleSelected = lipgloss.NewStyle().
			Background(lipgloss.Color(colSurface)).
			Foreground(lipgloss.Color(colBright))
)

func statusStyle(s fleet.Status) lipgloss.Style {
	switch s {
	case fleet.StatusWorking:
		return lipgloss.NewStyle().Foreground(lipgloss.Color(colCyan))
	case fleet.StatusNeedsYou:
		return lipgloss.NewStyle().Foreground(lipgloss.Color(colAmber)).Bold(true)
	case fleet.StatusError:
		return lipgloss.NewStyle().Foreground(lipgloss.Color(colRed)).Bold(true)
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color(colSlate))
	}
}

// statusGlyph animates per state: braille spinner while working, a
// breathing dot while blocked.
func statusGlyph(s fleet.Status, frame int) string {
	switch s {
	case fleet.StatusWorking:
		spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		return spinner[frame%len(spinner)]
	case fleet.StatusNeedsYou:
		breathe := []string{"●", "●", "◉", "◉", "●", "●"} // always filled: never confusable with idle ○
		return breathe[frame%len(breathe)]
	case fleet.StatusError:
		return "✕"
	default:
		return "○"
	}
}

// pulseAmber breathes the attention color between amber and pale gold.
func pulseAmber(frame int) lipgloss.Style {
	t := pulsePhase(frame, 8)
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(lerpHex(colAmber, colAmberHi, t))).
		Bold(true)
}

// pulsePhase produces a 0→1→0 triangle wave over `period` frames.
func pulsePhase(frame, period int) float64 {
	pos := frame % period
	half := period / 2
	if pos <= half {
		return float64(pos) / float64(half)
	}
	return float64(period-pos) / float64(half)
}

func agentStyle(a fleet.Agent) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(fleet.HexFor(a)))
}

// gradient renders text with a per-rune color blend between two hex colors.
func gradient(text, fromHex, toHex string) string {
	return gradientPhase(text, fromHex, toHex, 0)
}

// gradientPhase shifts the blend by phase (0..1), animating a shimmer.
func gradientPhase(text, fromHex, toHex string, phase float64) string {
	runes := []rune(text)
	if len(runes) == 0 {
		return ""
	}
	var b strings.Builder
	for i, r := range runes {
		t := float64(i)/float64(maxInt(len(runes)-1, 1)) + phase
		t -= float64(int(t)) // wrap into 0..1
		b.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color(lerpHex(fromHex, toHex, fold(t)))).
			Bold(true).
			Render(string(r)))
	}
	return b.String()
}

// fold mirrors 0..1 into 0..1..0 so wrapped gradients have no hard seam.
func fold(t float64) float64 {
	if t < 0.5 {
		return t * 2
	}
	return (1 - t) * 2
}

// wave renders an animated water line: characters and colors both drift.
// Glyphs are deliberately common (~ ≈) so every terminal font resolves them.
func wave(width, frame int) string {
	if width <= 0 {
		return ""
	}
	glyphs := []rune("~~≈~")
	var b strings.Builder
	for i := 0; i < width; i++ {
		g := glyphs[(i+frame/3)%len(glyphs)]
		t := fold(floatMod(float64(i)/28.0+float64(frame)/45.0, 1))
		color := darken(lerpHex(colDeep, colViolet, t), 0.45)
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(string(g)))
	}
	return b.String()
}

// sparkline renders recent activity as quiet mini bars — an accent, not a
// billboard: capped height, dimmed hue, dots for silence.
func sparkline(samples []int, width int) string {
	bars := []rune("▂▃▄▅")
	if len(samples) > width {
		samples = samples[len(samples)-width:]
	}
	peak := 0
	for _, s := range samples {
		if s > peak {
			peak = s
		}
	}
	var b strings.Builder
	for i := 0; i < width-len(samples); i++ {
		b.WriteString(styleFaint.Render("·"))
	}
	for _, s := range samples {
		if peak == 0 || s == 0 {
			b.WriteString(styleFaint.Render("·"))
			continue
		}
		level := int(float64(s) / float64(peak) * float64(len(bars)-1))
		color := darken(colCyan, 0.4)
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(string(bars[level])))
	}
	return b.String()
}

func lerpHex(fromHex, toHex string, t float64) string {
	fr, fg, fb := hexRGB(fromHex)
	tr, tg, tb := hexRGB(toHex)
	lerp := func(a, b int) int { return a + int(float64(b-a)*t) }
	return fmt.Sprintf("#%02X%02X%02X", lerp(fr, tr), lerp(fg, tg), lerp(fb, tb))
}

// darken blends a color toward the abyss background.
func darken(hex string, amount float64) string {
	return lerpHex(hex, colAbyss, amount)
}

func hexRGB(hex string) (r, g, b int) {
	fmt.Sscanf(strings.TrimPrefix(hex, "#"), "%02x%02x%02x", &r, &g, &b)
	return r, g, b
}

// contextBar renders context-window pressure as a small water-gradient meter.
func contextBar(pct float64, width int) string {
	if pct < 0 {
		return styleFaint.Render(strings.Repeat("·", width))
	}
	filled := int(pct*float64(width) + 0.5)
	var b strings.Builder
	for i := 0; i < width; i++ {
		if i < filled {
			t := float64(i) / float64(maxInt(width-1, 1))
			color := lerpHex(colDeep, colViolet, t)
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render("▰"))
		} else {
			b.WriteString(styleFaint.Render("▱"))
		}
	}
	return b.String()
}

func floatMod(v, m float64) float64 {
	return v - m*float64(int(v/m))
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
