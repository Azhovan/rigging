package rigging

import (
	"context"
	"errors"
	"time"
)

// Source represents a pluggable configuration source.
// Implementations provide configuration data from various sources
// such as environment variables, files, or remote stores.
type Source interface {
	// Load returns a flat map of configuration keys to values.
	// Keys must be normalized to dot-separated paths (e.g., "database.host").
	// The context can be used for cancellation and timeouts.
	Load(ctx context.Context) (map[string]any, error)

	// Watch returns a channel of change events if the source supports watching.
	// Returns (nil, ErrWatchNotSupported) if watching is not supported.
	// The channel is closed when the context is cancelled.
	Watch(ctx context.Context) (<-chan ChangeEvent, error)
}

// ChangeEvent represents a configuration change notification.
type ChangeEvent struct {
	At    time.Time // When the change occurred
	Cause string    // Description of what caused the change (e.g., "file-changed", "env-updated")
}

// ErrWatchNotSupported is returned by Source.Watch when the source
// does not support watching for configuration changes.
var ErrWatchNotSupported = errors.New("rigging: watch not supported by this source")

// Optional wraps a value that may or may not be explicitly set.
// This allows distinguishing between "field not set" and "field set to zero value".
type Optional[T any] struct {
	Value T    // The wrapped value
	Set   bool // Whether the value was explicitly set
}

// Get returns the value and whether it was set.
func (o Optional[T]) Get() (T, bool) {
	return o.Value, o.Set
}

// OrDefault returns the value if set, otherwise returns the provided default.
func (o Optional[T]) OrDefault(defaultVal T) T {
	if o.Set {
		return o.Value
	}
	return defaultVal
}

// Validator performs validation on a loaded configuration.
// Custom validators can implement this interface to perform
// cross-field or semantic validation beyond tag-based rules.
type Validator[T any] interface {
	// Validate checks the configuration and returns an error if validation fails.
	// The error should typically be a *ValidationError for consistency.
	Validate(ctx context.Context, cfg *T) error
}

// ValidatorFunc is a function adapter for Validator.
// It allows using functions as validators without defining a new type.
type ValidatorFunc[T any] func(ctx context.Context, cfg *T) error

// Validate implements the Validator interface for ValidatorFunc.
func (f ValidatorFunc[T]) Validate(ctx context.Context, cfg *T) error {
	return f(ctx, cfg)
}

// Snapshot represents a loaded configuration with metadata.
// It is used by the Watch functionality to provide versioned configuration updates.
type Snapshot[T any] struct {
	Config   *T        // The loaded and validated configuration
	Version  int64     // Version number, incremented on each reload
	LoadedAt time.Time // When this configuration was loaded
	Source   string    // Description of what triggered this load (e.g., "initial", "file-changed")
}
