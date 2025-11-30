package rigging

import (
	"context"
	"errors"
	"time"
)

// Source provides configuration data from backends (env vars, files, remote stores).
// Keys must be normalized to lowercase dot-separated paths (e.g., "database.host").
type Source interface {
	// Load returns configuration as a flat map. Missing optional sources should return empty map.
	Load(ctx context.Context) (map[string]any, error)

	// Watch emits ChangeEvent when configuration changes. Returns ErrWatchNotSupported if not supported.
	Watch(ctx context.Context) (<-chan ChangeEvent, error)
}

// ChangeEvent notifies of configuration changes.
type ChangeEvent struct {
	At    time.Time
	Cause string // Description (e.g., "file-changed")
}

// ErrWatchNotSupported is returned when watching is not supported.
var ErrWatchNotSupported = errors.New("rigging: watch not supported by this source")

// Optional distinguishes "not set" from "zero value".
type Optional[T any] struct {
	Value T
	Set   bool
}

// Get returns the wrapped value and whether it was set.
func (o Optional[T]) Get() (T, bool) {
	return o.Value, o.Set
}

// OrDefault returns the wrapped value or the provided default.
func (o Optional[T]) OrDefault(defaultVal T) T {
	if o.Set {
		return o.Value
	}
	return defaultVal
}

// Validator performs custom validation after tag-based validation.
// Use for cross-field, semantic, or external validation.
type Validator[T any] interface {
	// Validate checks configuration. Return *ValidationError for field-level errors.
	Validate(ctx context.Context, cfg *T) error
}

// ValidatorFunc is a function adapter for Validator interface.
type ValidatorFunc[T any] func(ctx context.Context, cfg *T) error

func (f ValidatorFunc[T]) Validate(ctx context.Context, cfg *T) error {
	return f(ctx, cfg)
}

// Snapshot represents a configuration version emitted by Watch().
type Snapshot[T any] struct {
	Config   *T
	Version  int64 // Increments on reload (starts at 1)
	LoadedAt time.Time
	Source   string // What triggered the load
}
