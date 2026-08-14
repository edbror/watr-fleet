package ui

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/edbror/watr-fleet/internal/fleet"
)

// Monitor mode: the ambient, leave-it-on-a-screen view. An animated ocean
// where every session is a vessel riding the waves — wave energy follows
// real token burn, blocked vessels fly a pulsing flag, a ticker streams
// fleet events. Data as weather.

const legendWidth = 30

// mcell is one canvas cell: a rune and its color.
type mcell struct {
	ch  rune
	hex string
}

func (m Model) viewMonitor(l layout) string {
	w := l.width - 2
	h := maxInt(m.height-6, 14)
	canvas := make([][]mcell, h)
	for y := range canvas {
		canvas[y] = make([]mcell, w)
		for x := range canvas[y] {
			canvas[y][x] = mcell{' ', ""}
		}
	}

	m.paintFrame(canvas)
	m.paintWaves(canvas)
	m.paintVessels(canvas)
	m.paintHUD(canvas)
	m.paintLegend(canvas)

	var b strings.Builder
	b.WriteString("\n " + m.monitorHeader(w) + "\n")
	b.WriteString(renderCanvas(canvas))
	b.WriteString(" " + m.ticker(w) + "\n")
	b.WriteString(" " + styleFaint.Render("m exit monitor · fleet "+version) + "\n")
	return b.String()
}

func (m Model) monitorHeader(w int) string {
	title := gradientPhase("≈ F L E E T · O P E N  S E A", colDeep, colViolet, float64(m.frame)/55.0)
	return title
}

// paintFrame draws the thin HUD border and a faint starfield grid.
func (m Model) paintFrame(c [][]mcell) {
	h, w := len(c), len(c[0])
	edge := darken(colCyan, 0.6)
	for x := 1; x < w-1; x++ {
		c[0][x] = mcell{'─', edge}
		c[h-1][x] = mcell{'─', edge}
	}
	for y := 1; y < h-1; y++ {
		c[y][0] = mcell{'│', edge}
		c[y][w-1] = mcell{'│', edge}
	}
	c[0][0], c[0][w-1] = mcell{'╭', edge}, mcell{'╮', edge}
	c[h-1][0], c[h-1][w-1] = mcell{'╰', edge}, mcell{'╯', edge}
	// Sparse drifting stars above the waterline.
	for y := 1; y < h*45/100; y++ {
		for x := 1; x < w-1; x++ {
			if (x*7+y*13+m.frame/6)%97 == 0 {
				c[y][x] = mcell{'·', darken(colBright, 0.55)}
			}
		}
	}
}

// waveY returns the surface height of wave layer k at column x. Amplitude
// breathes with the fleet's real burn rate.
func (m Model) waveY(k, x, h int) int {
	energy := 1.0 + math.Min(float64(m.burnPerMinute())/1_500_000.0, 1.0)*1.6
	base := float64(h)*0.52 + float64(k)*2.6
	amp := (1.1 + float64(k)*0.7) * energy
	wavelength := 16.0 + float64(k)*7.0
	phase := float64(m.frame) / (7.0 - float64(k)*1.5)
	y := base + amp*math.Sin(2*math.Pi*float64(x)/wavelength+phase)
	return int(y)
}

// paintWaves draws three parallax water layers plus depth shimmer.
func (m Model) paintWaves(c [][]mcell) {
	h, w := len(c), len(c[0])
	glyphs := []rune{'≈', '~', '≈'}
	for k := 2; k >= 0; k-- {
		for x := 1; x < w-1; x++ {
			y := m.waveY(k, x, h)
			if y <= 1 || y >= h-1 {
				continue
			}
			t := fold(floatMod(float64(x)/30.0+float64(m.frame)/50.0, 1))
			color := darken(lerpHex(colDeep, colViolet, t), 0.15+float64(k)*0.22)
			c[y][x] = mcell{glyphs[k], color}
		}
	}
	// Shimmer under the surface: sparse drifting motes.
	for y := 2; y < h-1; y++ {
		for x := 1; x < w-1; x++ {
			if y > m.waveY(0, x, h) && c[y][x].ch == ' ' && (x*5+y*11+m.frame/4)%53 == 0 {
				c[y][x] = mcell{'˙', darken(colDeep, 0.55)}
			}
		}
	}
}

// The open sea is a living chart, Gestaltung style: every visual channel
// carries a variable. Hull size = total tokens (log tiers), drift speed
// and glow = recency of output (half-life decay), wake length = speed.
const cruiseSpeed = 0.5 // cells per animation frame at full activity

// driftSpeed maps how long a session has been silent to horizontal
// velocity. Fresh wakes race; dormant hulls barely hold steerage way.
func driftSpeed(idle time.Duration, unknown bool) float64 {
	const halfLifeSeconds, floor = 90.0, 0.05
	if unknown {
		return cruiseSpeed * floor
	}
	if idle < 0 {
		idle = 0
	}
	f := math.Pow(0.5, idle.Seconds()/halfLifeSeconds)
	if f < floor {
		f = floor
	}
	return cruiseSpeed * f
}

// activityDim converts the same recency signal into brightness: recently
// active vessels burn full brand color, dormant ones fade toward the sea.
func activityDim(idle time.Duration, unknown bool) float64 {
	return (1 - driftSpeed(idle, unknown)/cruiseSpeed) * 0.55
}

// shipHull picks the hull for a token magnitude: quick errands are dinghies,
// hundred-million-token voyages are capital ships.
func shipHull(tokens int) []rune {
	switch {
	case tokens >= 50_000_000:
		return []rune{'◢', '▲', '◣'}
	case tokens >= 1_000_000:
		return []rune{'▲'}
	default:
		return []rune{'▴'}
	}
}

// advanceVoyage integrates each vessel's drift phase once per animation
// frame; integration keeps speed changes smooth instead of teleporting.
func (m *Model) advanceVoyage() {
	now := time.Now()
	for _, s := range m.sessions {
		if _, ok := m.voyage[s.ID]; !ok {
			m.voyage[s.ID] = float64(hashString(s.ID) % 4096)
		}
		m.voyage[s.ID] += driftSpeed(now.Sub(s.LastActive), s.LastActive.IsZero())
	}
}

// paintVessels places one ship per session on the surface wave, bobbing,
// with status regalia: pulsing amber flag when blocked, red cross on error.
func (m Model) paintVessels(c [][]mcell) {
	h, w := len(c), len(c[0])
	ships := append([]fleet.Session(nil), m.sessions...)
	if len(ships) == 0 {
		return
	}
	now := time.Now()
	speed := func(s fleet.Session) float64 {
		return driftSpeed(now.Sub(s.LastActive), s.LastActive.IsZero())
	}
	// Slow, dim hulls paint first: fresh vessels sail over dormant ones.
	sort.SliceStable(ships, func(i, j int) bool { return speed(ships[i]) < speed(ships[j]) })
	left, right := 4, w-legendWidth-6
	span := maxInt(right-left, 8)
	for i, s := range ships {
		phase, ok := m.voyage[s.ID]
		if !ok {
			phase = float64(hashString(s.ID) % 4096)
		}
		x := left + int(math.Mod(phase, float64(span)))
		bob := int(math.Round(math.Sin(float64(m.frame)/4.0+float64(i)*1.3) * 1.2))
		y := clampInt(m.waveY(0, x, h)-1+bob, 3, h-3)
		hull := shipHull(s.Tokens)
		hex := fleet.HexFor(s.Agent)
		dim := activityDim(now.Sub(s.LastActive), s.LastActive.IsZero())
		for k, r := range hull {
			if x+k < w-1 {
				c[y][x+k] = mcell{r, darken(hex, dim)}
			}
		}
		if bow := x + len(hull); bow < w-1 {
			c[y][bow] = mcell{'▸', darken(hex, 0.3+dim*0.5)}
		}
		// The wake trails proportionally to speed: fast vessels tear water.
		for k := 1; k <= int(speed(s)/cruiseSpeed*4); k++ {
			if wx := x - k - (m.frame/2+k)%2; wx > 0 && y+1 < h-1 {
				c[y+1][wx] = mcell{'·', darken(colCyan, 0.25+float64(k)*0.12)}
			}
		}
		mast := x + len(hull)/2
		switch s.Status {
		case fleet.StatusNeedsYou:
			flag := colAmber
			if m.frame%6 < 3 {
				flag = colAmberHi
			}
			c[y-1][mast] = mcell{'⚑', flag}
		case fleet.StatusError:
			c[y-1][mast] = mcell{'✕', colRed}
		}
	}
}

// hashString gives each vessel a stable berth on the sea, independent of
// roster order: re-sorts must not teleport ships.
func hashString(s string) uint64 {
	const offset, prime = 14695981039346656037, 1099511628211
	h := uint64(offset)
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= prime
	}
	return h
}

// paintHUD writes the big readouts along the top edge.
func (m Model) paintHUD(c [][]mcell) {
	sum := fleet.Summarize(m.sessions)
	burn := m.burnPerMinute()
	readout := fmt.Sprintf(" %s tok │ %s │ %s/min │ %s ",
		compactInt(sum.Tokens), costLabel(sum.CostUSD), compactInt(burn), throughputLabel(burn))
	writeString(c, 0, 3, readout, colCyan)
	if sum.NeedsYou > 0 {
		alert := fmt.Sprintf(" ⚠ %d NEED YOU ", sum.NeedsYou)
		flag := colAmber
		if m.frame%6 < 3 {
			flag = colAmberHi
		}
		writeString(c, 0, len(c[0])-len([]rune(alert))-3, alert, flag)
	}
}

// paintLegend draws the vessel roster on the right: color, name, context.
// The panel rectangle is wiped first so waves and shimmer never wash
// through the text, attention sorts to the top, and a fleet larger than
// the panel ends in a "+N more" line instead of silently vanishing.
func (m Model) paintLegend(c [][]mcell) {
	ships := append([]fleet.Session(nil), m.sessions...)
	sort.SliceStable(ships, func(i, j int) bool {
		ri, rj := legendRank(ships[i].Status), legendRank(ships[j].Status)
		if ri != rj {
			return ri > rj
		}
		return ships[i].Project < ships[j].Project
	})
	h, w := len(c), len(c[0])
	capacity := h - 6
	if capacity < 1 {
		return
	}
	shown, more := len(ships), 0
	if shown > capacity {
		shown = capacity - 1
		more = len(ships) - shown
	}
	rows := shown
	if more > 0 {
		rows++
	}
	// Wipe the roster panel: the sea stays out of the text.
	for y := 1; y <= clampInt(4+rows, 1, h-2); y++ {
		for x := maxInt(w-legendWidth-2, 1); x <= w-2; x++ {
			c[y][x] = mcell{' ', ""}
		}
	}
	x := w - legendWidth
	writeString(c, 2, x, "VESSELS", darken(colInk, 0.2))
	for i, s := range ships[:shown] {
		ctx := "  —"
		if s.ContextPct >= 0 {
			ctx = fmt.Sprintf("%3.0f%%", s.ContextPct*100)
		}
		row := fmt.Sprintf("▲ %-14s %s %s", truncate(s.Project, 14), ctx, statusGlyph(s.Status, m.frame))
		hex := fleet.HexFor(s.Agent)
		if s.Status == fleet.StatusNeedsYou {
			hex = colAmber
		}
		writeString(c, 3+i, x, row, hex)
	}
	if more > 0 {
		writeString(c, 3+shown, x, fmt.Sprintf("  +%d more…", more), darken(colInk, 0.2))
	}
}

// legendRank orders the roster by who deserves eyes first: blocked, then
// errored, then working, then idle.
func legendRank(s fleet.Status) int {
	switch s {
	case fleet.StatusNeedsYou:
		return 3
	case fleet.StatusError:
		return 2
	case fleet.StatusWorking:
		return 1
	default:
		return 0
	}
}

func writeString(c [][]mcell, y, x int, s string, hex string) {
	if y < 0 || y >= len(c) {
		return
	}
	for i, r := range []rune(s) {
		if x+i < 1 || x+i >= len(c[y])-0 {
			return
		}
		c[y][x+i] = mcell{r, hex}
	}
}

// ticker streams every session's latest event across the bottom, airport
// departures style.
func (m Model) ticker(w int) string {
	now := time.Now()
	var items []string
	for _, s := range m.sessions {
		item := s.Project + " · " + s.LastEvent
		if s.Status == fleet.StatusNeedsYou {
			item += " · waiting " + humanDuration(s.WaitingFor(now))
		}
		items = append(items, item)
	}
	if len(items) == 0 {
		items = []string{"open sea · no vessels · press n to launch"}
	}
	tape := "  ✦  " + strings.Join(items, "  ✦  ") + "  ✦  "
	runes := []rune(tape)
	offset := (m.frame / 2) % len(runes)
	var window []rune
	for i := 0; i < w; i++ {
		window = append(window, runes[(offset+i)%len(runes)])
	}
	return styleInk.Render(string(window))
}

// renderCanvas flattens the cell grid, batching same-color runs so the
// animation stays cheap even at full screen.
func renderCanvas(c [][]mcell) string {
	var b strings.Builder
	for _, row := range c {
		b.WriteString(" ")
		runStart := 0
		for x := 1; x <= len(row); x++ {
			if x == len(row) || row[x].hex != row[runStart].hex {
				var chunk strings.Builder
				for _, cell := range row[runStart:x] {
					chunk.WriteRune(cell.ch)
				}
				if row[runStart].hex == "" {
					b.WriteString(chunk.String())
				} else {
					b.WriteString(lipgloss.NewStyle().
						Foreground(lipgloss.Color(row[runStart].hex)).
						Render(chunk.String()))
				}
				runStart = x
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
