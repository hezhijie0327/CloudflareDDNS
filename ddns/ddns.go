// Package ddns orchestrates DNS record upsert and delete operations,
// combining DDNS providers with WAN IP detection.
package ddns

import (
	"fmt"
	"zjddns/config"
	"zjddns/internal/ipdetect"
)

// Runner 执行DDNS更新与删除操作的编排器
type Runner struct {
	providers []Provider
	detector  *ipdetect.Detector
	cfg       *config.Config
}

// New returns a Runner bound to the given providers and configuration.
func New(providers []Provider, cfg *config.Config) *Runner {
	return &Runner{
		providers: providers,
		detector:  ipdetect.New(cfg.IP),
		cfg:       cfg,
	}
}

// recordTypes 获取要处理的记录类型列表
func (r *Runner) recordTypes() []string {
	if r.cfg.Type == "A_AAAA" {
		return []string{"A", "AAAA"}
	}
	return []string{r.cfg.Type}
}

// Upsert 处理更新模式（自动创建或更新）。
// WAN IP 每种记录类型检测一次，推送给所有已配置的提供商；
// 记录级别的输出（Record ID、创建/更新详情）由各 Provider 打印。
func (r *Runner) Upsert() {
	for _, recordType := range r.recordTypes() {
		fmt.Printf("🔍 Checking %s record...\n", recordType)

		// 获取IP
		wanIP, err := r.detector.WANIP(recordType)
		if err != nil {
			fmt.Printf("❌ Failed to get WAN IP: %v\n\n", err)
			continue
		}
		fmt.Printf("🌍 WAN IP: %s\n", wanIP)

		// 各 Provider 自行处理创建/更新细节与输出
		for _, p := range r.providers {
			_ = p.Upsert(recordType, wanIP)
		}
	}
}

// Delete 处理删除模式。记录级别的输出由各 Provider 打印。
func (r *Runner) Delete() {
	types := r.recordTypes()
	if r.cfg.Type == "" {
		// 如果没有指定类型，删除所有类型
		types = []string{"A", "AAAA"}
	}

	for _, recordType := range types {
		fmt.Printf("🗑️  Deleting %s record...\n", recordType)

		// 各 Provider 自行处理删除细节与输出
		for _, p := range r.providers {
			_ = p.Delete(recordType)
		}
	}
}
