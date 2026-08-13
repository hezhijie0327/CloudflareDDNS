// Package config loads and validates the Cloudflare DDNS configuration.
package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config 配置结构
type Config struct {
	APIToken string `json:"api_token"`
	// Deprecated: Use api_token instead
	XAuthEmail string `json:"x_auth_email,omitempty"`
	// Deprecated: Use api_token instead
	XAuthKey       string `json:"x_auth_key,omitempty"`
	ZoneName       string `json:"zone_name"`
	RecordName     string `json:"record_name"`
	Type           string `json:"type,omitempty"`            // A, AAAA, or A_AAAA (default: A)
	TTL            int    `json:"ttl,omitempty"`             // 1, 120, 300, 600, 900, 1800, 3600, 7200, 18000, 43200, 86400 (default: 1)
	IP             string `json:"ip,omitempty"`              // "auto" or specific IP or "ipv4,ipv6" (default: auto)
	ProxyStatus    bool   `json:"proxy_status,omitempty"`    // true or false (default: false)
	Mode           string `json:"mode,omitempty"`            // upsert (default), delete
	UpdateInterval *int   `json:"update_interval,omitempty"` // Update interval in seconds (nil/default: 300, 0: run once)
}

// Load 从JSON文件加载配置
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
