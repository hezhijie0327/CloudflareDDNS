package main

import (
	"cloudflareddns/cloudflare"
	"cloudflareddns/config"
	"cloudflareddns/ddns"
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	// 自定义帮助信息
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Cloudflare DDNS Tool - Dynamic DNS Update Client\n\n")
		fmt.Fprintf(os.Stderr, "Version: %s\n\n", getVersion())
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  %s -config <config file>     # Start with config file\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -generate-config          # Generate example config\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -version                  # Show version information\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s                            # Start with default config\n\n", os.Args[0])
	}

	// 解析命令行参数
	configPath := flag.String("config", "config.json", "Path to config file")
	generateConfig := flag.Bool("generate-config", false, "Generate example config file")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	// 处理特殊参数
	if *showVersion {
		fmt.Printf("Cloudflare DDNS Tool %s\n", getVersion())
		return
	}

	if *generateConfig {
		printExampleConfig()
		return
	}

	fmt.Printf("🚀 Cloudflare DDNS Tool %s\n\n", getVersion())

	// 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Printf("❌ Failed to load config: %v\n", err)
		return
	}

	// 设置默认值
	cfg.SetDefaults()

	// 检查是否使用了已弃用的认证方式
	if cfg.XAuthEmail != "" && cfg.XAuthKey != "" && cfg.APIToken == "" {
		fmt.Printf("⚠️  WARNING: Using deprecated authentication method (x_auth_email + x_auth_key)\n")
		fmt.Printf("⚠️  Please migrate to using 'api_token' instead\n")
		fmt.Printf("⚠️  You can create an API token at: https://dash.cloudflare.com/profile/api-tokens\n\n")
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		fmt.Printf("❌ Invalid config: %v\n", err)
		return
	}

	client := cloudflare.New(cfg)
	runner := ddns.New(client, cfg)

	// 获取 Zone ID
	zoneID, err := client.ZoneID()
	if err != nil {
		fmt.Printf("❌ Failed to get zone ID: %v\n", err)
		return
	}
	fmt.Printf("🌐 Zone ID: %s\n", zoneID)

	// 执行操作
	fmt.Println()

	// 获取更新间隔
	updateInterval := cfg.Interval()

	// 执行更新的函数
	runUpdate := func() {
		switch cfg.Mode {
		case "upsert":
			runner.Upsert(zoneID)
		case "delete":
			runner.Delete(zoneID)
		default:
			fmt.Printf("❌ Invalid mode: %s\n", cfg.Mode)
		}
	}

	// 如果 update_interval 为 0，只运行一次
	if updateInterval <= 0 {
		runUpdate()
		return
	}

	// 定期执行更新
	interval := time.Duration(updateInterval) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	fmt.Printf("⏰ Running every %d seconds. Press Ctrl+C to stop.\n\n", updateInterval)

	// 立即执行一次
	runUpdate()

	// 循环执行
	for range ticker.C {
		fmt.Printf("\n🔄 %s - Starting scheduled update...\n", time.Now().Format(time.DateTime))
		runUpdate()
	}
}

// printExampleConfig 打印示例配置文件
func printExampleConfig() {
	data, err := config.Example()
	if err != nil {
		fmt.Printf("❌ Failed to generate example config: %v\n", err)
		return
	}

	fmt.Println(data)
}
