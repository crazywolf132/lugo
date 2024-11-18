package lugo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvironmentManager(t *testing.T) {
	// Create temporary config directory
	configDir := t.TempDir()

	// Create test configuration files
	baseConfig := `
		config = {
			app = {
				name = "test-app",
				version = "1.0.0"
			},
			defaults = {
				timeout = 30,
				retries = 3
			}
		}
	`
	err := os.WriteFile(filepath.Join(configDir, "base.lua"), []byte(baseConfig), 0644)
	require.NoError(t, err)

	devConfig := `
		config.environment = "development"
		config.debug = true
		config.app.url = "http://localhost:8080"
	`
	err = os.WriteFile(filepath.Join(configDir, "dev.lua"), []byte(devConfig), 0644)
	require.NoError(t, err)

	prodConfig := `
		config.environment = "production"
		config.debug = false
		config.app.url = "https://example.com"
	`
	err = os.WriteFile(filepath.Join(configDir, "prod.lua"), []byte(prodConfig), 0644)
	require.NoError(t, err)

	// Create config instance
	cfg := New()
	defer cfg.Close()

	// Create environment manager
	envManager := cfg.NewEnvManager(configDir)

	// Register environments
	err = envManager.RegisterEnvironment(&Environment{
		Name:       "dev",
		BaseConfig: "base.lua",
		EnvConfig:  "dev.lua",
		EnvPrefix:  "APP_",
	})
	require.NoError(t, err)

	err = envManager.RegisterEnvironment(&Environment{
		Name:       "prod",
		BaseConfig: "base.lua",
		EnvConfig:  "prod.lua",
		EnvPrefix:  "APP_",
	})
	require.NoError(t, err)

	// Test development environment
	t.Run("development environment", func(t *testing.T) {
		os.Setenv("APP_API_KEY", "dev-key")
		defer os.Unsetenv("APP_API_KEY")

		err := envManager.ActivateEnvironment("dev")
		require.NoError(t, err)

		var config struct {
			Environment string `lua:"environment"`
			Debug       bool   `lua:"debug"`
			App         struct {
				Name    string `lua:"name"`
				Version string `lua:"version"`
				URL     string `lua:"url"`
			} `lua:"app"`
			Defaults struct {
				Timeout int `lua:"timeout"`
				Retries int `lua:"retries"`
			} `lua:"defaults"`
		}

		err = cfg.Get(nil, "config", &config)
		require.NoError(t, err)

		assert.Equal(t, "development", config.Environment)
		assert.True(t, config.Debug)
		assert.Equal(t, "test-app", config.App.Name)
		assert.Equal(t, "1.0.0", config.App.Version)
		assert.Equal(t, "http://localhost:8080", config.App.URL)
		assert.Equal(t, 30, config.Defaults.Timeout)
		assert.Equal(t, 3, config.Defaults.Retries)

		// Verify environment variable was loaded
		var apiKey string
		err = cfg.GetGlobal("api.key", &apiKey)
		require.NoError(t, err)
		assert.Equal(t, "dev-key", apiKey)
	})

	// Test production environment
	t.Run("production environment", func(t *testing.T) {
		os.Setenv("APP_API_KEY", "prod-key")
		defer os.Unsetenv("APP_API_KEY")

		err := envManager.ActivateEnvironment("prod")
		require.NoError(t, err)

		var config struct {
			Environment string `lua:"environment"`
			Debug       bool   `lua:"debug"`
			App         struct {
				URL string `lua:"url"`
			} `lua:"app"`
		}

		err = cfg.Get(nil, "config", &config)
		require.NoError(t, err)

		assert.Equal(t, "production", config.Environment)
		assert.False(t, config.Debug)
		assert.Equal(t, "https://example.com", config.App.URL)

		// Verify environment variable was loaded
		var apiKey string
		err = cfg.GetGlobal("api.key", &apiKey)
		require.NoError(t, err)
		assert.Equal(t, "prod-key", apiKey)
	})
}
