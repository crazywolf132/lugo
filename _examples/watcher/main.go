package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yourorg/lugo"
)

type DynamicConfig struct {
	Features struct {
		EnableCache   bool    `lua:"enable_cache"`
		CacheTTL      int     `lua:"cache_ttl"`
		SamplingRate  float64 `lua:"sampling_rate"`
		EnableMetrics bool    `lua:"enable_metrics"`
	} `lua:"features"`
	Limits struct {
		MaxConnections int           `lua:"max_connections"`
		RequestTimeout time.Duration `lua:"request_timeout"`
		RateLimit      int           `lua:"rate_limit"`
	} `lua:"limits"`
}

func main() {
	// Initialize Lugo
	cfg := lugo.New()
	defer cfg.Close()

	// Create configuration watcher
	watcher, err := cfg.NewWatcher(lugo.WatcherConfig{
		Paths:        []string{"config.lua"},
		PollInterval: 5 * time.Second,
		OnReload: func(err error) {
			if err != nil {
				log.Printf("Error reloading config: %v", err)
				return
			}
			log.Println("Configuration reloaded successfully")
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	// Load initial configuration
	var config DynamicConfig
	if err := cfg.Get(context.Background(), "config", &config); err != nil {
		log.Fatal(err)
	}

	// Wait for signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Watching for configuration changes. Press Ctrl+C to exit.")
	<-sigChan
}
