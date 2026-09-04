// Package buildinfo resolves fleet's version once, for every part of the
// binary that needs to show it. It exists because the version used to live in
// two places — a const in the UI and a flag in main — and they drifted: v0.10.5
// shipped rendering "v0.10.4" in its own header.
package buildinfo

import "runtime/debug"

// version is stamped at release time with
// -ldflags "-X github.com/edbror/watr-fleet/internal/buildinfo.version=<tag>".
var version string

// Version prefers the release stamp, then the module version the Go toolchain
// records for `go install ...@latest`, so both install paths report something
// real instead of an empty string.
func Version() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return info.Main.Version
	}
	return "dev"
}
