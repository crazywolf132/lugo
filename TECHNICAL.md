# Lugo Technical Implementation Guide
## Library Pattern Analysis and Usage Documentation

## Core Concepts Implementation

### 1. Configuration State Management

The Lugo library manages configuration state through the `Config` struct, which maintains:
- Lua state (`*lua.LState`)
- Thread-safe access (`sync.RWMutex`)
- Hooks registry (`map[HookType][]Hook`)
- Middleware chain
- Sandbox restrictions

Example implementation pattern:

```go
// Initialize with sandbox and logger
config := lugo.New(
    lugo.WithSandbox(&lugo.Sandbox{
        MaxMemory: 50 * 1024 * 1024,
        MaxExecutionTime: 10 * time.Second,
    }),
    lugo.WithLogger(logger),
)

// Register shutdown handler
defer config.Close()

// Thread-safe operations pattern
config.mu.RLock()
// Read operations
config.mu.RUnlock()

config.mu.Lock()
// Write operations
config.mu.Unlock()
```

### 2. Type Registration System

Type registration follows a reflection-based pattern for mapping Go structs to Lua tables:

```go
// Base configuration type pattern
type DatabaseConfig struct {
    Host     string   `lua:"host"`
    Port     int      `lua:"port"`
    Replicas []string `lua:"replicas"`
    Options  map[string]interface{} `lua:"options"`
}

// Registration with validation
ctx := context.Background()
err := config.RegisterType(ctx, "database", DatabaseConfig{
    Host: "localhost",  // Default value
    Port: 5432,        // Default value
    Replicas: []string{},
    Options: map[string]interface{}{
        "max_connections": 100,
    },
})
```

Corresponding Lua configuration:
```lua
database = {
    host = "prod.db.example.com",
    port = 5432,
    replicas = {
        "replica1.db.example.com",
        "replica2.db.example.com"
    },
    options = {
        max_connections = 200,
        idle_timeout = 300
    }
}
```

### 3. Hook System Implementation

Hooks provide lifecycle event handling. Implementation pattern:

```go
// Hook registration pattern
config.RegisterHook(lugo.BeforeLoad, func(ctx context.Context, event lugo.HookEvent) error {
    // Pre-validation logic
    return validateConfigPath(event.Name)
})

config.RegisterHook(lugo.AfterLoad, func(ctx context.Context, event lugo.HookEvent) error {
    // Post-load processing
    metrics.ObserveConfigLoadTime(event.Elapsed)
    return nil
})

// Hook event handling pattern
type processHook struct {
    logger *zap.Logger
    metrics MetricsCollector
}

func (h *processHook) HandleLoad(ctx context.Context, event lugo.HookEvent) error {
    start := time.Now()
    defer func() {
        h.metrics.RecordHookDuration("load", time.Since(start))
    }()
    
    h.logger.Info("processing configuration",
        zap.String("file", event.Name),
        zap.Duration("elapsed", event.Elapsed))
    
    return nil
}
```

### 4. Middleware Chain Pattern

Middleware implementation follows a chain-of-responsibility pattern:

```go
// Middleware definition pattern
func loggingMiddleware(logger *zap.Logger) lugo.Middleware {
    return func(next lugo.LuaFunction) lugo.LuaFunction {
        return func(ctx context.Context, L *lua.LState) ([]lua.LValue, error) {
            funcName := L.GetStack(0).Func.String()
            logger.Debug("executing lua function", zap.String("func", funcName))
            
            start := time.Now()
            results, err := next(ctx, L)
            
            logger.Debug("lua function completed",
                zap.String("func", funcName),
                zap.Duration("duration", time.Since(start)),
                zap.Error(err))
            
            return results, err
        }
    }
}

// Validation middleware pattern
func validationMiddleware(validators map[string]func(lua.LValue) error) lugo.Middleware {
    return func(next lugo.LuaFunction) lugo.LuaFunction {
        return func(ctx context.Context, L *lua.LState) ([]lua.LValue, error) {
            // Pre-execution validation
            for i := 1; i <= L.GetTop(); i++ {
                if err := validators[L.Get(i).Type().String()](L.Get(i)); err != nil {
                    return nil, err
                }
            }
            
            return next(ctx, L)
        }
    }
}
```

### 5. Error Handling Patterns

Comprehensive error handling implementation:

```go
// Error type pattern
type ConfigError struct {
    Code    ErrorCode
    Message string
    Cause   error
}

// Error handling pattern
func (c *Config) LoadFile(ctx context.Context, filename string) error {
    if err := c.validatePath(filename); err != nil {
        return &Error{
            Code:    ErrValidation,
            Message: "invalid configuration path",
            Cause:   err,
        }
    }
    
    // Wrap lower-level errors
    if err := c.L.DoFile(filename); err != nil {
        return &Error{
            Code:    ErrExecution,
            Message: "failed to execute configuration file",
            Cause:   err,
        }
    }
    
    return nil
}

// Error handling usage pattern
if err := config.LoadFile(ctx, "config.lua"); err != nil {
    var configErr *lugo.Error
    if errors.As(err, &configErr) {
        switch configErr.Code {
        case lugo.ErrValidation:
            // Handle validation error
        case lugo.ErrExecution:
            // Handle execution error
        case lugo.ErrSandbox:
            // Handle sandbox violation
        }
    }
    return err
}
```

### 6. Type Conversion System

Implementation patterns for Go-Lua type conversion:

```go
// Go to Lua conversion pattern
func (c *Config) goToLua(v interface{}) (lua.LValue, error) {
    val := reflect.ValueOf(v)
    switch val.Kind() {
    case reflect.Struct:
        return c.structToTable(v)
    case reflect.Map:
        table := c.L.NewTable()
        iter := val.MapRange()
        for iter.Next() {
            k, v := iter.Key(), iter.Value()
            if lv, err := c.goToLua(v.Interface()); err == nil {
                table.RawSetString(k.String(), lv)
            }
        }
        return table, nil
    case reflect.Slice:
        table := c.L.NewTable()
        for i := 0; i < val.Len(); i++ {
            if lv, err := c.goToLua(val.Index(i).Interface()); err == nil {
                table.Append(lv)
            }
        }
        return table, nil
    }
    return nil, fmt.Errorf("unsupported type: %T", v)
}

// Lua to Go conversion pattern
func (c *Config) luaToGo(lv lua.LValue, t reflect.Type) (interface{}, error) {
    switch lv.Type() {
    case lua.LTTable:
        switch t.Kind() {
        case reflect.Struct:
            return c.tableToStruct(lv.(*lua.LTable), t)
        case reflect.Map:
            return c.tableToMap(lv.(*lua.LTable), t)
        case reflect.Slice:
            return c.tableToSlice(lv.(*lua.LTable), t)
        }
    }
    return nil, fmt.Errorf("unsupported conversion: %s to %s", lv.Type(), t)
}
```

### 7. Sandbox Implementation

Sandbox security pattern implementation:

```go
// Sandbox configuration pattern
sandbox := &lugo.Sandbox{
    EnableFileIO:     false,
    EnableNetworking: false,
    EnableSyscalls:   false,
    MaxMemory:        100 * 1024 * 1024,
    MaxExecutionTime: 30 * time.Second,
    AllowedPaths:     []string{"/config"},
    BlockedPaths:     []string{"/etc", "/var"},
}

// Restricted environment setup pattern
func (c *Config) createRestrictedEnv() *lua.LTable {
    env := c.L.NewTable()
    
    // Safe basic functions
    safeFuncs := map[string]lua.LGFunction{
        "assert":   c.L.GetGlobal("assert").(*lua.LFunction).GFunction,
        "tonumber": c.L.GetGlobal("tonumber").(*lua.LFunction).GFunction,
        "tostring": c.L.GetGlobal("tostring").(*lua.LFunction).GFunction,
    }
    
    // Safe libraries
    safeLibs := map[string]*lua.LTable{
        "string": c.L.GetGlobal("string").(*lua.LTable),
        "table":  c.L.GetGlobal("table").(*lua.LTable),
        "math":   c.L.GetGlobal("math").(*lua.LTable),
    }
    
    // Apply restrictions
    for name, fn := range safeFuncs {
        env.RawSetString(name, c.L.NewFunction(fn))
    }
    
    for name, lib := range safeLibs {
        env.RawSetString(name, lib)
    }
    
    return env
}
```

## Common Usage Patterns

### 1. Configuration Loading Pattern

```go
func LoadConfiguration(ctx context.Context, configPath string) (*AppConfig, error) {
    cfg := lugo.New(
        lugo.WithSandbox(DefaultSandbox()),
        lugo.WithMiddleware(ValidationMiddleware()),
        lugo.WithMiddleware(LoggingMiddleware()),
    )
    defer cfg.Close()
    
    // Register configuration types
    if err := cfg.RegisterType(ctx, "app", AppConfig{}); err != nil {
        return nil, fmt.Errorf("failed to register config type: %w", err)
    }
    
    // Load and validate
    if err := cfg.LoadFile(ctx, configPath); err != nil {
        return nil, fmt.Errorf("failed to load config: %w", err)
    }
    
    var appConfig AppConfig
    if err := cfg.Get(ctx, "app", &appConfig); err != nil {
        return nil, fmt.Errorf("failed to parse config: %w", err)
    }
    
    return &appConfig, nil
}
```

### 2. Dynamic Configuration Pattern

```go
func NewDynamicConfig(ctx context.Context) (*DynamicConfig, error) {
    cfg := lugo.New(
        lugo.WithSandbox(RestrictedSandbox()),
    )
    
    // Register update function
    err := cfg.RegisterFunction(ctx, "updateConfig", func(ctx context.Context, key string, value interface{}) error {
        return cfg.UpdateConfigValue(key, value)
    })
    
    // Register watch function
    err = cfg.RegisterFunction(ctx, "watchConfig", func(ctx context.Context, key string) (interface{}, error) {
        return cfg.WatchConfigValue(key)
    })
    
    return &DynamicConfig{config: cfg}, nil
}
```

## Event Handling Patterns

### 1. Configuration Change Events

```go
type ConfigChangeEvent struct {
    Key      string
    OldValue interface{}
    NewValue interface{}
    Time     time.Time
}

func (c *Config) handleConfigChange(ctx context.Context, event ConfigChangeEvent) error {
    // Notify hooks
    for _, hook := range c.hooks[AfterExec] {
        if err := hook(ctx, HookEvent{
            Type:   AfterExec,
            Name:   event.Key,
            Args:   []interface{}{event.OldValue, event.NewValue},
            Result: event.NewValue,
        }); err != nil {
            return err
        }
    }
    
    return nil
}
```

### 2. Error Recovery Pattern

```go
func (c *Config) safeExecution(ctx context.Context, fn func() error) (err error) {
    defer func() {
        if r := recover(); r != nil {
            err = &Error{
                Code:    ErrExecution,
                Message: fmt.Sprintf("panic recovered: %v", r),
                Cause:   fmt.Errorf("%v", r),
            }
        }
    }()
    
    return fn()
}
```

## Testing Patterns

```go
func TestConfigurationLoading(t *testing.T) {
    tests := []struct {
        name    string
        config  string
        want    AppConfig
        wantErr bool
    }{
        {
            name: "valid configuration",
            config: `
                app = {
                    name = "test",
                    port = 8080
                }
            `,
            want: AppConfig{
                Name: "test",
                Port: 8080,
            },
        },
        // Add more test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cfg := lugo.New()
            defer cfg.Close()
            
            if err := cfg.RegisterType(context.Background(), "app", AppConfig{}); err != nil {
                t.Fatal(err)
            }
            
            // Create temporary file with config
            f, err := os.CreateTemp("", "config-*.lua")
            if err != nil {
                t.Fatal(err)
            }
            defer os.Remove(f.Name())
            
            if _, err := f.WriteString(tt.config); err != nil {
                t.Fatal(err)
            }
            
            if err := cfg.LoadFile(context.Background(), f.Name()); err != nil {
                if !tt.wantErr {
                    t.Errorf("LoadFile() error = %v", err)
                }
                return
            }
            
            var got AppConfig
            if err := cfg.Get(context.Background(), "app", &got); err != nil {
                t.Fatal(err)
            }
            
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("Get() got = %v, want %v", got, tt.want)
            }
        })
    }
}
```