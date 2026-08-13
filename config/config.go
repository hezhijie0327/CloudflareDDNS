// Package config loads and validates the ZJDDNS configuration.
package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config 配置结构：顶层为所有提供商公用的设置，
// 每个提供商一个子段（如 "cloudflare": {...}），可同时配置多个。
type Config struct {
	IP             string            `json:"ip,omitempty"`              // "auto" or specific IP or "ipv4,ipv6" (default: auto)
	UpdateInterval *int              `json:"update_interval,omitempty"` // Update interval in seconds (nil/default: 300, 0: run once)
	LogLevel       string            `json:"log_level,omitempty"`       // e.g. "info" or "debug:CLOUDFLARE,IPDETECT" (default: info)
	Cloudflare     *CloudflareConfig `json:"cloudflare,omitempty"`
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
