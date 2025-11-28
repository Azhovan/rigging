package rigging

import "context"

// Loader loads and validates configuration of type T from multiple sources.
// It provides a fluent API for configuring sources, validators, and loading behavior.
// Loader instances are not safe for concurrent use during configuration,
// but loaded configuration instances are safe for concurrent reads.
type Loader[T any] struct {
	sources    []Source       // Configuration sources, processed in order
	validators []Validator[T] // Custom validators, executed in order
	strict     bool           // Whether to fail on unknown keys (default: true)
}

// NewLoader creates a new Loader for configuration type T.
// The loader starts with no sources or validators and strict mode enabled by default.
func NewLoader[T any]() *Loader[T] {
	return &Loader[T]{
		sources:    make([]Source, 0),
		validators: make([]Validator[T], 0),
		strict:     true, // Default to strict mode
	}
}

// WithSource adds a configuration source to the loader.
// Sources are processed in the order they are added, with later sources
// overriding values from earlier sources.
// Returns the loader for method chaining (fluent API).
func (l *Loader[T]) WithSource(src Source) *Loader[T] {
	l.sources = append(l.sources, src)
	return l
}

// WithValidator adds a custom validator for cross-field validation.
// Validators are executed in the order they are added, after tag-based validation.
// Returns the loader for method chaining (fluent API).
func (l *Loader[T]) WithValidator(v Validator[T]) *Loader[T] {
	l.validators = append(l.validators, v)
	return l
}

// Strict controls whether unknown keys cause loading to fail.
// When strict is true (default), any keys in sources that don't map to struct fields
// will cause Load to return an error.
// When strict is false, unknown keys are silently ignored.
// Returns the loader for method chaining (fluent API).
func (l *Loader[T]) Strict(strict bool) *Loader[T] {
	l.strict = strict
	return l
}

// Load loads, merges, binds, and validates configuration from all sources.
// It processes sources in order, merges their data, binds values to the typed struct,
// performs tag-based validation, and runs custom validators.
// Returns the typed configuration or a structured error.
func (l *Loader[T]) Load(ctx context.Context) (*T, error) {
	// TODO: Implementation in task 11
	return nil, nil
}

// Watch monitors all sources for changes and reloads configuration automatically.
// Returns two channels:
//   - snapshots: emits new Snapshot[T] on successful reload
//   - errors: emits errors when reload/validation fails
// The previous valid configuration is retained on validation failures.
// Both channels are closed when ctx is cancelled.
func (l *Loader[T]) Watch(ctx context.Context) (<-chan Snapshot[T], <-chan error, error) {
	// TODO: Implementation in task 15 (optional)
	return nil, nil, nil
}
