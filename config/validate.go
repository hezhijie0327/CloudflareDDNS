package config

import (
	"errors"
	"fmt"
)

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
	if len(c.Cloudflare) == 0 {
		return errors.New(`no provider configured (add a "cloudflare": [...] section)`)
	}

	for i := range c.Cloudflare {
		if err := c.Cloudflare[i].validate(); err != nil {
			return fmt.Errorf("cloudflare[%d]: %w", i, err)
		}
	}

	return nil
}
