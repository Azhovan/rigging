# Quick Start

This guide gets you from zero to a successful load with validation, provenance, and safe dumps.

## 1. Install

```bash
go get github.com/Azhovan/rigging
go get github.com/Azhovan/rigging/sourcefile
go get github.com/Azhovan/rigging/sourceenv
```

## 2. Define a Typed Schema

```go
type Config struct {
    Server struct {
        Host string `conf:"default:0.0.0.0"`
        Port int    `conf:"default:8080,min:1024,max:65535"`
    } `conf:"prefix:server"`

    Database struct {
        Host     string `conf:"required"`
        Port     int    `conf:"default:5432"`
        Password string `conf:"required,secret"`
    } `conf:"prefix:database"`
}
```

## 3. Provide Inputs

Create `config.yaml`:

```yaml
server:
  host: 0.0.0.0
database:
  host: localhost
```

Set required secret from env:

```bash
export APP_DATABASE__PASSWORD=secret123
```

## 4. Load Configuration

```go
loader := rigging.NewLoader[Config]().
    WithSource(sourcefile.New("config.yaml", sourcefile.Options{})).
    WithSource(sourceenv.New(sourceenv.Options{Prefix: "APP_"}))

cfg, err := loader.Load(context.Background())
if err != nil {
    log.Fatal(err)
}

log.Printf("server at %s:%d", cfg.Server.Host, cfg.Server.Port)
```

## 5. Observe and Debug Safely

### Provenance

```go
prov, ok := rigging.GetProvenance(cfg)
if ok {
    for _, field := range prov.Fields {
        log.Printf("%s <- %s", field.FieldPath, field.SourceName)
    }
}
```

### Redacted dump

```go
rigging.DumpEffective(os.Stdout, cfg, rigging.WithSources())
```

Secrets tagged with `conf:"secret"` are redacted.

## 6. Key Mapping Rules (Important)

Rigging matches using normalized lowercase key paths.

- Field `MaxConnections` maps to key `max_connections`
- Field `APIKey` maps to key `api_key`
- Nested fields use dots (`database.host`)
- `prefix:` prepends nested paths
- `name:` overrides derived key paths entirely

Environment source normalization:
- `APP_DATABASE__HOST` -> `database.host`
- `APP_API_KEY` -> `api_key`
- single `_` is preserved; double `__` becomes `.`

File source behavior:
- keys are flattened from file structure and lowercased
- separators are not rewritten (for example, `max_connections` stays `max_connections`)

If your file keys are snake_case, map explicitly:

```go
type Config struct {
    MaxConnections int `conf:"name:max_connections"`
}
```

## 7. Fail-Fast Validation

Tag validation and custom validators run during `Load`:

```go
loader.WithValidator(rigging.ValidatorFunc[Config](func(ctx context.Context, cfg *Config) error {
    if cfg.Server.Port == 5432 {
        return errors.New("server port conflicts with postgres")
    }
    return nil
}))
```

If validation fails, `Load` returns a `*rigging.ValidationError` with all field errors.

## 8. Next Steps

- Source layering, watch/reload: [Configuration Sources](configuration-sources.md)
- Tag strategy and schema patterns: [Configuration Patterns](patterns.md)
- Full API details: [API Reference](api-reference.md)
- Runnable demo: [`examples/basic`](../examples/basic)
