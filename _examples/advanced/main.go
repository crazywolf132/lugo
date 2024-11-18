package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/yourorg/lugo"
)

// Complex configuration example with multiple features
type ServiceConfig struct {
	// Basic service info
	Name        string `lua:"name" validate:"required"`
	Version     string `lua:"version" validate:"semver"`
	Environment string `lua:"environment" validate:"oneof=dev staging prod"`

	// Networking
	Network struct {
		Host         string        `lua:"host" validate:"hostname"`
		Port         int           `lua:"port" validate:"min=1024,max=65535"`
		ReadTimeout  time.Duration `lua:"read_timeout" validate:"min=1s,max=1m"`
		WriteTimeout time.Duration `lua:"write_timeout" validate:"min=1s,max=1m"`
		TLS          struct {
			Enabled    bool     `lua:"enabled"`
			CertFile   string   `lua:"cert_file" validate:"required_if=Enabled true"`
			KeyFile    string   `lua:"key_file" validate:"required_if=Enabled true"`
			MinVersion string   `lua:"min_version" validate:"oneof=1.2 1.3"`
			Ciphers    []string `lua:"ciphers" validate:"dive,oneof=TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256 TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"`
		} `lua:"tls"`
	} `lua:"network"`

	// Database configurations
	Databases map[string]struct {
		Driver         string            `lua:"driver" validate:"oneof=postgres mysql sqlite"`
		Host           string            `lua:"host" validate:"required_if=Driver postgres mysql"`
		Port           int               `lua:"port" validate:"required_if=Driver postgres mysql"`
		Name           string            `lua:"name" validate:"required"`
		Username       string            `lua:"username"`
		Password       string            `lua:"password"`
		Parameters     map[string]string `lua:"parameters"`
		MaxConnections int               `lua:"max_connections" validate:"min=1,max=1000"`
		IdleTimeout    time.Duration     `lua:"idle_timeout" validate:"min=1s,max=1h"`
	} `lua:"databases" validate:"required,min=1"`

	// Cache configuration
	Cache struct {
		Type      string        `lua:"type" validate:"oneof=redis memcached memory"`
		Servers   []string      `lua:"servers" validate:"required_if=Type redis memcached,dive,hostname_port"`
		KeyPrefix string        `lua:"key_prefix"`
		TTL       time.Duration `lua:"ttl" validate:"required,min=1s,max=24h"`
	} `lua:"cache"`

	// Feature flags and toggles
	Features map[string]struct {
		Enabled      bool      `lua:"enabled"`
		Percentage   float64   `lua:"percentage" validate:"min=0,max=100"`
		AllowedIPs   []string  `lua:"allowed_ips" validate:"dive,ip"`
		AllowedUsers []string  `lua:"allowed_users" validate:"dive,email"`
		StartTime    time.Time `lua:"start_time"`
		EndTime      time.Time `lua:"end_time" validate:"gtfield=StartTime"`
	} `lua:"features"`

	// Monitoring and metrics
	Monitoring struct {
		Enabled bool `lua:"enabled"`
		Metrics struct {
			Port     int           `lua:"port" validate:"min=1024,max=65535"`
			Path     string        `lua:"path" validate:"startswith=/"`
			Interval time.Duration `lua:"interval" validate:"min=1s,max=1m"`
		} `lua:"metrics"`
		Tracing struct {
			Enabled    bool    `lua:"enabled"`
			Endpoint   string  `lua:"endpoint" validate:"required_if=Enabled true,url"`
			SampleRate float64 `lua:"sample_rate" validate:"min=0,max=1"`
			BatchSize  int     `lua:"batch_size" validate:"min=1,max=1000"`
		} `lua:"tracing"`
		HealthCheck struct {
			Path     string        `lua:"path" validate:"startswith=/"`
			Interval time.Duration `lua:"interval" validate:"min=1s,max=1m"`
			Timeout  time.Duration `lua:"timeout" validate:"min=100ms,max=10s"`
		} `lua:"health_check"`
	} `lua:"monitoring"`

	// Resource limits
	Limits struct {
		MaxRequests     int           `lua:"max_requests" validate:"min=1"`
		MaxConcurrent   int           `lua:"max_concurrent" validate:"min=1"`
		RateLimit       float64       `lua:"rate_limit" validate:"min=0"`
		BurstLimit      int           `lua:"burst_limit" validate:"min=1"`
		RequestTimeout  time.Duration `lua:"request_timeout" validate:"min=1s,max=5m"`
		ShutdownTimeout time.Duration `lua:"shutdown_timeout" validate:"min=1s,max=5m"`
		MaxRequestSize  int64         `lua:"max_request_size" validate:"min=1024,max=104857600"` // 1KB to 100MB
	} `lua:"limits"`
}

func main() {
	ctx := context.Background()

	// Initialize with all features enabled
	cfg := lugo.New(
		lugo.WithValidation(true),
		lugo.WithTemplating(true),
		lugo.WithEnvironmentOverrides(true),
	)
	defer cfg.Close()

	// Load base configuration
	if err := cfg.LoadFile(ctx, "config.lua"); err != nil {
		log.Fatal(err)
	}

	// Load environment-specific overrides
	if err := cfg.LoadFile(ctx, fmt.Sprintf("config.%s.lua", cfg.Environment())); err != nil && !os.IsNotExist(err) {
		log.Fatal(err)
	}

	// Parse configuration
	var config ServiceConfig
	if err := cfg.Get(ctx, "service", &config); err != nil {
		log.Fatal(err)
	}

	// Start configuration watcher
	watcher, err := cfg.NewWatcher(lugo.WatcherConfig{
		Paths:        []string{"config.lua", fmt.Sprintf("config.%s.lua", cfg.Environment())},
		PollInterval: 30 * time.Second,
		OnReload: func(err error) {
			if err != nil {
				log.Printf("Error reloading config: %v", err)
				return
			}
			// Re-parse configuration
			var newConfig ServiceConfig
			if err := cfg.Get(ctx, "service", &newConfig); err != nil {
				log.Printf("Error parsing new config: %v", err)
				return
			}
			// Apply new configuration (implement your own logic)
			applyNewConfiguration(newConfig)
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	// ... rest of the application
}

func applyNewConfiguration(config ServiceConfig) {
	// Implementation for applying new configuration
	log.Printf("Applying new configuration: %+v", config)
}
