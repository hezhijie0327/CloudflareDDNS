// Package cloudflare implements the Cloudflare DDNS provider (API v4)
// with zone lookup and DNS record management.
package cloudflare

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"zjddns/config"
	"zjddns/internal/log"
)

// Client 封装的HTTP客户端
type Client struct {
	httpClient *http.Client
	cfg        *config.Config
	zoneID     string
}

// Response API响应结构
type Response struct {
	Success bool             `json:"success"`
	Result  any              `json:"result"`
	Errors  []map[string]any `json:"errors"`
}

const (
	// APIBase is the base URL of the Cloudflare API v4.
	APIBase = "https://api.cloudflare.com"

	// RequestTimeout bounds every API call.
	RequestTimeout = 5 * time.Second
)

// New resolves the Zone ID and returns a Client bound to the given
// configuration.
func New(cfg *config.Config) (*Client, error) {
	c := &Client{
		httpClient: &http.Client{Timeout: RequestTimeout},
		cfg:        cfg,
	}

	// 检查是否使用了已弃用的认证方式
	if cfg.Cloudflare.XAuthEmail != "" && cfg.Cloudflare.XAuthKey != "" && cfg.Cloudflare.APIToken == "" {
		log.Warnf("CLOUDFLARE: Using deprecated authentication method (x_auth_email + x_auth_key)")
		log.Warnf("CLOUDFLARE: Please migrate to using 'api_token' instead")
		log.Warnf("CLOUDFLARE: You can create an API token at: https://dash.cloudflare.com/profile/api-tokens")
	}

	zoneID, err := c.ZoneID()
	if err != nil {
		return nil, fmt.Errorf("get zone ID: %w", err)
	}
	log.Infof("CLOUDFLARE: Zone ID: %s", zoneID)
	c.zoneID = zoneID

	return c, nil
}

// request 发送HTTP请求
func (c *Client) request(method, path string, payload any) (*Response, error) {
	url := APIBase + path

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
	if c.cfg.Cloudflare.APIToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.Cloudflare.APIToken)
	} else {
		// 向后兼容旧的认证方式
		req.Header.Set("X-Auth-Email", c.cfg.Cloudflare.XAuthEmail)
		req.Header.Set("X-Auth-Key", c.cfg.Cloudflare.XAuthKey)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
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

	var cfResp Response
	if err := json.Unmarshal(body, &cfResp); err != nil {
		return nil, err
	}

	if !cfResp.Success {
		return &cfResp, fmt.Errorf("API error: %v", cfResp.Errors)
	}

	return &cfResp, nil
}
