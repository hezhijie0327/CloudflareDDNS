package cloudflare

import "fmt"

// Upsert 确保指定类型的记录指向 ip，不存在则创建，存在且内容不同则更新。
// 控制台输出（含失败信息）由本方法打印，返回的 error 供调用方程序化判断。
func (c *Client) Upsert(recordType, ip string) error {
	// 检查记录是否存在
	recordID, _ := c.RecordID(c.zoneID, recordType)

	record := Record{
		Type:    recordType,
		Name:    c.cfg.Cloudflare.RecordName,
		Content: ip,
		TTL:     c.cfg.TTL,
		Proxied: c.cfg.Cloudflare.ProxyStatus,
	}

	if recordID == "" {
		// 记录不存在，创建新记录
		fmt.Printf("📝 Record does not exist, creating...\n")

		if err := c.CreateRecord(c.zoneID, record); err != nil {
			fmt.Printf("❌ Failed to create record: %v\n\n", err)
			return err
		}

		fmt.Printf("✅ Successfully created %s record\n\n", recordType)
		return nil
	}

	// 记录存在，检查是否需要更新
	fmt.Printf("🔖 Record ID: %s\n", recordID)

	dnsContent, err := c.RecordContent(c.zoneID, recordID)
	if err != nil {
		fmt.Printf("❌ Failed to get DNS record: %v\n\n", err)
		return err
	}

	if dnsContent == ip {
		fmt.Printf("ℹ️  IP unchanged, no upsert needed\n\n")
		return nil
	}

	fmt.Printf("📊 Current DNS: %s\n", dnsContent)
	fmt.Printf("🔄 Updating record...\n")

	if err := c.UpdateRecord(c.zoneID, recordID, record); err != nil {
		fmt.Printf("❌ Failed to upsert record: %v\n\n", err)
		return err
	}

	fmt.Printf("✅ Successfully upserted %s record\n\n", recordType)
	return nil
}

// Delete 删除指定类型的记录。
// 控制台输出（含失败信息）由本方法打印，返回的 error 供调用方程序化判断。
func (c *Client) Delete(recordType string) error {
	// 获取记录ID
	recordID, _ := c.RecordID(c.zoneID, recordType)
	if recordID == "" {
		fmt.Printf("ℹ️  %s record does not exist\n\n", recordType)
		return nil
	}
	fmt.Printf("🔖 Record ID: %s\n", recordID)

	// 删除记录
	if err := c.DeleteRecord(c.zoneID, recordID); err != nil {
		fmt.Printf("❌ Failed to delete record: %v\n\n", err)
		return err
	}

	fmt.Printf("✅ Successfully deleted %s record\n\n", recordType)
	return nil
}
