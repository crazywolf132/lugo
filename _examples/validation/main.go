package main

import (
	"context"
	"log"
	"time"

	"github.com/yourorg/lugo"
)

type EmailConfig struct {
	SMTP struct {
		Host     string `lua:"host" validate:"required,hostname"`
		Port     int    `lua:"port" validate:"required,min=1,max=65535"`
		Username string `lua:"username" validate:"required,email"`
		Password string `lua:"password" validate:"required,min=8"`
		TLS      bool   `lua:"tls"`
	} `lua:"smtp"`
	Defaults struct {
		From    string        `lua:"from" validate:"required,email"`
		ReplyTo string        `lua:"reply_to" validate:"omitempty,email"`
		Timeout time.Duration `lua:"timeout" validate:"required,min=1s,max=1m"`
		Retries int           `lua:"retries" validate:"min=0,max=5"`
		MaxSize int           `lua:"max_size" validate:"required,min=1048576"` // 1MB
	} `lua:"defaults"`
	Templates []struct {
		Name    string   `lua:"name" validate:"required"`
		Subject string   `lua:"subject" validate:"required"`
		CC      []string `lua:"cc" validate:"omitempty,dive,email"`
		BCC     []string `lua:"bcc" validate:"omitempty,dive,email"`
	} `lua:"templates" validate:"required,dive"`
}

func main() {
	// Initialize Lugo with validation
	cfg := lugo.New(
		lugo.WithValidation(true),
	)
	defer cfg.Close()

	// Load and validate configuration
	if err := cfg.LoadFile(context.Background(), "email.lua"); err != nil {
		log.Fatal(err)
	}

	var config EmailConfig
	if err := cfg.Get(context.Background(), "email", &config); err != nil {
		log.Fatal(err)
	}

	log.Println("Configuration validated successfully!")
}
