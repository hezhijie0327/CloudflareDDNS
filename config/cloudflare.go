package config

import (
	"errors"
	"fmt"
)

// CloudflareConfig Cloudflare 提供商配置
type CloudflareConfig struct {
	APIToken string `json:"api_token"`
	// Deprecated: Use api_token instead
	XAuthEmail string `json:"x_auth_email,omitempty"`
	// Deprecated: Use api_token instead
	XAuthKey    string `json:"x_auth_key,omitempty"`
	ZoneName    string `json:"zone_name"`
	RecordName  string `json:"record_name"`
	Mode        string `json:"mode,omitempty"`         // upsert (default), delete
	Type        string `json:"type,omitempty"`         // A, AAAA, or A_AAAA (default: A)
	TTL         int    `json:"ttl,omitempty"`          // 1, 120, 300, 600, 900, 1800, 3600, 7200, 18000, 43200, 86400 (default: 1)
	ProxyStatus bool   `json:"proxy_status,omitempty"` // true or false (default: false)
}

// setDefaults 设置默认值
func (c *CloudflareConfig) setDefaults() {
	if c.Mode == "" {
		c.Mode = "upsert"
	}
	if c.Type == "" {
		c.Type = "A"
	}
	if c.TTL == 0 {
		c.TTL = 1
	}
}

// validate 验证 Cloudflare 提供商配置
func (c *CloudflareConfig) validate() error {
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
