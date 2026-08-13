package cloudflare

import "errors"

// ZoneID 获取Zone ID
func (c *Client) ZoneID() (string, error) {
	resp, err := c.request("GET", "/client/v4/zones?name="+c.cfg.ZoneName, nil)
	if err != nil {
		return "", err
	}

	if results, ok := resp.Result.([]any); ok && len(results) > 0 {
		if result, ok := results[0].(map[string]any); ok {
			if id, ok := result["id"].(string); ok {
				return id, nil
			}
		}
	}

	return "", errors.New("zone not found")
}
