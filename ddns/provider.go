package ddns

import "zjddns/internal/ipdetect"

// Provider performs DNS record operations for a DDNS provider.
//
// Implementations own the record-level logging (record IDs, failure
// details) and report failures there as well; the returned error is
// informational for callers. New providers live under providers/ and
// are constructed by providers.All from their config section.
type Provider interface {
	// Mode returns the operation mode from its config section
	// (config.ModeUpsert or config.ModeDelete).
	Mode() string
	// Types returns the record types this provider handles (from its
	// config section, e.g. config.TypeA / config.TypeAAAA or both for
	// config.TypeAAndAAAA).
	Types() []string
	// Upsert ensures the record of the given type points to ip.Value,
	// creating it if missing or updating it when the current content
	// differs; ip.Source records how the IP was obtained.
	Upsert(recordType string, ip ipdetect.IP) error
	// Delete removes the record of the given type.
	Delete(recordType string) error
}
