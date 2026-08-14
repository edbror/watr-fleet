package ui

import (
	"testing"
	"time"
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
