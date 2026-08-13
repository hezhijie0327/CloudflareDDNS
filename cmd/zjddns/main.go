package main

import (
	"flag"
	"fmt"
	"os"
	"time"
	"zjddns/config"
	"zjddns/ddns"
	"zjddns/internal/log"
	"zjddns/providers"
)

func main() {
	// Custom usage text.
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "ZJDDNS - Dynamic DNS Update Client\n\n")
		fmt.Fprintf(os.Stderr, "Version: %s\n\n", getVersion())
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  %s -config <config file>     # Start with config file\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -generate-config          # Generate example config\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -version                  # Show version information\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s                            # Start with default config\n\n", os.Args[0])
	}

	// Parse the command line flags.
	configPath := flag.String("config", "config.json", "Path to config file")
	generateConfig := flag.Bool("generate-config", false, "Generate example config file")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	// Handle the special commands.
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

	// Load the configuration.
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Errorf("CONFIG: Failed to load config: %v", err)
		return
	}

	// Apply the defaults.
	cfg.SetDefaults()

	// Apply the log level (supports "debug:COMP1,COMP2" component filters).
	lvl, components := log.ParseLevelFilter(cfg.LogLevel, log.Info)
	log.SetLevelFilter(lvl, components)

	// Validate the configuration.
	if err := cfg.Validate(); err != nil {
		log.Errorf("CONFIG: Invalid config: %v", err)
		return
	}

	// Construct every configured provider; multiple providers may run at
	// once, each updating its own records.
	ps, err := providers.All(cfg)
	if err != nil {
		log.Errorf("CONFIG: Failed to initialize provider: %v", err)
		return
	}
	runner := ddns.New(ps, cfg)

	// The update interval drives the scheduling loop.
	updateInterval := cfg.Interval()

	// Each provider runs according to its own configured mode.
	runUpdate := func() {
		runner.Run()
	}

	// An interval of 0 means run once.
	if updateInterval <= 0 {
		runUpdate()
		return
	}

	// Run periodically.
	interval := time.Duration(updateInterval) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Infof("DDNS: Running every %d seconds. Press Ctrl+C to stop.", updateInterval)

	// Run immediately, then on every tick.
	runUpdate()

	for range ticker.C {
		log.Infof("DDNS: %s - Starting scheduled update...", time.Now().Format(time.DateTime))
		runUpdate()
	}
}

// printExampleConfig writes the example configuration to stdout.
func printExampleConfig() {
	data, err := config.Example()
	if err != nil {
		log.Errorf("CONFIG: Failed to generate example config: %v", err)
		return
	}

	fmt.Println(data)
}
