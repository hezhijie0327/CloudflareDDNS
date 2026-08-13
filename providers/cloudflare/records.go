package cloudflare

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Record is the DNS record payload sent to the API.
type Record struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
	Proxied bool   `json:"proxied"`
}

// dnsRecord is a DNS record entry as returned by the API.
type dnsRecord struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
	Proxied bool   `json:"proxied"`
}

// RecordID returns the ID of the DNS record matching the configured name
// and type, or "" when the record does not exist.
func (c *Client) RecordID(zoneID, recordType string) (string, error) {
	resp, err := c.request(http.MethodGet, fmt.Sprintf("/client/v4/zones/%s/dns_records?name=%s&type=%s", zoneID, c.section.RecordName, recordType), nil)
	if err != nil {
		return "", err
	}

	var records []dnsRecord
	if err := json.Unmarshal(resp.Result, &records); err != nil {
		return "", fmt.Errorf("decode DNS records: %w", err)
	}
	if len(records) == 0 {
		return "", nil // an absent record is not an error
	}

	return records[0].ID, nil
}

// RecordContent returns the current content of the given DNS record.
func (c *Client) RecordContent(zoneID, recordID string) (string, error) {
	resp, err := c.request(http.MethodGet, fmt.Sprintf("/client/v4/zones/%s/dns_records/%s", zoneID, recordID), nil)
	if err != nil {
		return "", err
	}

	var record dnsRecord
	if err := json.Unmarshal(resp.Result, &record); err != nil {
		return "", fmt.Errorf("decode DNS record: %w", err)
	}
	if record.Content == "" {
		return "", ErrRecordNotFound
	}

	return record.Content, nil
}

// CreateRecord creates a new DNS record in the zone.
func (c *Client) CreateRecord(zoneID string, record Record) error {
	_, err := c.request(http.MethodPost, fmt.Sprintf("/client/v4/zones/%s/dns_records", zoneID), record)
	return err
}

// UpdateRecord updates an existing DNS record.
func (c *Client) UpdateRecord(zoneID, recordID string, record Record) error {
	_, err := c.request(http.MethodPut, fmt.Sprintf("/client/v4/zones/%s/dns_records/%s", zoneID, recordID), record)
	return err
}

// DeleteRecord deletes a DNS record.
func (c *Client) DeleteRecord(zoneID, recordID string) error {
	_, err := c.request(http.MethodDelete, fmt.Sprintf("/client/v4/zones/%s/dns_records/%s", zoneID, recordID), nil)
	return err
}
