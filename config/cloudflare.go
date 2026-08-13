package config

import (
	"errors"
	"fmt"
)

// CloudflareConfig is the Cloudflare provider section.
type CloudflareConfig struct {
	APIToken    string `json:"api_token"`
	ZoneName    string `json:"zone_name"`
	RecordName  string `json:"record_name"`
	Mode        string `json:"mode,omitempty"`         // upsert (default), delete
	Type        string `json:"type,omitempty"`         // A, AAAA, or A_AAAA (default: A)
	TTL         int    `json:"ttl,omitempty"`          // TTL in seconds (default: 1, which means auto for Cloudflare)
	ProxyStatus bool   `json:"proxy_status,omitempty"` // true or false (default: false)
}

// validTTLs are the TTL values accepted by the Cloudflare API
// (1 = auto; other providers define their own TTL rules).
var validTTLs = map[int]bool{1: true, 120: true, 300: true, 600: true, 900: true, 1800: true, 3600: true, 7200: true, 18000: true, 43200: true, 86400: true}

// setDefaults fills empty settings with their default values.
func (c *CloudflareConfig) setDefaults() {
	if c.Mode == "" {
		c.Mode = DefaultMode
	}
	if c.Type == "" {
		c.Type = DefaultType
	}
	if c.TTL == 0 {
		c.TTL = 1
	}
}

// validate checks the Cloudflare provider section for completeness and
// self-consistency.
func (c *CloudflareConfig) validate() error {
	if c.APIToken == "" {
		return errors.New("missing required authentication (api_token)")
	}
	if c.ZoneName == "" || c.RecordName == "" {
		return errors.New("missing required fields (zone_name, record_name)")
	}
	if !validModes[c.Mode] {
		return fmt.Errorf("invalid mode: %s (must be upsert/delete)", c.Mode)
	}

	// Type and TTL only apply to upsert mode.
	if c.Mode == ModeUpsert {
		if !validTypes[c.Type] {
			return fmt.Errorf("invalid type: %s (must be A/AAAA/A_AAAA)", c.Type)
		}
		if !validTTLs[c.TTL] {
			return fmt.Errorf("invalid TTL: %d", c.TTL)
		}
	}

	return nil
}
