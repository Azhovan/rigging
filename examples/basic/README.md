# Basic Example

Hands-on demonstration of Rigging's core features with a complete working example.

## Quick Start

```bash
# Run with default config.yaml
go run main.go

# Override with environment variables
export APP_DATABASE__PASSWORD=secret123
export APP_SERVER__PORT=9090
export APP_CLICKHOUSE__PRIMARY__HOST=ch-primary-override.internal
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
export APP_FEATURES__RATE_LIMIT=5000
go run main.go

# Override a named ClickHouse cluster entry from YAML map config
export APP_CLICKHOUSE__ANALYTICS__PORT=9010
go run main.go
```

## What This Example Shows

- Multi-source loading (YAML + environment variables)
- Tag-based and custom validation
- Provenance tracking output
- Secret redaction in config dumps
- List/map of structured objects loaded from YAML (`clickhouse_nodes`, `clickhouse`)

See the [main README](../../README.md) for complete documentation.
