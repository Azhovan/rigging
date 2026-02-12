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

## 4. Validate at Startup, Not Mid-Request

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

## 5. Mark and Handle Secrets Explicitly

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

## 6. Use Provenance During Incident Response

```go
prov, _ := rigging.GetProvenance(cfg)
for _, field := range prov.Fields {
    log.Printf("%s <- %s", field.FieldPath, field.SourceName)
}
```

This quickly answers "why is this value set?" without guesswork.

## 7. Model Repeated and Named Backends

Use `[]Struct` for ordered/repeated entries and `map[string]Struct` for named dynamic entries.

```go
type ClickHouseConfig struct {
    Host string
    Port int
}

type Config struct {
    // Ordered list of backends
    ClickhouseNodes []ClickHouseConfig `conf:"name:clickhouse_nodes"`

    // Named backends keyed by environment/role
    Clickhouse map[string]ClickHouseConfig `conf:"name:clickhouse"`
}
```

```yaml
clickhouse_nodes:
  - host: ch1.internal
    port: 9000
  - host: ch2.internal
    port: 9000

clickhouse:
  primary:
    host: ch-primary.internal
    port: 9000
  analytics:
    host: ch-analytics.internal
    port: 9001
```

When strict mode is enabled, keys under declared maps (for example `clickhouse.primary.host`) are treated as valid.

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
