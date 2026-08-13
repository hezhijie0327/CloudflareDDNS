// Package ipdetect detects the public WAN IP address of the requested
// family.
//
// DNS detection (a whoami.cloudflare CH TXT query) is tried first — the
// authoritative answer carries the querying client's egress IP — with the
// Cloudflare trace endpoint (api.cloudflare.com/cdn-cgi/trace) as
// automatic fallback.
package ipdetect

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
	"zjddns/internal/log"
)

// Detector resolves the public WAN IPv4 and IPv6 addresses.
type Detector struct{}

const (
	// DNS detection servers: the whoami.cloudflare answer carries the
	// querying client's egress IP of the requested family.
	dns4Server   = "1.1.1.1"
	dns6Server   = "2606:4700:4700::1111"
	whoamiDomain = "whoami.cloudflare"

	// traceURL is the HTTP fallback endpoint.
	traceURL = "https://api.cloudflare.com/cdn-cgi/trace"

	detectTimeout = 5 * time.Second
)

// IPv4 returns the detected public WAN IPv4 address.
func (d *Detector) IPv4() (string, error) { return d.detect(false) }

// IPv6 returns the detected public WAN IPv6 address.
func (d *Detector) IPv6() (string, error) { return d.detect(true) }

// detect runs DNS-first detection with the HTTP trace fallback.
func (d *Detector) detect(forceIPv6 bool) (string, error) {
	if ip, err := d.detectViaDNS(forceIPv6); err == nil {
		return ip, nil
	} else {
		log.Warnf("IPDETECT: DNS detection failed (%v), falling back to HTTP", err)
	}

	return d.detectViaTrace(forceIPv6)
}

// detectViaDNS queries the family-forced Cloudflare DNS server for the
// whoami CH TXT record and validates the answer against the family.
func (d *Detector) detectViaDNS(forceIPv6 bool) (string, error) {
	dnsServer := dns4Server
	if forceIPv6 {
		dnsServer = dns6Server
	}

	txt, err := dnsQueryTXT(dnsServer, whoamiDomain)
	if err != nil {
		return "", err
	}

	if !validFamilyIP(txt, forceIPv6) {
		return "", fmt.Errorf("invalid IP from DNS: %s", txt)
	}

	return txt, nil
}

// detectViaTrace queries the Cloudflare trace endpoint with the dialer
// forced to the requested address family.
func (d *Detector) detectViaTrace(forceIPv6 bool) (string, error) {
	networkType := "tcp4"
	if forceIPv6 {
		networkType = "tcp6"
	}

	// Force the requested address family on the dialer.
	client := &http.Client{
		Timeout: detectTimeout,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: func(ctx context.Context, _, address string) (net.Conn, error) {
				dialer := net.Dialer{
					Timeout: detectTimeout,
				}
				return dialer.DialContext(ctx, networkType, address)
			},
		},
	}
	defer client.CloseIdleConnections()

	resp, err := client.Get(traceURL) //nolint:gosec // fixed Cloudflare endpoint, not user input
	if err != nil {
		return "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	for line := range strings.SplitSeq(string(body), "\n") {
		if ip, found := strings.CutPrefix(line, "ip="); found {
			if validFamilyIP(ip, forceIPv6) {
				return ip, nil
			}
		}
	}

	return "", errors.New("no valid IP found")
}

// validFamilyIP reports whether ip parses and matches the requested
// address family.
func validFamilyIP(ip string, forceIPv6 bool) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	isIPv4 := parsedIP.To4() != nil
	if forceIPv6 {
		return !isIPv4
	}
	return isIPv4
}
