package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/yourorg/lugo"
)

type ServiceConfig struct {
	Environment string `lua:"environment"`
	Database    struct {
		Host     string `lua:"host"`
		Port     int    `lua:"port"`
		Name     string `lua:"name"`
		Username string `lua:"username"`
		Password string `lua:"password"`
	} `lua:"database"`
	Redis struct {
		Host string `lua:"host"`
		Port int    `lua:"port"`
	} `lua:"redis"`
}

func main() {
	// Initialize Lugo
	cfg := lugo.New()
	defer cfg.Close()

	// Template variables
	vars := lugo.TemplateConfig{
		"env":        os.Getenv("APP_ENV"),
		"db_host":    os.Getenv("DB_HOST"),
		"db_pass":    os.Getenv("DB_PASSWORD"),
		"redis_host": os.Getenv("REDIS_HOST"),
	}

	// Load and process template
	if err := cfg.LoadTemplate(context.Background(), "config.lua.tmpl", vars); err != nil {
		log.Fatal(err)
	}

	// Parse configuration
	var config ServiceConfig
	if err := cfg.Get(context.Background(), "service", &config); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Environment: %s\n", config.Environment)
	fmt.Printf("Database: %s:%d/%s\n",
		config.Database.Host,
		config.Database.Port,
		config.Database.Name,
	)
}
