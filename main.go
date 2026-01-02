package main

import (
	"context"
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
	Version        = "1.5.0"
	CommitHash     = "dirty"
	BuildTime      = "dev"
	CloudflareAPI  = "https://api.cloudflare.com"
	RequestTimeout = 5 * time.Second
)

// Config 配置结构
type Config struct {
	XAuthEmail     string `json:"x_auth_email"`
	XAuthKey       string `json:"x_auth_key"`
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

	// 验证配置
	if err := config.configValidate(); err != nil {
		fmt.Printf("❌ Invalid config: %v\n", err)
		return
	}

	// 检查网络连接
	if err := checkConnectivity(); err != nil {
		fmt.Printf("❌ Network connectivity check failed: %v\n", err)
		return
	}

	client := &HTTPClient{
		client: &http.Client{Timeout: RequestTimeout},
		config: config,
	}

	// 获取账户名
	accountName, err := client.getAccountName()
	if err != nil {
		fmt.Printf("❌ Failed to get account name: %v\n", err)
		return
	}
	fmt.Printf("👤 Account: %s\n", accountName)

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

	// 如果 update_interval 为 0，只运行一次
	if updateInterval <= 0 {
		switch config.Mode {
		case "upsert":
			client.handleUpsert(zoneID)
		case "delete":
			client.handleDelete(zoneID)
		default:
			fmt.Printf("❌ Invalid mode: %s\n", config.Mode)
		}
		return
	}

	// 定期执行更新
	interval := time.Duration(updateInterval) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	fmt.Printf("⏰ Running every %d seconds. Press Ctrl+C to stop.\n\n", updateInterval)

	// 立即执行一次
	switch config.Mode {
	case "upsert":
		client.handleUpsert(zoneID)
	case "delete":
		client.handleDelete(zoneID)
	default:
		fmt.Printf("❌ Invalid mode: %s\n", config.Mode)
		return
	}

	// 循环执行
	for range ticker.C {
		fmt.Printf("\n🔄 %s - Starting scheduled update...\n", time.Now().Format("2006-01-02 15:04:05"))
		switch config.Mode {
		case "upsert":
			client.handleUpsert(zoneID)
		case "delete":
			client.handleDelete(zoneID)
		}
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

	if c.XAuthEmail == "" || c.XAuthKey == "" || c.ZoneName == "" || c.RecordName == "" {
		return fmt.Errorf("missing required fields (x_auth_email, x_auth_key, zone_name, record_name)")
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
		XAuthEmail:     "your-cloudflare-email@example.com",
		XAuthKey:       "your-cloudflare-api-key",
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

// checkConnectivity 检查网络连接
func checkConnectivity() error {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(CloudflareAPI)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	return nil
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

	req.Header.Set("X-Auth-Email", h.config.XAuthEmail)
	req.Header.Set("X-Auth-Key", h.config.XAuthKey)
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

// getAccountName 获取账户名
func (h *HTTPClient) getAccountName() (string, error) {
	resp, err := h.request("GET", "/client/v4/accounts?page=1&per_page=5", nil)
	if err != nil {
		return "", err
	}

	if results, ok := resp.Result.([]interface{}); ok && len(results) > 0 {
		if result, ok := results[0].(map[string]interface{}); ok {
			if name, ok := result["name"].(string); ok {
				return name, nil
			}
		}
	}

	return "", fmt.Errorf("no account found")
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

	// 从Cloudflare trace获取
	return h.getIPFromCloudflareTrace(recordType)
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
