# Basic Configuration Example

This example demonstrates the core features of the `rigging` configuration library through a realistic application configuration scenario.

## Overview

This example shows how to build a production-ready configuration system with:
- Type-safe configuration structs with compile-time guarantees
- Deterministic multi-source precedence (files → environment variables)
- Comprehensive validation (structural + business logic)
- Security-first design (automatic secret redaction, provenance tracking)
- Zero reflection in hot paths (validation happens once at startup)

### When to Use This Library

✅ **Good fit:**
- Production services where configuration errors are expensive
- Teams that value type safety and compile-time checks
- Applications with complex validation requirements
- Services that need audit trails (provenance tracking)
- Multi-environment deployments (dev/staging/prod)

❌ **Not a good fit:**
- Highly dynamic configuration that changes shape at runtime
- Configuration from untrusted sources (use dedicated validation libraries)
- Applications that need hot-reload of configuration structure (not just values)
- Prototypes where schema is rapidly evolving

## What This Example Demonstrates

1. **Strongly-Typed Configuration**: Define configuration as Go structs with compile-time type safety
2. **Multi-Source Loading**: Load configuration from YAML files and environment variables with deterministic precedence
3. **Tag-Based Validation**: Use struct tags for field validation (required, min, max, oneof, default)
4. **Custom Validators**: Implement cross-field validation logic
5. **Secret Redaction**: Automatically redact sensitive fields in dumps and logs
6. **Provenance Tracking**: Track where each configuration value came from
7. **Configuration Dumping**: Export effective configuration with source attribution

## Configuration Structure

The example demonstrates **nested configuration structs** for better organization and maintainability:

```go
type AppConfig struct {
    Environment string         `conf:"default:development,oneof:development,staging,production"`
    Database    DatabaseConfig `conf:"prefix:database"`
    Server      ServerConfig   `conf:"prefix:server"`
    Logging     LoggingConfig  `conf:"prefix:logging"`
    Features    FeaturesConfig `conf:"prefix:features"`
}
```

Each nested struct represents a logical grouping:

- **DatabaseConfig**: Connection parameters, pool settings, SSL configuration
- **ServerConfig**: HTTP server host, port, and timeout configurations
- **LoggingConfig**: Log level, format, and output destination
- **FeaturesConfig**: Optional feature toggles using `Optional[T]` type

The `prefix:` tag maps each nested struct to a configuration namespace (e.g., `prefix:database` means all fields in `DatabaseConfig` are prefixed with `database.`)

## Running the Example

### Prerequisites

- Go 1.21+ (requires generics support)
- No external dependencies beyond standard library and `gopkg.in/yaml.v3`

### Basic Run

Run the example with the provided YAML configuration:

```bash
cd examples/basic
go run main.go
```

This will load configuration from `config.yaml` and display:
- The loaded configuration values
- Provenance information (which source provided each value)
- Effective configuration dumps in both text and JSON formats

### Override with Environment Variables

The example demonstrates configuration layering. Environment variables override file values:

```bash
# Set some environment variables (note the APP_ prefix and __ for nesting)
export APP_ENVIRONMENT=production
export APP_DATABASE__PASSWORD=super_secret_password
export APP_DATABASE__HOST=prod-db.example.com
export APP_SERVER__PORT=9090
export APP_FEATURES__ENABLEMETRICS=true
export APP_FEATURES__RATELIMIT=5000

# Run the example
go run main.go
```

You'll see that:
- Environment variables override the YAML file values
- Provenance tracking shows which values came from env vs file
- The password is redacted in all output

### Test Validation Errors

Try setting invalid values to see validation in action:

```bash
# Invalid environment (not in oneof list)
export APP_ENVIRONMENT=testing
go run main.go
# Error: value "testing" must be one of: development, staging, production

# Port out of range
export APP_SERVER__PORT=100
go run main.go
# Error: value 100 is below minimum 1024

# Production with insecure settings (custom validator)
export APP_ENVIRONMENT=production
export APP_DATABASE__HOST=localhost
export APP_DATABASE__SSLMODE=disable
go run main.go
# Error: production environment cannot use localhost database
# Error: production environment must use SSL for database connections
```

### Run Without Config File

The example makes the YAML file optional. You can run with only environment variables:

```bash
# Remove or rename the config file
mv config.yaml config.yaml.bak

# Set required fields via environment
export APP_DATABASE__HOST=db.example.com
export APP_DATABASE__NAME=myapp
export APP_DATABASE__USER=appuser
export APP_DATABASE__PASSWORD=secret

# Run the example
go run main.go
```

Note: All required fields must be provided either via file or environment variables.

## Key Features Demonstrated

### 1. Nested Structs with Prefix Tag

```go
type AppConfig struct {
    Database DatabaseConfig `conf:"prefix:database"`
    Server   ServerConfig   `conf:"prefix:server"`
}

type DatabaseConfig struct {
    Host     string `conf:"required"`
    Port     int    `conf:"default:5432"`
    Password string `conf:"secret,required"`
}
```

The `prefix:` tag creates a namespace for nested configuration:
- `Database.Host` maps to configuration key `database.host`
- `Database.Port` maps to configuration key `database.port`
- In YAML: nest under `database:` section
- In env vars: use `APP_DATABASE__HOST`, `APP_DATABASE__PORT`

This approach keeps your configuration organized and type-safe.

### 2. Struct Tag Validation

```go
type DatabaseConfig struct {
    Host     string `conf:"required"`
    Port     int    `conf:"default:5432,min:1024,max:65535"`
    Password string `conf:"secret,required"`
    Sslmode  string `conf:"default:disable,oneof:disable,require,verify-ca,verify-full"`
}
```

Available validation tags:
- `required`: Field must have a value
- `default:X`: Use X if no source provides a value
- `min:N, max:M`: Numeric range validation
- `oneof:a,b,c`: Value must be one of the listed options
- `secret`: Mark field for automatic redaction
- `name:key`: Override the derived field name (use lowercase for consistency)

### 3. Multi-Source Configuration

```go
loader := rigging.NewLoader[AppConfig]().
    WithSource(sourcefile.New("config.yaml", sourcefile.Options{
        Required: false,
    })).
    WithSource(sourceenv.New(sourceenv.Options{
        Prefix: "APP_",
    }))
```

Sources are processed in order with later sources taking precedence:
1. YAML file provides base configuration
2. Environment variables override file values
3. Later sources always win (deterministic precedence)

This layering approach is perfect for:
- Development: Use YAML defaults
- Production: Override sensitive values via environment variables
- Testing: Inject test-specific configuration

### 4. Custom Cross-Field Validation

```go
func customValidator(ctx context.Context, cfg *AppConfig) error {
    if cfg.Environment == "production" && cfg.Database.Host == "localhost" {
        return &rigging.ValidationError{
            FieldErrors: []rigging.FieldError{{
                FieldPath: "Database.Host",
                Code:      "invalid_prod_host",
                Message:   "production environment cannot use localhost database",
            }},
        }
    }
    return nil
}
```

Custom validators enable:
- **Cross-field validation**: Check relationships between fields
- **Business logic validation**: Enforce domain-specific rules
- **Environment-specific rules**: Different validation for dev/staging/prod
- **Complex constraints**: Validate combinations that struct tags can't express

Note: Use the full field path (e.g., `Database.Host`) when referencing nested struct fields.

### 5. Optional Fields

```go
type FeaturesConfig struct {
    Enablemetrics rigging.Optional[bool]
    Enabletracing rigging.Optional[bool]
    Ratelimit     rigging.Optional[int] `conf:"min:1"`
}

// Check if value was explicitly set
if metrics, ok := cfg.Features.Enablemetrics.Get(); ok {
    fmt.Printf("Metrics enabled: %v\n", metrics)
} else {
    fmt.Println("Metrics setting not provided")
}
```

`Optional[T]` distinguishes between "not set" and "set to zero value":
- Perfect for feature flags that should be explicitly enabled/disabled
- Useful when you need to know if a user provided a value
- Avoids ambiguity with zero values (e.g., `false` vs "not set")

### 6. Provenance Tracking

```go
if prov, ok := rigging.GetProvenance(cfg); ok {
    for _, field := range prov.Fields {
        fmt.Printf("%s came from %s\n", field.FieldPath, field.SourceName)
    }
}
```

Provenance tracking shows:
- Which source provided each value (file, env, default)
- The original key path used
- Whether the field is marked as secret

Example output:
```
Database.Host = database.host (from source-0)
Database.Password = database.password (from source-1) [SECRET]
Database.Maxconnections = database.maxconnections (from source-0)
```

This is invaluable for:
- Debugging configuration issues
- Auditing where sensitive values come from
- Understanding configuration precedence in action

### 7. Configuration Dumping

```go
// Text format with source attribution
rigging.DumpEffective(os.Stdout, cfg, rigging.WithSources())

// JSON format with secret redaction
rigging.DumpEffective(os.Stdout, cfg, rigging.AsJSON())
```

Safely dump configuration for debugging without exposing secrets:
- Secrets are automatically redacted (fields marked with `secret` tag)
- Source attribution shows where each value came from
- JSON format for structured logging or external tools
- Text format for human-readable output

Perfect for:
- Startup logging to verify configuration
- Debugging configuration issues in production
- Generating configuration documentation

## Field Naming Conventions

### Struct Field Names

For fields with multi-word names, use **all lowercase** to ensure consistent key matching:

```go
type DatabaseConfig struct {
    Maxconnections int    // Derives to "maxconnections"
    Sslmode        string // Derives to "sslmode"
    Connecttimeout time.Duration // Derives to "connecttimeout"
}
```

**Why lowercase?** The library normalizes all configuration keys to lowercase for case-insensitive matching. Using lowercase field names ensures the derived key matches exactly.

**Alternative:** Use the `name:` tag to explicitly specify the key:
```go
MaxConnections int `conf:"name:maxconnections"`
```

### YAML Key Names

Match your struct field names (in lowercase):

```yaml
database:
  host: localhost
  port: 5432
  maxconnections: 25  # Matches Maxconnections field
  sslmode: require    # Matches Sslmode field
```

### Environment Variable Naming

The example uses `APP_` prefix. Environment variable names map to struct fields:

| Struct Field | Configuration Key | Environment Variable |
|--------------|-------------------|---------------------|
| `Environment` | `environment` | `APP_ENVIRONMENT` |
| `Database.Host` | `database.host` | `APP_DATABASE__HOST` |
| `Database.Port` | `database.port` | `APP_DATABASE__PORT` |
| `Database.Maxconnections` | `database.maxconnections` | `APP_DATABASE__MAXCONNECTIONS` |
| `Server.Port` | `server.port` | `APP_SERVER__PORT` |
| `Features.Enablemetrics` | `features.enablemetrics` | `APP_FEATURES__ENABLEMETRICS` |

**Key Points:**
- Use double underscore (`__`) to represent nesting levels (dots in config keys)
- Environment variables are case-insensitive on most systems
- The prefix (`APP_`) is configurable via `sourceenv.Options`

## Expected Output

When you run the example, you should see output similar to:

```
=== Configuration Library Example ===

Loading configuration from:
  1. config.yaml (if present)
  2. Environment variables (APP_* prefix)

✓ Configuration loaded successfully!

=== Loaded Configuration ===

Environment: staging

Database:
  Host: db.staging.example.com
  Port: 5432
  Name: myapp_staging
  User: app_user
  Password: [REDACTED]
  Max Connections: 25
  SSL Mode: require
  Connect Timeout: 5s

Server:
  Host: 0.0.0.0
  Port: 8080
  Read Timeout: 30s
  Write Timeout: 30s
  Shutdown Timeout: 10s

Logging:
  Level: info
  Format: json
  Output: stdout

Features:
  Enable Metrics: true
  Enable Tracing: false
  Rate Limit: 1000

=== Configuration Provenance ===

Source information for each field:
  Environment = environment (from file:config.yaml)
  Database.Host = database.host (from file:config.yaml)
  Database.Port = database.port (from file:config.yaml)
  Database.Name = database.name (from file:config.yaml)
  Database.User = database.user (from file:config.yaml)
  Database.Password = database.password (from env:APP_) [SECRET]
  ...

=== Effective Configuration Dump ===

Text format with source attribution:
---
environment: "staging" (source: file:config.yaml)
database.host: "db.staging.example.com" (source: file:config.yaml)
database.port: 5432 (source: file:config.yaml)
database.password: "***redacted***" (source: env:APP_)
...
---

JSON format (secrets redacted):
---
{
  "environment": "staging",
  "database": {
    "host": "db.staging.example.com",
    "port": 5432,
    "password": "***redacted***",
    ...
  }
}
---
```

## Design Decisions & Trade-offs

### Why Nested Structs?

The library uses nested structs with `prefix:` tags rather than flat structures or map-based configuration:

**Advantages:**
- **Type safety**: Compiler catches typos and type mismatches
- **IDE support**: Autocomplete, refactoring, go-to-definition all work
- **Documentation**: Struct fields are self-documenting with types and tags
- **Performance**: No runtime type assertions or map lookups in application code
- **Testability**: Easy to create test fixtures with struct literals

**Trade-offs:**
- **Verbosity**: More code than map[string]interface{} approaches
- **Rigidity**: Schema changes require code changes (this is intentional)
- **Learning curve**: Developers must understand struct tags

**When to use this approach:**
- Production services where configuration errors are costly
- Teams that value compile-time safety over runtime flexibility
- Applications with complex validation requirements

**When to consider alternatives:**
- Highly dynamic configuration that changes shape at runtime
- Prototypes where schema is still evolving rapidly
- Configuration that comes from untrusted sources (use validation libraries)

### Field Naming: Why Lowercase?

The library normalizes all keys to lowercase for case-insensitive matching. This design choice:

**Rationale:**
- Environment variables are case-insensitive on Windows
- YAML/JSON keys are case-sensitive, leading to confusion
- Lowercase normalization provides consistent behavior across sources

**Implications:**
- Struct fields with multi-word names should be lowercase (e.g., `Maxconnections`)
- Or use explicit `name:` tags (e.g., `MaxConnections int \`conf:"name:maxconnections"\``)
- YAML keys should match the normalized form

**Alternative considered:** CamelCase preservation was rejected because it creates platform-specific behavior.

## Best Practices

### Organizing Configuration with Nested Structs

Group related configuration into separate structs:

```go
// ✓ Good: Organized by domain
type AppConfig struct {
    Database DatabaseConfig `conf:"prefix:database"`
    Cache    CacheConfig    `conf:"prefix:cache"`
    API      APIConfig      `conf:"prefix:api"`
}

// ✗ Avoid: Flat structure with many fields
type AppConfig struct {
    DatabaseHost string
    DatabasePort int
    CacheHost    string
    CachePort    int
    APITimeout   time.Duration
    // ... 50 more fields
}
```

Benefits of nested structs:
- **Better organization**: Related fields are grouped together
- **Easier testing**: Mock individual config sections
- **Clearer ownership**: Each struct can be owned by a different team/module
- **Type safety**: Compile-time checks for field access
- **Reusability**: Share config structs across services

### Field Naming Tips

1. **Use lowercase for multi-word fields** to avoid key matching issues
2. **Be consistent** with your naming convention across all config structs
3. **Use `name:` tag** when you need a specific key name that differs from the field
4. **Document complex fields** with comments explaining valid values
5. **Avoid deeply nested structs** (>3 levels) - they become hard to override via env vars

### Security Considerations

1. **Always mark sensitive fields with `secret` tag** - prevents accidental logging
2. **Use environment variables for secrets in production** - never commit secrets to YAML
3. **Enable strict mode in production** - catches typos that could disable security features
4. **Validate early** - fail fast at startup rather than discovering bad config at runtime
5. **Log effective configuration at startup** - but ensure secrets are redacted

### Performance Characteristics

- **Startup cost**: O(n) where n = number of fields (one-time reflection cost)
- **Runtime cost**: Zero - configuration is a plain Go struct after loading
- **Memory**: One allocation per config struct + provenance metadata (optional)
- **Validation**: Happens once at load time, not on every access

For a typical service with ~100 config fields:
- Load time: <1ms
- Memory overhead: ~10KB (including provenance)
- No GC pressure after initial load

## Migration Guide

### From Environment Variables Only

If you're currently using `os.Getenv()` directly:

```go
// Before
dbHost := os.Getenv("DATABASE_HOST")
if dbHost == "" {
    dbHost = "localhost"
}

// After
type Config struct {
    Database struct {
        Host string `conf:"default:localhost"`
    } `conf:"prefix:database"`
}
```

Benefits: Type safety, validation, defaults in one place.

### From Viper or Similar

If you're using Viper or similar map-based libraries:

```go
// Before
viper.SetDefault("database.host", "localhost")
host := viper.GetString("database.host") // Runtime type assertion

// After
type Config struct {
    Database struct {
        Host string `conf:"default:localhost"`
    } `conf:"prefix:database"`
}
// Compile-time type safety, no runtime assertions
```

Benefits: Compile-time safety, better IDE support, explicit schema.

### Incremental Adoption

You can adopt this library incrementally:

1. **Start with a subset**: Migrate one config section at a time
2. **Keep existing code working**: Use both systems during transition
3. **Validate equivalence**: Compare old vs new config in tests
4. **Remove old system**: Once all config is migrated

## Common Pitfalls

### 1. Field Name Mismatches

```go
// ❌ Wrong: Field name doesn't match YAML key
type Config struct {
    MaxConnections int // Derives to "maxConnections"
}
// YAML: max_connections: 10  (won't match!)

// ✅ Correct: Use lowercase or explicit name
type Config struct {
    Maxconnections int // Derives to "maxconnections"
    // OR
    MaxConnections int `conf:"name:maxconnections"`
}
```

### 2. Forgetting Required Fields

```go
// ❌ Wrong: No default, not marked required
type Config struct {
    APIKey string
}
// Silently uses empty string if not provided!

// ✅ Correct: Mark as required or provide default
type Config struct {
    APIKey string `conf:"required"`
    // OR
    APIKey string `conf:"default:dev-key-123"`
}
```

### 3. Secrets in Logs

```go
// ❌ Wrong: Password will appear in dumps
type Config struct {
    Password string
}

// ✅ Correct: Mark as secret
type Config struct {
    Password string `conf:"secret,required"`
}
```

### 4. Validation Order

Custom validators run AFTER struct tag validation. Design accordingly:

```go
func validator(ctx context.Context, cfg *Config) error {
    // This runs AFTER required/min/max/oneof validation
    // So you can assume those constraints are already satisfied
    if cfg.Environment == "prod" && cfg.Database.Host == "localhost" {
        return errors.New("prod can't use localhost")
    }
    return nil
}
```

## Next Steps

After exploring this example:

1. **Adapt to your needs**: Modify the configuration struct for your application
2. **Add validation**: Implement custom validators for your business logic
3. **Test thoroughly**: Write tests for config loading and validation
4. **Document your schema**: Add comments to struct fields explaining valid values
5. **Monitor in production**: Log effective config at startup (with secrets redacted)
6. **Consider config reloading**: Explore `Watch()` for dynamic configuration updates

## Quick Reference

### Struct Tags

| Tag | Example | Description |
|-----|---------|-------------|
| `required` | `conf:"required"` | Field must have a value from some source |
| `default:X` | `conf:"default:8080"` | Use X if no source provides a value |
| `min:N` | `conf:"min:1024"` | Minimum value for numeric types |
| `max:N` | `conf:"max:65535"` | Maximum value for numeric types |
| `oneof:a,b,c` | `conf:"oneof:dev,prod"` | Value must be one of the listed options |
| `secret` | `conf:"secret"` | Mark field for automatic redaction |
| `name:key` | `conf:"name:maxconn"` | Override derived field name |
| `prefix:ns` | `conf:"prefix:database"` | Namespace for nested struct |

### Environment Variable Mapping

| Pattern | Example | Maps To |
|---------|---------|---------|
| Top-level field | `APP_ENVIRONMENT` | `Environment` |
| Nested field | `APP_DATABASE__HOST` | `Database.Host` |
| Deep nesting | `APP_A__B__C` | `A.B.C` |

**Rules:**
- Double underscore (`__`) = nesting level separator
- Single underscore in field name = part of the field name
- Prefix is configurable (default: none, example uses `APP_`)

## Further Reading

- [Library Design Document](../../.kiro/specs/go-config-library/design.md) - Architecture and design decisions
- [Requirements Document](../../.kiro/specs/go-config-library/requirements.md) - Feature requirements and use cases
- [API Reference](https://pkg.go.dev/github.com/Azhovan/rigging) - Complete API documentation

## Contributing

Found an issue or have a suggestion? Please open an issue or submit a pull request.

## Troubleshooting

### "Failed to load configuration: validation failed"

Check that all required fields are provided either in the YAML file or via environment variables. The error message will list which fields failed validation.

### "Unknown configuration keys"

If strict mode is enabled (default), the library will reject unknown keys. Either:
- Remove the unknown keys from your configuration sources
- Disable strict mode: `loader.Strict(false)`

### Environment variables not being read

Ensure you're using the correct prefix (`APP_`) and double underscores (`__`) for nested fields:
- ✓ Correct: `APP_DATABASE__HOST`
- ✗ Wrong: `APP_DATABASE_HOST` (single underscore)
- ✗ Wrong: `DATABASE__HOST` (missing prefix)

Also verify the field name matches (case-insensitive):
- ✓ Correct: `APP_DATABASE__MAXCONNECTIONS` or `APP_DATABASE__maxconnections`
- ✗ Wrong: `APP_DATABASE__MAX_CONNECTIONS` (underscores in field name)