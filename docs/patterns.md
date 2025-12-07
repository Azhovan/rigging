# Configuration Patterns

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

## Best Practices

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
```

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
