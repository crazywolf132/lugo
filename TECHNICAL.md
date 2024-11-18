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
```

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

### Validation

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

## Security Considerations

Security is a critical aspect of configuration management. Lugo provides several layers of security that should be carefully considered for your application.

### Sandbox Security

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

### Secret Management

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

### Best Practices for Secrets

1. Never Store Secrets in Version Control
   - Use template placeholders
   - Document required secrets
   - Provide example values

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

## Performance Optimization

Performance in configuration management is about balancing speed, resource usage, and functionality.

### When to Optimize

Consider optimization when:
1. Configuration files are large (>1MB)
2. Updates are frequent (multiple times per minute)
3. Many goroutines access configuration
4. Memory usage is a concern

### Memory Management

1. Value Pooling
   - Reuse common objects
   - Reduce GC pressure
   - But adds complexity

```go
var valuePool = sync.Pool{
    New: func() interface{} {
        return make(map[string]interface{}, 32)
    },
}

func (c *Config) acquireMap() map[string]interface{} {
    m := valuePool.Get().(map[string]interface{})
    for k := range m {
        delete(m, k)
    }
    return m
}

func (c *Config) releaseMap(m map[string]interface{}) {
    valuePool.Put(m)
}
```

2. Lazy Loading
   - Load values only when needed
   - Reduce initial memory usage
   - But may increase complexity

```go
type LazyConfig struct {
    loader func() (interface{}, error)
    value  atomic.Value
    once   sync.Once
}

func (c *LazyConfig) Get() (interface{}, error) {
    var err error
    c.once.Do(func() {
        var v interface{}
        v, err = c.loader()
        if err == nil {
            c.value.Store(v)
        }
    })
    if err != nil {
        return nil, err
    }
    return c.value.Load(), nil
}
```

3. Memory Limits
   - Set appropriate limits
   - Monitor usage
   - Handle out-of-memory gracefully

```go
type MemoryLimiter struct {
    limit uint64
    used  atomic.Uint64
}

func (m *MemoryLimiter) Allocate(size uint64) error {
    if !m.used.CompareAndSwap(
        m.used.Load(),
        m.used.Load()+size,
    ) {
        return ErrMemoryLimit
    }
    return nil
}
```