# Configuration Sources

## Environment Variables

```go
source := sourceenv.New(sourceenv.Options{
    Prefix:        "APP_",  // Only load APP_* variables
    CaseSensitive: false,   // Prefix matching is case-insensitive (default)
})

// Maps environment variables to struct fields:
// APP_DATABASE__HOST → Database.Host
// APP_SERVER__PORT → Server.Port
```

**Prefix Matching:**
- By default (`CaseSensitive: false`), prefix matching is case-insensitive
- `APP_`, `app_`, and `App_` all match when prefix is `"APP_"`
- Set `CaseSensitive: true` for exact case matching
- Keys are always normalized to lowercase after prefix stripping

```go
// Case-insensitive (default) - matches all variations
sourceenv.New(sourceenv.Options{
    Prefix:        "APP_",
    CaseSensitive: false,
})
// Matches: APP_HOST, app_host, App_Host

// Case-sensitive - exact match only
sourceenv.New(sourceenv.Options{
    Prefix:        "APP_",
    CaseSensitive: true,
})
// Matches: APP_HOST only
// Ignores: app_host, App_Host
```

## Files (YAML/JSON/TOML)

```go
source := sourcefile.New("config.yaml", sourcefile.Options{
    Required: true,  // Error if file missing
})

// Auto-detects format from extension
// Flattens nested structures to dot-separated keys
```

## Custom Sources

Implement the `Source` interface:

```go
type Source interface {
    Load(ctx context.Context) (map[string]any, error)
    Watch(ctx context.Context) (<-chan ChangeEvent, error)
    Name() string // Returns human-readable identifier
}

// Example: Consul KV store
type ConsulSource struct {
    client *consul.Client
}

func (s *ConsulSource) Load(ctx context.Context) (map[string]any, error) {
    // Fetch from Consul
}

func (s *ConsulSource) Name() string {
    return "consul:kv"
}
```

**Enhanced Provenance (Optional):**

Implement `SourceWithKeys` to provide detailed source attribution:

```go
type SourceWithKeys interface {
    Source
    LoadWithKeys(ctx context.Context) (data map[string]any, originalKeys map[string]string, err error)
}

func (s *ConsulSource) LoadWithKeys(ctx context.Context) (map[string]any, map[string]string, error) {
    data := make(map[string]any)
    originalKeys := make(map[string]string)

    // Load from Consul
    data["database.host"] = "localhost"
    originalKeys["database.host"] = "config/database/host" // Original Consul key

    return data, originalKeys, nil
}
```

This enables detailed provenance like `consul:kv:config/database/host` for non-file sources (environment variables, remote stores, etc.). For file sources, just the source name is sufficient.

## Watch and Reload

The Watch API allows monitoring sources for changes and reloading configuration automatically:

```go
snapshots, errors, err := loader.Watch(ctx)
if err != nil {
    log.Fatal(err)
}

go func() {
    for {
        select {
        case snapshot := <-snapshots:
            log.Printf("Config reloaded: v%d", snapshot.Version)
            applyNewConfig(snapshot.Config)

        case err := <-errors:
            log.Printf("Reload failed: %v", err)
            // Previous config still valid
        }
    }
}()
```

**Note**: Built-in sources (sourcefile, sourceenv) return `ErrWatchNotSupported`. To use watch with custom sources:

```go
type MySource struct{}

func (s *MySource) Watch(ctx context.Context) (<-chan rigging.ChangeEvent, error) {
    ch := make(chan rigging.ChangeEvent)
    go func() {
        // Emit events when config changes
        ch <- rigging.ChangeEvent{At: time.Now(), Cause: "updated"}
    }()
    return ch, nil
}
```
