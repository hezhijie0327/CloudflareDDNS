package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

const (
	CloudflareAPI  = "https://api.cloudflare.com"
	RequestTimeout = 5 * time.Second

	// DNS检测配置：通过 whoami.cloudflare 的 TXT 记录获取WAN IP
	DNS4Server   = "1.1.1.1"              // IPv4 DNS服务器，返回IPv4出口地址
	DNS6Server   = "2606:4700:4700::1111" // IPv6 DNS服务器，返回IPv6出口地址
	WhoamiDomain = "whoami.cloudflare"
)

var (
	Version    = "1.6.0"
	CommitHash = "dirty"
	BuildTime  = "dev"
)

// Config 配置结构
type Config struct {
	APIToken string `json:"api_token"`
	// Deprecated: Use api_token instead
	XAuthEmail string `json:"x_auth_email,omitempty"`
	// Deprecated: Use api_token instead
	XAuthKey       string `json:"x_auth_key,omitempty"`
	ZoneName       string `json:"zone_name"`
	RecordName     string `json:"record_name"`
	Type           string `json:"type,omitempty"`            // A, AAAA, or A_AAAA (default: A)
	TTL            int    `json:"ttl,omitempty"`             // 1, 120, 300, 600, 900, 1800, 3600, 7200, 18000, 43200, 86400 (default: 1)
	IP             string `json:"ip,omitempty"`              // "auto" or specific IP or "ipv4,ipv6" (default: auto)
	ProxyStatus    bool   `json:"proxy_status,omitempty"`    // true or false (default: false)
	Mode           string `json:"mode,omitempty"`            // upsert (default), delete
	UpdateInterval *int   `json:"update_interval,omitempty"` // Update interval in seconds (nil/default: 300, 0: run once)
}

// CloudflareResponse API响应结构
type CloudflareResponse struct {
	Success bool                     `json:"success"`
	Result  interface{}              `json:"result"`
	Errors  []map[string]interface{} `json:"errors"`
}

// HTTPClient 封装的HTTP客户端
type HTTPClient struct {
	client *http.Client
	config *Config
}

func main() {
	// 自定义帮助信息
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Cloudflare DDNS Tool - Dynamic DNS Update Client\n\n")
		fmt.Fprintf(os.Stderr, "Version: %s\n\n", getVersion())
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  %s -config <config file>     # Start with config file\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -generate-config          # Generate example config\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -version                  # Show version information\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s                            # Start with default config\n\n", os.Args[0])
	}

	// 解析命令行参数
	configPath := flag.String("config", "config.json", "Path to config file")
	generateConfig := flag.Bool("generate-config", false, "Generate example config file")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	// 处理特殊参数
	if *showVersion {
		fmt.Printf("Cloudflare DDNS Tool %s\n", getVersion())
		return
	}

	if *generateConfig {
		generateExampleConfig()
		return
	}

	fmt.Printf("🚀 Cloudflare DDNS Tool %s\n\n", getVersion())

	// 加载配置
	config, err := loadConfig(*configPath)
	if err != nil {
		fmt.Printf("❌ Failed to load config: %v\n", err)
		return
	}

	// 设置默认值
	config.setDefaults()

	// 检查是否使用了已弃用的认证方式
	if config.XAuthEmail != "" && config.XAuthKey != "" && config.APIToken == "" {
		fmt.Printf("⚠️  WARNING: Using deprecated authentication method (x_auth_email + x_auth_key)\n")
		fmt.Printf("⚠️  Please migrate to using 'api_token' instead\n")
		fmt.Printf("⚠️  You can create an API token at: https://dash.cloudflare.com/profile/api-tokens\n\n")
	}

	// 验证配置
	if err := config.configValidate(); err != nil {
		fmt.Printf("❌ Invalid config: %v\n", err)
		return
	}

	client := &HTTPClient{
		client: &http.Client{Timeout: RequestTimeout},
		config: config,
	}

	// 获取 Zone ID
	zoneID, err := client.getZoneID()
	if err != nil {
		fmt.Printf("❌ Failed to get zone ID: %v\n", err)
		return
	}
	fmt.Printf("🌐 Zone ID: %s\n", zoneID)

	// 执行操作
	fmt.Println()

	// 获取更新间隔
	updateInterval := config.getUpdateInterval()

	// 执行更新的函数
	runUpdate := func() {
		switch config.Mode {
		case "upsert":
			client.handleUpsert(zoneID)
		case "delete":
			client.handleDelete(zoneID)
		default:
			fmt.Printf("❌ Invalid mode: %s\n", config.Mode)
		}
	}

	// 如果 update_interval 为 0，只运行一次
	if updateInterval <= 0 {
		runUpdate()
		return
	}

	// 定期执行更新
	interval := time.Duration(updateInterval) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	fmt.Printf("⏰ Running every %d seconds. Press Ctrl+C to stop.\n\n", updateInterval)

	// 立即执行一次
	runUpdate()

	// 循环执行
	for range ticker.C {
		fmt.Printf("\n🔄 %s - Starting scheduled update...\n", time.Now().Format("2006-01-02 15:04:05"))
		runUpdate()
	}
}

// loadConfig 从JSON文件加载配置
func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &config, nil
}

// setDefaults 设置默认值
func (c *Config) setDefaults() {
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
		defaultInterval := 300
		c.UpdateInterval = &defaultInterval
	}
}

// getUpdateInterval 获取更新间隔（秒）
func (c *Config) getUpdateInterval() int {
	if c.UpdateInterval == nil {
		return 300
	}
	return *c.UpdateInterval
}

// configValidate 验证配置有效性
func (c *Config) configValidate() error {
	validTTLs := map[int]bool{1: true, 120: true, 300: true, 600: true, 900: true, 1800: true, 3600: true, 7200: true, 18000: true, 43200: true, 86400: true}
	validModes := map[string]bool{"upsert": true, "delete": true}
	validTypes := map[string]bool{"A": true, "AAAA": true, "A_AAAA": true}

	if c.APIToken == "" && (c.XAuthEmail == "" || c.XAuthKey == "") {
		return fmt.Errorf("missing required authentication (api_token or x_auth_email + x_auth_key)")
	}
	if c.ZoneName == "" || c.RecordName == "" {
		return fmt.Errorf("missing required fields (zone_name, record_name)")
	}

	if !validModes[c.Mode] {
		return fmt.Errorf("invalid mode: %s (must be upsert/delete)", c.Mode)
	}

	if c.Mode == "upsert" {
		if !validTypes[c.Type] {
			return fmt.Errorf("invalid type: %s (must be A/AAAA/A_AAAA)", c.Type)
		}
		if !validTTLs[c.TTL] {
			return fmt.Errorf("invalid TTL: %d", c.TTL)
		}
	}

	return nil
}

// generateExampleConfig 打印示例配置文件
func generateExampleConfig() {
	updateInterval := 300
	config := &Config{
		APIToken:       "your-cloudflare-api-token",
		ZoneName:       "example.com",
		RecordName:     "ddns.example.com",
		Type:           "A_AAAA",
		TTL:            1,
		IP:             "auto",
		ProxyStatus:    false,
		Mode:           "upsert",
		UpdateInterval: &updateInterval,
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		fmt.Printf("❌ Failed to generate example config: %v\n", err)
		return
	}

	fmt.Println(string(data))
}

// getVersion 获取版本信息
func getVersion() string {
	return fmt.Sprintf("v%s-%s@%s (%s)", Version, CommitHash, BuildTime, runtime.Version())
}

// request 发送HTTP请求
func (h *HTTPClient) request(method, path string, payload interface{}) (*CloudflareResponse, error) {
	url := CloudflareAPI + path

	var bodyReader io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		bodyReader = strings.NewReader(string(data))
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	// 支持新的 API Token 认证方式
	if h.config.APIToken != "" {
		req.Header.Set("Authorization", "Bearer "+h.config.APIToken)
	} else {
		// 向后兼容旧的认证方式
		req.Header.Set("X-Auth-Email", h.config.XAuthEmail)
		req.Header.Set("X-Auth-Key", h.config.XAuthKey)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var cfResp CloudflareResponse
	if err := json.Unmarshal(body, &cfResp); err != nil {
		return nil, err
	}

	if !cfResp.Success {
		return &cfResp, fmt.Errorf("API error: %v", cfResp.Errors)
	}

	return &cfResp, nil
}

// getZoneID 获取Zone ID
func (h *HTTPClient) getZoneID() (string, error) {
	resp, err := h.request("GET", fmt.Sprintf("/client/v4/zones?name=%s", h.config.ZoneName), nil)
	if err != nil {
		return "", err
	}

	if results, ok := resp.Result.([]interface{}); ok && len(results) > 0 {
		if result, ok := results[0].(map[string]interface{}); ok {
			if id, ok := result["id"].(string); ok {
				return id, nil
			}
		}
	}

	return "", fmt.Errorf("zone not found")
}

// getRecordID 获取DNS记录ID
func (h *HTTPClient) getRecordID(zoneID, recordType string) (string, error) {
	resp, err := h.request("GET", fmt.Sprintf("/client/v4/zones/%s/dns_records?name=%s&type=%s", zoneID, h.config.RecordName, recordType), nil)
	if err != nil {
		return "", err
	}

	if results, ok := resp.Result.([]interface{}); ok && len(results) > 0 {
		if result, ok := results[0].(map[string]interface{}); ok {
			if id, ok := result["id"].(string); ok {
				return id, nil
			}
		}
	}

	return "", nil // 不存在返回空字符串，不报错
}

// getDNSRecordContent 获取DNS记录内容
func (h *HTTPClient) getDNSRecordContent(zoneID, recordID string) (string, error) {
	resp, err := h.request("GET", fmt.Sprintf("/client/v4/zones/%s/dns_records/%s", zoneID, recordID), nil)
	if err != nil {
		return "", err
	}

	if result, ok := resp.Result.(map[string]interface{}); ok {
		if content, ok := result["content"].(string); ok {
			return content, nil
		}
	}

	return "", fmt.Errorf("record content not found")
}

// getWANIP 获取WAN IP地址
func (h *HTTPClient) getWANIP(recordType string) (string, error) {
	// 如果是静态IP
	if h.config.IP != "auto" {
		return h.parseStaticIP(recordType)
	}

	// 优先通过DNS检测，失败时回退到Cloudflare trace
	if ip, err := h.getIPFromDNS(recordType); err == nil {
		return ip, nil
	} else {
		fmt.Printf("⚠️  DNS detection failed (%v), falling back to HTTP...\n", err)
	}

	return h.getIPFromCloudflareTrace(recordType)
}

// getIPFromDNS 通过DNS查询获取WAN IP
// A记录查询IPv4 DNS服务器，AAAA记录查询IPv6 DNS服务器，
// 因为DNS服务器返回的是发起查询的网络出口地址
func (h *HTTPClient) getIPFromDNS(recordType string) (string, error) {
	dnsServer := DNS4Server
	if recordType == "AAAA" {
		dnsServer = DNS6Server
	}

	txt, err := dnsQueryTXT(dnsServer, WhoamiDomain)
	if err != nil {
		return "", err
	}

	if !isValidIP(txt, recordType) {
		return "", fmt.Errorf("invalid IP from DNS: %s", txt)
	}

	return txt, nil
}

// parseStaticIP 解析静态IP
func (h *HTTPClient) parseStaticIP(recordType string) (string, error) {
	ipResult := h.config.IP
	if strings.Contains(h.config.IP, ",") {
		parts := strings.Split(h.config.IP, ",")
		if recordType == "A" && len(parts) >= 1 {
			ipResult = parts[0]
		} else if recordType == "AAAA" && len(parts) >= 2 {
			ipResult = parts[1]
		}
	}

	if !isValidIP(ipResult, recordType) {
		return "", fmt.Errorf("invalid static IP format")
	}

	return ipResult, nil
}

// getIPFromCloudflareTrace 从Cloudflare trace获取IP
func (h *HTTPClient) getIPFromCloudflareTrace(recordType string) (string, error) {
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
		Timeout: RequestTimeout,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: func(ctx context.Context, _, address string) (net.Conn, error) {
				dialer := net.Dialer{
					Timeout: RequestTimeout,
				}
				return dialer.DialContext(ctx, networkType, address)
			},
		},
	}

	resp, err := client.Get(CloudflareAPI + "/cdn-cgi/trace")
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

	for _, line := range strings.Split(string(body), "\n") {
		if ip, found := strings.CutPrefix(line, "ip="); found {
			if isValidIP(ip, recordType) {
				return ip, nil
			}
		}
	}

	return "", fmt.Errorf("no valid IP found")
}

// dnsQueryTXT 发送DNS TXT查询并返回记录内容
func dnsQueryTXT(server, name string) (string, error) {
	id := uint16(time.Now().UnixNano())
	query := buildDNSQuery(id, name)

	// 优先UDP查询，响应截断（TC标志）时回退到TCP
	return dnsQueryUDP(server, query, id)
}

// buildDNSQuery 构建DNS查询包（TXT类型，CH类）
func buildDNSQuery(id uint16, name string) []byte {
	// Header: ID(2) + Flags(2) + QDCOUNT(2) + ANCOUNT(2) + NSCOUNT(2) + ARCOUNT(2)
	query := make([]byte, 12)
	binary.BigEndian.PutUint16(query[0:2], id)
	query[2], query[3] = 0x01, 0x00 // 设置RD（期望递归）标志
	binary.BigEndian.PutUint16(query[4:6], 1)

	// 编码域名
	for _, label := range strings.Split(name, ".") {
		query = append(query, byte(len(label)))
		query = append(query, label...)
	}
	query = append(query, 0)

	// 查询类型TXT(16)和类CH(3)
	query = append(query, 0x00, 0x10, 0x00, 0x03)

	return query
}

// dnsQueryUDP 通过UDP发送DNS查询
func dnsQueryUDP(server string, query []byte, id uint16) (string, error) {
	conn, err := net.DialTimeout("udp", net.JoinHostPort(server, "53"), RequestTimeout)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(RequestTimeout))

	if _, err := conn.Write(query); err != nil {
		return "", err
	}

	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return "", err
	}

	// 响应被截断（TC标志）时使用TCP重试
	if n >= 4 && binary.BigEndian.Uint16(buf[2:4])&0x0200 != 0 {
		return dnsQueryTCP(server, query, id)
	}

	return parseDNSResponse(buf[:n], id)
}

// dnsQueryTCP 通过TCP发送DNS查询（用于UDP响应截断的情况）
func dnsQueryTCP(server string, query []byte, id uint16) (string, error) {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(server, "53"), RequestTimeout)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(RequestTimeout))

	// TCP查询需要在消息前附加2字节长度前缀
	length := make([]byte, 2)
	binary.BigEndian.PutUint16(length, uint16(len(query)))
	if _, err := conn.Write(append(length, query...)); err != nil {
		return "", err
	}

	// 先读取2字节响应长度前缀
	header := make([]byte, 2)
	if _, err := io.ReadFull(conn, header); err != nil {
		return "", err
	}

	buf := make([]byte, binary.BigEndian.Uint16(header))
	if _, err := io.ReadFull(conn, buf); err != nil {
		return "", err
	}

	return parseDNSResponse(buf, id)
}

// parseDNSResponse 解析DNS响应，返回TXT记录内容
func parseDNSResponse(resp []byte, id uint16) (string, error) {
	if len(resp) < 12 {
		return "", fmt.Errorf("DNS response too short")
	}

	if binary.BigEndian.Uint16(resp[0:2]) != id {
		return "", fmt.Errorf("DNS response ID mismatch")
	}

	flags := binary.BigEndian.Uint16(resp[2:4])
	if flags&0x8000 == 0 {
		return "", fmt.Errorf("not a DNS response")
	}
	if flags&0x000F != 0 {
		return "", fmt.Errorf("DNS response error: rcode=%d", flags&0x000F)
	}

	// 跳过问题段
	offset := skipDNSName(resp, 12)
	if offset < 0 || offset+4 > len(resp) {
		return "", fmt.Errorf("invalid DNS question section")
	}
	offset += 4 // 跳过Type和Class

	// 解析答案段
	anCount := int(binary.BigEndian.Uint16(resp[6:8]))
	var txts []string
	for i := 0; i < anCount; i++ {
		offset = skipDNSName(resp, offset)
		if offset < 0 || offset+10 > len(resp) {
			return "", fmt.Errorf("invalid DNS answer header")
		}

		recordType := binary.BigEndian.Uint16(resp[offset : offset+2])
		rdLength := int(binary.BigEndian.Uint16(resp[offset+8 : offset+10]))
		offset += 10
		if offset+rdLength > len(resp) {
			return "", fmt.Errorf("invalid DNS answer data")
		}

		if recordType == 16 { // TXT
			for i := 0; i < rdLength; {
				length := int(resp[offset+i])
				i++
				if i+length > rdLength {
					return "", fmt.Errorf("invalid TXT record data")
				}
				txts = append(txts, string(resp[offset+i:offset+i+length]))
				i += length
			}
		}

		offset += rdLength
	}

	if len(txts) == 0 {
		return "", fmt.Errorf("no TXT record found")
	}

	return strings.Join(txts, ""), nil
}

// skipDNSName 跳过DNS名称，支持压缩指针
func skipDNSName(msg []byte, offset int) int {
	for offset < len(msg) {
		length := int(msg[offset])
		if length == 0 {
			return offset + 1
		}
		if length&0xC0 == 0xC0 {
			// 压缩指针，固定占用2字节
			return offset + 2
		}
		if length > 63 {
			return -1
		}
		offset += 1 + length
	}
	return -1
}

// isValidIP 验证IP地址格式
func isValidIP(ip, recordType string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	if recordType == "A" {
		return parsedIP.To4() != nil
	}
	return parsedIP.To4() == nil
}

// getRecordTypes 获取要处理的记录类型列表
func (h *HTTPClient) getRecordTypes() []string {
	if h.config.Type == "A_AAAA" {
		return []string{"A", "AAAA"}
	}
	return []string{h.config.Type}
}

// handleUpsert 处理更新模式（自动创建或更新）
func (h *HTTPClient) handleUpsert(zoneID string) {
	for _, recordType := range h.getRecordTypes() {
		fmt.Printf("🔍 Checking %s record...\n", recordType)

		// 获取IP
		wanIP, err := h.getWANIP(recordType)
		if err != nil {
			fmt.Printf("❌ Failed to get WAN IP: %v\n\n", err)
			continue
		}
		fmt.Printf("🌍 WAN IP: %s\n", wanIP)

		// 检查记录是否存在
		recordID, _ := h.getRecordID(zoneID, recordType)

		payload := map[string]interface{}{
			"type":    recordType,
			"name":    h.config.RecordName,
			"content": wanIP,
			"ttl":     h.config.TTL,
			"proxied": h.config.ProxyStatus,
		}

		if recordID == "" {
			// 记录不存在，创建新记录
			fmt.Printf("📝 Record does not exist, creating...\n")

			if _, err := h.request("POST", fmt.Sprintf("/client/v4/zones/%s/dns_records", zoneID), payload); err != nil {
				fmt.Printf("❌ Failed to create record: %v\n\n", err)
				continue
			}

			fmt.Printf("✅ Successfully created %s record\n\n", recordType)
		} else {
			// 记录存在，检查是否需要更新
			fmt.Printf("🔖 Record ID: %s\n", recordID)

			dnsContent, err := h.getDNSRecordContent(zoneID, recordID)
			if err != nil {
				fmt.Printf("❌ Failed to get DNS record: %v\n\n", err)
				continue
			}

			if dnsContent == wanIP {
				fmt.Printf("ℹ️  IP unchanged, no upsert needed\n\n")
				continue
			}

			fmt.Printf("📊 Current DNS: %s\n", dnsContent)
			fmt.Printf("🔄 Updating record...\n")

			if _, err := h.request("PUT", fmt.Sprintf("/client/v4/zones/%s/dns_records/%s", zoneID, recordID), payload); err != nil {
				fmt.Printf("❌ Failed to upsert record: %v\n\n", err)
				continue
			}

			fmt.Printf("✅ Successfully upsertd %s record\n\n", recordType)
		}
	}
}

// handleDelete 处理删除模式
func (h *HTTPClient) handleDelete(zoneID string) {
	types := h.getRecordTypes()
	if h.config.Type == "" {
		// 如果没有指定类型，删除所有类型
		types = []string{"A", "AAAA"}
	}

	for _, recordType := range types {
		fmt.Printf("🗑️  Deleting %s record...\n", recordType)

		// 获取记录ID
		recordID, _ := h.getRecordID(zoneID, recordType)
		if recordID == "" {
			fmt.Printf("ℹ️  %s record does not exist\n\n", recordType)
			continue
		}
		fmt.Printf("🔖 Record ID: %s\n", recordID)

		// 删除记录
		if _, err := h.request("DELETE", fmt.Sprintf("/client/v4/zones/%s/dns_records/%s", zoneID, recordID), nil); err != nil {
			fmt.Printf("❌ Failed to delete record: %v\n\n", err)
			continue
		}

		fmt.Printf("✅ Successfully deleted %s record\n\n", recordType)
	}
}
