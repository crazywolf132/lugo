package lugo

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateProcessing(t *testing.T) {
	// Create a temporary config file
	tmpFile := filepath.Join(t.TempDir(), "config.lua.tmpl")
	tmpl := `
    -- Set global configuration table
    config = {
        name = "{{ .name }}",
        api_key = "{{ env "API_KEY" }}",
        greeting = "{{ hello .name }}",
        timeout = {{ default 30 .timeout }},
        server = {
            host = "{{ .server.host }}",
            port = {{ .server.port }},
            {{ if .server.debug }}
            debug = true,
            {{ end }}
            endpoints = {
                {{ range .server.endpoints }}
                "{{ . }}",
                {{ end }}
            }
        }
    }
`
	err := os.WriteFile(tmpFile, []byte(tmpl), 0644)
	require.NoError(t, err)

	// Set up environment
	os.Setenv("API_KEY", "test-key")
	defer os.Unsetenv("API_KEY")

	// Create config instance
	cfg := New()
	defer cfg.Close()

	// Process template
	err = cfg.ProcessTemplate(tmpFile, TemplateConfig{
		Variables: map[string]interface{}{
			"name": "test",
			"server": map[string]interface{}{
				"host":      "localhost",
				"port":      8080,
				"debug":     true,
				"endpoints": []string{"/api", "/health", "/metrics"},
			},
		},
		Functions: template.FuncMap{
			"hello": func(name string) string {
				return "Hello, " + name + "!"
			},
			"env": os.Getenv,
			"default": func(def interface{}, val interface{}) interface{} {
				if val == nil {
					return def
				}
				return val
			},
		},
	})
	require.NoError(t, err)

	// Get the returned configuration
	var result struct {
		Name     string `lua:"name"`
		APIKey   string `lua:"api_key"`
		Greeting string `lua:"greeting"`
		Timeout  int    `lua:"timeout"`
		Server   struct {
			Host      string   `lua:"host"`
			Port      int      `lua:"port"`
			Debug     bool     `lua:"debug"`
			Endpoints []string `lua:"endpoints"`
		} `lua:"server"`
	}

	err = cfg.Get(context.Background(), "config", &result)
	require.NoError(t, err)

	assert.Equal(t, "test", result.Name)
	assert.Equal(t, "test-key", result.APIKey)
	assert.Equal(t, "Hello, test!", result.Greeting)
	assert.Equal(t, 30, result.Timeout)
	assert.Equal(t, "localhost", result.Server.Host)
	assert.Equal(t, 8080, result.Server.Port)
	assert.True(t, result.Server.Debug)
	assert.Equal(t, []string{"/api", "/health", "/metrics"}, result.Server.Endpoints)
}
