# API Reference

## Core Types

### Loader[T]

The main entry point for loading configuration.

```go
loader := rigging.NewLoader[Config]()
```

**Methods:**

- `WithSource(src Source) *Loader[T]` - Add a configuration source
- `WithValidator(v Validator[T]) *Loader[T]` - Add a custom validator
- `Strict(strict bool) *Loader[T]` - Enable/disable strict mode
- `Load(ctx context.Context) (*T, error)` - Load and validate configuration
- `Watch(ctx context.Context) (<-chan Snapshot[T], <-chan error, error)` - Watch for changes

### Source

Interface for configuration sources.

```go
type Source interface {
    Load(ctx context.Context) (map[string]any, error)
    Watch(ctx context.Context) (<-chan ChangeEvent, error)
}
```

**Built-in sources:**
- `sourcefile.New(path string, opts sourcefile.Options)` - YAML/JSON/TOML files
- `sourceenv.New(opts sourceenv.Options)` - Environment variables

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

### Validator[T]

Interface for custom validation.

```go
type Validator[T any] interface {
    Validate(ctx context.Context, cfg *T) error
}
```

**Helper:**
- `ValidatorFunc[T](func(ctx context.Context, cfg *T) error)` - Function adapter

## Observability

### GetProvenance

Track where configuration values came from.

```go
func GetProvenance[T any](cfg *T) (*Provenance, bool)
```

Returns provenance metadata with field-level source information.

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
- `prefix:` applies to nested struct fields
- Without `name:`, keys are derived from field names (lowercased first letter)

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

### ChangeEvent

Notification of configuration change.

```go
type ChangeEvent struct {
    At    time.Time // When the change occurred
    Cause string    // Description of the change
}
```

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
