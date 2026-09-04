package main

import "testing"

// resolveVersion has three sources; the stamp must win, and an unstamped
// binary must never report an empty version.
func TestResolveVersion(t *testing.T) {
	original := version
	defer func() { version = original }()

	version = "v9.9.9"
	if got := resolveVersion(); got != "v9.9.9" {
		t.Errorf("stamped build: got %q, want %q", got, "v9.9.9")
	}

	version = ""
	if got := resolveVersion(); got == "" {
		t.Error("unstamped build returned an empty version")
	}
}
