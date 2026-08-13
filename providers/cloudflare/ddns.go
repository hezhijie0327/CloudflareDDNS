package cloudflare

import (
	"zjddns/config"
	"zjddns/internal/log"
)

// Mode returns the operation mode configured for this provider.
func (c *Client) Mode() string {
	return c.cfg.Cloudflare.Mode
}

// Types returns the record types this provider handles.
func (c *Client) Types() []string {
	if c.cfg.Cloudflare.Type == config.TypeAAndAAAA {
		return []string{config.TypeA, config.TypeAAAA}
	}
	return []string{c.cfg.Cloudflare.Type}
}

// Upsert ensures the record of the given type points to ip, creating it
// when missing or updating it when the current content differs.
//
// Record-level logging (including failures) is owned by the provider;
// the returned error is informational for callers.
func (c *Client) Upsert(recordType, ip string) error {
	// Check whether the record exists.
	recordID, _ := c.RecordID(c.zoneID, recordType)

	record := Record{
		Type:    recordType,
		Name:    c.cfg.Cloudflare.RecordName,
		Content: ip,
		TTL:     c.cfg.Cloudflare.TTL,
		Proxied: c.cfg.Cloudflare.ProxyStatus,
	}

	if recordID == "" {
		// Missing record: create it.
		log.Infof("CLOUDFLARE: Record does not exist, creating...")

		if err := c.CreateRecord(c.zoneID, record); err != nil {
			log.Errorf("CLOUDFLARE: Failed to create record: %v", err)
			return err
		}

		log.Infof("CLOUDFLARE: Successfully created %s record", recordType)
		return nil
	}

	// Existing record: update it when the content differs.
	log.Infof("CLOUDFLARE: Record ID: %s", recordID)

	dnsContent, err := c.RecordContent(c.zoneID, recordID)
	if err != nil {
		log.Errorf("CLOUDFLARE: Failed to get DNS record: %v", err)
		return err
	}

	if dnsContent == ip {
		log.Infof("CLOUDFLARE: IP unchanged, no upsert needed")
		return nil
	}

	log.Infof("CLOUDFLARE: Current DNS: %s", dnsContent)
	log.Infof("CLOUDFLARE: Updating record...")

	if err := c.UpdateRecord(c.zoneID, recordID, record); err != nil {
		log.Errorf("CLOUDFLARE: Failed to upsert record: %v", err)
		return err
	}

	log.Infof("CLOUDFLARE: Successfully upserted %s record", recordType)
	return nil
}

// Delete removes the record of the given type.
//
// Record-level logging (including failures) is owned by the provider;
// the returned error is informational for callers.
func (c *Client) Delete(recordType string) error {
	// Look up the record ID.
	recordID, _ := c.RecordID(c.zoneID, recordType)
	if recordID == "" {
		log.Infof("CLOUDFLARE: %s record does not exist", recordType)
		return nil
	}
	log.Infof("CLOUDFLARE: Record ID: %s", recordID)

	if err := c.DeleteRecord(c.zoneID, recordID); err != nil {
		log.Errorf("CLOUDFLARE: Failed to delete record: %v", err)
		return err
	}

	log.Infof("CLOUDFLARE: Successfully deleted %s record", recordType)
	return nil
}
