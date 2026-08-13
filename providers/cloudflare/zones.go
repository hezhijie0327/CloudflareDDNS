package cloudflare

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// zone is a Cloudflare zone entry.
type zone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ZoneID returns the ID of the configured zone.
func (c *Client) ZoneID() (string, error) {
	resp, err := c.request(http.MethodGet, "/client/v4/zones?name="+c.cfg.Cloudflare.ZoneName, nil)
	if err != nil {
		return "", err
	}

	var zones []zone
	if err := json.Unmarshal(resp.Result, &zones); err != nil {
		return "", fmt.Errorf("decode zones: %w", err)
	}
	if len(zones) == 0 {
		return "", ErrZoneNotFound
	}

	return zones[0].ID, nil
}
