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

## Key Mapping + Precedence + `required` Decision Table

| Situation | Result | Validation outcome |
|---|---|---|
| Field `Database.Host` (no tags) + env `APP_DATABASE__HOST=db.internal` | Env key normalizes to `database.host` and binds directly. | `required` passes when tagged, because the key is present. |
| Nested struct `conf:"prefix:database"` + child `Port int` | Child key derives to `database.port`. | Normal conversion and constraints (`min/max/oneof`) apply. |
| Nested struct `conf:"prefix:database"` + child `Port int \`conf:"name:db.port"\`` | `name:` wins over derived/prefix key; binding uses `db.port`. | `required`/constraints evaluate on `db.port`. |
| Same key appears in `defaults.yaml` and env (env source added later) | Later source wins; final value comes from env. | `required` passes as long as one source (or default) supplies the key. |
| Field `Port int \`conf:"default:8080,required"\`` and key absent from all sources | Default injects `8080`. | `required` passes because presence includes defaults. |
| Field `Port int \`conf:"required,min:1"\`` and key present as `0` or wrong type (`"abc"`) | Key is present, but value is invalid for constraints/conversion. | `required` passes; you get `min` (for `0`) or `invalid_type` (for `"abc"`), not an extra `required` error. |

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
ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer cancel()

snapshots, errors, err := loader.Watch(ctx)
if err != nil {
    return fmt.Errorf("start config watch: %w", err)
}

for snapshots != nil || errors != nil {
    select {
    case <-ctx.Done():
        return nil
    case snapshot, ok := <-snapshots:
        if !ok {
            snapshots = nil
            continue
        }
        log.Printf("Config reloaded: v%d (%s)", snapshot.Version, snapshot.Source)
        if err := applyNewConfig(snapshot.Config); err != nil {
            log.Printf("Apply failed (keeping last good config): %v", err)
            continue
        }
    case err, ok := <-errors:
        if !ok {
            errors = nil
            continue
        }
        log.Printf("Reload failed: %v", err)
    }
}
```

Production notes:
- Keep a "last good config" in your app and only swap it after successful `applyNewConfig`.
- Record reload success/failure metrics and include `snapshot.Source` in logs for incident response.
- Use `Load` once at startup before entering watch mode so boot fails fast on invalid config.

Note:
- Built-in `sourcefile` and `sourceenv` return `ErrWatchNotSupported`.
- Custom sources can implement watch support today.
