package config

const (
	// DefaultIP is the default WAN IP resolution setting.
	DefaultIP = "auto"

	// DefaultMode is the default provider operation mode.
	DefaultMode = ModeUpsert
	// ModeUpsert creates DNS records or updates them when the content changed.
	ModeUpsert = "upsert"
	// ModeDelete removes DNS records.
	ModeDelete = "delete"

	// DefaultType is the default DNS record type.
	DefaultType = TypeA
	// TypeA is an IPv4 address record.
	TypeA = "A"
	// TypeAAAA is an IPv6 address record.
	TypeAAAA = "AAAA"
	// TypeAAndAAAA updates both IPv4 and IPv6 records.
	TypeAAndAAAA = "A_AAAA"

	// DefaultLogLevel is the default logging level.
	DefaultLogLevel = "info"

	// defaultUpdateIntervalSeconds is used when update_interval is omitted.
	defaultUpdateIntervalSeconds = 300
)

// SetDefaults fills empty settings with their default values.
func (c *Config) SetDefaults() {
	if c.IP == "" {
		c.IP = DefaultIP
	}
	// UpdateInterval nil means the default interval.
	if c.UpdateInterval == nil {
		c.UpdateInterval = new(int)
		*c.UpdateInterval = defaultUpdateIntervalSeconds
	}
	// Per-provider section defaults.
	if c.Cloudflare != nil {
		c.Cloudflare.setDefaults()
	}
}

// Interval returns the update interval in seconds.
func (c *Config) Interval() int {
	if c.UpdateInterval == nil {
		return defaultUpdateIntervalSeconds
	}
	return *c.UpdateInterval
}
