package main

import (
	"context"
	"fmt"
	"log"

	"github.com/yourorg/lugo"
)

type AppConfig struct {
	Name     string `lua:"name" validate:"required"`
	Version  string `lua:"version" validate:"semver"`
	LogLevel string `lua:"log_level" validate:"oneof=debug info warn error"`
	Server   struct {
		Host string `lua:"host" validate:"hostname"`
		Port int    `lua:"port" validate:"min=1024,max=65535"`
	} `lua:"server"`
}

func main() {
	// Initialize Lugo
	cfg := lugo.New()
	defer cfg.Close()

	// Load configuration
	err := cfg.LoadFile(context.Background(), "config.lua")
	if err != nil {
		log.Fatal(err)
	}

	// Parse into struct
	var appConfig AppConfig
	if err := cfg.Get(context.Background(), "app", &appConfig); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("App: %s v%s\n", appConfig.Name, appConfig.Version)
	fmt.Printf("Server: %s:%d\n", appConfig.Server.Host, appConfig.Server.Port)
}
