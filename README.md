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
        Port     int    `conf:"default:5432,min:1024,max:65535"`
        Password string `conf:"required,secret"`
    } `conf:"prefix:database"`
}

cfg, err := rigging.NewLoader[Config]().
    WithSource(sourcefile.New("config.yaml", sourcefile.Options{})).
    WithSource(sourceenv.New(sourceenv.Options{Prefix: "APP_"})).
    Load(ctx)
// Type-safe: cfg.Database.Port is an int, guaranteed by compiler
// Observable: Track where Database.Host came from (file vs env)
// Policy-driven: Port validated to be within 1024-65535 range
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

## Comparison with Other Libraries

| Feature | Rigging | Viper | envconfig |
|---------|---------|-------|-----------|
| Type safety | Compile-time | Runtime | Compile-time |
| Multi-source | Explicit order | Implicit | Env only |
| Provenance | Full tracking | No | No |
| Validation | Tags + custom | Manual | Tags only |
| Secret redaction | Automatic | Manual | Manual |
| Global state | None | Singleton | None |
| Watch/reload | Custom sources | Built-in | No |

\* Rigging provides the `Watch()` API for custom configuration sources. Built-in file and environment sources don't support watching yet.

## Installation

```bash
# Core library (zero dependencies)
go get github.com/Azhovan/rigging

# File support (YAML/JSON/TOML)
go get github.com/Azhovan/rigging/sourcefile

# Environment variables
go get github.com/Azhovan/rigging/sourceenv
```

## Documentation

- **[Quick Start Guide](docs/quick-start.md)** - Get started with installation, basic usage, validation, and observability
- **[Configuration Sources](docs/configuration-sources.md)** - Learn about environment variables, file sources, custom sources, and watch/reload
- **[API Reference](docs/api-reference.md)** - Complete API documentation for all types, methods, and struct tags
- **[Configuration Patterns](docs/patterns.md)** - Best practices and design patterns for organizing your configuration
- **[Architecture Overview](docs/architecture.md)** - Understand Rigging's internals, core components, and data models
- **[Examples](examples/)** - Complete working examples

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

## Reference

- [Go Package Documentation](https://pkg.go.dev/github.com/Azhovan/rigging)
- [Contributing Guide](CONTRIBUTING.md)

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Contributing

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.