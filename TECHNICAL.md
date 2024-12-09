# Lugo Documentation

## Getting Started

Lugo is a configuration management library for Go that uses Lua as its configuration language. It provides type-safe configuration with the flexibility of a full programming language.

## Why Lugo?

Most configuration formats like JSON, YAML, or TOML are static. They can't compute values or include logic. Lugo uses Lua, giving you the power to:

- Compute values dynamically
- Share configuration across environments
- Include conditional logic
- Keep your configuration DRY

## Quick Start

1. Install Lugo:
```bash
go get github.com/yourorg/lugo
```

2. Create a configuration file (config.lua):
```lua
config = {
    app_name = "myapp",
    port = 8080,
    
    -- Compute values
    worker_count = math.min(8, os.getenv("CPU_COUNT") or 4),
    
    -- Environment-specific settings
    database = {
        host = ENV == "production" and "db.prod" or "localhost",
        port = 5432
    }
}
```

3. Use it in your Go code:
```go
package main

import (
    "context"
    "log"
    
    "github.com/yourorg/lugo"
)

type Config struct {
    AppName     string `lua:"app_name"`
    Port        int    `lua:"port"`
    WorkerCount int    `lua:"worker_count"`
    Database struct {
        Host string `lua:"host"`
        Port int    `lua:"port"`
    } `lua:"database"`
}

func main() {
    // Initialize Lugo
    cfg := lugo.New()
    
    // Load configuration
    if err := cfg.LoadFile(context.Background(), "config.lua"); err != nil {
        log.Fatal(err)
    }
    
    // Parse into struct
    var config Config
    if err := cfg.Get(context.Background(), "config", &config); err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Starting %s on port %d", config.AppName, config.Port)
}

## Core Concepts

### Configuration Loading

Lugo loads configuration in a pipeline:

1. Read the file
2. Process any templates
3. Execute the Lua code in a sandbox
4. Convert values to Go types
5. Validate the configuration

Example with all features:
```go
cfg := lugo.New(
    // Enable template processing
    lugo.WithTemplating(true),
    
    // Configure sandbox
    lugo.WithSandbox(&lugo.Sandbox{
        EnableFileIO: false,
        MaxMemory: 10 * 1024 * 1024,
    }),
    
    // Enable validation
    lugo.WithValidation(true),
)

// Load with environment variables
env := map[string]string{
    "ENV": "production",
    "DB_PASSWORD": "secret",
}

if err := cfg.LoadFile(context.Background(), "config.lua", lugo.LoadOptions{
    Environment: env,
}); err != nil {
    log.Fatal(err)
}
```

### Type Safety

Lugo maps Lua values to Go types automatically:

| Lua Type | Go Type |
|----------|---------|
| string | string |
| number | int, int64, float64 |
| boolean | bool |
| table (array) | slice |
| table (hash) | struct, map |

Example:
```lua
config = {
    -- Maps to string
    name = "myapp",
    
    -- Maps to int
    port = 8080,
    
    -- Maps to []string
    tags = {"web", "api"},
    
    -- Maps to map[string]string
    labels = {
        env = "prod",
        region = "us-west"
    },
    
    -- Maps to struct
    database = {
        host = "localhost",
        port = 5432
    }
}
```

```go
type Config struct {
    Name     string            `lua:"name"`
    Port     int              `lua:"port"`
    Tags     []string         `lua:"tags"`
    Labels   map[string]string `lua:"labels"`
    Database struct {
        Host string `lua:"host"`
        Port int    `lua:"port"`
    } `lua:"database"`
}
```

### Type Conversion Utilities

Lugo provides robust type conversion utilities through the `TypeConverter` interface, which safely converts values between different types:

```go
type Config struct {
    converter *TypeConverter
}
```

#### Built-in Conversions

The following conversions are supported:

| Source Type | Target Types |
|-------------|--------------|
| string | string, int64, float64, bool |
| int/int8/.../int64 | string, int64, float64, bool |
| uint/uint8/.../uint64 | string, int64, float64, bool |
| float32/float64 | string, int64, float64, bool |
| bool | string, int64, float64, bool |
| time.Time | string (RFC3339 format) |

#### Conversion Methods

- `ToString(v interface{}) (string, error)`: Converts any supported value to string
- `ToInt(v interface{}) (int64, error)`: Converts any supported value to int64
- `ToFloat(v interface{}) (float64, error)`: Converts any supported value to float64
- `ToBool(v interface{}) (bool, error)`: Converts any supported value to bool

Example usage:
```go
// String conversion
str, err := converter.ToString(123) // "123"
str, err := converter.ToString(true) // "true"
str, err := converter.ToString(time.Now()) // "2024-01-20T15:04:05Z07:00"

// Numeric conversion
num, err := converter.ToInt("123") // 123
num, err := converter.ToInt(true) // 1
num, err := converter.ToInt("invalid") // error

// Boolean conversion
bool, err := converter.ToBool(1) // true
bool, err := converter.ToBool("yes") // true
bool, err := converter.ToBool("no") // false
```

### Error Handling and Propagation

Lugo uses a structured error handling system with error codes and optional error chaining.

#### Error Types

```go
type Error struct {
    Code    ErrorCode
    Message string
    Cause   error
}
```

#### Error Codes

| Code | Description |
|------|-------------|
| ErrInvalidType | Type conversion or assertion failed |
| ErrNotFound | Configuration or value not found |
| ErrValidation | Configuration validation failed |
| ErrSandbox | Sandbox security violation |
| ErrExecution | Lua execution error |
| ErrTimeout | Operation timed out |
| ErrCanceled | Operation was canceled |
| ErrIO | File I/O error |
| ErrParse | Configuration parsing error |
| ErrConversion | Data conversion error |

#### Error Utilities

- `NewError(code ErrorCode, message string) *Error`: Creates a new error
- `WrapError(code ErrorCode, message string, cause error) *Error`: Wraps an existing error
- `IsErrorCode(err error, code ErrorCode) bool`: Checks if an error matches a specific code

Example error handling:
```go
// Creating errors
err := NewError(ErrNotFound, "configuration not found")
err := WrapError(ErrValidation, "invalid port", portErr)

// Checking error types
if IsErrorCode(err, ErrNotFound) {
    // Handle not found error
}

// Error chaining
if err := cfg.Get("database", &dbConfig); err != nil {
    return WrapError(ErrValidation, "database config invalid", err)
}
```

#### Best Practices

1. **Error Context**: Always provide meaningful error messages that include:
   - The operation being performed
   - The values or identifiers involved
   - Any relevant context

2. **Error Wrapping**: When propagating errors up the call stack:
   - Wrap errors to add context
   - Preserve the original error as the cause
   - Use appropriate error codes

3. **Error Handling**: When handling errors:
   - Check specific error codes for recoverable errors
   - Log errors with full context
   - Provide user-friendly error messages

Example:
```go
func (c *Config) GetDatabaseConfig() (*DatabaseConfig, error) {
    var config DatabaseConfig
    if err := c.Get("database", &config); err != nil {
        if IsErrorCode(err, ErrNotFound) {
            // Use defaults
            return DefaultDatabaseConfig(), nil
        }
        return nil, WrapError(
            ErrValidation,
            "failed to load database configuration",
            err,
        )
    }
    return &config, nil
}
```

## Stack Manipulation

Lugo provides direct access to the Lua stack through a set of methods that allow you to manipulate and inspect stack values safely.

### Stack Operations

1. **Push Operations**
   ```go
   // Push values onto the stack
   cfg.PushValue("string")  // Push string
   cfg.PushValue(42)        // Push number
   cfg.PushValue(true)      // Push boolean
   cfg.PushLuaValue(table)  // Push raw Lua value
   ```

2. **Pop Operations**
   ```go
   // Pop value from the stack
   val, err := cfg.PopValue()
   if err != nil {
       // Handle error
   }
   ```

3. **Peek Operations**
   ```go
   // Peek at stack values without removing them
   // Position 1 is top of stack, increasing numbers go deeper
   val, err := cfg.PeekValue(1)  // Look at top value
   val, err := cfg.PeekValue(2)  // Look at second from top
   ```

4. **Stack Information**
   ```go
   size := cfg.GetStackSize()  // Get current stack size
   ```

5. **Raw Value Access**
   ```go
   // Get raw Lua value at position
   lv, err := cfg.GetRawLuaValue(1)
   if err != nil {
       // Handle error
   }
   ```

### Stack Management

1. **Stack Clearing**
   ```go
   cfg.ClearStack()  // Remove all values from stack
   ```

2. **Type Conversion**
   - Values pushed onto the stack are automatically converted between Go and Lua types
   - Supported type mappings:
     - Go string ↔ Lua string
     - Go int/float64 ↔ Lua number
     - Go bool ↔ Lua boolean
     - Go map ↔ Lua table
     - Go slice ↔ Lua table (array)

### Error Handling

Stack operations include comprehensive error handling for common scenarios:

1. **Empty Stack Errors**
   - Attempting to pop or peek from an empty stack
   - Attempting to access values beyond stack size

2. **Type Conversion Errors**
   - Pushing unsupported Go types
   - Converting incompatible Lua types

3. **Stack Position Errors**
   - Invalid stack positions (negative or too large)
   - Stack overflow protection

Example error handling:
```go
// Safe stack manipulation with error handling
if err := cfg.PushValue(value); err != nil {
    switch e := err.(type) {
    case *lugo.StackError:
        log.Printf("Stack error: %v", e)
    case *lugo.TypeError:
        log.Printf("Type conversion error: %v", e)
    default:
        log.Printf("Unknown error: %v", err)
    }
}
```

### Best Practices

1. **Stack Balance**
   - Always maintain stack balance in your operations
   - Use `defer cfg.ClearStack()` in long-running operations
   - Check stack size before and after operations

2. **Error Handling**
   - Always check errors from stack operations
   - Use appropriate error types for specific handling
   - Clean up stack on errors

3. **Performance**
   - Minimize stack operations in tight loops
   - Use bulk operations when possible
   - Clear stack when no longer needed

4. **Thread Safety**
   - Stack operations are not thread-safe by default
   - Use appropriate synchronization when sharing config
   - Consider using separate configs for concurrent operations

## Function Injection and Extension

Lugo provides several ways to extend functionality by injecting Go functions into the Lua environment.

### Basic Function Registration

1. **Direct Function Registration**
   ```go
   // Register a simple Go function
   cfg.RegisterLuaFunction("add", func(L *lua.LState) int {
       a := L.CheckNumber(1)
       b := L.CheckNumber(2)
       L.Push(lua.LNumber(a + b))
       return 1
   })
   ```

2. **String-Based Function Registration**
   ```go
   // Register a Lua function from string
   cfg.RegisterLuaFunctionString("multiply", `
       local x, y = ...
       return x * y
   `)
   ```

### Advanced Function Registration

1. **Function Tables**
   ```go
   // Register a table of functions
   funcs := map[string]interface{}{
       "add": func(x, y int) int {
           return x + y
       },
       "greet": func(name string) string {
           return "Hello, " + name
       },
   }
   
   cfg.RegisterFunctionTable(ctx, "utils", funcs)
   ```

2. **Namespaced Functions**
   ```go
   // Register function with namespace
   cfg.RegisterLuaFunctionWithOptions("get", httpGetFunc, FunctionOptions{
       Namespace: "http.client",
       Metadata: &FunctionMetadata{
           Description: "Makes an HTTP GET request",
           Params: []ParamMetadata{
               {Name: "url", Type: "string", Description: "The URL to request"},
           },
           Returns: []ReturnMetadata{
               {Name: "response", Type: "string", Description: "The response body"},
           },
       },
   })
   ```

### Middleware and Function Composition

1. **Function Middleware**
   ```go
   // Register function with middleware
   cfg.RegisterLuaFunctionWithOptions("secure", func(L *lua.LState) int {
       L.Push(lua.LString("secret"))
       return 1
   }, FunctionOptions{
       BeforeCall: func(L *lua.LState) error {
           // Authentication check
           return nil
       },
       Middleware: []string{"logging", "metrics"},
   })
   ```

2. **Function Composition**
   ```go
   // Compose multiple functions
   cfg.RegisterLuaFunctionString("double", `
       local x = ...
       return x * 2
   `)
   
   cfg.RegisterLuaFunctionString("addOne", `
       local x = ...
       return x + 1
   `)
   
   // Create a new function that combines both
   cfg.ComposeFunctions("doubleAndAddOne", "double", "addOne")
   ```

### Function Metadata and Documentation

1. **Function Documentation**
   ```go
   FunctionMetadata{
       Description: "Performs secure data encryption",
       Params: []ParamMetadata{
           {Name: "data", Type: "string", Description: "Data to encrypt"},
           {Name: "key", Type: "string", Description: "Encryption key"},
       },
       Returns: []ReturnMetadata{
           {Name: "encrypted", Type: "string", Description: "Encrypted data"},
           {Name: "error", Type: "string", Description: "Error message if any"},
       },
   }
   ```

2. **Type Validation**
   ```go
   // Register function with type validation
   cfg.RegisterLuaFunctionWithOptions("setPort", func(L *lua.LState) int {
       port := L.CheckNumber(1)
       if port < 1 || port > 65535 {
           L.ArgError(1, "port must be between 1 and 65535")
       }
       // Set port logic here
       return 0
   }, FunctionOptions{
       ValidateTypes: true,
   })
   ```

### Best Practices

1. **Error Handling**
   - Always validate function arguments
   - Return meaningful error messages
   - Use appropriate error types
   - Handle panics gracefully

2. **Performance**
   - Cache frequently used functions
   - Minimize allocations in hot paths
   - Use appropriate middleware ordering
   - Profile function performance

3. **Security**
   - Validate all inputs
   - Limit function capabilities
   - Use middleware for authentication
   - Implement rate limiting

4. **Maintainability**
   - Document function behavior
   - Use consistent naming conventions
   - Keep functions focused and small
   - Test edge cases thoroughly

## Middleware System

Lugo's middleware system allows you to intercept and modify configuration values during loading and processing. This is useful for tasks such as decryption, validation, transformation, and logging.

### Basic Middleware Usage

```go
// Initialize Lugo with middleware
cfg := lugo.New(
    lugo.WithMiddleware(decryptSecrets),
    lugo.WithMiddleware(validateConfig),
    lugo.WithMiddleware(transformValues),
)
```

### Creating Middleware

Middleware functions have the following signature:
```go
type Middleware func(ctx context.Context, config *Config) error
```

Example middleware implementation:
```go
func decryptSecrets(ctx context.Context, cfg *Config) error {
    // Get the current value from the stack
    val, err := cfg.PopValue()
    if err != nil {
        return err
    }

    // Process the value (e.g., decrypt secrets)
    processed, err := processValue(val)
    if err != nil {
        return err
    }

    // Push the processed value back
    return cfg.PushValue(processed)
}
```

### Common Middleware Patterns

1. **Secret Decryption**
```go
func decryptSecrets(ctx context.Context, cfg *Config) error {
    val, err := cfg.PopValue()
    if err != nil {
        return err
    }

    // Check if value needs decryption
    str, ok := val.(string)
    if !ok || !strings.HasPrefix(str, "enc:") {
        return cfg.PushValue(val)
    }

    // Decrypt the value
    decrypted, err := decrypt(strings.TrimPrefix(str, "enc:"))
    if err != nil {
        return err
    }

    return cfg.PushValue(decrypted)
}

// Usage in Lua
config = {
    database = {
        password = "enc:AES256_encrypted_value_here",
        api_key = "enc:another_encrypted_value"
    }
}
```

2. **Value Transformation**
```go
func transformValues(ctx context.Context, cfg *Config) error {
    val, err := cfg.PopValue()
    if err != nil {
        return err
    }

    switch v := val.(type) {
    case string:
        // Transform string values
        transformed := strings.ToLower(v)
        return cfg.PushValue(transformed)
    case map[string]interface{}:
        // Transform map values recursively
        for k, mapVal := range v {
            if str, ok := mapVal.(string); ok {
                v[k] = strings.ToLower(str)
            }
        }
        return cfg.PushValue(v)
    default:
        return cfg.PushValue(val)
    }
}
```

3. **Validation Middleware**
```go
func validateConfig(ctx context.Context, cfg *Config) error {
    val, err := cfg.PopValue()
    if err != nil {
        return err
    }

    // Validate the configuration
    if err := validate(val); err != nil {
        return err
    }

    return cfg.PushValue(val)
}

func validate(val interface{}) error {
    switch v := val.(type) {
    case map[string]interface{}:
        return validateMap(v)
    case []interface{}:
        return validateArray(v)
    default:
        return validateScalar(v)
    }
}
```

### Advanced Middleware Features

1. **Middleware Chaining**
```go
// Chain multiple middleware functions
cfg := lugo.New(
    lugo.WithMiddleware(
        logAccess,                // Log access to configuration
        decryptSecrets,          // Decrypt sensitive values
        validateConfig,          // Validate configuration
        transformValues,         // Transform values if needed
        auditChanges,           // Audit configuration changes
    ),
)
```

2. **Context-Aware Middleware**
```go
func contextAwareMiddleware(ctx context.Context, cfg *Config) error {
    // Get values from context
    userID := ctx.Value("user_id").(string)
    role := ctx.Value("role").(string)

    // Process based on context
    val, err := cfg.PopValue()
    if err != nil {
        return err
    }

    // Apply context-specific transformations
    processed, err := processWithContext(val, userID, role)
    if err != nil {
        return err
    }

    return cfg.PushValue(processed)
}
```

3. **Error Handling Middleware**
```go
func errorHandler(ctx context.Context, cfg *Config) error {
    val, err := cfg.PopValue()
    if err != nil {
        // Log the error
        log.Printf("Error in middleware: %v", err)
        
        // Provide fallback value
        return cfg.PushValue(getDefaultValue())
    }

    return cfg.PushValue(val)
}
```

### Best Practices

1. **Middleware Design**
   - Keep middleware functions focused and single-purpose
   - Handle errors appropriately
   - Maintain stack balance (pop/push operations)
   - Document middleware behavior and requirements

2. **Performance**
   - Order middleware for optimal performance
   - Cache processed values when possible
   - Avoid unnecessary processing
   - Profile middleware performance

3. **Security**
   - Validate all inputs
   - Limit function capabilities
   - Use middleware for authentication
   - Implement rate limiting

4. **Maintainability**
   - Use clear naming conventions
   - Document middleware dependencies
   - Keep middleware stateless when possible
   - Write tests for middleware functions

5. **Error Handling**
   - Provide meaningful error messages
   - Implement proper error recovery
   - Log errors appropriately
   - Consider fallback strategies

## Validation

Lugo supports struct tag validation:

```go
type ServerConfig struct {
    Host     string        `lua:"host" validate:"required,hostname"`
    Port     int           `lua:"port" validate:"required,min=1024,max=65535"`
    Timeout  time.Duration `lua:"timeout" validate:"required,min=1s,max=1m"`
}
```

Built-in validators:
- required: Field must be present
- min, max: Numeric bounds
- oneof: Value must be one of given options
- email: Valid email address
- hostname: Valid hostname
- url: Valid URL
- duration: Valid time duration

Custom validators:
```go
cfg.RegisterValidator("port_range", func(v interface{}) bool {
    port, ok := v.(int)
    if !ok {
        return false
    }
    return port >= 1024 && port <= 65535
})

type Config struct {
    Port int `lua:"port" validate:"port_range"`
}
```

## Templates and Environment Variables

Lugo supports template processing for configuration files, making it easy to customize configuration based on environment variables or other dynamic values.

### Basic Template Syntax

Templates use the Go template syntax with `{{` and `}}` delimiters:

```lua
config = {
    database = {
        host = "{{ .DB_HOST }}",
        port = {{ .DB_PORT | default "5432" }},
        name = "myapp_{{ .ENV }}"
    }
}
```

### Available Template Functions

Lugo provides several built-in template functions:

- `env "NAME"`: Get environment variable
- `default VALUE`: Provide default value
- `required`: Ensure value is provided
- `join SEPARATOR LIST`: Join list elements
- `upper`: Convert to uppercase
- `lower`: Convert to lowercase

Example usage:
```lua
config = {
    -- Get required environment variable
    api_key = "{{ env "API_KEY" | required }}",
    
    -- Provide default value
    log_level = "{{ env "LOG_LEVEL" | default "info" }}",
    
    -- Join array elements
    allowed_origins = {{ .CORS_ORIGINS | default "localhost,127.0.0.1" | split "," | printf "%#v" }},
    
    -- Compute based on environment
    pool_size = {{ if eq .ENV "production" }}100{{ else }}10{{ end }}
}
```

### Loading with Environment

You can provide environment variables when loading configuration:

```go
cfg := lugo.New(lugo.WithTemplating(true))

err := cfg.LoadFile(context.Background(), "config.lua", lugo.LoadOptions{
    Environment: map[string]string{
        "ENV": "production",
        "DB_HOST": "db.example.com",
        "DB_PORT": "5432",
        "API_KEY": "secret",
    },
})
```

## Dynamic Configuration

Dynamic configuration allows your application to adapt to changes without requiring a restart. However, this power comes with complexity that needs to be carefully managed.

### When to Use Dynamic Configuration

Dynamic configuration is most valuable when:
1. You need to adjust application behavior in production without downtime
2. You're implementing feature flags or A/B testing
3. You need to respond to changing conditions (load, resource availability, etc.)
4. Different environments require different settings

However, be cautious about making too many settings dynamic. Each dynamic setting:
- Increases complexity in your application
- Requires careful consideration of thread safety
- Needs proper validation and safety checks
- Should be monitored and audited

### Implementation Strategies

There are several ways to implement dynamic configuration, each with its own trade-offs:

1. Full Reload
   - Simplest to implement
   - Safest for consistency
   - But may briefly pause configuration access
   - Best for infrequent changes

2. Atomic Updates
   - More complex implementation
   - No pauses in configuration access
   - Requires more memory (keeps two copies)
   - Best for frequent changes

3. Partial Updates
   - Most complex to implement
   - Lowest resource usage
   - Highest risk of inconsistency
   - Best for large configurations with small, frequent changes

### Best Practices

1. Validate Before Applying
```go
// Always validate new configuration before applying
func (s *Server) validateNewConfig(cfg *Config) error {
    // Schema validation
    if err := cfg.Validate(); err != nil {
        return fmt.Errorf("schema validation failed: %w", err)
    }
    
    // Business logic validation
    if cfg.MaxConnections < cfg.MinConnections {
        return fmt.Errorf("max connections must be >= min connections")
    }
    
    // Resource validation
    if cfg.MaxMemory > systemMemory*0.8 {
        return fmt.Errorf("max memory setting too high for system")
    }
    
    return nil
}
```

2. Implement Rollback Capability
```go
type ConfigManager struct {
    current  atomic.Value  // Current config
    previous atomic.Value  // Previous config
    mu       sync.RWMutex
}

func (cm *ConfigManager) Update(newCfg *Config) error {
    cm.mu.Lock()
    defer cm.mu.Unlock()
    
    // Store current as previous
    if curr := cm.current.Load(); curr != nil {
        cm.previous.Store(curr)
    }
    
    // Update current
    cm.current.Store(newCfg)
    
    return nil
}

func (cm *ConfigManager) Rollback() error {
    cm.mu.Lock()
    defer cm.mu.Unlock()
    
    prev := cm.previous.Load()
    if prev == nil {
        return errors.New("no previous configuration available")
    }
    
    cm.current.Store(prev)
    return nil
}
```

3. Monitor Changes
```go
func (cm *ConfigManager) monitorChanges() {
    metrics := NewMetrics()
    
    cm.OnChange(func(old, new *Config) {
        // Record change
        metrics.ConfigChangesTotal.Inc()
        
        // Compare values
        changes := diffConfigs(old, new)
        for _, change := range changes {
            metrics.ConfigValueChanges.WithLabelValues(change.Path).Inc()
            
            // Log significant changes
            if isSignificantChange(change) {
                log.Printf("Significant config change: %s: %v -> %v",
                    change.Path, change.OldValue, change.NewValue)
            }
        }
    })
}
```

### Common Pitfalls

1. Race Conditions
   - Always use atomic operations or proper locking
   - Consider using read-write locks for better performance
   - Be careful with goroutines accessing configuration

2. Inconsistent State
   - Update related values atomically
   - Consider using transactions for complex updates
   - Validate configuration consistency

3. Resource Leaks
   - Clean up old configurations
   - Watch for goroutine leaks in watchers
   - Monitor memory usage during updates

4. Cascading Failures
   - Implement circuit breakers
   - Don't make critical systems fully dynamic
   - Have fallback configurations

### When Not to Use Dynamic Configuration

Dynamic configuration might not be appropriate when:
1. The configuration is critical for system stability
2. Changes require careful coordination across services
3. The overhead of dynamic updates outweighs the benefits
4. You need strict audit trails for all changes

In these cases, consider:
- Using static configuration with deployment-time updates
- Implementing a more formal change management process
- Using feature flags instead of full dynamic configuration
- Breaking your configuration into static and dynamic parts

### Integration with Other Systems

Dynamic configuration often needs to work with other systems:

1. Service Discovery
```go
// Example integration with Consul
func NewConsulWatcher(cfg *Config, consul *api.Client) (*Watcher, error) {
    return cfg.Watch(WatchOptions{
        Provider: &ConsulProvider{
            client: consul,
            prefix: "config/",
        },
        Interval: 30 * time.Second,
        OnChange: func(changes []Change) {
            // Update service registration if needed
            if shouldUpdateRegistration(changes) {
                updateServiceRegistration(consul, changes)
            }
        },
    })
}
```

2. Metrics and Monitoring
```go
// Track configuration changes in metrics
func (cm *ConfigManager) recordMetrics() {
    cm.OnChange(func(old, new *Config) {
        // Record change counts
        metrics.ConfigurationChanges.Inc()
        
        // Record value changes
        if old.MaxConnections != new.MaxConnections {
            metrics.ConfigValueChange.With(prometheus.Labels{
                "parameter": "max_connections",
            }).Set(float64(new.MaxConnections))
        }
        
        // Record timing
        metrics.TimeSinceLastChange.Set(
            time.Since(cm.lastChangeTime).Seconds(),
        )
    })
}
```

3. Audit Logging
```go
// Log all configuration changes
func (cm *ConfigManager) auditChanges() {
    cm.OnChange(func(old, new *Config) {
        diff := diffConfigs(old, new)
        
        // Create audit entry
        audit.Log(audit.ConfigChange{
            Time:     time.Now(),
            User:     cm.currentUser,
            Changes:  diff,
            Metadata: map[string]string{
                "source": "dynamic_update",
                "reason": cm.changeReason,
            },
        })
    })
}
```

### Security Considerations

Security is a critical aspect of configuration management. Lugo provides several layers of security that should be carefully considered for your application.

#### Sandbox Security

The Lua sandbox is your first line of defense. It's crucial to understand what to enable and what to restrict:

1. File System Access
   - Default: Disabled
   - Enable only if you need to load additional configuration files
   - Consider using virtual filesystem for controlled access
   - Never enable in untrusted environments

Example of secure sandbox configuration:
```go
cfg := lugo.New(
    lugo.WithSandbox(&lugo.Sandbox{
        // Disable dangerous operations
        EnableFileIO: false,
        EnableNetworking: false,
        EnableSyscalls: false,
        
        // Set resource limits
        MaxMemory: 10 * 1024 * 1024,  // 10MB
        MaxExecutionTime: 5 * time.Second,
        
        // Whitelist safe packages
        AllowedPackages: []string{
            "string",
            "table",
            "math",
        },
    }),
)
```

When to adjust these settings:
- Enable FileIO: Only for trusted environments where configurations need to reference other files
- Enable Networking: Rarely needed; consider middleware instead
- Increase Memory: For large configurations or complex computations
- Extend Timeout: For configurations with heavy computation

#### Secret Management

Lugo provides several approaches to handling sensitive data:

1. Environment Variables (Simplest)
   - Good for small applications
   - Easy to manage
   - But limited in features
   - No encryption at rest

```lua
database = {
    password = os.getenv("DB_PASSWORD"),
    api_key = os.getenv("API_KEY")
}
```

2. Encrypted Values (More Secure)
   - Supports encryption at rest
   - Key rotation
   - Audit logging
   - But more complex to manage

```go
type SecretProvider interface {
    Encrypt(value string) (string, error)
    Decrypt(value string) (string, error)
}

cfg := lugo.New(
    lugo.WithSecretProvider(&VaultProvider{
        Client: vaultClient,
        Path: "secret/myapp",
    }),
)
```

3. External Secret Stores (Most Secure)
   - Best for production environments
   - Centralized management
   - Access control
   - Audit logging
   - But highest complexity

```go
// Integration with HashiCorp Vault
type VaultConfig struct {
    Address     string
    Role        string
    SecretPath  string
    MaxRetries  int
    Timeout     time.Duration
}

func NewVaultProvider(config VaultConfig) (*VaultProvider, error) {
    // Initialize Vault client
    client, err := vault.NewClient(&vault.Config{
        Address: config.Address,
    })
    if err != nil {
        return nil, err
    }
    
    return &VaultProvider{
        client: client,
        config: config,
    }, nil
}
```

#### Best Practices for Secrets

1. Never Store Secrets in Version Control
   - Use template placeholders
   - Document required secrets
   - Provide sensible defaults for optional secrets

2. Implement Secret Rotation
   - Support multiple versions
   - Graceful rotation
   - Audit logging

```go
type RotatableSecret struct {
    Current     string
    Previous    string
    NextUpdate  time.Time
}

func (s *RotatableSecret) Rotate(newValue string) {
    s.Previous = s.Current
    s.Current = newValue
    s.NextUpdate = time.Now().Add(24 * time.Hour)
}
```

3. Access Control
   - Limit who can read/write secrets
   - Audit all access
   - Implement least privilege

```go
type SecretAccess struct {
    Path      string
    Operation string
    User      string
    Time      time.Time
    Allowed   bool
}

func (p *VaultProvider) auditAccess(access SecretAccess) {
    if access.Allowed {
        audit.Log("secret_access",
            "user", access.User,
            "path", access.Path,
            "operation", access.Operation,
        )
    } else {
        audit.Log("secret_access_denied",
            "user", access.User,
            "path", access.Path,
            "operation", access.Operation,
        )
    }
}
```

### Configuration Watcher

The configuration watcher allows you to automatically reload configuration files when they change on disk. This is useful for updating configuration without restarting your application.

#### Basic Usage

```go
// Create a watcher with basic configuration
watcher, err := cfg.NewWatcher(lugo.WatcherConfig{
    Paths:        []string{"config.lua"},
    PollInterval: 5 * time.Second,
    OnReload: func(err error) {
        if err != nil {
            log.Printf("Error reloading config: %v", err)
            return
        }
        log.Println("Configuration reloaded successfully")
    },
})
if err != nil {
    log.Fatal(err)
}
defer watcher.Close()
```

#### Watcher Configuration Options

- `Paths`: List of configuration files to watch
- `PollInterval`: How often to check for changes (default: 5 seconds)
- `DebounceInterval`: Time to wait after a change before reloading (default: 100ms)
- `OnReload`: Callback function executed after configuration reload

#### Advanced Usage

```go
watcher, err := cfg.NewWatcher(lugo.WatcherConfig{
    // Watch multiple configuration files
    Paths: []string{
        "config.lua",
        fmt.Sprintf("config.%s.lua", cfg.Environment()),
    },
    PollInterval: 30 * time.Second,
    DebounceInterval: 500 * time.Millisecond,
    OnReload: func(err error) {
        if err != nil {
            log.Printf("Error reloading config: %v", err)
            return
        }
        
        // Re-parse configuration after reload
        var newConfig Config
        if err := cfg.Get(context.Background(), "config", &newConfig); err != nil {
            log.Printf("Error parsing new config: %v", err)
            return
        }
        
        // Apply new configuration
        applyNewConfiguration(newConfig)
    },
})
```

#### Dynamic Path Management

The watcher supports adding and removing paths at runtime:

```go
// Add a new path to watch
if err := watcher.AddPath("additional-config.lua"); err != nil {
    log.Printf("Error adding path: %v", err)
}

// Remove a path from watching
if err := watcher.RemovePath("config.lua"); err != nil {
    log.Printf("Error removing path: %v", err)
}
```

#### Best Practices

1. **Error Handling**
   - Always handle reload errors in the OnReload callback
   - Consider implementing retry logic for transient failures
   - Log errors with appropriate context

2. **Resource Management**
   - Close watchers when they're no longer needed using `defer watcher.Close()`
   - Avoid creating multiple watchers for the same files
   - Use appropriate poll intervals to balance responsiveness and system load

3. **Configuration Updates**
   - Implement proper synchronization when applying new configuration
   - Consider versioning or checksums to track configuration changes
   - Validate new configuration before applying it

4. **Performance Considerations**
   - Use reasonable poll intervals (5-30 seconds) to minimize system load
   - Set appropriate debounce intervals to prevent rapid reloads
   - Consider the impact of configuration reloads on your application

### Environment Management

The environment management system in Lugo provides robust handling of different configuration environments (development, staging, production, etc.) and environment variable integration.

#### Basic Environment Setup

```go
// Initialize Lugo with environment support
cfg := lugo.New(
    lugo.WithEnvironmentOverrides(true),
)

// Set current environment (defaults to "development")
cfg.SetEnvironment("production")

// Load base configuration
if err := cfg.LoadFile(ctx, "config.lua"); err != nil {
    log.Fatal(err)
}

// Load environment-specific overrides
envConfig := fmt.Sprintf("config.%s.lua", cfg.Environment())
if err := cfg.LoadFile(ctx, envConfig); err != nil && !os.IsNotExist(err) {
    log.Fatal(err)
}
```

#### Environment Variables

Lugo provides seamless integration with environment variables:

```go
// Initialize with environment variable support
cfg := lugo.New(
    lugo.WithEnvironmentOverrides(true),
    lugo.WithEnvironmentPrefix("APP_"), // Only load vars with APP_ prefix
)

// Access in Lua configuration
config = {
    database = {
        host = os.getenv("APP_DB_HOST"),
        port = tonumber(os.getenv("APP_DB_PORT")) or 5432,
        name = string.format("myapp_%s", ENV)  -- ENV is automatically set
    }
}
```

#### Environment-Specific Configuration Files

Structure your configuration files to support different environments:

```lua
-- config.lua (base configuration)
config = {
    app_name = "myapp",
    log_level = "info",
    database = {
        pool_size = 10,
        timeout = "30s"
    }
}

-- config.production.lua (production overrides)
config.log_level = "warn"
config.database.pool_size = 50
config.database.timeout = "60s"

-- config.development.lua (development overrides)
config.log_level = "debug"
config.database.pool_size = 5
```

#### Advanced Environment Features

1. **Environment Detection**
```go
// Auto-detect environment from ENV variable
cfg := lugo.New(
    lugo.WithEnvironmentDetection(true),
    lugo.WithEnvironmentVariable("APP_ENV"),
)
```

2. **Environment-Specific Validation**
```go
type Config struct {
    LogLevel  string `lua:"log_level" validate:"required,oneof=debug info warn error"`
    Database struct {
        PoolSize int    `lua:"pool_size" validate:"required,min=5"`
        Timeout  string `lua:"timeout" validate:"required,duration"`
    } `lua:"database"`
}

// Validation rules can vary by environment
if cfg.Environment() == "production" {
    // Stricter validation for production
    config.Database.PoolSize = validatePoolSize(config.Database.PoolSize, 20, 100)
} else {
    // Relaxed validation for development
    config.Database.PoolSize = validatePoolSize(config.Database.PoolSize, 5, 50)
}
```

### Advanced Type Conversion

Lugo provides a sophisticated type conversion system that handles complex transformations between Lua and Go types, with support for custom conversions, edge cases, and error handling.

#### Basic Type Mappings

| Lua Type | Go Types |
|----------|----------|
| nil | nil, pointer types, interface{} |
| string | string, []byte, time.Duration, custom string-based types |
| number | int, int8-64, uint, uint8-64, float32, float64 |
| boolean | bool, string ("true"/"false") |
| table (array) | slice, array, custom slice types |
| table (hash) | map, struct, custom struct types |

#### Using Type Converter

```go
// Create a type converter
converter := &lugo.TypeConverter{}

// Basic conversions
str, err := converter.ToString("hello")           // string -> string
num, err := converter.ToInt("123")               // string -> int
float, err := converter.ToFloat(42)              // int -> float
bool, err := converter.ToBool("yes")             // string -> bool
duration, err := converter.ToDuration("30s")     // string -> time.Duration
```

#### Custom Type Conversions

1. **String-Based Types**
```go
type Status string

func (s *Status) FromString(str string) error {
    switch strings.ToLower(str) {
    case "active", "enabled", "on":
        *s = Status("active")
    case "inactive", "disabled", "off":
        *s = Status("inactive")
    default:
        return fmt.Errorf("invalid status: %s", str)
    }
    return nil
}

// Register custom converter
cfg.RegisterConverter(func(val string) (Status, error) {
    var s Status
    err := s.FromString(val)
    return s, err
})
```

2. **Complex Types**
```go
type IPAddress struct {
    IP   net.IP
    Port int
}

// Register function handling complex types
vm.RegisterConverter(func(val interface{}) (IPAddress, error) {
    switch v := val.(type) {
    case string:
        host, port, err := net.SplitHostPort(v)
        if err != nil {
            return IPAddress{}, err
        }
        portNum, err := strconv.Atoi(port)
        if err != nil {
            return IPAddress{}, err
        }
        return IPAddress{net.ParseIP(host), portNum}, nil
    case map[string]interface{}:
        // Handle table conversion
        return parseIPAddressFromTable(v)
    default:
        return IPAddress{}, fmt.Errorf("unsupported type: %T", val)
    }
})
```

#### Type Conversion with Validation

1. **Validated Types**
```go
type Port int

func (p *Port) UnmarshalLua(val interface{}) error {
    num, err := lugo.ToInt(val)
    if err != nil {
        return err
    }
    
    if num < 1 || num > 65535 {
        return fmt.Errorf("port must be between 1 and 65535")
    }
    
    *p = Port(num)
    return nil
}

type Email string

func (e *Email) UnmarshalLua(val interface{}) error {
    str, err := lugo.ToString(val)
    if err != nil {
        return err
    }
    
    if !isValidEmail(str) {
        return fmt.Errorf("invalid email format: %s", str)
    }
    
    *e = Email(str)
    return nil
}
```

2. **Composite Validation**
```go
type ServerConfig struct {
    LogLevel  string `lua:"log_level" validate:"required,oneof=debug info warn error"`
    Database struct {
        PoolSize int    `lua:"pool_size" validate:"required,min=5"`
        Timeout  string `lua:"timeout" validate:"required,duration"`
    } `lua:"database"`
}

func (sc *ServerConfig) UnmarshalLua(val interface{}) error {
    // First, convert the basic structure
    if err := convertBasicStructure(val, sc); err != nil {
        return err
    }
    
    // Then, validate the complete structure
    return sc.Validate()
}

func (sc *ServerConfig) Validate() error {
    if sc.LogLevel == "" {
        return fmt.Errorf("log level is required")
    }
    if sc.Database.PoolSize == 0 {
        return fmt.Errorf("pool size is required")
    }
    if sc.Database.Timeout == "" {
        return fmt.Errorf("timeout is required")
    }
    return nil
}
```

#### Advanced Type Conversion Patterns

1. **Union Types**
```go
type Value struct {
    Type ValueType
    Data interface{}
}

type ValueType int

const (
    StringValue ValueType = iota
    NumberValue
    BoolValue
    ArrayValue
    MapValue
)

func (v *Value) UnmarshalLua(val interface{}) error {
    switch actual := val.(type) {
    case string:
        v.Type = StringValue
        v.Data = actual
    case float64:
        v.Type = NumberValue
        v.Data = actual
    case bool:
        v.Type = BoolValue
        v.Data = actual
    case []interface{}:
        v.Type = ArrayValue
        v.Data = actual
    case map[string]interface{}:
        v.Type = MapValue
        v.Data = actual
    default:
        return fmt.Errorf("unsupported type: %T", val)
    }
    return nil
}
```

2. **Type Conversion with Context**
```go
type ContextualConverter struct {
    Environment string
    Debug       bool
    context     map[string]interface{}
}

func (cc *ContextualConverter) Convert(val interface{}) (interface{}, error) {
    switch v := val.(type) {
    case string:
        return cc.convertStringWithContext(v)
    case map[string]interface{}:
        return cc.convertMapWithContext(v)
    default:
        return cc.convertDefault(v)
    }
}

func (cc *ContextualConverter) convertStringWithContext(s string) (interface{}, error) {
    // Apply environment-specific transformations
    if cc.Environment == "production" {
        // Apply production rules
    } else {
        // Apply development rules
    }
    
    // Apply debug-specific transformations
    if cc.Debug {
        // Add debug information
    }
    
    return s, nil
}
```

3. **Lazy Type Conversion**
```go
type LazyValue struct {
    raw       interface{}
    converted interface{}
    converter func(interface{}) (interface{}, error)
    once      sync.Once
    err       error
}

func NewLazyValue(raw interface{}, converter func(interface{}) (interface{}, error)) *LazyValue {
    return &LazyValue{
        raw:       raw,
        converter: converter,
    }
}

func (lv *LazyValue) Get() (interface{}, error) {
    lv.once.Do(func() {
        lv.converted, lv.err = lv.converter(lv.raw)
    })
    return lv.converted, lv.err
}
```

#### Performance Optimization Patterns

1. **Cached Type Conversion**
```go
type CachedConverter struct {
    cache  sync.Map
    parser func(string) (interface{}, error)
}

func (cc *CachedConverter) Convert(key string) (interface{}, error) {
    // Check cache first
    if cached, ok := cc.cache.Load(key); ok {
        return cached, nil
    }
    
    // Parse and cache
    parsed, err := cc.parser(key)
    if err != nil {
        return nil, err
    }
    
    cc.cache.Store(key, parsed)
    return parsed, nil
}
```

2. **Batch Type Conversion**
```go
type BatchConverter struct {
    batchSize int
    converter func(interface{}) (interface{}, error)
}

func (bc *BatchConverter) ConvertBatch(items []interface{}) ([]interface{}, error) {
    results := make([]interface{}, len(items))
    errors := make([]error, 0)
    
    // Process in batches
    for i := 0; i < len(items); i += bc.batchSize {
        end := i + bc.batchSize
        if end > len(items) {
            end = len(items)
        }
        
        // Convert batch
        batch := items[i:end]
        for j, item := range batch {
            result, err := bc.converter(item)
            if err != nil {
                errors = append(errors, fmt.Errorf("item %d: %w", i+j, err))
                continue
            }
            results[i+j] = result
        }
    }
    
    if len(errors) > 0 {
        return results, fmt.Errorf("batch conversion errors: %v", errors)
    }
    return results, nil
}
```

### Hook System

The Lugo hook system provides a powerful way to extend and customize the behavior of your Lua scripts at runtime. Hooks allow you to intercept and modify the execution flow, add custom functionality, and implement cross-cutting concerns.

#### Basic Hook Usage

1. **Registering Hooks**
```go
// Define a hook function
hook := func(L *lua.State) error {
    // Hook implementation
    return nil
}

// Register the hook
vm.AddHook("before_call", hook)
```

2. **Available Hook Points**
```go
const (
    // Called before executing any Lua function
    HookBeforeCall = "before_call"
    
    // Called after executing any Lua function
    HookAfterCall = "after_call"
    
    // Called before loading any module
    HookBeforeRequire = "before_require"
    
    // Called after loading a module
    HookAfterRequire = "after_require"
    
    // Called before executing any statement
    HookBeforeStatement = "before_statement"
    
    // Called when an error occurs
    HookOnError = "on_error"
)
```

#### Advanced Hook Patterns

1. **Stack Manipulation in Hooks**
```go
// Hook that adds debug information to function calls
vm.AddHook(HookBeforeCall, func(L *lua.State) error {
    // Get the function name
    if L.IsFunction(-1) {
        funcInfo := L.Debug()
        
        // Add debug context to the stack
        L.PushString(fmt.Sprintf("Debug: calling %s at line %d", 
            funcInfo.Name, funcInfo.CurrentLine))
        L.Insert(-2)
        
        // Now the debug string is below the function
    }
    return nil
})
```

2. **Error Handling Hooks**
```go
// Hook that provides detailed error information
vm.AddHook(HookOnError, func(L *lua.State) error {
    if err := L.ToString(-1); err != nil {
        // Get the stack trace
        debug := L.GetStack(0)
        if debug != nil {
            L.GetInfo("Sl", debug)
            file := debug.Source
            line := debug.CurrentLine
            
            // Log detailed error information
            log.Printf("Error in %s at line %d: %s", 
                file, line, err)
        }
    }
    return nil
})
```

3. **Module Loading Hooks**
```go
// Hook that controls module loading
vm.AddHook(HookBeforeRequire, func(L *lua.State) error {
    if moduleName := L.ToString(-1); moduleName != "" {
        // Check if module is allowed
        if !isAllowedModule(moduleName) {
            return fmt.Errorf("module %s is not allowed", 
                moduleName)
        }
        
        // Add custom module path
        L.GetGlobal("package")
        path := L.GetField(-1, "path")
        newPath := path.(string) + ";" + 
            "/custom/module/path/?.lua"
        L.PushString(newPath)
        L.SetField(-2, "path")
    }
    return nil
})
```

#### Hook Composition

1. **Chaining Multiple Hooks**
```go
// Define hook chain
type HookChain struct {
    hooks []HookFunc
}

func (hc *HookChain) Execute(L *lua.State) error {
    for _, hook := range hc.hooks {
        if err := hook(L); err != nil {
            return err
        }
    }
    return nil
}

// Usage example
chain := &HookChain{
    hooks: []HookFunc{
        validateInput,
        addDebugInfo,
        measurePerformance,
    },
}

vm.AddHook(HookBeforeCall, chain.Execute)
```

2. **Conditional Hooks**
```go
// Hook that only executes under certain conditions
func conditionalHook(condition func() bool) HookFunc {
    return func(L *lua.State) error {
        if !condition() {
            return nil
        }
        
        // Hook implementation
        return nil
    }
}

// Usage example
vm.AddHook(HookBeforeCall, conditionalHook(func() bool {
    return os.Getenv("DEBUG") == "true"
}))
```

#### Performance Considerations

1. **Lightweight Hooks**
```go
// Efficient hook implementation
vm.AddHook(HookBeforeStatement, func(L *lua.State) error {
    // Use sync.Pool for frequently allocated objects
    bufPool := sync.Pool{
        New: func() interface{} {
            return new(bytes.Buffer)
        },
    }
    
    // Get buffer from pool
    buf := bufPool.Get().(*bytes.Buffer)
    defer bufPool.Put(buf)
    
    // Use buffer for string operations
    buf.WriteString("Statement executed: ")
    buf.WriteString(L.ToString(-1))
    
    return nil
})
```

2. **Hook Priority**
```go
// Define hook priority levels
const (
    PriorityHigh   = 100
    PriorityNormal = 50
    PriorityLow    = 0
)

// Hook with priority
type PrioritizedHook struct {
    priority int
    hook     HookFunc
}

// Sort hooks by priority
type HookManager struct {
    hooks []PrioritizedHook
}

func (hm *HookManager) AddHook(priority int, hook HookFunc) {
    hm.hooks = append(hm.hooks, PrioritizedHook{
        priority: priority,
        hook:     hook,
    })
    sort.Slice(hm.hooks, func(i, j int) bool {
        return hm.hooks[i].priority > hm.hooks[j].priority
    })
}
```

#### Best Practices

1. **Hook Documentation**
   - Always document the purpose and expected behavior of hooks
   - Specify any assumptions about the Lua stack state
   - Document any side effects or stack modifications

2. **Error Handling**
   - Always return errors rather than panicking
   - Clean up resources in case of errors
   - Maintain stack balance even in error cases

3. **Performance**
   - Keep hooks lightweight
   - Use sync.Pool for frequently allocated objects
   - Consider using conditional hooks in performance-critical code

4. **Testing**
   - Write unit tests for hooks
   - Test error conditions
   - Verify stack balance after hook execution

### Function Injection System

Lugo provides a powerful function injection system that allows you to dynamically inject Go functions into Lua scripts. This enables seamless integration between Go and Lua code, with support for complex types, error handling, and state management.

#### Basic Function Injection

1. **Simple Function Injection**
```go
// Inject a simple function
vm.Inject("math", map[string]interface{}{
    "add": func(a, b int) int {
        return a + b
    },
    "subtract": func(a, b int) int {
        return a - b
    },
})

-- In Lua
local math = require("math")
local result = math.add(10, 5)  -- returns 15
```

2. **Module Injection**
```go
// Create a module with multiple functions
type HTTPClient struct {
    client *http.Client
    baseURL string
}

func NewHTTPModule(baseURL string) map[string]interface{} {
    client := &HTTPClient{
        client: &http.Client{},
        baseURL: baseURL,
    }
    
    return map[string]interface{}{
        "get":    client.Get,
        "post":   client.Post,
        "delete": client.Delete,
    }
}

// Inject the module
vm.Inject("http", NewHTTPModule("https://api.example.com"))

-- In Lua
local http = require("http")
local response = http.get("/users")  -- Makes HTTP GET request
```

#### Advanced Injection Patterns

1. **Context-Aware Functions**
```go
// Inject functions with context
type AppContext struct {
    logger *log.Logger
    config *Config
}

func (ac *AppContext) InjectFunctions(vm *lugo.VM) {
    vm.Inject("app", map[string]interface{}{
        "log": func(level, msg string) {
            ac.logger.Printf("[%s] %s", level, msg)
        },
        "getConfig": func(key string) interface{} {
            return ac.config.Get(key)
        },
    })
}

-- In Lua
local app = require("app")
app.log("info", "Starting process")
local dbConfig = app.getConfig("database")
```

2. **Chainable Functions**
```go
// Inject chainable query builder
type QueryBuilder struct {
    query string
    params []interface{}
}

func NewQueryBuilder() *QueryBuilder {
    return &QueryBuilder{}
}

func (qb *QueryBuilder) Where(condition string, args ...interface{}) *QueryBuilder {
    qb.query += " WHERE " + condition
    qb.params = append(qb.params, args...)
    return qb
}

// Inject the builder
vm.Inject("db", map[string]interface{}{
    "query": func() *QueryBuilder {
        return NewQueryBuilder()
    },
})

-- In Lua
local db = require("db")
local result = db.query()
    :where("age > ?", 18)
    :orderBy("name")
    :limit(10)
    :execute()
```

#### State Management

1. **Persistent State**
```go
// Inject functions with shared state
type StateManager struct {
    state map[string]interface{}
    mutex sync.RWMutex
}

func NewStateManager() *StateManager {
    return &StateManager{
        state: make(map[string]interface{}),
    }
}

func (sm *StateManager) InjectFunctions(vm *lugo.VM) {
    vm.Inject("state", map[string]interface{}{
        "set": func(key string, value interface{}) {
            sm.mutex.Lock()
            defer sm.mutex.Unlock()
            sm.state[key] = value
        },
        "get": func(key string) interface{} {
            sm.mutex.RLock()
            defer sm.mutex.RUnlock()
            return sm.state[key]
        },
    })
}

-- In Lua
local state = require("state")
state.set("user", {name = "Alice", age = 30})
local user = state.get("user")
```

2. **Resource Management**
```go
// Inject functions with resource cleanup
type ResourceManager struct {
    resources map[string]io.Closer
    mutex    sync.RWMutex
}

func (rm *ResourceManager) InjectFunctions(vm *lugo.VM) {
    vm.Inject("resource", map[string]interface{}{
        "acquire": func(name string) (interface{}, error) {
            resource, err := rm.acquireResource(name)
            if err != nil {
                return nil, err
            }
            
            rm.mutex.Lock()
            rm.resources[name] = resource
            rm.mutex.Unlock()
            
            return resource, nil
        },
        "release": func(name string) error {
            rm.mutex.Lock()
            defer rm.mutex.Unlock()
            
            if resource, ok := rm.resources[name]; ok {
                delete(rm.resources, name)
                return resource.Close()
            }
            return nil
        },
    })
}

-- In Lua
local resource = require("resource")
local file = resource.acquire("data.txt")
-- Use file
resource.release("data.txt")
```

#### Error Handling

1. **Error Propagation**
```go
// Inject functions with error handling
vm.Inject("safe", map[string]interface{}{
    "call": func(fn func() error) (err error) {
        defer func() {
            if r := recover(); r != nil {
                err = fmt.Errorf("panic recovered: %v", r)
            }
        }()
        return fn()
    },
})

-- In Lua
local safe = require("safe")
local err = safe.call(function()
    -- Potentially dangerous operation
    error("something went wrong")
end)
if err then
    print("Error:", err)
end
```

#### Best Practices

1. **Function Organization**
   - Group related functions into modules
   - Use consistent naming conventions
   - Document function signatures and behavior
   - Provide example usage in comments

2. **State Management**
   - Use thread-safe state containers
   - Clean up resources properly
   - Handle concurrent access safely
   - Document state dependencies

3. **Error Handling**
   - Return meaningful error messages
   - Use error wrapping for context
   - Recover from panics gracefully
   - Log errors appropriately

4. **Type Safety**
   - Validate input parameters
   - Use strong typing where possible
   - Document type constraints
   - Handle type conversions safely

#### Common Patterns

1. **Configuration Injection**
```go
// Inject configuration functions
type Config struct {
    values map[string]interface{}
    env    string
}

func (c *Config) InjectFunctions(vm *lugo.VM) {
    vm.Inject("config", map[string]interface{}{
        "get": func(key string) interface{} {
            if value, ok := c.values[key]; ok {
                return value
            }
            return nil
        },
        "getEnv": func() string {
            return c.env
        },
    })
}
```

2. **Event System**
```go
// Inject event system functions
type EventSystem struct {
    handlers map[string][]func(interface{})
    mutex    sync.RWMutex
}

func (es *EventSystem) InjectFunctions(vm *lugo.VM) {
    vm.Inject("events", map[string]interface{}{
        "on": func(event string, handler func(interface{})) {
            es.mutex.Lock()
            defer es.mutex.Unlock()
            es.handlers[event] = append(es.handlers[event], handler)
        },
        "emit": func(event string, data interface{}) {
            es.mutex.RLock()
            handlers := es.handlers[event]
            es.mutex.RUnlock()
            
            for _, handler := range handlers {
                handler(data)
            }
        },
    })
}
```

3. **Middleware Pattern**
```go
// Inject middleware system
type MiddlewareSystem struct {
    middlewares []func(func() error) func() error
}

func (ms *MiddlewareSystem) InjectFunctions(vm *lugo.VM) {
    vm.Inject("middleware", map[string]interface{}{
        "use": func(middleware func(func() error) func() error) {
            ms.middlewares = append(ms.middlewares, middleware)
        },
        "execute": func(fn func() error) error {
            // Apply middlewares in reverse order
            wrapped := fn
            for i := len(ms.middlewares) - 1; i >= 0; i-- {
                wrapped = ms.middlewares[i](wrapped)
            }
            return wrapped()
        },
    })
}

-- In Lua
local middleware = require("middleware")
middleware.use(function(next)
    return function()
        print("Before")
        local err = next()
        print("After")
        return err
    end
end)
```

### Middleware System

Lugo's middleware system allows you to intercept and modify configuration values during loading and processing. This is useful for tasks such as decryption, validation, transformation, and logging.

#### Basic Middleware Usage

```go
// Initialize Lugo with middleware
cfg := lugo.New(
    lugo.WithMiddleware(decryptSecrets),
    lugo.WithMiddleware(validateConfig),
    lugo.WithMiddleware(transformValues),
)
```

#### Creating Middleware

Middleware functions have the following signature:
```go
type Middleware func(ctx context.Context, config *Config) error
```

Example middleware implementation:
```go
func decryptSecrets(ctx context.Context, cfg *Config) error {
    // Get the current value from the stack
    val, err := cfg.PopValue()
    if err != nil {
        return err
    }

    // Process the value (e.g., decrypt secrets)
    processed, err := processValue(val)
    if err != nil {
        return err
    }

    // Push the processed value back
    return cfg.PushValue(processed)
}
```

#### Common Middleware Patterns

1. **Secret Decryption**
```go
func decryptSecrets(ctx context.Context, cfg *Config) error {
    val, err := cfg.PopValue()
    if err != nil {
        return err
    }

    // Check if value needs decryption
    str, ok := val.(string)
    if !ok || !strings.HasPrefix(str, "enc:") {
        return cfg.PushValue(val)
    }

    // Decrypt the value
    decrypted, err := decrypt(strings.TrimPrefix(str, "enc:"))
    if err != nil {
        return err
    }

    return cfg.PushValue(decrypted)
}

// Usage in Lua
config = {
    database = {
        password = "enc:AES256_encrypted_value_here",
        api_key = "enc:another_encrypted_value"
    }
}
```

2. **Value Transformation**
```go
func transformValues(ctx context.Context, cfg *Config) error {
    val, err := cfg.PopValue()
    if err != nil {
        return err
    }

    switch v := val.(type) {
    case string:
        // Transform string values
        transformed := strings.ToLower(v)
        return cfg.PushValue(transformed)
    case map[string]interface{}:
        // Transform map values recursively
        for k, mapVal := range v {
            if str, ok := mapVal.(string); ok {
                v[k] = strings.ToLower(str)
            }
        }
        return cfg.PushValue(v)
    default:
        return cfg.PushValue(val)
    }
}
```

3. **Validation Middleware**
```go
func validateConfig(ctx context.Context, cfg *Config) error {
    val, err := cfg.PopValue()
    if err != nil {
        return err
    }

    // Validate the configuration
    if err := validate(val); err != nil {
        return err
    }

    return cfg.PushValue(val)
}

func validate(val interface{}) error {
    switch v := val.(type) {
    case map[string]interface{}:
        return validateMap(v)
    case []interface{}:
        return validateArray(v)
    default:
        return validateScalar(v)
    }
}
```

#### Advanced Middleware Features

1. **Middleware Chaining**
```go
// Chain multiple middleware functions
cfg := lugo.New(
    lugo.WithMiddleware(
        logAccess,                // Log access to configuration
        decryptSecrets,          // Decrypt sensitive values
        validateConfig,          // Validate configuration
        transformValues,         // Transform values if needed
        auditChanges,           // Audit configuration changes
    ),
)
```

2. **Context-Aware Middleware**
```go
func contextAwareMiddleware(ctx context.Context, cfg *Config) error {
    // Get values from context
    userID := ctx.Value("user_id").(string)
    role := ctx.Value("role").(string)

    // Process based on context
    val, err := cfg.PopValue()
    if err != nil {
        return err
    }

    // Apply context-specific transformations
    processed, err := processWithContext(val, userID, role)
    if err != nil {
        return err
    }

    return cfg.PushValue(processed)
}
```

3. **Error Handling Middleware**
```go
func errorHandler(ctx context.Context, cfg *Config) error {
    val, err := cfg.PopValue()
    if err != nil {
        // Log the error
        log.Printf("Error in middleware: %v", err)
        
        // Provide fallback value
        return cfg.PushValue(getDefaultValue())
    }

    return cfg.PushValue(val)
}
```

#### Best Practices

1. **Middleware Design**
   - Keep middleware functions focused and single-purpose
   - Handle errors appropriately
   - Maintain stack balance (pop/push operations)
   - Document middleware behavior and requirements

2. **Performance**
   - Order middleware for optimal performance
   - Cache processed values when possible
   - Avoid unnecessary processing
   - Profile middleware performance

3. **Security**
   - Validate all inputs
   - Limit function capabilities
   - Use middleware for authentication
   - Implement rate limiting

4. **Maintainability**
   - Use clear naming conventions
   - Document middleware dependencies
   - Keep middleware stateless when possible
   - Write tests for middleware functions

5. **Error Handling**
   - Provide meaningful error messages
   - Implement proper error recovery
   - Log errors appropriately
   - Consider fallback strategies

## Version Compatibility

- Go version: 1.16 or later
- Lua version: 5.1 compatible
- Operating Systems: Linux, macOS, Windows

## Performance Considerations

- Memory Usage: Default sandbox limit is 10MB per configuration
- Execution Time: Typical load time < 100ms
- Caching: Built-in caching for compiled Lua code
- Concurrent Access: Thread-safe configuration access

### Best Practices for Performance

1. Use precompiled configurations when possible
2. Enable caching for frequently accessed values
3. Limit template processing to necessary files
4. Use appropriate memory limits for your use case

## Additional Error Handling Examples

```go
// Example 1: Handle template errors
cfg := lugo.New(lugo.WithTemplating(true))
if err := cfg.LoadFile(ctx, "config.lua"); err != nil {
    switch e := err.(type) {
    case *lugo.TemplateError:
        log.Printf("Template error at line %d: %v", e.Line, e.Message)
    case *lugo.SyntaxError:
        log.Printf("Lua syntax error: %v", e)
    default:
        log.Printf("Unknown error: %v", err)
    }
}

// Example 2: Validation with custom error handling
type CustomConfig struct {
    Port int `lua:"port" validate:"required,min=1024,max=65535"`
}

if err := cfg.Get(ctx, "config", &customConfig); err != nil {
    if vErr, ok := err.(validator.ValidationErrors); ok {
        for _, e := range vErr {
            log.Printf("Validation failed for field: %s", e.Field())
        }
    }
}