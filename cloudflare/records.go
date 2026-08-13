package cloudflare

import (
	"errors"
	"fmt"
)

// Record is the DNS record payload sent to the API.
type Record struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
	Proxied bool   `json:"proxied"`
}

// RecordID 获取DNS记录ID，记录不存在时返回空字符串
func (c *Client) RecordID(zoneID, recordType string) (string, error) {
	resp, err := c.request("GET", fmt.Sprintf("/client/v4/zones/%s/dns_records?name=%s&type=%s", zoneID, c.cfg.RecordName, recordType), nil)
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

	return "", nil // 不存在返回空字符串，不报错
}

// RecordContent 获取DNS记录内容
func (c *Client) RecordContent(zoneID, recordID string) (string, error) {
	resp, err := c.request("GET", fmt.Sprintf("/client/v4/zones/%s/dns_records/%s", zoneID, recordID), nil)
	if err != nil {
		return "", err
	}

	if result, ok := resp.Result.(map[string]any); ok {
		if content, ok := result["content"].(string); ok {
			return content, nil
		}
	}

	return "", errors.New("record content not found")
}

// CreateRecord creates a new DNS record in the zone.
func (c *Client) CreateRecord(zoneID string, record Record) error {
	_, err := c.request("POST", fmt.Sprintf("/client/v4/zones/%s/dns_records", zoneID), record)
	return err
}

// UpdateRecord updates an existing DNS record.
func (c *Client) UpdateRecord(zoneID, recordID string, record Record) error {
	_, err := c.request("PUT", fmt.Sprintf("/client/v4/zones/%s/dns_records/%s", zoneID, recordID), record)
	return err
}

// DeleteRecord deletes a DNS record.
func (c *Client) DeleteRecord(zoneID, recordID string) error {
	_, err := c.request("DELETE", fmt.Sprintf("/client/v4/zones/%s/dns_records/%s", zoneID, recordID), nil)
	return err
}
