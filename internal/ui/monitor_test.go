package ui

import (
	"testing"
	"time"

	"github.com/edbror/watr-fleet/internal/fleet"
)

func TestDriftSpeedDecaysWithSilence(t *testing.T) {
	fresh := driftSpeed(0, false)
	warm := driftSpeed(2*time.Minute, false)
	cold := driftSpeed(30*time.Minute, false)
	if fresh != cruiseSpeed {
		t.Errorf("fresh vessel speed = %f, want cruise %f", fresh, cruiseSpeed)
	}
	if !(fresh > warm && warm > cold) {
		t.Errorf("speed must decay monotonically: %f, %f, %f", fresh, warm, cold)
	}
	if cold < cruiseSpeed*0.05 {
		t.Errorf("dormant vessel below the floor: %f", cold)
	}
	if got := driftSpeed(0, true); got != cruiseSpeed*0.05 {
		t.Errorf("unknown activity should crawl at the floor, got %f", got)
	}
}

func TestShipHullTiers(t *testing.T) {
	if got := string(shipHull(200_000)); got != "▴" {
		t.Errorf("dinghy hull = %q", got)
	}
	if got := string(shipHull(5_000_000)); got != "▲" {
		t.Errorf("mid hull = %q", got)
	}
	if got := string(shipHull(120_000_000)); got != "◢▲◣" {
		t.Errorf("capital hull = %q", got)
	}
}

func TestVesselBerthIsStable(t *testing.T) {
	if hashString("cc:abc") != hashString("cc:abc") {
		t.Fatal("hash must be deterministic")
	}
	if hashString("cc:abc") == hashString("cc:abd") {
		t.Error("distinct sessions should berth apart (hash collision on trivial pair)")
	}
}

func TestDepthRecedesWithSilenceButPinsAttention(t *testing.T) {
	now := time.Now()
	fresh := fleet.Session{ID: "a", Status: fleet.StatusWorking, LastActive: now}
	silent := fleet.Session{ID: "b", Status: fleet.StatusIdle, LastActive: now.Add(-30 * time.Minute)}
	blocked := fleet.Session{ID: "c", Status: fleet.StatusNeedsYou, LastActive: now.Add(-30 * time.Minute)}

	if z := depthOf(fresh, now); z > 0.25 {
		t.Errorf("fresh vessel too far: z=%f", z)
	}
	if z := depthOf(silent, now); z < 0.7 {
		t.Errorf("silent vessel too near: z=%f", z)
	}
	// Blocked 30 minutes: still pinned to the near water, flag in view.
	if z := depthOf(blocked, now); z > 0.25 {
		t.Errorf("blocked vessel drifted to the horizon: z=%f", z)
	}
}

func TestPerspectiveShrinksAndDiffusesFarHulls(t *testing.T) {
	if got := string(hullAtDepth(120_000_000, 0.1)); got != "◢▲◣" {
		t.Errorf("near capital hull = %q", got)
	}
	if got := string(hullAtDepth(120_000_000, 0.7)); got != "▲" {
		t.Errorf("far capital hull should shrink a tier, got %q", got)
	}
	if got := string(hullAtDepth(120_000_000, 0.9)); got != "△" {
		t.Errorf("horizon hull should be a hollow silhouette, got %q", got)
	}
	if got := string(hullAtDepth(200_000, 0.7)); got != "▴" {
		t.Errorf("far dinghy stays a dinghy, got %q", got)
	}
}

func TestPerspectiveRowMapsDepthToBand(t *testing.T) {
	h := 50
	near := perspectiveRow(0, h)
	far := perspectiveRow(1, h)
	if int(far) != horizonRow(h)+1 {
		t.Errorf("z=1 should sit at the horizon: row %f", far)
	}
	if int(near) != shoreRow(h) {
		t.Errorf("z=0 should sit at the shore: row %f", near)
	}
	if !(near > far) {
		t.Errorf("perspective inverted: near %f, far %f", near, far)
	}
}
