// Package ipdetect detects the WAN IP address of the requested family.
//
// Detection tries a DNS CHAOS TXT query (whoami.cloudflare) first — the
// answer carries the querying client's egress IP — with the Cloudflare
// trace endpoint (api.cloudflare.com/cdn-cgi/trace) as automatic fallback.
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

// Detector resolves WAN IP addresses for DNS records.
// StaticIP is used when it is not "auto" (empty behaves as "auto"
// after config defaults are applied).
type Detector struct {
	StaticIP string
}

const (
	// DNS检测配置：通过 whoami.cloudflare 的 TXT 记录获取WAN IP
	dns4Server   = "1.1.1.1"              // IPv4 DNS服务器，返回IPv4出口地址
	dns6Server   = "2606:4700:4700::1111" // IPv6 DNS服务器，返回IPv6出口地址
	whoamiDomain = "whoami.cloudflare"

	// traceURL is the Cloudflare trace endpoint for the HTTP fallback path.
	traceURL = "https://api.cloudflare.com/cdn-cgi/trace"

	detectTimeout = 5 * time.Second
)

// New returns a Detector with the given static IP setting.
func New(staticIP string) *Detector {
	return &Detector{StaticIP: staticIP}
}

// WANIP 获取WAN IP地址
func (d *Detector) WANIP(recordType string) (string, error) {
	// 如果是静态IP
	if d.StaticIP != "auto" {
		return d.staticIP(recordType)
	}

	// 优先通过DNS检测，失败时回退到Cloudflare trace
	if ip, err := d.ipFromDNS(recordType); err == nil {
		return ip, nil
	} else {
		log.Warnf("IPDETECT: DNS detection failed (%v), falling back to HTTP", err)
	}

	return d.ipFromCloudflareTrace(recordType)
}

// ipFromDNS 通过DNS查询获取WAN IP
// A记录查询IPv4 DNS服务器，AAAA记录查询IPv6 DNS服务器，
// 因为DNS服务器返回的是发起查询的网络出口地址
func (d *Detector) ipFromDNS(recordType string) (string, error) {
	dnsServer := dns4Server
	if recordType == "AAAA" {
		dnsServer = dns6Server
	}

	txt, err := dnsQueryTXT(dnsServer, whoamiDomain)
	if err != nil {
		return "", err
	}

	if !validIP(txt, recordType) {
		return "", fmt.Errorf("invalid IP from DNS: %s", txt)
	}

	return txt, nil
}

// staticIP 解析静态IP，"ipv4,ipv6"按记录类型取对应部分
func (d *Detector) staticIP(recordType string) (string, error) {
	ipResult := d.StaticIP
	if strings.Contains(d.StaticIP, ",") {
		parts := strings.Split(d.StaticIP, ",")
		if recordType == "A" && len(parts) >= 1 {
			ipResult = parts[0]
		} else if recordType == "AAAA" && len(parts) >= 2 {
			ipResult = parts[1]
		}
	}

	if !validIP(ipResult, recordType) {
		return "", errors.New("invalid static IP format")
	}

	return ipResult, nil
}

// ipFromCloudflareTrace 从Cloudflare trace获取IP
func (d *Detector) ipFromCloudflareTrace(recordType string) (string, error) {
	// 根据记录类型确定网络类型
	var networkType string
	switch recordType {
	case "A":
		networkType = "tcp4" // 强制 IPv4
	case "AAAA":
		networkType = "tcp6" // 强制 IPv6
	default:
		networkType = "tcp" // 默认双栈
	}

	// 创建强制使用指定网络协议的HTTP客户端
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

	resp, err := client.Get(traceURL)
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
			if validIP(ip, recordType) {
				return ip, nil
			}
		}
	}

	return "", errors.New("no valid IP found")
}

// validIP 验证IP地址格式与记录类型是否匹配
func validIP(ip, recordType string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	if recordType == "A" {
		return parsedIP.To4() != nil
	}
	return parsedIP.To4() == nil
}
