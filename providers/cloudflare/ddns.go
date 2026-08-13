package cloudflare

import (
	"zjddns/config"
	"zjddns/internal/ipdetect"
	"zjddns/internal/log"
)

// Mode returns the operation mode configured for this provider.
func (c *Client) Mode() string {
	return c.section.Mode
}

// Types returns the record types this provider handles.
func (c *Client) Types() []string {
	if c.section.Type == config.TypeAAndAAAA {
		return []string{config.TypeA, config.TypeAAAA}
	}
	return []string{c.section.Type}
}

// Upsert ensures the record of the given type points to ip.Value,
// creating it when missing or updating it when the current content
// differs.
//
// Record-level logging (including failures) is owned by the provider;
// the returned error is informational for callers. State changes are
// logged at Info carrying the record name and the IP change; routine
// per-cycle detail is logged at Debug.
func (c *Client) Upsert(recordType string, ip ipdetect.IP) error {
	recordName := c.section.RecordName

	// Check whether the record exists.
	recordID, _ := c.RecordID(c.zoneID, recordType)

	record := Record{
		Type:    recordType,
		Name:    recordName,
		Content: ip.Value,
		TTL:     c.section.TTL,
		Proxied: c.section.ProxyStatus,
	}

	if recordID == "" {
		// Missing record: create it.
		log.Debugf("CLOUDFLARE: Record does not exist, creating...")

		if err := c.CreateRecord(c.zoneID, record); err != nil {
			log.Errorf("CLOUDFLARE: Failed to create %s record %s: %v", recordType, recordName, err)
			return err
		}

		log.Infof("CLOUDFLARE: Created %s record %s: %s (%s)", recordType, recordName, ip.Value, ip.Source)
		return nil
	}

	// Existing record: update it when the content differs.
	log.Debugf("CLOUDFLARE: Record ID: %s", recordID)

	dnsContent, err := c.RecordContent(c.zoneID, recordID)
	if err != nil {
		log.Errorf("CLOUDFLARE: Failed to get DNS record: %v", err)
		return err
	}

	if dnsContent == ip.Value {
		log.Debugf("CLOUDFLARE: %s record %s unchanged", recordType, recordName)
		return nil
	}

	if err := c.UpdateRecord(c.zoneID, recordID, record); err != nil {
		log.Errorf("CLOUDFLARE: Failed to update %s record %s: %v", recordType, recordName, err)
		return err
	}

	log.Infof("CLOUDFLARE: Updated %s record %s: %s -> %s (%s)", recordType, recordName, dnsContent, ip.Value, ip.Source)
	return nil
}

// Delete removes the record of the given type.
//
// Record-level logging (including failures) is owned by the provider;
// the returned error is informational for callers. Deletions are logged
// at Info carrying the record name; routine detail at Debug.
func (c *Client) Delete(recordType string) error {
	recordName := c.section.RecordName

	// Look up the record ID.
	recordID, _ := c.RecordID(c.zoneID, recordType)
	if recordID == "" {
		log.Debugf("CLOUDFLARE: %s record %s does not exist", recordType, recordName)
		return nil
	}
	log.Debugf("CLOUDFLARE: Record ID: %s", recordID)

	if err := c.DeleteRecord(c.zoneID, recordID); err != nil {
		log.Errorf("CLOUDFLARE: Failed to delete %s record %s: %v", recordType, recordName, err)
		return err
	}

	log.Infof("CLOUDFLARE: Deleted %s record %s", recordType, recordName)
	return nil
}
