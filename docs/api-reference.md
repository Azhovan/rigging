# API Reference

## Core Types

### Loader[T]

The main entry point for loading configuration.

```go
loader := rigging.NewLoader[Config]()
```

**Methods:**

- `WithSource(src Source) *Loader[T]` - Add a configuration source
- `WithTransformer(t Transformer[T]) *Loader[T]` - Add a typed transform (after bind/defaults/conversion, before validation)
- `WithTransformerFunc(fn func(context.Context, *T) error) *Loader[T]` - Add a typed transform using a function (ergonomic helper for inline transforms)
- `WithValidator(v Validator[T]) *Loader[T]` - Add a custom validator
- `Strict(strict bool) *Loader[T]` - Enable/disable strict mode
- `Load(ctx context.Context) (*T, error)` - Load and validate configuration
- `LoadWithProvenance(ctx context.Context) (*T, *Provenance, error)` - Load config and return provenance without using global storage
- `Watch(ctx context.Context) (<-chan Snapshot[T], <-chan error, error)` - Watch for changes

### Source

Interface for configuration sources.

```go
type Source interface {
    Load(ctx context.Context) (map[string]any, error)
    Watch(ctx context.Context) (<-chan ChangeEvent, error)
    Name() string
}
```

`Watch` may return `ErrWatchNotSupported` for sources that don't support runtime change events.

**Built-in sources:**
- `sourcefile.New(path string, opts sourcefile.Options)` - YAML/JSON/TOML files
- `sourceenv.New(opts sourceenv.Options)` - Environment variables

`sourcefile.Options` fields:
- `Format string` - Optional explicit format (`yaml`, `json`, `toml`). When empty, inferred from file extension.
- `Required bool` - When true, missing files return an error. When false, missing files return an empty map.
- `Root string` - Optional dot-separated subtree path to load and flatten as the source root.

`sourcefile.Root` example:

```go
src := sourcefile.New("config.yaml", sourcefile.Options{
    Root: "root.section",
})

loader := rigging.NewLoader[Config]().WithSource(src)
cfg, err := loader.Load(ctx)
if err != nil {
    if errors.Is(err, sourcefile.ErrRootNotFound) ||
        errors.Is(err, sourcefile.ErrRootNotMap) ||
        errors.Is(err, sourcefile.ErrInvalidRoot) {
        // handle root configuration problems
    }
    return err
}
_ = cfg
```

`Name()` is used in provenance output (for example, `file:config.yaml`, `env:APP_PORT`).

### SourceWithKeys

Optional interface for richer provenance (original source keys).

```go
type SourceWithKeys interface {
    Source
    LoadWithKeys(ctx context.Context) (data map[string]any, originalKeys map[string]string, err error)
}
```

`originalKeys` maps normalized keys back to original source keys (for example, full env var names).

### Optional[T]

Distinguish "not set" from "zero value".

```go
type Optional[T any] struct {
    Value T
    Set   bool
}
```

**Methods:**
- `Get() (T, bool)` - Returns value and whether it was set
- `OrDefault(defaultVal T) T` - Returns value or default

### Transformer[T]

Interface for typed config transforms.

Transformers run after binding/defaults/type conversion and before tag-based validation.
Use them for canonicalization (for example trim/lowercase/derive fields), not source key normalization.

```go
type Transformer[T any] interface {
    Transform(ctx context.Context, cfg *T) error
}
```

**Helper:**
- `TransformerFunc[T](func(ctx context.Context, cfg *T) error)` - Function adapter

When registering an inline transform, prefer `WithTransformerFunc(...)` for a shorter call site:

```go
loader.WithTransformerFunc(func(ctx context.Context, cfg *Config) error {
    cfg.Environment = strings.ToLower(strings.TrimSpace(cfg.Environment))
    return nil
})
```

### Validator[T]

Interface for custom validation.

```go
type Validator[T any] interface {
    Validate(ctx context.Context, cfg *T) error
}
```

**Helper:**
- `ValidatorFunc[T](func(ctx context.Context, cfg *T) error)` - Function adapter

## Load Pipeline

Rigging uses a fixed load pipeline during `Loader.Load` and `Loader.Watch` reloads:

`merge sources -> strict unknown-key check -> bind/type conversion/defaults -> typed transforms -> tag validation -> custom validators`

Behavior notes:

- Sources are merged in registration order; later sources override earlier ones.
- `Strict(true)` is the default, and unknown-key checks run before binding.
- Binding applies defaults and type conversion before transformers run.
- `WithTransformer(...)` mutates typed config values before tag validation.
- `WithValidator(...)` runs after tag validation for cross-field/business checks.
- Source/key normalization (for example env key rewriting/aliasing) belongs in sources or source wrappers, not typed transformers.

## Validation Semantics

Rigging distinguishes **field presence** from Go zero values during `Loader.Load`.

- `required` checks whether a key was provided by any source (or default), not whether the final value is non-zero.
- `min`, `max`, and `oneof` run for fields that are explicitly provided, even when the value is a zero value (for example, `0`, `""`, `false`).
- Optional fields that are absent skip value constraints.

Example behavior:

- Absent optional `port int \`conf:"min:1"\`` -> no validation error.
- Provided `port=0` -> `min` validation error.

## Observability

### GetProvenance

Track where configuration values came from.

```go
func GetProvenance[T any](cfg *T) (*Provenance, bool)
```

```go
func ReleaseProvenance[T any](cfg *T)
```

Returns provenance metadata with field-level source information.
`ReleaseProvenance` removes stored provenance for a config instance.
`LoadWithProvenance` returns provenance directly and does not populate global storage.

Lifecycle notes:
- `Load` stores provenance for the returned config pointer until `ReleaseProvenance` is called.
- `Watch` keeps global provenance for the most recently emitted snapshot; when a newer snapshot is emitted, the superseded snapshot's provenance is released.
- If you need historical provenance across snapshots, persist/copy it in your own state.

```go
type Provenance struct {
    Fields []FieldProvenance
}

type FieldProvenance struct {
    FieldPath  string // e.g., "Database.Host"
    KeyPath    string // e.g., "database.host"
    SourceName string // e.g., "file:config.yaml" or "env:APP_DATABASE__PASSWORD"
    Secret     bool   // true if marked as secret
}
```

### DumpEffective

Safely dump configuration with secret redaction.

```go
func DumpEffective[T any](w io.Writer, cfg *T, opts ...DumpOption) error
```

**Options:**
- `WithSources()` - Include source attribution
- `AsJSON()` - Output as JSON instead of text
- `WithIndent(indent string)` - Set JSON indentation

When `AsJSON()` and `WithSources()` are combined, leaf fields are wrapped as:

```json
{"value":"...","source":"..."}
```

**Examples:**

```go
// Text format
rigging.DumpEffective(os.Stdout, cfg)

// With source attribution
rigging.DumpEffective(os.Stdout, cfg, rigging.WithSources())

// JSON format
rigging.DumpEffective(os.Stdout, cfg, rigging.AsJSON())

// JSON with custom indent
rigging.DumpEffective(os.Stdout, cfg,
    rigging.AsJSON(),
    rigging.WithIndent("    "))
```

## Snapshots

Capture configuration state for debugging and auditing.

### CreateSnapshot

```go
func CreateSnapshot[T any](cfg *T, opts ...SnapshotOption) (*ConfigSnapshot, error)
```

Creates a point-in-time capture with flattened config, provenance, and automatic secret redaction.

```go
snapshot, err := rigging.CreateSnapshot(cfg)
// snapshot.Config["database.host"] = "localhost"
// snapshot.Config["database.password"] = "***redacted***"
```

**Options:**
- `WithExcludeFields(paths ...string)` - Exclude specific field paths

```go
snapshot, err := rigging.CreateSnapshot(cfg,
    rigging.WithExcludeFields("debug", "internal.metrics"))
```

### WriteSnapshot / ReadSnapshot

```go
func WriteSnapshot(snapshot *ConfigSnapshot, pathTemplate string) error
func ReadSnapshot(path string) (*ConfigSnapshot, error)
```

Persist and restore snapshots with atomic writes and `{{timestamp}}` template support.

```go
// Write with timestamp in filename
err := rigging.WriteSnapshot(snapshot, "snapshots/config-{{timestamp}}.json")
// Creates: snapshots/config-20240115-103000.json

// Read back
restored, err := rigging.ReadSnapshot("snapshots/config-20240115-103000.json")
```

### Path Helpers

```go
func ExpandPath(template string) string
func ExpandPathWithTime(template string, t time.Time) string
```

`WriteSnapshot` uses the snapshot's internal `Timestamp` when expanding `{{timestamp}}` to keep filename and snapshot metadata consistent.

### ConfigSnapshot

```go
type ConfigSnapshot struct {
    Version    string                 // Format version ("1.0")
    Timestamp  time.Time              // Creation time
    Config     map[string]any         // Flattened config (secrets redacted)
    Provenance []FieldProvenance      // Source tracking
}
```

### Constants and Errors

```go
const MaxSnapshotSize = 100 * 1024 * 1024  // 100MB limit
const SnapshotVersion = "1.0"

var ErrSnapshotTooLarge    // Snapshot exceeds size limit
var ErrNilConfig           // Nil config passed
var ErrUnsupportedVersion  // Unknown snapshot version
```

## Error Types

### ValidationError

Aggregates all validation failures.

```go
type ValidationError struct {
    FieldErrors []FieldError
}
```

### FieldError

Represents a single field validation failure.

```go
type FieldError struct {
    FieldPath string // e.g., "Database.Port"
    Code      string // e.g., "required", "min", "max"
    Message   string // Human-readable error
}
```

**Standard error codes:**
- `required` - Field is required but not provided
- `min` - Value below minimum
- `max` - Value exceeds maximum
- `oneof` - Value not in allowed set
- `invalid_type` - Type conversion failed
- `unknown_key` - Configuration key doesn't map to any field (strict mode)

## Struct Tags

Configure binding and validation with the `conf` tag:

| Tag | Description | Example |
|-----|-------------|---------|
| `required` | Field must have a value | `conf:"required"` |
| `env:VAR` | Bind from an explicit env-style key path (normalized with `__` -> `.` and lowercased) | `conf:"env:APP__DATABASE__HOST"` |
| `default:X` | Default value if not provided | `conf:"default:8080"` |
| `min:N` | Minimum value (numeric) or length (string) | `conf:"min:1024"` |
| `max:N` | Maximum value (numeric) or length (string) | `conf:"max:65535"` |
| `oneof:a,b,c` | Value must be one of the options (duplicates removed, empty values ignored) | `conf:"oneof:prod,staging,dev"` |
| `secret` | Mark field for redaction | `conf:"secret"` |
| `prefix:path` | Prefix for nested struct fields | `conf:"prefix:database"` |
| `name:path` | Override derived key path | `conf:"name:custom.path"` |

**Combining tags:**

```go
type Config struct {
    Port     int    `conf:"default:8080,min:1024,max:65535"`
    Env      string `conf:"required,oneof:prod,staging,dev"`
    Password string `conf:"required,secret"`
}
```

**Tag precedence:**

- `name:` overrides all key derivation (ignores `prefix:` and field name)
- `env:` is used when `name:` is not set
- otherwise keys are derived from field names (snake_case), with parent `prefix:` for nested structs
- env-style normalization converts `__` to `.` and preserves single `_`

```go
type Config struct {
    Database struct {
        Host string              // Key: database.host (prefix applied)
        Port int `conf:"name:db.port"` // Key: db.port (name overrides prefix)
    } `conf:"prefix:database"`
}
```

## Watch and Reload

### Snapshot[T]

Represents a loaded configuration with metadata.

```go
type Snapshot[T any] struct {
    Config   *T        // The loaded configuration
    Version  int64     // Incremented on each reload
    LoadedAt time.Time // When loaded
    Source   string    // What triggered the load
}
```

Watch behavior:
- `Watch` emits an initial snapshot immediately (`Version=1`, `Source="initial"`).
- Reloads from source change events are debounced by 100ms.

Provenance note:
- For configs emitted by `Watch`, `GetProvenance` is intended for the latest snapshot.
- Older snapshots may no longer have global provenance after subsequent reloads.

### ChangeEvent

Notification of configuration change.

```go
type ChangeEvent struct {
    At    time.Time // When the change occurred
    Cause string    // Description of the change
}
```

`ErrWatchNotSupported` is returned by `Source.Watch` for sources that don't support watch mode.

## Strict Mode

Catch typos and deprecated keys:

```go
loader.Strict(true)  // Fail on unknown keys (default)
loader.Strict(false) // Ignore unknown keys
```

## Error Handling

All validation errors include field paths and codes:

```go
cfg, err := loader.Load(ctx)
if err != nil {
    if valErr, ok := err.(*rigging.ValidationError); ok {
        for _, fe := range valErr.FieldErrors {
            log.Printf("%s: %s (code: %s)",
                fe.FieldPath, fe.Message, fe.Code)
        }
    }
}
```
