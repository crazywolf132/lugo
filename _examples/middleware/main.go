package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/yourorg/lugo"
)

// SecretConfig demonstrates secret handling and transformation
type SecretConfig struct {
	Database struct {
		Password string `lua:"password"`
		APIKey   string `lua:"api_key"`
	} `lua:"database"`
	JWT struct {
		Secret string `lua:"secret"`
	} `lua:"jwt"`
}

// Custom middleware to mask secrets in logs
func maskSecrets(next lugo.Handler) lugo.Handler {
	return func(ctx context.Context, cfg *lugo.Config) error {
		// Call the next handler first
		if err := next(ctx, cfg); err != nil {
			return err
		}

		// Get all variables
		vars := cfg.GetAll()
		for k, v := range vars {
			if isSecret(k) {
				// Mask secret values
				cfg.Set(k, maskValue(v))
			}
		}
		return nil
	}
}

// Decrypt middleware using external KMS
func decryptSecrets(next lugo.Handler) lugo.Handler {
	return func(ctx context.Context, cfg *lugo.Config) error {
		if err := next(ctx, cfg); err != nil {
			return err
		}

		vars := cfg.GetAll()
		for k, v := range vars {
			if isEncrypted(v) {
				decrypted, err := decryptValue(v)
				if err != nil {
					return fmt.Errorf("failed to decrypt %s: %w", k, err)
				}
				cfg.Set(k, decrypted)
			}
		}
		return nil
	}
}

func main() {
	// Initialize with middlewares
	cfg := lugo.New(
		lugo.WithMiddleware(decryptSecrets),
		lugo.WithMiddleware(maskSecrets),
	)
	defer cfg.Close()

	if err := cfg.LoadFile(context.Background(), "config.lua"); err != nil {
		log.Fatal(err)
	}

	var config SecretConfig
	if err := cfg.Get(context.Background(), "config", &config); err != nil {
		log.Fatal(err)
	}

	// Secrets are now decrypted but masked in logs
	log.Printf("Configuration loaded: %+v", config)
}

// Helper functions
func isSecret(key string) bool {
	secrets := []string{"password", "secret", "key", "token"}
	key = strings.ToLower(key)
	for _, s := range secrets {
		if strings.Contains(key, s) {
			return true
		}
	}
	return false
}

func maskValue(v interface{}) string {
	if s, ok := v.(string); ok {
		if len(s) <= 4 {
			return "****"
		}
		return s[:4] + strings.Repeat("*", len(s)-4)
	}
	return "****"
}

func isEncrypted(v interface{}) bool {
	s, ok := v.(string)
	return ok && strings.HasPrefix(s, "enc:")
}

func decryptValue(v interface{}) (string, error) {
	// Simulate decryption
	s := v.(string)
	return strings.TrimPrefix(s, "enc:"), nil
}
