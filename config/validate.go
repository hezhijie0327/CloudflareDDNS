package config

import "errors"

var (
	validModes = map[string]bool{ModeUpsert: true, ModeDelete: true}
	validTTLs  = map[int]bool{1: true, 120: true, 300: true, 600: true, 900: true, 1800: true, 3600: true, 7200: true, 18000: true, 43200: true, 86400: true}
	validTypes = map[string]bool{TypeA: true, TypeAAAA: true, TypeAAndAAAA: true}
)

// Validate checks the configuration for completeness and self-consistency.
func (c *Config) Validate() error {
	if c.Cloudflare == nil {
		return errors.New(`no provider configured (add a "cloudflare": {...} section)`)
	}

	if c.Cloudflare != nil {
		if err := c.Cloudflare.validate(); err != nil {
			return err
		}
	}

	return nil
}
