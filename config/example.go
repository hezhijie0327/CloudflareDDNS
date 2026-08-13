package config

import "encoding/json"

// Example 返回示例配置的缩进JSON
func Example() (string, error) {
	cfg := &Config{
		Type:           "A_AAAA",
		TTL:            1,
		IP:             "auto",
		Mode:           "upsert",
		UpdateInterval: new(int),
		Cloudflare: &CloudflareConfig{
			APIToken:   "your-cloudflare-api-token",
			ZoneName:   "example.com",
			RecordName: "ddns.example.com",
		},
	}
	*cfg.UpdateInterval = defaultUpdateIntervalSeconds

	data, err := json.MarshalIndent(cfg, "", "  ") //nolint:gosec // serializes placeholder values, no real secrets
	if err != nil {
		return "", err
	}

	return string(data), nil
}
