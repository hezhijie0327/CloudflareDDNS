package config

import "errors"

// CloudflareConfig Cloudflare 提供商配置
type CloudflareConfig struct {
	APIToken string `json:"api_token"`
	// Deprecated: Use api_token instead
	XAuthEmail string `json:"x_auth_email,omitempty"`
	// Deprecated: Use api_token instead
	XAuthKey    string `json:"x_auth_key,omitempty"`
	ZoneName    string `json:"zone_name"`
	RecordName  string `json:"record_name"`
	ProxyStatus bool   `json:"proxy_status,omitempty"`
}

// validate 验证 Cloudflare 提供商配置
func (c *CloudflareConfig) validate() error {
	if c.APIToken == "" && (c.XAuthEmail == "" || c.XAuthKey == "") {
		return errors.New("missing required authentication (api_token or x_auth_email + x_auth_key)")
	}
	if c.ZoneName == "" || c.RecordName == "" {
		return errors.New("missing required fields (zone_name, record_name)")
	}

	return nil
}
