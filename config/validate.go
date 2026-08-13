package config

import (
	"errors"
	"fmt"
)

var (
	validModes = map[string]bool{"upsert": true, "delete": true}
	validTTLs  = map[int]bool{1: true, 120: true, 300: true, 600: true, 900: true, 1800: true, 3600: true, 7200: true, 18000: true, 43200: true, 86400: true}
	validTypes = map[string]bool{"A": true, "AAAA": true, "A_AAAA": true}
)

// Validate 验证配置有效性
func (c *Config) Validate() error {
	if c.Cloudflare == nil {
		return errors.New(`no provider configured (add a "cloudflare": {...} section)`)
	}

	if c.Cloudflare != nil {
		if err := c.Cloudflare.validate(); err != nil {
			return fmt.Errorf("cloudflare: %w", err)
		}
	}

	return nil
}
