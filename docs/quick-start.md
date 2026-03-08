# Quick Start

This guide gets you from zero to a successful load with validation, provenance, and safe dumps.

## 1. Install

```bash
go get github.com/Azhovan/rigging@latest
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

<a id="redacted-dump"></a>
### Redacted Dump

```go
rigging.DumpEffective(os.Stdout, cfg, rigging.WithSources())
```

Secrets tagged with `conf:"secret"` are redacted.

<a id="snapshots-debugging-audits"></a>
### Snapshots for Debugging and Audits

```go
snapshot, err := rigging.CreateSnapshot(cfg)
if err != nil {
    log.Fatal(err)
}

if err := rigging.WriteSnapshot(snapshot, "snapshots/config-{{timestamp}}.json"); err != nil {
    log.Fatal(err)
}
```

Snapshots include flattened config values, provenance, and secret redaction.
Use `WithExcludeFields(...)` to omit noisy fields when sharing or storing snapshots.

<a id="key-mapping-rules"></a>
## 6. Key Mapping Rules

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

Derived field keys are already snake_case, so `MaxConnections` matches `max_connections` without extra tags.
Use `name:` only when the source key path differs, or adapt file keys at the source boundary with `sourcefile.Options{SnakeCaseKeys: true}` / `KeyPrefix`.

```go
type Config struct {
    MaxConnections int // matches max_connections
    LegacyPort     int `conf:"name:service.port"`
}
```

If you need to bind a field to a specific environment-style key path, use `env:`.
The tag matches the normalized key after any `sourceenv.Options.Prefix` stripping, so with `Prefix: "APP_"`, `APP_DATABASE__HOST` maps via `conf:"env:DATABASE__HOST"`.

## 7. Fail-Fast Validation

Tag validation and custom validators run during `Load`.

<a id="typed-transforms"></a>
### Typed Transforms (Before Tag Validation)

Use `WithTransformer(...)` when you need to normalize or derive typed values before tag validation runs.
This is useful for canonicalization such as trimming whitespace or normalizing enum casing.
Use `WithTransformerFunc(...)` for inline functions (as shown below), and `WithTransformer(...)` when registering a reusable `Transformer[T]` implementation.

```go
loader.WithTransformerFunc(func(ctx context.Context, cfg *Config) error {
    cfg.Environment = strings.ToLower(strings.TrimSpace(cfg.Environment))
    return nil
})
```

Use typed transforms for post-bind value normalization.
For source key aliasing/normalization (for example renaming keys before strict mode), use a custom source wrapper instead.

<a id="optional-fields"></a>
### Optional Fields (`Optional[T]`)

Use `rigging.Optional[T]` when you need to distinguish "not set" from a zero value (`false`, `0`, `""`).

```go
type Config struct {
    Features struct {
        EnableMetrics rigging.Optional[bool]
        RateLimit     rigging.Optional[int] `conf:"min:1"`
    } `conf:"prefix:features"`
}
```

```go
if rateLimit, ok := cfg.Features.RateLimit.Get(); ok {
    log.Printf("rate limit explicitly set: %d", rateLimit)
} else {
    log.Printf("rate limit not set")
}
```

<a id="custom-validators"></a>
### Custom Validators (After Tag Validation)

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
- Transformer demo: [`examples/transformer`](../examples/transformer)
