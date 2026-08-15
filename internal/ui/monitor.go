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
	// Sparse drifting stars, sky only: everything below the horizon is sea.
	for y := 1; y < horizonRow(h); y++ {
		for x := 1; x < w-1; x++ {
			if (x*7+y*13+m.frame/6)%97 == 0 {
				c[y][x] = mcell{'·', darken(colBright, 0.55)}
			}
		}
	}
}

// The sea is drawn in perspective: a horizon high on the canvas and water
// filling the whole plane below it. Depth carries data like every other
// channel — far means silent, near means alive.
func horizonRow(h int) int { return maxInt(h*30/100, 2) }
func shoreRow(h int) int   { return h - 4 }

// perspectiveRow projects depth z (0 = at the viewer's feet, 1 = horizon)
// onto a canvas row; the easing bunches far rows against the horizon the
// way distance compresses the real sea.
func perspectiveRow(z float64, h int) float64 {
	top, bottom := float64(horizonRow(h))+1, float64(shoreRow(h))
	return top + math.Pow(1-z, 1.6)*(bottom-top)
}

// paintWaves fills the whole plane below the horizon with parallax water:
// far layers short, dim and slow against the horizon; near layers wide,
// bright and fast at the viewer's feet. Amplitude breathes with real burn.
func (m Model) paintWaves(c [][]mcell) {
	h, w := len(c), len(c[0])
	top := horizonRow(h)
	if shoreRow(h) <= top {
		return
	}
	layers := clampInt(h/8, 3, 6)
	energy := 1.0 + math.Min(float64(m.burnPerMinute())/1_500_000.0, 1.0)*0.6
	glyphs := []rune{'≈', '~'}
	for k := 0; k < layers; k++ { // horizon first, near water paints over
		z := 1.0
		if layers > 1 {
			z = 1.0 - float64(k)/float64(layers-1) // 1 = horizon, 0 = near
		}
		// Gentle rolling swell: amplitude grows toward the viewer but the
		// wavelength grows faster, so the sea never turns into mountains.
		amp := (0.4 + (1-z)*1.4) * energy
		wavelength := 14.0 + (1-z)*34.0
		phase := float64(m.frame) / (9.0 - (1-z)*4.5)
		for x := 1; x < w-1; x++ {
			base := perspectiveRow(z, h)
			y := int(base + amp*math.Sin(2*math.Pi*float64(x)/wavelength+phase))
			if y <= top || y >= h-1 {
				continue
			}
			t := fold(floatMod(float64(x)/30.0+float64(m.frame)/50.0, 1))
			color := darken(lerpHex(colDeep, colViolet, t), 0.12+z*0.5)
			c[y][x] = mcell{glyphs[k%len(glyphs)], color}
		}
	}
	// Shimmer: sparse drifting motes anywhere on the water.
	for y := top + 1; y < h-1; y++ {
		for x := 1; x < w-1; x++ {
			if c[y][x].ch == ' ' && (x*5+y*11+m.frame/4)%53 == 0 {
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

// shipTier grades a token magnitude: quick errands are dinghies,
// hundred-million-token voyages are capital ships.
func shipTier(tokens int) int {
	switch {
	case tokens >= 50_000_000:
		return 2
	case tokens >= 1_000_000:
		return 1
	default:
		return 0
	}
}

var hullByTier = [][]rune{{'▴'}, {'▲'}, {'◢', '▲', '◣'}}

// shipHull picks the hull glyphs for a token magnitude.
func shipHull(tokens int) []rune {
	return hullByTier[shipTier(tokens)]
}

// hullAtDepth applies perspective to the hull: distance shrinks a vessel
// one tier, and on the horizon line only a hollow silhouette remains.
func hullAtDepth(tokens int, z float64) []rune {
	if z > 0.82 {
		return []rune{'△'}
	}
	tier := shipTier(tokens)
	if z > 0.62 && tier > 0 {
		tier--
	}
	return hullByTier[tier]
}

// depthOf places a vessel on the z axis: fresh wakes sail the near water,
// silent sessions recede toward the horizon. Blocked and errored vessels
// stay pinned near regardless — attention must never fade with distance.
func depthOf(s fleet.Session, now time.Time) float64 {
	f := driftSpeed(now.Sub(s.LastActive), s.LastActive.IsZero()) / cruiseSpeed
	z := 1 - f
	if s.Status == fleet.StatusNeedsYou || s.Status == fleet.StatusError {
		z = math.Min(z, 0.12)
	}
	// A deterministic lane offset keeps simultaneous actives off one row.
	lane := float64(hashString(s.ID)%5)/5.0*0.14 - 0.07
	return clampUnit(z + lane)
}

// parallax scales on-screen speed by depth: near water slides past the
// viewer faster than the horizon, even at equal true speed.
func parallax(z float64) float64 { return 0.45 + 0.55*(1-z) }

func clampUnit(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// advanceVoyage integrates each vessel's drift phase once per animation
// frame; integration keeps speed changes smooth instead of teleporting.
func (m *Model) advanceVoyage() {
	now := time.Now()
	for _, s := range m.sessions {
		if _, ok := m.voyage[s.ID]; !ok {
			m.voyage[s.ID] = float64(hashString(s.ID) % 4096)
		}
		v := driftSpeed(now.Sub(s.LastActive), s.LastActive.IsZero())
		m.voyage[s.ID] += v * parallax(depthOf(s, now))
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
	// Far, diffuse hulls paint first: near vessels sail over the distance.
	sort.SliceStable(ships, func(i, j int) bool { return depthOf(ships[i], now) > depthOf(ships[j], now) })
	left, right := 4, w-legendWidth-6
	span := maxInt(right-left, 8)
	for _, s := range ships {
		z := depthOf(s, now)
		phase, ok := m.voyage[s.ID]
		if !ok {
			phase = float64(hashString(s.ID) % 4096)
		}
		x := left + int(math.Mod(phase, float64(span)))
		// Bob softens with distance; phase comes from the ID, not the sort.
		bobPhase := float64(hashString(s.ID)%628) / 100.0
		bob := math.Round(math.Sin(float64(m.frame)/4.0+bobPhase) * 1.2 * (0.4 + 0.6*(1-z)))
		y := clampInt(int(perspectiveRow(z, h)+bob), horizonRow(h)+1, h-3)
		hull := hullAtDepth(s.Tokens, z)
		hex := fleet.HexFor(s.Agent)
		dim := math.Min(activityDim(now.Sub(s.LastActive), s.LastActive.IsZero())+0.30*z, 0.68)
		for k, r := range hull {
			if x+k < w-1 {
				c[y][x+k] = mcell{r, darken(hex, dim)}
			}
		}
		// Definition fades with distance: only near vessels keep their bow
		// marker and tear a wake in the water.
		if z <= 0.62 {
			if bow := x + len(hull); bow < w-1 {
				c[y][bow] = mcell{'▸', darken(hex, 0.3+dim*0.5)}
			}
			v := driftSpeed(now.Sub(s.LastActive), s.LastActive.IsZero())
			for k := 1; k <= int(v/cruiseSpeed*4); k++ {
				if wx := x - k - (m.frame/2+k)%2; wx > 0 && y+1 < h-1 {
					c[y+1][wx] = mcell{'·', darken(colCyan, 0.25+float64(k)*0.12)}
				}
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
