package main

import (
	"flag"
	"fmt"
	"os"
	"time"
	"zjddns/config"
	"zjddns/ddns"
	"zjddns/providers"
)

func main() {
	// 自定义帮助信息
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "ZJDDNS - Dynamic DNS Update Client\n\n")
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
		fmt.Printf("%s %s\n", ProjectName, getVersion())
		return
	}

	if *generateConfig {
		printExampleConfig()
		return
	}

	versionStr := getVersion()
	if _, err := fmt.Print(banner(versionStr)); err != nil {
		fmt.Fprintf(os.Stderr, "warning: writing banner: %v\n", err)
	}

	// 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Printf("❌ Failed to load config: %v\n", err)
		return
	}

	// 设置默认值
	cfg.SetDefaults()

	// 验证配置
	if err := cfg.Validate(); err != nil {
		fmt.Printf("❌ Invalid config: %v\n", err)
		return
	}

	// 构造所有已配置的 Provider（可同时使用多个提供商）
	ps, err := providers.All(cfg)
	if err != nil {
		fmt.Printf("❌ Failed to initialize provider: %v\n", err)
		return
	}
	runner := ddns.New(ps, cfg)

	// 执行操作
	fmt.Println()

	// 获取更新间隔
	updateInterval := cfg.Interval()

	// 执行更新的函数
	runUpdate := func() {
		switch cfg.Mode {
		case "upsert":
			runner.Upsert()
		case "delete":
			runner.Delete()
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
