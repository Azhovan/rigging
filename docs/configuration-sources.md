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
    Root:     "root.section",
})
```

- Supports YAML/JSON/TOML (extension auto-detected unless `Format` is set)
- Nested objects are flattened to dot paths (`database.host`)
- `Root` loads from a dot-separated subtree (for example, `root.section`) and flattens keys relative to that subtree
- `SnakeCaseKeys` (opt-in) rewrites flattened keys to underscore snake_case (for example, `pollInterval` -> `poll_interval`, `http.clientTimeout` -> `http_client_timeout`)
- `KeyPrefix` (optional) prepends a prefix to each flattened key after optional `SnakeCaseKeys` conversion (for example, `msa_`)
- Missing file returns empty map unless `Required: true`
- `Root` errors are typed: `sourcefile.ErrRootNotFound` (missing path), `sourcefile.ErrRootNotMap` (non-map path), `sourcefile.ErrInvalidRoot` (invalid syntax such as leading/trailing dot, empty segment, wildcard)

### Load a Subtree with `Root`

Use `Root` when one file contains multiple sections but this loader should bind only one section.

Example file:

```yaml
root:
  section:
    enabled: true
    pollInterval: 5s
  sibling:
    enabled: false
```

Example source setup:

```go
source := sourcefile.New("config.yaml", sourcefile.Options{
    Root: "root.section",
})
```

With this setup, file keys are flattened relative to `root.section`:
- `root.section.enabled` -> `enabled`
- `root.section.pollInterval` -> `pollInterval`

### Optional Subtree Key Adaptation (Opt-In)

```go
source := sourcefile.New("config.yaml", sourcefile.Options{
    Root:          "root.section",
    SnakeCaseKeys: true,
    KeyPrefix:     "msa_",
})
```

With `SnakeCaseKeys: true` and `KeyPrefix: "msa_"`:
- `root.section.pollInterval` -> `msa_poll_interval`

Notes:
- `Root` uses dot-separated path segments only.
- `Required` behavior is unchanged; if the file is missing and `Required` is false, load returns an empty map.
- `Root` is applied inside `sourcefile` before flattening, and key adaptation is disabled by default.

### Nested Collections from File Sources

Rigging can bind nested collection fields from file-source values, including common YAML shapes for slices and dynamic named maps.

Example schema:

```go
type ClickHouseConfig struct {
    Host string
    Port int
}

type Config struct {
    ClickhouseList []ClickHouseConfig
    ClickhouseMap  map[string]ClickHouseConfig
}
```

Example YAML:

```yaml
clickhouse_list:
  - host: ch1
    port: 9000
  - host: ch2
    port: 9000

clickhouse_map:
  primary:
    host: ch1
    port: 9000
  replica:
    host: ch2
    port: 9000
```

Behavior notes:
- `[]Struct` fields (for example, `[]ClickHouseConfig`) bind from YAML arrays of nested objects.
- `map[string]Struct` fields (for example, `map[string]ClickHouseConfig`) bind from nested YAML maps even though the file source flattens them to dotted keys (for example, `clickhouse_map.primary.host`).
- In strict mode, valid nested keys under dynamic map entries are accepted (for example, `clickhouse_map.primary.host`), but unknown nested fields are rejected (for example, `clickhouse_map.primary.unknown`).
- With `Strict(false)`, unknown nested keys do not fail the load.

<a id="key-normalization-by-source"></a>
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
- File source flattens nested objects but does not rewrite separators.
- File source can optionally adapt flattened keys via `SnakeCaseKeys` and `KeyPrefix` in `sourcefile.Options`.
- Rigging lowercases keys during loader merge for consistent matching across sources.
- Derived field keys are already snake_case, so `MaxConnections` matches `max_connections` without tags.
- Use `name:` only when the source key path differs from the derived path, or use `SnakeCaseKeys` / `KeyPrefix` to adapt file keys at the source boundary.

```go
type Config struct {
    MaxConnections int           // matches max_connections
    LegacyPort     int `conf:"name:service.port"`
}
```

Typed transforms are a separate stage from source key normalization:
- Source normalization/aliasing changes keys before merge/strict mode (this page).
- `WithTransformer(...)` normalizes typed values after binding/defaults/conversion and before tag validation.

See [API Reference](api-reference.md) (`Load Pipeline`, `Transformer[T]`) and [Configuration Patterns](patterns.md) for when to use typed transforms.

<a id="key-mapping-precedence-required"></a>
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

<a id="watch-reload"></a>
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

Behavior notes:
- `Watch` emits an initial snapshot immediately (`Version=1`, `Source="initial"`).
- Subsequent reloads are debounced for 100ms.

Production notes:
- Keep a "last good config" in your app and only swap it after successful `applyNewConfig`.
- Record reload success/failure metrics and include `snapshot.Source` in logs for incident response.
- Use `Load` once at startup before entering watch mode so boot fails fast on invalid config.

Note:
- Built-in `sourcefile` and `sourceenv` return `ErrWatchNotSupported`.
- Custom sources can implement watch support today.
