# Basic Example

Hands-on demonstration of Rigging's core features with a complete working example.

## Quick Start

```bash
# Run with default config.yaml
go run main.go

# Override with environment variables
export APP_DATABASE__PASSWORD=secret123
export APP_SERVER__PORT=9090
go run main.go
```

## Example Scenarios

```bash
# Switch to production environment
export APP_ENVIRONMENT=production
export APP_DATABASE__HOST=prod-db.example.com
go run main.go

# Enable feature flags
export APP_FEATURES__ENABLE_METRICS=true
export APP_FEATURES__RATELIMIT=5000
go run main.go
```

## What This Example Shows

- Multi-source loading (YAML + environment variables)
- Tag-based and custom validation
- Provenance tracking output
- Secret redaction in config dumps

See the [main README](../../README.md) for complete documentation.
