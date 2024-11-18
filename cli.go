package lugo

import (
	lua "github.com/yuin/gopher-lua"
)

// CLIConfig represents CLI-specific configuration
type CLIConfig struct {
	cfg      *Config
	commands map[string]lua.LValue
	plugins  map[string]*lua.LTable
	bindings map[string]string
}

// NewCLIConfig creates a new CLI configuration
func (c *Config) NewCLIConfig(opts CLIConfigOptions) *CLIConfig {
	cli := &CLIConfig{
		cfg:      c,
		commands: make(map[string]lua.LValue),
		plugins:  make(map[string]*lua.LTable),
		bindings: make(map[string]string),
	}

	// Register the register_command function
	c.L.SetGlobal("register_command", c.L.NewFunction(func(L *lua.LState) int {
		name := L.CheckString(1)
		fn := L.CheckFunction(2)
		cli.commands[name] = fn
		return 0
	}))

	return cli
}

// CLIConfigOptions holds options for CLI configuration
type CLIConfigOptions struct {
	AppName       string
	PluginDir     string
	DefaultConfig string
}

// GetPlugin returns a plugin by name
func (c *CLIConfig) GetPlugin(name string) (lua.LValue, bool) {
	cmd, ok := c.commands[name]
	return cmd, ok
}

// LoadPlugins loads all plugins from the plugin directory
func (c *CLIConfig) LoadPlugins() error {
	// Implementation would go here
	return nil
}
