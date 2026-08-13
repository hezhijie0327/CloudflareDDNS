// Package config loads and validates the ZJDDNS configuration.
//
// The top level holds provider-agnostic settings (IP resolution, update
// scheduling, logging); each provider owns a nested section (e.g.
// "cloudflare": {...}). Multiple provider sections may be configured at
// once — every configured section runs concurrently and updates its own
// records.
package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config is the top-level configuration structure.
type Config struct {
	IP             string            `json:"ip,omitempty"`              // "auto" or a static IP or "ipv4,ipv6" (default: auto)
	UpdateInterval *int              `json:"update_interval,omitempty"` // Update interval in seconds (nil/default: 300, 0: run once)
	LogLevel       string            `json:"log_level,omitempty"`       // e.g. "info" or "debug:CLOUDFLARE,IPDETECT" (default: info)
	Cloudflare     *CloudflareConfig `json:"cloudflare,omitempty"`
}

// Load reads and parses the JSON config file at path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path is the user-supplied config file location
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}
