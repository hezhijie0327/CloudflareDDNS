package config

// defaultUpdateIntervalSeconds is used when update_interval is omitted.
const defaultUpdateIntervalSeconds = 300

// SetDefaults 设置默认值
func (c *Config) SetDefaults() {
	if c.Mode == "" {
		c.Mode = "upsert"
	}
	if c.Type == "" {
		c.Type = "A"
	}
	if c.TTL == 0 {
		c.TTL = 1
	}
	if c.IP == "" {
		c.IP = "auto"
	}
	// UpdateInterval 为 nil 时设置为默认值 300
	if c.UpdateInterval == nil {
		c.UpdateInterval = new(int)
		*c.UpdateInterval = defaultUpdateIntervalSeconds
	}
}

// Interval 获取更新间隔（秒）
func (c *Config) Interval() int {
	if c.UpdateInterval == nil {
		return defaultUpdateIntervalSeconds
	}
	return *c.UpdateInterval
}
