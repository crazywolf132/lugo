package lugo

import (
	"bytes"
	"os"
	"text/template"
)

// ProcessTemplate processes a Lua configuration file as a template
func (c *Config) ProcessTemplate(filename string, tcfg TemplateConfig) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	// Create template
	t := template.New(filename)

	// Add default functions if none provided
	if tcfg.Functions == nil {
		tcfg.Functions = make(template.FuncMap)
	}

	// Add built-in functions if not overridden
	if _, ok := tcfg.Functions["env"]; !ok {
		tcfg.Functions["env"] = os.Getenv
	}
	if _, ok := tcfg.Functions["default"]; !ok {
		tcfg.Functions["default"] = func(def interface{}, val interface{}) interface{} {
			if val == nil {
				return def
			}
			return val
		}
	}

	if tcfg.LeftDelim != "" && tcfg.RightDelim != "" {
		t = t.Delims(tcfg.LeftDelim, tcfg.RightDelim)
	}

	// Add functions
	t = t.Funcs(tcfg.Functions)

	// Parse template
	t, err = t.Parse(string(content))
	if err != nil {
		return err
	}

	// Execute template
	var buf bytes.Buffer
	if err := t.Execute(&buf, tcfg.Variables); err != nil {
		return err
	}

	// Set the config global before executing the template output
	c.L.SetGlobal("config", c.L.CreateTable(0, 0))

	// Execute the processed template as Lua code
	return c.DoString(buf.String())
}

// TemplateConfig holds configuration for template processing
type TemplateConfig struct {
	Variables  map[string]interface{}
	Functions  template.FuncMap
	LeftDelim  string
	RightDelim string
}
