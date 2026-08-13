package ddns

// Provider performs DNS record operations for a DDNS provider.
//
// Implementations own the provider-specific console output (record IDs,
// failure details) and report failures there as well; the returned error
// is informational for callers. New providers live under providers/ and
// are constructed by providers.All from their config section.
type Provider interface {
	// Mode returns the operation mode from its config section
	// ("upsert" or "delete").
	Mode() string
	// Types returns the record types this provider handles (from its
	// config section, e.g. "A" / "AAAA" / both for "A_AAAA").
	Types() []string
	// Upsert ensures the record of the given type points to ip,
	// creating it if missing or updating it when the current content
	// differs.
	Upsert(recordType, ip string) error
	// Delete removes the record of the given type.
	Delete(recordType string) error
}
