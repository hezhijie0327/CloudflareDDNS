// Package ddns orchestrates DNS record upsert and delete operations,
// combining DDNS providers with WAN IP detection.
package ddns

import (
	"slices"
	"zjddns/config"
	"zjddns/internal/ipdetect"
	"zjddns/internal/log"
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

// unionTypes 收集给定 Provider 列表的记录类型（去重、保持配置顺序）
func (r *Runner) unionTypes(providers []Provider) []string {
	var types []string
	for _, p := range providers {
		for _, t := range p.Types() {
			if !slices.Contains(types, t) {
				types = append(types, t)
			}
		}
	}
	return types
}

// Run 按各 Provider 配置的操作模式执行 upsert 或 delete
func (r *Runner) Run() {
	var upserters, deleters []Provider
	for _, p := range r.providers {
		switch p.Mode() {
		case "upsert":
			upserters = append(upserters, p)
		case "delete":
			deleters = append(deleters, p)
		default:
			log.Errorf("DDNS: Invalid mode: %s", p.Mode())
		}
	}

	r.upsert(upserters)
	r.delete(deleters)
}

// upsert 处理更新模式（自动创建或更新）。
// WAN IP 每种记录类型检测一次，推送给所有处理该类型的 Provider；
// 记录级别的输出由各 Provider 打印。
func (r *Runner) upsert(providers []Provider) {
	if len(providers) == 0 {
		return
	}

	for _, recordType := range r.unionTypes(providers) {
		log.Infof("DDNS: Checking %s record...", recordType)

		// 获取IP
		wanIP, err := r.detector.WANIP(recordType)
		if err != nil {
			log.Errorf("DDNS: Failed to get WAN IP: %v", err)
			continue
		}
		log.Infof("DDNS: WAN IP: %s", wanIP)

		// 各 Provider 自行处理创建/更新细节与输出
		for _, p := range providers {
			if slices.Contains(p.Types(), recordType) {
				_ = p.Upsert(recordType, wanIP)
			}
		}
	}
}

// delete 处理删除模式。各 Provider 删除自己配置的记录类型。
func (r *Runner) delete(providers []Provider) {
	for _, p := range providers {
		for _, recordType := range p.Types() {
			log.Infof("DDNS: Deleting %s record...", recordType)

			// Provider 自行处理删除细节与输出
			_ = p.Delete(recordType)
		}
	}
}
