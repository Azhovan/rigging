# Rigging

[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org/dl/)
[![Go Reference](https://pkg.go.dev/badge/github.com/Azhovan/rigging.svg)](https://pkg.go.dev/github.com/Azhovan/rigging)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/Azhovan/rigging)](https://goreportcard.com/report/github.com/Azhovan/rigging)

**A typed, observable, policy-driven configuration system for Go services.**

Stop debugging configuration errors in production. Rigging gives you compile-time safety, runtime observability, and policy enforcement for your service configuration.

```go
type Config struct {
    Database struct {
        Host     string `conf:"required"`
        Password string `conf:"required,secret"`
    } `conf:"prefix:database"`
}

cfg, err := rigging.NewLoader[Config]().
    WithSource(sourcefile.New("config.yaml", sourcefile.Options{})).
    WithSource(sourceenv.New(sourceenv.Options{Prefix: "APP_"})).
    Load(ctx)
// Type-safe: cfg.Database.Host is a string, not interface{}
// Observable: Know exactly where each value came from
// Policy-driven: Validation rules enforced at load time
```

## Why Rigging?

Configuration management in production services faces several challenges:

- **Type Safety**: String-based key access loses compile-time guarantees
- **Observability**: Difficult to trace where configuration values originated
- **Validation**: Business rules scattered throughout the codebase
- **Testing**: Global state makes configuration hard to test
- **Precedence**: Unclear which source wins when values conflict

Rigging addresses these through three core principles:

### 1. Typed Configuration

Define your configuration schema as Go structs. The compiler catches errors, your IDE provides autocomplete, and refactoring tools work correctly.

```go
type Config struct {
    Database struct {
        Host string `conf:"required"`
        Port int    `conf:"default:5432"`
    } `conf:"prefix:database"`
}

cfg, err := loader.Load(ctx)
// cfg.Database.Port is an int, guaranteed by the compiler
// No runtime type assertions needed
```

### 2. Observable Configuration

Track the source of every configuration value. Know exactly where each value came from for debugging and compliance.

```go
prov, _ := rigging.GetProvenance(cfg)
for _, field := range prov.Fields {
    log.Printf("%s from %s", field.FieldPath, field.SourceName)
}
// Output:
// Database.Host from file:config.yaml
// Database.Password from env:APP_DATABASE__PASSWORD
// Database.Port from default
```

### 3. Policy-Driven Validation

Enforce validation rules at startup. All configuration is validated before your application runs.

```go
type Config struct {
    Environment string `conf:"required,oneof:prod,staging,dev"`
    Database struct {
        Port int `conf:"min:1024,max:65535"`
    } `conf:"prefix:database"`
}

// Custom cross-field validation
loader.WithValidator(rigging.ValidatorFunc[Config](func(ctx context.Context, cfg *Config) error {
    if cfg.Environment == "prod" && cfg.Database.Host == "localhost" {
        return errors.New("production cannot use localhost")
    }
    return nil
}))

cfg, err := loader.Load(ctx)
// If we reach here, all validation passed
```

## Installation

```bash
# Core library (zero dependencies)
go get github.com/Azhovan/rigging

# File support (YAML/JSON/TOML)
go get github.com/Azhovan/rigging/sourcefile

# Environment variables
go get github.com/Azhovan/rigging/sourceenv
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log"
    
    "github.com/Azhovan/rigging"
    "github.com/Azhovan/rigging/sourcefile"
    "github.com/Azhovan/rigging/sourceenv"
)

type Config struct {
    Server struct {
        Port int    `conf:"default:8080"`
        Host string `conf:"default:0.0.0.0"`
    } `conf:"prefix:server"`
    
    Database struct {
        Host     string `conf:"required"`
        Port     int    `conf:"default:5432"`
        Password string `conf:"required,secret"`
    } `conf:"prefix:database"`
}

func main() {
    loader := rigging.NewLoader[Config]().
        WithSource(sourcefile.New("config.yaml", sourcefile.Options{})).
        WithSource(sourceenv.New(sourceenv.Options{Prefix: "APP_"}))
    
    cfg, err := loader.Load(context.Background())
    if err != nil {
        log.Fatal(err)
    }
    
    // Use your configuration
    log.Printf("Starting server on %s:%d", cfg.Server.Host, cfg.Server.Port)
}
```

### Multi-Source Configuration

Sources are processed in order. Later sources override earlier ones.

```go
loader := rigging.NewLoader[Config]().
    WithSource(sourcefile.New("defaults.yaml", sourcefile.Options{})).  // Base configuration
    WithSource(sourcefile.New("config.yaml", sourcefile.Options{})).    // Environment-specific
    WithSource(sourceenv.New(sourceenv.Options{Prefix: "APP_"}))        // Runtime overrides
```

### Validation

Tag-based validation:

```go
type Config struct {
    Port        int    `conf:"required,min:1024,max:65535"`
    Environment string `conf:"required,oneof:prod,staging,dev"`
    Timeout     time.Duration `conf:"default:30s"`
}
```

Custom validation:

```go
loader.WithValidator(rigging.ValidatorFunc[Config](func(ctx context.Context, cfg *Config) error {
    if cfg.Environment == "prod" && cfg.Database.Host == "localhost" {
        return errors.New("production cannot use localhost")
    }
    return nil
}))
```

### Observability

Track configuration sources:

```go
cfg, _ := loader.Load(ctx)

prov, _ := rigging.GetProvenance(cfg)
for _, field := range prov.Fields {
    log.Printf("%s from %s", field.FieldPath, field.SourceName)
}
```

Dump configuration safely:

```go
// Secrets are automatically redacted
rigging.DumpEffective(os.Stdout, cfg, rigging.WithSources())

// Output:
// server.host: "0.0.0.0" (source: file:config.yaml)
// server.port: 8080 (source: default)
// database.host: "localhost" (source: file:config.yaml)
// database.port: 5432 (source: default)
// database.password: "***redacted***" (source: env:APP_DATABASE__PASSWORD)
```

### Optional Fields

Distinguish "not set" from "zero value":

```go
type Config struct {
    Timeout rigging.Optional[time.Duration]
}

cfg, _ := loader.Load(ctx)

if timeout, ok := cfg.Timeout.Get(); ok {
    // Value was explicitly set
    client.SetTimeout(timeout)
} else {
    // Value not set, use computed default
    client.SetTimeout(computeDefault())
}
```

## Core Concepts

### Struct Tags

Control binding and validation with struct tags:

```go
type Config struct {
    // Required field
    ApiKey string `conf:"required"`
    
    // Default value
    Port int `conf:"default:8080"`
    
    // Validation constraints
    MaxConns int `conf:"min:1,max:100"`
    
    // Allowed values
    Environment string `conf:"oneof:prod,staging,dev"`
    
    // Secret (auto-redacted)
    Password string `conf:"secret"`
    
    // Nested with prefix
    Database DatabaseConfig `conf:"prefix:database"`
}
```

### Source Precedence

Sources are processed in order, later sources override earlier ones:

```go
loader := rigging.NewLoader[Config]().
    WithSource(source1).  // Base layer
    WithSource(source2).  // Overrides source1
    WithSource(source3)   // Overrides source2
```

Common pattern:
1. Defaults (hardcoded or file)
2. Environment-specific file (dev.yaml, prod.yaml)
3. Environment variables (for secrets and overrides)

### Validation Order

1. **Type conversion**: String → target type
2. **Tag validation**: required, min, max, oneof
3. **Custom validators**: Your business rules

All errors are collected and returned together.

## Comparison with Other Libraries

| Feature | Rigging | Viper | envconfig |
|---------|---------|-------|-----------|
| Type safety | Compile-time | Runtime | Compile-time |
| Multi-source | Explicit order | Implicit | Env only |
| Provenance | Full tracking | No | No |
| Validation | Tags + custom | Manual | Tags only |
| Secret redaction | Automatic | Manual | Manual |
| Global state | None | Singleton | None |
| Watch/reload | API ready* | Built-in | No |

\* `loader.Watch()` is implemented. Built-in sources return `ErrWatchNotSupported`. Implement `Source.Watch()` in custom sources to enable hot-reload.

## Configuration Sources

### Environment Variables

```go
source := sourceenv.New(sourceenv.Options{
    Prefix:        "APP_",  // Only load APP_* variables
    CaseSensitive: false,   // Prefix matching is case-insensitive (default)
})

// Maps environment variables to struct fields:
// APP_DATABASE__HOST → Database.Host
// APP_SERVER__PORT → Server.Port
```

**Prefix Matching:**
- By default (`CaseSensitive: false`), prefix matching is case-insensitive
- `APP_`, `app_`, and `App_` all match when prefix is `"APP_"`
- Set `CaseSensitive: true` for exact case matching
- Keys are always normalized to lowercase after prefix stripping

```go
// Case-insensitive (default) - matches all variations
sourceenv.New(sourceenv.Options{
    Prefix:        "APP_",
    CaseSensitive: false,
})
// Matches: APP_HOST, app_host, App_Host

// Case-sensitive - exact match only
sourceenv.New(sourceenv.Options{
    Prefix:        "APP_",
    CaseSensitive: true,
})
// Matches: APP_HOST only
// Ignores: app_host, App_Host
```

### Files (YAML/JSON/TOML)

```go
source := sourcefile.New("config.yaml", sourcefile.Options{
    Required: true,  // Error if file missing
})

// Auto-detects format from extension
// Flattens nested structures to dot-separated keys
```

### Custom Sources

Implement the `Source` interface:

```go
type Source interface {
    Load(ctx context.Context) (map[string]any, error)
    Watch(ctx context.Context) (<-chan ChangeEvent, error)
}

// Example: Consul KV store
type ConsulSource struct {
    client *consul.Client
}

func (s *ConsulSource) Load(ctx context.Context) (map[string]any, error) {
    // Fetch from Consul
}
```

## Watch and Reload

The Watch API allows monitoring sources for changes and reloading configuration automatically:

```go
snapshots, errors, err := loader.Watch(ctx)
if err != nil {
    log.Fatal(err)
}

go func() {
    for {
        select {
        case snapshot := <-snapshots:
            log.Printf("Config reloaded: v%d", snapshot.Version)
            applyNewConfig(snapshot.Config)
            
        case err := <-errors:
            log.Printf("Reload failed: %v", err)
            // Previous config still valid
        }
    }
}()
```

**Note**: Built-in sources (sourcefile, sourceenv) return `ErrWatchNotSupported`. To use watch with custom sources:

```go
type MySource struct{}

func (s *MySource) Watch(ctx context.Context) (<-chan rigging.ChangeEvent, error) {
    ch := make(chan rigging.ChangeEvent)
    go func() {
        // Emit events when config changes
        ch <- rigging.ChangeEvent{At: time.Now(), Cause: "updated"}
    }()
    return ch, nil
}
```

## Strict Mode

Catch typos and deprecated keys:

```go
loader.Strict(true)  // Fail on unknown keys (default)
loader.Strict(false) // Ignore unknown keys
```

## Error Handling

All validation errors include field paths and codes:

```go
cfg, err := loader.Load(ctx)
if err != nil {
    if valErr, ok := err.(*rigging.ValidationError); ok {
        for _, fe := range valErr.FieldErrors {
            log.Printf("%s: %s (code: %s)", 
                fe.FieldPath, fe.Message, fe.Code)
        }
    }
}
```

## Configuration Patterns

### Organize with Nested Structs

```go
// Good: Clear schema
type Config struct {
    Server   ServerConfig   `conf:"prefix:server"`
    Database DatabaseConfig `conf:"prefix:database"`
}

// Avoid: Flat structure
type Config struct {
    ServerPort int
    ServerHost string
    DatabaseHost string
    DatabasePort int
    // ... 50 more fields
}
```

### Field Naming

Use idiomatic Go names - keys are automatically normalized:

```go
type Config struct {
    MaxConnections int           // Matches: maxconnections
    APIKey         string         // Matches: apikey
    RetryTimeout   time.Duration  // Matches: retrytimeout
}
```

**Key normalization**: All keys are fully lowercased for matching. Field name `MaxConnections` automatically matches config key `maxconnections`, `MAXCONNECTIONS`, or `max_connections` (after normalization). Use `name:` tag only when you need a different key path:

```go
type Config struct {
    MaxConnections int `conf:"name:max.connections"` // Matches: max.connections
}

### Handling Secrets

```go
// Good: Secrets marked
type Config struct {
    Password string `conf:"secret"`
    ApiKey   string `conf:"secret"`
}
```

### Startup Validation

```go
// ✓ Good: Validate at startup
func main() {
    cfg, err := loader.Load(ctx)
    if err != nil {
        log.Fatal(err)  // Fail fast
    }
    
    // Config guaranteed valid
    startServer(cfg)
}
```

### Production Logging

```go
// ✓ Good: Log configuration sources at startup
cfg, _ := loader.Load(ctx)
prov, _ := rigging.GetProvenance(cfg)

log.Info("Configuration loaded:")
for _, field := range prov.Fields {
    if !field.Secret {
        log.Infof("  %s from %s", field.FieldPath, field.SourceName)
    }
}
```

## API Reference

### Core Types

#### Loader[T]

The main entry point for loading configuration.

```go
loader := rigging.NewLoader[Config]()
```

**Methods:**

- `WithSource(src Source) *Loader[T]` - Add a configuration source
- `WithValidator(v Validator[T]) *Loader[T]` - Add a custom validator
- `Strict(strict bool) *Loader[T]` - Enable/disable strict mode
- `Load(ctx context.Context) (*T, error)` - Load and validate configuration
- `Watch(ctx context.Context) (<-chan Snapshot[T], <-chan error, error)` - Watch for changes

#### Source

Interface for configuration sources.

```go
type Source interface {
    Load(ctx context.Context) (map[string]any, error)
    Watch(ctx context.Context) (<-chan ChangeEvent, error)
}
```

**Built-in sources:**
- `sourcefile.New(path string, opts sourcefile.Options)` - YAML/JSON/TOML files
- `sourceenv.New(opts sourceenv.Options)` - Environment variables

#### Optional[T]

Distinguish "not set" from "zero value".

```go
type Optional[T any] struct {
    Value T
    Set   bool
}
```

**Methods:**
- `Get() (T, bool)` - Returns value and whether it was set
- `OrDefault(defaultVal T) T` - Returns value or default

#### Validator[T]

Interface for custom validation.

```go
type Validator[T any] interface {
    Validate(ctx context.Context, cfg *T) error
}
```

**Helper:**
- `ValidatorFunc[T](func(ctx context.Context, cfg *T) error)` - Function adapter

### Observability

#### GetProvenance

Track where configuration values came from.

```go
func GetProvenance[T any](cfg *T) (*Provenance, bool)
```

Returns provenance metadata with field-level source information.

```go
type Provenance struct {
    Fields []FieldProvenance
}

type FieldProvenance struct {
    FieldPath  string // e.g., "Database.Host"
    KeyPath    string // e.g., "database.host"
    SourceName string // e.g., "file:config.yaml"
    Secret     bool   // true if marked as secret
}
```

#### DumpEffective

Safely dump configuration with secret redaction.

```go
func DumpEffective[T any](w io.Writer, cfg *T, opts ...DumpOption) error
```

**Options:**
- `WithSources()` - Include source attribution
- `AsJSON()` - Output as JSON instead of text
- `WithIndent(indent string)` - Set JSON indentation

**Examples:**

```go
// Text format
rigging.DumpEffective(os.Stdout, cfg)

// With source attribution
rigging.DumpEffective(os.Stdout, cfg, rigging.WithSources())

// JSON format
rigging.DumpEffective(os.Stdout, cfg, rigging.AsJSON())

// JSON with custom indent
rigging.DumpEffective(os.Stdout, cfg, 
    rigging.AsJSON(), 
    rigging.WithIndent("    "))
```

### Error Types

#### ValidationError

Aggregates all validation failures.

```go
type ValidationError struct {
    FieldErrors []FieldError
}
```

#### FieldError

Represents a single field validation failure.

```go
type FieldError struct {
    FieldPath string // e.g., "Database.Port"
    Code      string // e.g., "required", "min", "max"
    Message   string // Human-readable error
}
```

**Standard error codes:**
- `required` - Field is required but not provided
- `min` - Value below minimum
- `max` - Value exceeds maximum
- `oneof` - Value not in allowed set
- `invalid_type` - Type conversion failed
- `unknown_key` - Configuration key doesn't map to any field (strict mode)

### Struct Tags

Configure binding and validation with the `conf` tag:

| Tag | Description | Example |
|-----|-------------|---------|
| `required` | Field must have a value | `conf:"required"` |
| `default:X` | Default value if not provided | `conf:"default:8080"` |
| `min:N` | Minimum value (numeric) or length (string) | `conf:"min:1024"` |
| `max:N` | Maximum value (numeric) or length (string) | `conf:"max:65535"` |
| `oneof:a,b,c` | Value must be one of the options | `conf:"oneof:prod,staging,dev"` |
| `secret` | Mark field for redaction | `conf:"secret"` |
| `prefix:path` | Prefix for nested struct fields | `conf:"prefix:database"` |
| `name:path` | Override derived key path | `conf:"name:custom.path"` |

**Combining tags:**

```go
type Config struct {
    Port     int    `conf:"default:8080,min:1024,max:65535"`
    Env      string `conf:"required,oneof:prod,staging,dev"`
    Password string `conf:"required,secret"`
}
```

**Tag precedence:**

- `name:` overrides all key derivation (ignores `prefix:` and field name)
- `prefix:` applies to nested struct fields
- Without `name:`, keys are derived from field names (lowercased first letter)

```go
type Config struct {
    Database struct {
        Host string              // Key: database.host (prefix applied)
        Port int `conf:"name:db.port"` // Key: db.port (name overrides prefix)
    } `conf:"prefix:database"`
}
```

### Watch and Reload

#### Snapshot[T]

Represents a loaded configuration with metadata.

```go
type Snapshot[T any] struct {
    Config   *T        // The loaded configuration
    Version  int64     // Incremented on each reload
    LoadedAt time.Time // When loaded
    Source   string    // What triggered the load
}
```

#### ChangeEvent

Notification of configuration change.

```go
type ChangeEvent struct {
    At    time.Time // When the change occurred
    Cause string    // Description of the change
}
```

## Examples

See the [examples](examples/) directory for complete working examples:

- [Basic Example](examples/basic/) - Complete tutorial with all features

## Documentation

- [API Reference](https://pkg.go.dev/github.com/Azhovan/rigging)
- [Design Document](.kiro/specs/go-config-library/design.md)
- [Contributing Guide](CONTRIBUTING.md)

## FAQ

**Q: Why not just use Viper?**  
A: Viper uses `map[string]interface{}` which loses type safety. Rigging gives you compile-time guarantees and provenance tracking.

**Q: Can I use this with existing config files?**  
A: Yes! Rigging supports YAML, JSON, and TOML files. Just define a struct that matches your file structure.

**Q: How do I handle secrets?**  
A: Mark fields with `secret` tag and load from environment variables. Secrets are automatically redacted in dumps.

**Q: Does Rigging support hot-reload?**  
A: The `loader.Watch()` API is implemented and ready to use. However, built-in sources (sourcefile, sourceenv) don't emit change events yet. You can implement custom sources with watch support, or wait for file watching (planned via fsnotify).

**Q: Is this production-ready?**  
A: Rigging is designed for production use with comprehensive error handling, validation, and observability. The API is currently v0.x - expect minor breaking changes as we incorporate feedback from early adopters.

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Contributing

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

---

**Built with ❤️ for Go services that need reliable configuration.**
