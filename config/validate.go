package config

import (
	"errors"
	"fmt"
)

var (
	validTTLs  = map[int]bool{1: true, 120: true, 300: true, 600: true, 900: true, 1800: true, 3600: true, 7200: true, 18000: true, 43200: true, 86400: true}
	validModes = map[string]bool{"upsert": true, "delete": true}
	validTypes = map[string]bool{"A": true, "AAAA": true, "A_AAAA": true}
)

// Validate 验证配置有效性
func (c *Config) Validate() error {
	if c.APIToken == "" && (c.XAuthEmail == "" || c.XAuthKey == "") {
		return errors.New("missing required authentication (api_token or x_auth_email + x_auth_key)")
	}
	if c.ZoneName == "" || c.RecordName == "" {
		return errors.New("missing required fields (zone_name, record_name)")
	}

	if !validModes[c.Mode] {
		return fmt.Errorf("invalid mode: %s (must be upsert/delete)", c.Mode)
	}

	if c.Mode == "upsert" {
		if !validTypes[c.Type] {
			return fmt.Errorf("invalid type: %s (must be A/AAAA/A_AAAA)", c.Type)
		}
		if !validTTLs[c.TTL] {
			return fmt.Errorf("invalid TTL: %d", c.TTL)
		}
	}

	return nil
}
