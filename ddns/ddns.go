// Package ddns orchestrates DNS record upsert and delete operations,
// combining the Cloudflare API client with WAN IP detection.
package ddns

import (
	"cloudflareddns/cloudflare"
	"cloudflareddns/config"
	"cloudflareddns/internal/ipdetect"
	"fmt"
)

// Runner 执行DDNS更新与删除操作的编排器
type Runner struct {
	client   *cloudflare.Client
	detector *ipdetect.Detector
	cfg      *config.Config
}

// New returns a Runner bound to the given client and configuration.
func New(client *cloudflare.Client, cfg *config.Config) *Runner {
	return &Runner{
		client:   client,
		detector: ipdetect.New(cfg.IP),
		cfg:      cfg,
	}
}

// recordTypes 获取要处理的记录类型列表
func (r *Runner) recordTypes() []string {
	if r.cfg.Type == "A_AAAA" {
		return []string{"A", "AAAA"}
	}
	return []string{r.cfg.Type}
}

// Upsert 处理更新模式（自动创建或更新）
func (r *Runner) Upsert(zoneID string) {
	for _, recordType := range r.recordTypes() {
		fmt.Printf("🔍 Checking %s record...\n", recordType)

		// 获取IP
		wanIP, err := r.detector.WANIP(recordType)
		if err != nil {
			fmt.Printf("❌ Failed to get WAN IP: %v\n\n", err)
			continue
		}
		fmt.Printf("🌍 WAN IP: %s\n", wanIP)

		// 检查记录是否存在
		recordID, _ := r.client.RecordID(zoneID, recordType)

		record := cloudflare.Record{
			Type:    recordType,
			Name:    r.cfg.RecordName,
			Content: wanIP,
			TTL:     r.cfg.TTL,
			Proxied: r.cfg.ProxyStatus,
		}

		if recordID == "" {
			// 记录不存在，创建新记录
			fmt.Printf("📝 Record does not exist, creating...\n")

			if err := r.client.CreateRecord(zoneID, record); err != nil {
				fmt.Printf("❌ Failed to create record: %v\n\n", err)
				continue
			}

			fmt.Printf("✅ Successfully created %s record\n\n", recordType)
		} else {
			// 记录存在，检查是否需要更新
			fmt.Printf("🔖 Record ID: %s\n", recordID)

			dnsContent, err := r.client.RecordContent(zoneID, recordID)
			if err != nil {
				fmt.Printf("❌ Failed to get DNS record: %v\n\n", err)
				continue
			}

			if dnsContent == wanIP {
				fmt.Printf("ℹ️  IP unchanged, no upsert needed\n\n")
				continue
			}

			fmt.Printf("📊 Current DNS: %s\n", dnsContent)
			fmt.Printf("🔄 Updating record...\n")

			if err := r.client.UpdateRecord(zoneID, recordID, record); err != nil {
				fmt.Printf("❌ Failed to upsert record: %v\n\n", err)
				continue
			}

			fmt.Printf("✅ Successfully upserted %s record\n\n", recordType)
		}
	}
}

// Delete 处理删除模式
func (r *Runner) Delete(zoneID string) {
	types := r.recordTypes()
	if r.cfg.Type == "" {
		// 如果没有指定类型，删除所有类型
		types = []string{"A", "AAAA"}
	}

	for _, recordType := range types {
		fmt.Printf("🗑️  Deleting %s record...\n", recordType)

		// 获取记录ID
		recordID, _ := r.client.RecordID(zoneID, recordType)
		if recordID == "" {
			fmt.Printf("ℹ️  %s record does not exist\n\n", recordType)
			continue
		}
		fmt.Printf("🔖 Record ID: %s\n", recordID)

		// 删除记录
		if err := r.client.DeleteRecord(zoneID, recordID); err != nil {
			fmt.Printf("❌ Failed to delete record: %v\n\n", err)
			continue
		}

		fmt.Printf("✅ Successfully deleted %s record\n\n", recordType)
	}
}
