package config

import "errors"

// validModes and validTypes are ZJDDNS-wide abstractions shared by every
// provider; provider-specific rules (e.g. valid TTLs) live with their
// section struct.
var (
	validModes = map[string]bool{ModeUpsert: true, ModeDelete: true}
	validTypes = map[string]bool{TypeA: true, TypeAAAA: true, TypeAAndAAAA: true}
)

// Validate checks the configuration for completeness and self-consistency.
// At least one provider section must be configured; each configured
// section is validated by its own rules.
func (c *Config) Validate() error {
	if c.Cloudflare == nil {
		return errors.New(`no provider configured (add a "cloudflare": {...} section)`)
	}

	return c.Cloudflare.validate()
}
