# Configuration Sources

Rigging supports multiple sources with explicit precedence.

```go
loader := rigging.NewLoader[Config]().
    WithSource(sourcefile.New("defaults.yaml", sourcefile.Options{})).
    WithSource(sourcefile.New("config.yaml", sourcefile.Options{})).
    WithSource(sourceenv.New(sourceenv.Options{Prefix: "APP_"}))
// Later sources override earlier ones.
```

## Built-in Sources

## Environment Variables (`sourceenv`)

```go
source := sourceenv.New(sourceenv.Options{
    Prefix:        "APP_", // load APP_* variables only
    CaseSensitive: false,  // default: case-insensitive prefix matching
})
```

Examples:
- `APP_DATABASE__HOST` -> `database.host`
- `APP_SERVER__PORT` -> `server.port`
- `APP_API_KEY` -> `api_key`

Prefix behavior:
- `CaseSensitive: false` (default): `APP_`, `app_`, `App_` all match
- `CaseSensitive: true`: exact prefix match only

## Files (`sourcefile`)

```go
source := sourcefile.New("config.yaml", sourcefile.Options{
    Required: true,
})
```

- Supports YAML/JSON/TOML (extension auto-detected unless `Format` is set)
- Nested objects are flattened to dot paths (`database.host`)
- Missing file returns empty map unless `Required: true`

## Key Normalization by Source

| Source | Example input | Normalized key |
|---|---|---|
| Env | `APP_DATABASE__HOST` | `database.host` |
| Env | `APP_MAX_CONNECTIONS` | `max_connections` |
| Env | `APP_API_KEY` | `api_key` |
| File | `database.host` | `database.host` |
| File | `max_connections` | `max_connections` |

Important:
- Env source strips the configured prefix first, then preserves single underscores and converts `__` to `.`.
- File source does not rewrite separators; it flattens and lowercases keys.
- If your file keys are snake_case, use `name:` tags to match them.

```go
type Config struct {
    MaxConnections int `conf:"name:max_connections"`
}
```

## Custom Sources

Implement `Source`:

```go
type Source interface {
    Load(ctx context.Context) (map[string]any, error)
    Watch(ctx context.Context) (<-chan ChangeEvent, error)
    Name() string
}
```

For stronger provenance, optionally implement `SourceWithKeys`:

```go
type SourceWithKeys interface {
    Source
    LoadWithKeys(ctx context.Context) (data map[string]any, originalKeys map[string]string, err error)
}
```

`originalKeys` lets Rigging report exact source keys in provenance (for example, full env var names).

## Watch and Reload

```go
snapshots, errors, err := loader.Watch(ctx)
if err != nil {
    log.Fatal(err)
}

for {
    select {
    case snapshot := <-snapshots:
        log.Printf("Config reloaded: v%d (%s)", snapshot.Version, snapshot.Source)
        applyNewConfig(snapshot.Config)
    case err := <-errors:
        log.Printf("Reload failed: %v", err)
    }
}
```

Note:
- Built-in `sourcefile` and `sourceenv` return `ErrWatchNotSupported`.
- Custom sources can implement watch support today.
