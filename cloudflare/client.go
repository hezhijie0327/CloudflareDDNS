// Package cloudflare implements a minimal Cloudflare API v4 client
// for zone lookup and DNS record management.
package cloudflare

import (
	"cloudflareddns/config"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client 封装的HTTP客户端
type Client struct {
	httpClient *http.Client
	cfg        *config.Config
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

// New returns a Client bound to the given configuration.
func New(cfg *config.Config) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: RequestTimeout},
		cfg:        cfg,
	}
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
	if c.cfg.APIToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIToken)
	} else {
		// 向后兼容旧的认证方式
		req.Header.Set("X-Auth-Email", c.cfg.XAuthEmail)
		req.Header.Set("X-Auth-Key", c.cfg.XAuthKey)
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
