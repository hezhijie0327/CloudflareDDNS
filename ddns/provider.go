package ddns

// Provider performs DNS record operations for a DDNS provider.
//
// Implementations own the provider-specific console output (record IDs,
// failure details) and report failures there as well; the returned error
// is informational for callers. New providers live under providers/ and
// are selected by config "provider" in providers.New.
type Provider interface {
	// Upsert ensures the record of the given type ("A"/"AAAA") points
	// to ip, creating it if missing or updating it when the current
	// content differs.
	Upsert(recordType, ip string) error
	// Delete removes the record of the given type.
	Delete(recordType string) error
}
