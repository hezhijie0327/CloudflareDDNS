// Package cloudflare implements the Cloudflare DDNS provider: a minimal
// Cloudflare API v4 client with zone lookup and DNS record management.
package cloudflare

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"zjddns/config"
	"zjddns/internal/log"
)

// Client is an HTTP client for the Cloudflare API v4, bound to a
// configured zone.
type Client struct {
	httpClient *http.Client
	baseURL    string
	cfg        *config.Config
	zoneID     string
}

// response is the common Cloudflare API envelope.
type response struct {
	Success bool            `json:"success"`
	Result  json.RawMessage `json:"result"`
	Errors  []apiError      `json:"errors"`
}

// apiError carries a single Cloudflare API error entry.
type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

const (
	// defaultAPIBase is the Cloudflare API v4 base URL.
	defaultAPIBase = "https://api.cloudflare.com"

	// requestTimeout bounds every API call.
	requestTimeout = 5 * time.Second
)

// Sentinel errors for provider states callers may check.
var (
	// ErrZoneNotFound is returned when the configured zone does not exist.
	ErrZoneNotFound = errors.New("zone not found")
	// ErrRecordNotFound is returned when a DNS record lookup has no content.
	ErrRecordNotFound = errors.New("record not found")
)

// New resolves the Zone ID and returns a Client bound to the given
// configuration.
func New(cfg *config.Config) (*Client, error) {
	c := &Client{
		httpClient: &http.Client{Timeout: requestTimeout},
		baseURL:    defaultAPIBase,
		cfg:        cfg,
	}

	zoneID, err := c.ZoneID()
	if err != nil {
		return nil, fmt.Errorf("get zone ID: %w", err)
	}
	log.Infof("CLOUDFLARE: Zone ID: %s", zoneID)
	c.zoneID = zoneID

	return c, nil
}

// request sends an authenticated API request and decodes the JSON
// envelope.
func (c *Client) request(method, path string, payload any) (*response, error) {
	url := c.baseURL + path

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

	// Bearer token authentication.
	req.Header.Set("Authorization", "Bearer "+c.cfg.Cloudflare.APIToken)
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

	var cfResp response
	if err := json.Unmarshal(body, &cfResp); err != nil {
		return nil, err
	}

	if !cfResp.Success {
		return &cfResp, fmt.Errorf("API error: %v", cfResp.Errors)
	}

	return &cfResp, nil
}
