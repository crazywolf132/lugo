package lugo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerator(t *testing.T) {
	tests := []struct {
		name     string
		generate func(*Generator)
		want     string
	}{
		{
			name: "simple table",
			generate: func(g *Generator) {
				g.Table("config").
					Field("name", "test").
					Field("value", 42).
					Field("enabled", true).
					EndTable()
			},
			want: `config = {
    name = "test",
    value = 42,
    enabled = true,
}
`,
		},
		{
			name: "nested tables",
			generate: func(g *Generator) {
				g.Table("server").
					Field("host", "localhost").
					Field("port", 8080).
					Table("tls").
					Field("enabled", true).
					Field("cert", "/path/to/cert").
					EndTable().
					EndTable()
			},
			want: `server = {
    host = "localhost",
    port = 8080,
    tls = {
        enabled = true,
        cert = "/path/to/cert",
    },
}
`,
		},
		{
			name: "arrays and comments",
			generate: func(g *Generator) {
				g.Comment("Server configuration").
					Table("config").
					Field("endpoints", []string{"/api", "/health"}).
					Field("allowed_ips", []string{"127.0.0.1", "::1"}).
					EndTable()
			},
			want: `-- Server configuration
config = {
    endpoints = { "/api", "/health" },
    allowed_ips = { "127.0.0.1", "::1" },
}
`,
		},
		{
			name: "functions",
			generate: func(g *Generator) {
				g.Function("process", "name", "value").
					Raw("    local result = name .. ': ' .. value").
					Raw("    return result").
					EndFunction()
			},
			want: `function process(name, value)
    local result = name .. ': ' .. value
    return result
end
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGenerator()
			tt.generate(g)
			assert.Equal(t, tt.want, g.String())
		})
	}
}
