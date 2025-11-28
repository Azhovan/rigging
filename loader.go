package rigging

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
)

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
	// Step 1: Load from all sources and merge
	mergedData := make(map[string]mergedEntry)
	
	for i, source := range l.sources {
		// Load data from source
		data, err := source.Load(ctx)
		if err != nil {
			return nil, fmt.Errorf("load source %d: %w", i, err)
		}
		
		// Merge data into mergedData map
		// Later sources override earlier ones
		for key, value := range data {
			// Normalize key to lowercase dot-separated path
			normalizedKey := strings.ToLower(key)
			
			// Store with source information
			sourceName := fmt.Sprintf("source-%d", i)
			mergedData[normalizedKey] = mergedEntry{
				value:      value,
				sourceName: sourceName,
			}
		}
	}
	
	// Step 2: In strict mode, detect unknown keys
	if l.strict {
		// Get all valid field keys from the struct
		var cfg T
		validKeys := collectValidKeys(reflect.TypeOf(cfg), "")
		
		// Check for unknown keys
		var unknownKeys []string
		for key := range mergedData {
			if !validKeys[key] {
				unknownKeys = append(unknownKeys, key)
			}
		}
		
		if len(unknownKeys) > 0 {
			return nil, fmt.Errorf("strict mode: unknown configuration keys: %v", unknownKeys)
		}
	}
	
	// Step 3: Create zero instance of T
	cfg := new(T)
	cfgValue := reflect.ValueOf(cfg).Elem()
	
	// Step 4: Bind struct fields from merged data
	var provenanceFields []FieldProvenance
	bindErrors := bindStruct(cfgValue, mergedData, &provenanceFields, "", "")
	
	// Step 5: Validate struct (tag-based validation)
	validationErrors := validateStruct(cfgValue)
	
	// Merge binding and validation errors
	allErrors := append(bindErrors, validationErrors...)
	
	// Step 6: Run custom validators
	for i, validator := range l.validators {
		err := validator.Validate(ctx, cfg)
		if err != nil {
			// Check if it's a ValidationError
			if valErr, ok := err.(*ValidationError); ok {
				allErrors = append(allErrors, valErr.FieldErrors...)
			} else {
				// Wrap other errors as validation errors
				return nil, fmt.Errorf("validator %d failed: %w", i, err)
			}
		}
	}
	
	// Step 7: Return error if any validation failed
	if len(allErrors) > 0 {
		return nil, &ValidationError{FieldErrors: allErrors}
	}
	
	// Step 8: Store provenance for the config instance
	storeProvenance(cfg, &Provenance{Fields: provenanceFields})
	
	// Step 9: Return the loaded configuration
	return cfg, nil
}

// Watch monitors all sources for changes and reloads configuration automatically.
// Returns two channels:
//   - snapshots: emits new Snapshot[T] on successful reload
//   - errors: emits errors when reload/validation fails
// The previous valid configuration is retained on validation failures.
// Both channels are closed when ctx is cancelled.
func (l *Loader[T]) Watch(ctx context.Context) (<-chan Snapshot[T], <-chan error, error) {
	// Load initial configuration
	initialCfg, err := l.Load(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("initial load failed: %w", err)
	}

	// Create channels for snapshots and errors
	snapshotCh := make(chan Snapshot[T])
	errorCh := make(chan error)

	// Start watch goroutine
	go l.watchLoop(ctx, initialCfg, snapshotCh, errorCh)

	return snapshotCh, errorCh, nil
}

// collectValidKeys recursively collects all valid configuration keys from a struct type.
// It returns a map of valid keys for use in strict mode validation.
func collectValidKeys(t reflect.Type, prefix string) map[string]bool {
	validKeys := make(map[string]bool)
	
	// Dereference pointer types
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	
	// Only process struct types
	if t.Kind() != reflect.Struct {
		return validKeys
	}
	
	// Walk through all fields
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		
		// Skip unexported fields
		if !field.IsExported() {
			continue
		}
		
		// Parse struct tag
		tag := field.Tag.Get("conf")
		tagCfg := parseTag(tag)
		
		// Determine key path
		keyPath := determineKeyPath(field.Name, tagCfg, prefix)
		
		// Add this key as valid
		validKeys[keyPath] = true
		
		// Handle nested structs
		fieldType := field.Type
		
		// Check if it's an Optional[T] type
		if isOptionalType(fieldType) {
			// For Optional[T], check the inner type
			innerType := fieldType.Field(0).Type
			if innerType.Kind() == reflect.Struct {
				// Recursively collect keys from nested struct
				nestedKeys := collectValidKeys(innerType, keyPath)
				for k := range nestedKeys {
					validKeys[k] = true
				}
			}
		} else if fieldType.Kind() == reflect.Struct {
			// Skip time.Time and time.Duration (they're structs but treated as primitives)
			if fieldType.PkgPath() == "time" {
				continue
			}
			
			// Determine prefix for nested struct
			nestedPrefix := keyPath
			if tagCfg.prefix != "" {
				nestedPrefix = tagCfg.prefix
			}
			
			// Recursively collect keys from nested struct
			nestedKeys := collectValidKeys(fieldType, nestedPrefix)
			for k := range nestedKeys {
				validKeys[k] = true
			}
		}
	}
	
	return validKeys
}

// watchLoop is the main goroutine that monitors sources for changes and reloads configuration.
// It handles debouncing, thread-safe snapshot emission, and cleanup.
func (l *Loader[T]) watchLoop(ctx context.Context, initialCfg *T, snapshotCh chan<- Snapshot[T], errorCh chan<- error) {
	defer close(snapshotCh)
	defer close(errorCh)

	// Emit initial snapshot
	currentVersion := int64(1)
	snapshotCh <- Snapshot[T]{
		Config:   initialCfg,
		Version:  currentVersion,
		LoadedAt: time.Now(),
		Source:   "initial",
	}

	// Start watching all sources
	changeChannels := make([]<-chan ChangeEvent, 0, len(l.sources))
	cancelFuncs := make([]context.CancelFunc, 0, len(l.sources))

	for i, source := range l.sources {
		// Create a child context for this source watcher
		sourceCtx, cancel := context.WithCancel(ctx)
		cancelFuncs = append(cancelFuncs, cancel)

		// Try to watch this source
		changeCh, err := source.Watch(sourceCtx)
		if err != nil {
			// If watch is not supported, skip this source
			if errors.Is(err, ErrWatchNotSupported) {
				cancel() // Clean up the context
				continue
			}
			// For other errors, send to error channel and skip
			select {
			case errorCh <- fmt.Errorf("watch source %d: %w", i, err):
			case <-ctx.Done():
				cancel()
				return
			}
			cancel()
			continue
		}

		changeChannels = append(changeChannels, changeCh)
	}

	// If no sources support watching, we're done
	if len(changeChannels) == 0 {
		return
	}

	// Create a debounce timer
	var debounceTimer *time.Timer
	const debounceDelay = 100 * time.Millisecond

	// Merge all change channels into one
	mergedChanges := make(chan ChangeEvent)
	go func() {
		defer close(mergedChanges)
		for {
			// Use reflection to select from multiple channels
			cases := make([]reflect.SelectCase, len(changeChannels)+1)
			
			// Add context.Done case
			cases[0] = reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(ctx.Done()),
			}
			
			// Add all change channels
			for i, ch := range changeChannels {
				cases[i+1] = reflect.SelectCase{
					Dir:  reflect.SelectRecv,
					Chan: reflect.ValueOf(ch),
				}
			}

			// Wait for any channel to receive
			chosen, value, ok := reflect.Select(cases)

			// Check if context was cancelled
			if chosen == 0 {
				return
			}

			// Check if channel was closed
			if !ok {
				// Remove this channel from the list
				changeChannels = append(changeChannels[:chosen-1], changeChannels[chosen:]...)
				// If all channels are closed, exit
				if len(changeChannels) == 0 {
					return
				}
				continue
			}

			// Extract the ChangeEvent
			event := value.Interface().(ChangeEvent)
			
			// Send to merged channel
			select {
			case mergedChanges <- event:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Main watch loop
	for {
		select {
		case <-ctx.Done():
			// Cancel all source watchers
			for _, cancel := range cancelFuncs {
				cancel()
			}
			return

		case event, ok := <-mergedChanges:
			if !ok {
				// All change channels closed
				return
			}

			// Debounce: reset timer on each event
			if debounceTimer != nil {
				debounceTimer.Stop()
			}

			debounceTimer = time.AfterFunc(debounceDelay, func() {
				// Reload configuration
				newCfg, err := l.Load(ctx)
				if err != nil {
					// Send error, keep previous config
					select {
					case errorCh <- fmt.Errorf("reload failed: %w", err):
					case <-ctx.Done():
					}
					return
				}

				// Increment version and emit new snapshot
				currentVersion++
				snapshot := Snapshot[T]{
					Config:   newCfg,
					Version:  currentVersion,
					LoadedAt: time.Now(),
					Source:   event.Cause,
				}

				select {
				case snapshotCh <- snapshot:
				case <-ctx.Done():
				}
			})
		}
	}
}
