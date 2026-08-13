package ui

import (
	"strings"
	"testing"

	"github.com/edbror/watr-fleet/internal/fleet"
)

func TestCompactIntTiers(t *testing.T) {
	cases := map[int]string{
		950:            "950",
		14_600:         "15k",
		2_400_000:      "2.4M",
		145_000_000:    "145M",
		14_347_900_000: "14.35B",
	}
	for in, want := range cases {
		if got := compactInt(in); got != want {
			t.Errorf("compactInt(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestCostLabelDropsCentsPastAGrand(t *testing.T) {
	cases := map[float64]string{
		0:        "—",
		12.34:    "$12.34",
		999.99:   "$999.99",
		31834.22: "$31,834",
	}
	for in, want := range cases {
		if got := costLabel(in); got != want {
			t.Errorf("costLabel(%f) = %q, want %q", in, got, want)
		}
	}
}

func TestGroupThousands(t *testing.T) {
	cases := map[int]string{0: "0", 999: "999", 1000: "1,000", 31834: "31,834", 1234567: "1,234,567"}
	for in, want := range cases {
		if got := groupThousands(in); got != want {
			t.Errorf("groupThousands(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestLegendPanelStaysDry(t *testing.T) {
	// A wide, short canvas with more ships than panel capacity: the panel
	// rectangle must contain no wave glyphs and must end in a "+N more"
	// row instead of silently dropping vessels.
	m := Model{}
	for i := 0; i < 20; i++ {
		m.sessions = append(m.sessions, fleet.Session{
			Project: string(rune('a'+i%26)) + "-proj",
			Agent:   fleet.AgentClaude,
			Status:  fleet.StatusIdle,
		})
	}
	m.sessions[7].Status = fleet.StatusNeedsYou

	h, w := 20, 120
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
	m.paintLegend(canvas)

	for y := 1; y <= 4+h-6; y++ {
		if y > h-2 {
			break
		}
		for x := w - legendWidth - 2; x <= w-2; x++ {
			switch canvas[y][x].ch {
			case '≈', '~', '˙':
				t.Fatalf("wave glyph %q inside legend panel at (%d,%d)", canvas[y][x].ch, y, x)
			}
		}
	}

	rows := canvasText(canvas)
	if !containsRow(rows, "more…") {
		t.Errorf("expected a +N more row, got none; capacity=%d ships=%d", h-6, len(m.sessions))
	}
	// The blocked vessel sorts to the top of the roster.
	if !strings.Contains(rows[3], "h-proj") {
		t.Errorf("blocked vessel not first in roster: %q", rows[3])
	}
}

func canvasText(c [][]mcell) []string {
	var rows []string
	for _, row := range c {
		var s []rune
		for _, cell := range row {
			s = append(s, cell.ch)
		}
		rows = append(rows, string(s))
	}
	return rows
}

func containsRow(rows []string, sub string) bool {
	for _, r := range rows {
		if strings.Contains(r, sub) {
			return true
		}
	}
	return false
}
