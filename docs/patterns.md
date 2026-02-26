# Configuration Patterns

This page focuses on practical schema choices for production services.

## 1. Organize by Domain

```go
type Config struct {
    Server   ServerConfig   `conf:"prefix:server"`
    Database DatabaseConfig `conf:"prefix:database"`
    Logging  LoggingConfig  `conf:"prefix:logging"`
}
```

Avoid large flat config structs. Grouping makes validation, ownership, and evolution easier.

## 2. Use Explicit Source Layers

```go
loader := rigging.NewLoader[Config]().
    WithSource(sourcefile.New("defaults.yaml", sourcefile.Options{})).
    WithSource(sourcefile.New("env.yaml", sourcefile.Options{})).
    WithSource(sourceenv.New(sourceenv.Options{Prefix: "APP_"}))
```

Recommended order:
1. defaults
2. environment file
3. env overrides (especially secrets)

## 3. Choose a Key Strategy Early

Default derived keys are snake_case field names.

```go
type Config struct {
    MaxConnections int // key: max_connections
    APIKey         string // key: api_key
}
```

If your source keys use snake_case or custom paths, map explicitly:

```go
type Config struct {
    MaxConnections int `conf:"name:max_connections"`
    APIKey         string `conf:"name:api.key"`
}
```

Use `env:` when you need a field to bind to a specific environment-style key path without changing your general source strategy.
This is useful during migrations or when a legacy env variable name must be preserved.

```go
type Config struct {
    DatabaseHost string `conf:"env:APP__DATABASE__HOST,required"`
}
```

`env:` normalizes env-style syntax (`__` -> `.`, lowercased) before matching.
Prefer `name:` for general cross-source key mapping; use `env:` when the intent is specifically env-key compatibility.

## 4. Normalize Typed Values Before Validation

Use typed transforms for startup-time canonicalization of already-bound values.
This keeps normalization logic out of request paths and lets tag validation operate on canonical values.

```go
loader := rigging.NewLoader[Config]().
    WithTransformerFunc(func(ctx context.Context, cfg *Config) error {
        cfg.Env = strings.ToLower(strings.TrimSpace(cfg.Env))
        return nil
    })
```

Typical uses:
- trim and lowercase enum-like strings before `oneof` validation
- derive convenience fields from typed config values
- dedupe/sort lists before custom validation or downstream use

Important:
- `WithTransformerFunc(...)` is the ergonomic helper for inline transform functions.
- `WithTransformer(...)` registers a reusable `Transformer[T]` implementation.
- `WithTransformer(...)` is for typed value normalization after binding/defaults/conversion.
- source key aliasing or key normalization belongs in sources/source wrappers, not typed transforms.

## 5. Validate at Startup, Not Mid-Request

```go
type Config struct {
    Port int `conf:"required,min:1024,max:65535"`
    Env  string `conf:"required,oneof:prod,staging,dev"`
}

loader.WithValidator(rigging.ValidatorFunc[Config](func(ctx context.Context, cfg *Config) error {
    if cfg.Env == "prod" && cfg.Port == 8080 {
        return errors.New("prod must not use default dev port")
    }
    return nil
}))
```

Treat config load as a startup gate.

## 6. Mark and Handle Secrets Explicitly

```go
type Config struct {
    DatabasePassword string `conf:"required,secret"`
    APIKey           string `conf:"required,secret"`
}
```

Then use safe outputs:

```go
rigging.DumpEffective(os.Stdout, cfg, rigging.WithSources())
snapshot, _ := rigging.CreateSnapshot(cfg)
```

Secrets are redacted in dump/snapshot outputs.

## 7. Use Provenance During Incident Response

```go
prov, _ := rigging.GetProvenance(cfg)
for _, field := range prov.Fields {
    log.Printf("%s <- %s", field.FieldPath, field.SourceName)
}
```

This quickly answers "why is this value set?" without guesswork.

## 8. Provenance Lifecycle for Long-Lived Processes

If you do not want global provenance retention:

```go
cfg, prov, err := loader.LoadWithProvenance(ctx)
_ = prov // pass to telemetry/logs
```

Or release after use:

```go
rigging.ReleaseProvenance(cfg)
```

## 9. Use `Optional[T]` When Absence Differs from Zero

Use `rigging.Optional[T]` for fields where "not set" must be distinct from valid zero values.
This is common for feature flags (`false` vs unset) and numeric limits (`0` vs unset).

```go
type Config struct {
    Features struct {
        EnableMetrics rigging.Optional[bool]
        RateLimit     rigging.Optional[int] `conf:"min:1"`
    } `conf:"prefix:features"`
}
```

```go
if enabled, ok := cfg.Features.EnableMetrics.Get(); ok {
    log.Printf("metrics explicitly set to %v", enabled)
} else {
    log.Printf("metrics flag not set")
}
```

Notes:
- `Optional[T]` works well when defaults should be applied by application logic rather than config tags.
- Tag validation still applies when the optional value is set (for example `min:1` on `Optional[int]`).

## 10. Capture Snapshots for Incident Response and Change Review

Snapshots give you a point-in-time, redacted, provenance-aware record of effective configuration.
Use them when debugging environment drift, reviewing rollout changes, or attaching config state to incident artifacts.

```go
snapshot, err := rigging.CreateSnapshot(cfg)
if err != nil {
    return err
}

if err := rigging.WriteSnapshot(snapshot, "snapshots/config-{{timestamp}}.json"); err != nil {
    return err
}
```

Tips:
- Secrets are redacted automatically in snapshots.
- Use `WithExcludeFields(...)` to omit noisy or non-essential fields from snapshots.
- Keep snapshots outside request paths; they are a diagnostic/audit tool, not a per-request operation.
