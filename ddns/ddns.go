// Package ddns orchestrates DNS record upsert and delete operations,
// combining DDNS providers with WAN IP detection.
package ddns

import (
	"fmt"
	"net"
	"slices"
	"strings"
	"zjddns/config"
	"zjddns/internal/ipdetect"
	"zjddns/internal/log"
)

// Runner orchestrates the configured providers.
type Runner struct {
	providers []Provider
	detector  *ipdetect.Detector
	cfg       *config.Config
}

// New returns a Runner bound to the given providers and configuration.
func New(providers []Provider, cfg *config.Config) *Runner {
	return &Runner{
		providers: providers,
		detector:  &ipdetect.Detector{},
		cfg:       cfg,
	}
}

// Run executes every provider according to its configured mode.
func (r *Runner) Run() {
	var upserters, deleters []Provider
	for _, p := range r.providers {
		switch p.Mode() {
		case config.ModeUpsert:
			upserters = append(upserters, p)
		case config.ModeDelete:
			deleters = append(deleters, p)
		default:
			log.Errorf("DDNS: Invalid mode: %s", p.Mode())
		}
	}

	r.upsert(upserters)
	r.delete(deleters)
}

// resolveIP returns the IP to publish for the given record type: the
// static setting when configured, otherwise WAN detection.
func (r *Runner) resolveIP(recordType string) (ipdetect.IP, error) {
	if r.cfg.IP != config.DefaultIP {
		ip, err := staticIP(r.cfg.IP, recordType)
		if err == nil {
			log.Debugf("DDNS: Using static IP: %s", ip.Value)
		}
		return ip, err
	}

	if recordType == config.TypeAAAA {
		return r.detector.IPv6()
	}
	return r.detector.IPv4()
}

// staticIP resolves the static IP setting for the given record type;
// the "ipv4,ipv6" dual form splits per address family.
func staticIP(setting, recordType string) (ipdetect.IP, error) {
	value := setting
	if before, after, found := strings.Cut(setting, ","); found {
		value = before
		if recordType == config.TypeAAAA {
			value = after
		}
	}

	parsed := net.ParseIP(value)
	isIPv4 := parsed != nil && parsed.To4() != nil
	if (recordType == config.TypeA && !isIPv4) || (recordType == config.TypeAAAA && isIPv4) {
		return ipdetect.IP{}, fmt.Errorf("invalid static IP %q for %s record", value, recordType)
	}

	return ipdetect.IP{Value: value, Source: "static"}, nil
}

// unionTypes collects the record types of the given providers, deduped
// and keeping the provider order.
func (r *Runner) unionTypes(providers []Provider) []string {
	var types []string
	for _, p := range providers {
		for _, t := range p.Types() {
			if !slices.Contains(types, t) {
				types = append(types, t)
			}
		}
	}
	return types
}

// upsert runs the upsert flow: the WAN IP is detected once per record
// type and pushed to every provider handling that type. Record-level
// logging is owned by each provider.
func (r *Runner) upsert(providers []Provider) {
	if len(providers) == 0 {
		return
	}

	for _, recordType := range r.unionTypes(providers) {
		log.Debugf("DDNS: Checking %s record...", recordType)

		ip, err := r.resolveIP(recordType)
		if err != nil {
			log.Errorf("DDNS: Failed to get WAN IP for %s record: %v", recordType, err)
			continue
		}

		for _, p := range providers {
			if slices.Contains(p.Types(), recordType) {
				_ = p.Upsert(recordType, ip)
			}
		}
	}
}

// delete runs the delete flow: each provider removes its configured
// record types. Record-level logging is owned by each provider.
func (r *Runner) delete(providers []Provider) {
	for _, p := range providers {
		for _, recordType := range p.Types() {
			log.Debugf("DDNS: Deleting %s record...", recordType)

			_ = p.Delete(recordType)
		}
	}
}
