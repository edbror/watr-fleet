// Package config loads fleet.toml: custom agents, pricing overrides,
// notification settings, and UI preferences.
package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config is the parsed fleet.toml.
type Config struct {
	Agents  map[string]AgentConfig `toml:"agents"`
	Pricing map[string]PriceConfig `toml:"pricing"`
	Notify  NotifyConfig           `toml:"notify"`
}

// AgentConfig registers a custom agent or overrides a built-in one.
//
//	[agents.mybot]
//	hex = "#FF00AA"
//	aliases = ["mybot-cli", "mb"]
type AgentConfig struct {
	Hex     string   `toml:"hex"`
	Aliases []string `toml:"aliases"`
}

// PriceConfig overrides USD-per-MTok pricing for a model family.
//
//	[pricing.opus]
//	input = 15.0
//	output = 75.0
//	cache_read = 1.5
//	cache_write = 18.75
type PriceConfig struct {
	Input      float64 `toml:"input"`
	Output     float64 `toml:"output"`
	CacheRead  float64 `toml:"cache_read"`
	CacheWrite float64 `toml:"cache_write"`
}

// NotifyConfig controls the attention-router escalation.
//
//	[notify]
//	threshold_seconds = 30
//	ntfy_topic = "watr-fleet"
//	command = "afplay /System/Library/Sounds/Ping.aiff"
type NotifyConfig struct {
	ThresholdSeconds int    `toml:"threshold_seconds"`
	NtfyTopic        string `toml:"ntfy_topic"`
	Command          string `toml:"command"`
}

// DefaultPath resolves ~/.config/fleet/fleet.toml.
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "fleet", "fleet.toml")
}

// Load reads the config file; a missing file yields usable defaults.
func Load(path string) (Config, error) {
	cfg := Config{
		Notify: NotifyConfig{ThresholdSeconds: 30},
	}
	if path == "" {
		return cfg, nil
	}
	if _, err := toml.DecodeFile(path, &cfg); err != nil && !os.IsNotExist(err) {
		return cfg, err
	}
	if cfg.Notify.ThresholdSeconds <= 0 {
		cfg.Notify.ThresholdSeconds = 30
	}
	return cfg, nil
}
