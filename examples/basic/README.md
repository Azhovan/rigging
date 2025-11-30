# Basic Example

Demonstrates core Rigging features:
- Multi-source loading (YAML + env vars)
- Struct tag validation
- Custom validators
- Secret redaction
- Provenance tracking

## Run

```bash
go run main.go

# Override with environment variables
export APP_DATABASE__PASSWORD=secret
export APP_ENVIRONMENT=production
go run main.go
```

## Key Points

**Struct Tags**:
```go
type Config struct {
    Port     int    `conf:"default:8080,min:1024"`
    Password string `conf:"required,secret"`
}
```

**Environment Variables**: Use `__` for nesting
- `APP_DATABASE__HOST` → `Database.Host`
- `APP_SERVER__PORT` → `Server.Port`

**YAML Keys**: Match struct fields (lowercase)
```yaml
database:
  host: localhost
  port: 5432
```

See [main README](../../README.md) for full documentation.
