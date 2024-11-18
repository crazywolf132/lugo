package lugo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"reflect"
	"sync"

	lua "github.com/yuin/gopher-lua"
)

// PluginAPI represents the interface that plugins can use to interact with the host application
type PluginAPI interface {
	// RegisterFunction registers a function that can be called from Lua
	RegisterFunction(name string, fn interface{}) error
	// RegisterHook registers a hook that will be called at specific points
	RegisterHook(hookType HookType, hook Hook) error
	// GetConfig returns the current configuration
	GetConfig() *Config
	// EmitEvent emits an event that other plugins can listen to
	EmitEvent(name string, data interface{}) error
}

// PluginManager handles plugin loading and lifecycle
type PluginManager struct {
	cfg           *Config
	plugins       map[string]*Plugin
	eventHandlers map[string][]EventHandler
	mu            sync.RWMutex
	sandbox       *Sandbox
	api           PluginAPI
}

// Plugin represents a loaded plugin
type Plugin struct {
	Name        string
	Version     string
	Description string
	Path        string
	Exports     map[string]interface{}
	State       *lua.LState
}

// EventHandler represents a function that handles plugin events
type EventHandler func(ctx context.Context, data interface{}) error

// PluginConfig holds configuration for plugin loading
type PluginConfig struct {
	// Directory containing plugins
	PluginDir string
	// Plugin-specific sandbox settings
	Sandbox *Sandbox
	// Custom API implementation
	API PluginAPI
	// Allowed plugin types (e.g., "lua", "so")
	AllowedTypes []string
	// Plugin metadata requirements
	RequiredMetadata []string
}

// NewPluginManager creates a new plugin manager
func (c *Config) NewPluginManager(pcfg PluginConfig) *PluginManager {
	if pcfg.Sandbox == nil {
		pcfg.Sandbox = &Sandbox{
			EnableFileIO:     false,
			EnableNetworking: false,
			MaxMemory:        1024 * 1024, // 1MB
		}
	}

	return &PluginManager{
		cfg:           c,
		plugins:       make(map[string]*Plugin),
		eventHandlers: make(map[string][]EventHandler),
		sandbox:       pcfg.Sandbox,
		api:           pcfg.API,
	}
}

// LoadPlugins loads all plugins from the configured directory
func (pm *PluginManager) LoadPlugins(ctx context.Context, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read plugin directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := filepath.Ext(entry.Name())
		switch ext {
		case ".lua":
			if err := pm.loadLuaPlugin(ctx, filepath.Join(dir, entry.Name())); err != nil {
				return fmt.Errorf("failed to load Lua plugin %s: %w", entry.Name(), err)
			}
		case ".so":
			if err := pm.loadGoPlugin(ctx, filepath.Join(dir, entry.Name())); err != nil {
				return fmt.Errorf("failed to load Go plugin %s: %w", entry.Name(), err)
			}
		}
	}

	return nil
}

func (pm *PluginManager) loadLuaPlugin(ctx context.Context, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Create sandboxed state for plugin
	L := lua.NewState()
	defer L.Close()

	// Register plugin API
	if err := pm.registerPluginAPI(L); err != nil {
		return err
	}

	// Execute plugin
	if err := L.DoString(string(content)); err != nil {
		return err
	}

	// Extract plugin metadata
	metadata := pm.extractPluginMetadata(L)
	if metadata == nil {
		return fmt.Errorf("plugin %s missing required metadata", path)
	}

	plugin := &Plugin{
		Name:        metadata["name"].(string),
		Version:     metadata["version"].(string),
		Description: metadata["description"].(string),
		Path:        path,
		Exports:     make(map[string]interface{}),
		State:       L,
	}

	// Extract exported functions
	exports := L.GetGlobal("exports")
	if exports != lua.LNil {
		if table, ok := exports.(*lua.LTable); ok {
			table.ForEach(func(k, v lua.LValue) {
				if fn, ok := v.(*lua.LFunction); ok {
					plugin.Exports[k.String()] = fn
				}
			})
		}
	}

	pm.mu.Lock()
	pm.plugins[plugin.Name] = plugin
	pm.mu.Unlock()

	return nil
}

func (pm *PluginManager) loadGoPlugin(ctx context.Context, path string) error {
	p, err := plugin.Open(path)
	if err != nil {
		return err
	}

	// Look for plugin metadata
	metadataSym, err := p.Lookup("Metadata")
	if err != nil {
		return fmt.Errorf("plugin %s does not export Metadata", path)
	}

	metadata, ok := metadataSym.(*map[string]interface{})
	if !ok {
		return fmt.Errorf("plugin %s has invalid Metadata type", path)
	}

	plugin := &Plugin{
		Name:        (*metadata)["name"].(string),
		Version:     (*metadata)["version"].(string),
		Description: (*metadata)["description"].(string),
		Path:        path,
		Exports:     make(map[string]interface{}),
	}

	// Look for Init function
	initSym, err := p.Lookup("Init")
	if err != nil {
		return fmt.Errorf("plugin %s does not export Init", path)
	}

	init, ok := initSym.(func(PluginAPI) error)
	if !ok {
		return fmt.Errorf("plugin %s has invalid Init function", path)
	}

	if err := init(pm.api); err != nil {
		return fmt.Errorf("plugin %s initialization failed: %w", path, err)
	}

	pm.mu.Lock()
	pm.plugins[plugin.Name] = plugin
	pm.mu.Unlock()

	return nil
}

// RegisterEventHandler registers a handler for plugin events
func (pm *PluginManager) RegisterEventHandler(event string, handler EventHandler) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.eventHandlers[event] = append(pm.eventHandlers[event], handler)
}

// EmitEvent emits an event to all registered handlers
func (pm *PluginManager) EmitEvent(ctx context.Context, event string, data interface{}) error {
	pm.mu.RLock()
	handlers := pm.eventHandlers[event]
	pm.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler(ctx, data); err != nil {
			return err
		}
	}
	return nil
}

// GetPlugin returns a loaded plugin by name
func (pm *PluginManager) GetPlugin(name string) (*Plugin, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	plugin, ok := pm.plugins[name]
	return plugin, ok
}

// CallPluginFunction calls an exported plugin function
func (pm *PluginManager) CallPluginFunction(ctx context.Context, pluginName, funcName string, args ...interface{}) (interface{}, error) {
	plugin, ok := pm.GetPlugin(pluginName)
	if !ok {
		return nil, fmt.Errorf("plugin %s not found", pluginName)
	}

	fn, ok := plugin.Exports[funcName]
	if !ok {
		return nil, fmt.Errorf("function %s not exported by plugin %s", funcName, pluginName)
	}

	switch fn := fn.(type) {
	case *lua.LFunction:
		return pm.callLuaFunction(plugin.State, fn, args...)
	case func(...interface{}) (interface{}, error):
		return fn(args...)
	default:
		return nil, fmt.Errorf("unsupported function type: %T", fn)
	}
}

func (pm *PluginManager) callLuaFunction(L *lua.LState, fn *lua.LFunction, args ...interface{}) (interface{}, error) {
	L.Push(fn)
	for _, arg := range args {
		lv, err := pm.cfg.goToLua(arg)
		if err != nil {
			return nil, err
		}
		L.Push(lv)
	}

	if err := L.PCall(len(args), 1, nil); err != nil {
		return nil, err
	}

	result := L.Get(-1)
	L.Pop(1)

	return pm.cfg.luaToGo(result, reflect.TypeOf(result))
}

func (pm *PluginManager) extractPluginMetadata(L *lua.LState) map[string]interface{} {
	metadata := L.GetGlobal("metadata")
	if metadata == lua.LNil {
		return nil
	}

	if table, ok := metadata.(*lua.LTable); ok {
		result := make(map[string]interface{})
		table.ForEach(func(k, v lua.LValue) {
			result[k.String()] = pm.luaValueToGo(v)
		})
		return result
	}

	return nil
}

func (pm *PluginManager) luaValueToGo(v lua.LValue) interface{} {
	switch v.Type() {
	case lua.LTString:
		return v.String()
	case lua.LTNumber:
		return float64(v.(lua.LNumber))
	case lua.LTBool:
		return bool(v.(lua.LBool))
	case lua.LTTable:
		table := v.(*lua.LTable)
		result := make(map[string]interface{})
		table.ForEach(func(k, v lua.LValue) {
			result[k.String()] = pm.luaValueToGo(v)
		})
		return result
	default:
		return nil
	}
}

func (pm *PluginManager) registerPluginAPI(L *lua.LState) error {
	api := L.NewTable()

	// Register API functions
	L.SetField(api, "register_function", L.NewFunction(func(L *lua.LState) int {
		name := L.CheckString(1)
		fn := L.CheckFunction(2)
		plugin := &Plugin{
			Name: name,
			Exports: map[string]interface{}{
				name: fn,
			},
			State: L,
		}
		pm.plugins[name] = plugin
		return 0
	}))

	L.SetField(api, "register_hook", L.NewFunction(func(L *lua.LState) int {
		hookType := HookType(L.CheckInt(1))
		fn := L.CheckFunction(2)

		pm.cfg.RegisterHook(hookType, func(ctx context.Context, event HookEvent) error {
			L.Push(fn)
			L.Push(lua.LString(event.Name))
			err := L.PCall(1, 0, nil)
			if err != nil {
				return fmt.Errorf("plugin hook error: %w", err)
			}
			return nil
		})
		return 0
	}))

	L.SetField(api, "emit_event", L.NewFunction(func(L *lua.LState) int {
		name := L.CheckString(1)
		data := pm.luaValueToGo(L.Get(2))

		err := pm.EmitEvent(context.Background(), name, data)
		if err != nil {
			L.Push(lua.LString(err.Error()))
			return 1
		}
		return 0
	}))

	L.SetField(api, "get_config", L.NewFunction(func(L *lua.LState) int {
		name := L.CheckString(1)
		value := pm.cfg.L.GetGlobal(name)
		L.Push(value)
		return 1
	}))

	// Set the API table as a global
	L.SetGlobal("api", api)
	return nil
}
