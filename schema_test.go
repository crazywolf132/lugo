package lugo

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemaValidation(t *testing.T) {
	type ServerConfig struct {
		Host string `lua:"host"`
		Port int    `lua:"port"`
		TLS  struct {
			Enabled bool   `lua:"enabled"`
			Cert    string `lua:"cert"`
			Key     string `lua:"key"`
		} `lua:"tls"`
		Timeout time.Duration `lua:"timeout"`
	}

	// Create schema validator
	validator := NewSchemaValidator()
	validator.Required = []string{"Host", "Port"}

	err := validator.AddPattern("Host", `^[a-zA-Z0-9\.-]+$`)
	require.NoError(t, err)

	validator.AddRange("Port", 1, 65535)

	tlsValidator := NewSchemaValidator()
	tlsValidator.Required = []string{"Cert", "Key"}
	validator.AddNestedValidator("TLS", tlsValidator)

	validator.AddCustomValidator("Timeout", func(v interface{}) error {
		duration := v.(time.Duration)
		if duration < time.Second || duration > time.Hour {
			return fmt.Errorf("timeout must be between 1 second and 1 hour")
		}
		return nil
	})

	tests := []struct {
		name    string
		config  ServerConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: ServerConfig{
				Host: "localhost",
				Port: 8080,
				TLS: struct {
					Enabled bool   `lua:"enabled"`
					Cert    string `lua:"cert"`
					Key     string `lua:"key"`
				}{
					Enabled: true,
					Cert:    "/path/to/cert",
					Key:     "/path/to/key",
				},
				Timeout: 5 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "invalid host",
			config: ServerConfig{
				Host: "invalid/host",
				Port: 8080,
			},
			wantErr: true,
		},
		{
			name: "invalid port",
			config: ServerConfig{
				Host: "localhost",
				Port: 70000,
			},
			wantErr: true,
		},
		{
			name: "missing required TLS fields",
			config: ServerConfig{
				Host: "localhost",
				Port: 8080,
				TLS: struct {
					Enabled bool   `lua:"enabled"`
					Cert    string `lua:"cert"`
					Key     string `lua:"key"`
				}{
					Enabled: true,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
