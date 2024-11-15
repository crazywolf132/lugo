# Lugo

![Build Status](https://github.com/crazywolf132/lugo/workflows/CI/badge.svg)
[![Go Reference](https://pkg.go.dev/badge/github.com/crazywolf132/lugo.svg)](https://pkg.go.dev/github.com/crazywolf132/lugo)
[![Go Report Card](https://goreportcard.com/badge/github.com/crazywolf132/lugo)](https://goreportcard.com/report/github.com/crazywolf132/lugo)
[![License](https://img.shields.io/github/license/crazywolf132/lugo)](LICENSE)

Lugo is a production-ready Go library for embedding Lua configurations with strong type safety, sandboxing, and seamless Go integration. It provides a robust solution for applications that need dynamic configuration, scripting, or business rule execution while maintaining type safety and security.

## 🌟 Key Features

- **🔒 Type-Safe Configuration**: Define configurations using Go structs and access them with full type safety
- **🎯 Simple API**: Easy-to-use interface for registering types and functions
- **🛡️ Sandboxing**: Fine-grained control over script capabilities and resource usage
- **🔌 Go Function Integration**: Expose Go functions to Lua scripts with automatic type conversion
- **⏱️ Context Support**: Built-in support for timeouts and cancellation
- **🔄 Middleware System**: Add cross-cutting concerns like logging or metrics
- **🪝 Hook System**: Monitor and interact with configuration loading and execution
- **🚀 Production Ready**: Thread-safe, well-tested, and battle-tested in production

## 📦 Installation

```bash
go get github.com/crazywolf132/lugo
```

## 🚀 Quick Start

### Basic Configuration

```go
package main

import (
    "context"
    "log"
    
    "github.com/crazywolf132/lugo"
)

// Define your configuration structure
type Config struct {
    Name     string   `lua:"name"`
    Version  string   `lua:"version"`
    Features []string `lua:"features"`
    Debug    bool     `lua:"debug"`
}

func main() {
    // Create a new Lugo instance
    cfg := lugo.New()
    defer cfg.Close()

    // Register your configuration type
    err := cfg.RegisterType(context.Background(), "config", Config{})
    if err != nil {
        log.Fatal(err)
    }

    // Load configuration from string
    err = cfg.L.DoString(`
        config = {
            name = "my-app",
            version = "1.0.0",
            features = {"feature1", "feature2"},
            debug = true
        }
    `)
    if err != nil {
        log.Fatal(err)
    }

    // Get the typed configuration
    var myConfig Config
    err = cfg.Get(context.Background(), "config", &myConfig)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Loaded config: %+v", myConfig)
}
```

### Advanced Usage

#### Registering Go Functions

```go
package main

import (
    "context"
    "log"
    
    "github.com/crazywolf132/lugo"
)

func main() {
    cfg := lugo.New()
    defer cfg.Close()

    // Register a Go function
    err := cfg.RegisterFunction(context.Background(), "processOrder", func(ctx context.Context, orderID string, amount float64) (string, error) {
        // Process the order...
        return "processed", nil
    })
    if err != nil {
        log.Fatal(err)
    }

    // Use the function in Lua
    err = cfg.L.DoString(`
        local status, err = processOrder("123", 99.99)
        assert(status == "processed")
    `)
    if err != nil {
        log.Fatal(err)
    }
}
```

#### Using Sandbox Restrictions

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/crazywolf132/lugo"
)

func main() {
    // Create a new instance with sandbox restrictions
    cfg := lugo.New(
        lugo.WithSandbox(&lugo.Sandbox{
            EnableFileIO:     false,
            EnableNetworking: false,
            MaxMemory:       50 * 1024 * 1024, // 50MB
            MaxExecutionTime: 10 * time.Second,
        }),
    )
    defer cfg.Close()

    // Try to execute restricted operations
    err := cfg.L.DoString(`
        -- This will fail due to sandbox restrictions
        local file = io.open("secret.txt", "r")
    `)
    if err != nil {
        log.Printf("Expected error: %v", err)
    }
}
```

#### Using Middleware for Metrics

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/crazywolf132/lugo"
)

func main() {
    // Create a new instance with middleware
    cfg := lugo.New(
        lugo.WithMiddleware(func(next lugo.LuaFunction) lugo.LuaFunction {
            return func(ctx context.Context, L *lua.LState) ([]lua.LValue, error) {
                start := time.Now()
                results, err := next(ctx, L)
                duration := time.Since(start)
                log.Printf("Function execution took: %v", duration)
                return results, err
            }
        }),
    )
    defer cfg.Close()

    // Register and execute a function
    cfg.RegisterFunction(context.Background(), "slowOperation", func() {
        time.Sleep(100 * time.Millisecond)
    })

    cfg.L.DoString(`slowOperation()`)
}
```

## 🔧 Use Cases

- **Dynamic Configuration**: Load and update application configuration at runtime
- **Business Rules Engine**: Implement complex business rules that can be modified without recompilation
- **Plugin System**: Create a plugin system where functionality can be extended through Lua scripts
- **Game Logic**: Implement game rules and behaviors in Lua while keeping core mechanics in Go
- **ETL Pipelines**: Define data transformation rules in Lua for flexible data processing
- **API Configuration**: Configure API routing, middleware, and handlers using Lua scripts

## 🔐 Security

Lugo provides comprehensive security features through its sandbox system:

- File system access control
- Network access restrictions
- Memory usage limits
- Execution time limits
- Function whitelist/blacklist
- Resource usage monitoring

## 📈 Performance

- Minimal overhead over raw Lua execution
- Efficient type conversion between Go and Lua
- Thread-safe for concurrent script execution
- Memory-efficient with configurable limits
- Fast configuration reloading

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📜 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

Built with [gopher-lua](https://github.com/yuin/gopher-lua)