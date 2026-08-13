package cloudflare

import (
	"zjddns/internal/log"
)

// Mode 返回本提供商配置的操作模式
func (c *Client) Mode() string {
	return c.cfg.Cloudflare.Mode
}

// Types 返回本提供商处理的记录类型列表
func (c *Client) Types() []string {
	if c.cfg.Cloudflare.Type == "A_AAAA" {
		return []string{"A", "AAAA"}
	}
	return []string{c.cfg.Cloudflare.Type}
}

// Upsert 确保指定类型的记录指向 ip，不存在则创建，存在且内容不同则更新。
// 控制台输出（含失败信息）由本方法打印，返回的 error 供调用方程序化判断。
func (c *Client) Upsert(recordType, ip string) error {
	// 检查记录是否存在
	recordID, _ := c.RecordID(c.zoneID, recordType)

	record := Record{
		Type:    recordType,
		Name:    c.cfg.Cloudflare.RecordName,
		Content: ip,
		TTL:     c.cfg.Cloudflare.TTL,
		Proxied: c.cfg.Cloudflare.ProxyStatus,
	}

	if recordID == "" {
		// 记录不存在，创建新记录
		log.Infof("CLOUDFLARE: Record does not exist, creating...")

		if err := c.CreateRecord(c.zoneID, record); err != nil {
			log.Errorf("CLOUDFLARE: Failed to create record: %v", err)
			return err
		}

		log.Infof("CLOUDFLARE: Successfully created %s record", recordType)
		return nil
	}

	// 记录存在，检查是否需要更新
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

// Delete 删除指定类型的记录。
// 控制台输出（含失败信息）由本方法打印，返回的 error 供调用方程序化判断。
func (c *Client) Delete(recordType string) error {
	// 获取记录ID
	recordID, _ := c.RecordID(c.zoneID, recordType)
	if recordID == "" {
		log.Infof("CLOUDFLARE: %s record does not exist", recordType)
		return nil
	}
	log.Infof("CLOUDFLARE: Record ID: %s", recordID)

	// 删除记录
	if err := c.DeleteRecord(c.zoneID, recordID); err != nil {
		log.Errorf("CLOUDFLARE: Failed to delete record: %v", err)
		return err
	}

	log.Infof("CLOUDFLARE: Successfully deleted %s record", recordType)
	return nil
}
