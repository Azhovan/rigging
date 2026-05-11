package rigging

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// Transformer mutates a typed config after binding/defaults/conversion and before validation.
// Use for canonicalization or deriving fields from already-typed values.
type Transformer[T any] interface {
	Transform(ctx context.Context, cfg *T) error
}

// TransformerFunc is a function adapter for Transformer interface.
type TransformerFunc[T any] func(ctx context.Context, cfg *T) error

func (f TransformerFunc[T]) Transform(ctx context.Context, cfg *T) error {
	return f(ctx, cfg)
}

// Loader loads and validates configuration from multiple sources.
// Sources are processed in order (later override earlier). Supports transformers, tag-based validation, and custom validation.
// Thread-safe for reads, not for concurrent configuration changes.
type Loader[T any] struct {
	sources      []Source
	transformers []Transformer[T]
	validators   []Validator[T]
	strict       bool // Fail on unknown keys (default: true)
}

// NewLoader creates a Loader with no sources, transformers, or validators and strict mode enabled.
func NewLoader[T any]() *Loader[T] {
	return &Loader[T]{
		sources:      make([]Source, 0),
		transformers: make([]Transformer[T], 0),
		validators:   make([]Validator[T], 0),
		strict:       true, // Default to strict mode
	}
}

// WithSource adds a source. Sources are processed in order (later override earlier).
func (l *Loader[T]) WithSource(src Source) *Loader[T] {
	l.sources = append(l.sources, src)
	return l
}

// WithTransformer adds a typed transform (executed after binding and before tag-based validation).
func (l *Loader[T]) WithTransformer(t Transformer[T]) *Loader[T] {
	l.transformers = append(l.transformers, t)
	return l
}

// WithTransformerFunc adds a typed transform function (executed after binding and before tag-based validation).
// This is a convenience wrapper around WithTransformer(TransformerFunc[T](fn)).
func (l *Loader[T]) WithTransformerFunc(fn func(context.Context, *T) error) *Loader[T] {
	return l.WithTransformer(TransformerFunc[T](fn))
}

// WithValidator adds a custom validator (executed after transformers and tag-based validation).
func (l *Loader[T]) WithValidator(v Validator[T]) *Loader[T] {
	l.validators = append(l.validators, v)
	return l
}

// Strict controls whether unknown keys cause errors. Default: true.
func (l *Loader[T]) Strict(strict bool) *Loader[T] {
	l.strict = strict
	return l
}

// Load loads, merges, binds, and validates configuration from all sources.
// Returns populated config or ValidationError with all field errors.
func (l *Loader[T]) Load(ctx context.Context) (*T, error) {
	cfg, _, err := l.loadInternal(ctx, true)
	return cfg, err
}

// LoadWithProvenance loads configuration and returns field provenance explicitly.
// Unlike Load, it does not store provenance in the global provenance map.
func (l *Loader[T]) LoadWithProvenance(ctx context.Context) (*T, *Provenance, error) {
	return l.loadInternal(ctx, false)
}

func (l *Loader[T]) loadInternal(ctx context.Context, store bool) (*T, *Provenance, error) {
	rootType := configType[T]()

	// Step 1: Validate the schema before any source I/O.
	schemaErrors := getSchemaValidationErrors(rootType)
	if len(schemaErrors) > 0 {
		return nil, nil, &ValidationError{FieldErrors: schemaErrors}
	}

	// Step 2: Load from all sources and merge
	mergedData := make(map[string]mergedEntry)

	for _, source := range l.sources {
		var data map[string]any
		var originalKeys map[string]string
		var err error

		// Check if source implements SourceWithKeys for better provenance
		if sourceWithKeys, ok := source.(SourceWithKeys); ok {
			data, originalKeys, err = sourceWithKeys.LoadWithKeys(ctx)
		} else {
			data, err = source.Load(ctx)
			originalKeys = nil
		}

		if err != nil {
			return nil, nil, fmt.Errorf("load source %s: %w", source.Name(), err)
		}

		// Merge data into mergedData map
		// Later sources override earlier ones
		for key, value := range data {
			// Normalize key to lowercase dot-separated path
			normalizedKey := strings.ToLower(key)

			// Determine source key for provenance
			sourceKey := source.Name()
			if originalKeys != nil {
				if origKey, ok := originalKeys[normalizedKey]; ok {
					// For env vars, use the full variable name (e.g., "env:APP_DATABASE__PASSWORD")
					// For files, just use the filename (e.g., "file:config.yaml")
					if strings.HasPrefix(source.Name(), "env") {
						sourceKey = "env:" + origKey
					}
					// For files, sourceKey remains just source.Name() (e.g., "file:config.yaml")
				}
			}

			mergedData[normalizedKey] = mergedEntry{
				value:      value,
				sourceName: source.Name(),
				sourceKey:  sourceKey,
			}
		}
	}

	// Step 3: In strict mode, detect unknown keys
	if l.strict {
		strictMetadataCache := make(map[strictValidationMetadataKey]strictValidationMetadata)
		strictMetadata := getStrictValidationMetadata(rootType, "", strictMetadataCache)

		// Check for unknown keys
		var unknownKeyErrors []FieldError
		for key, entry := range mergedData {
			if !strictMetadata.validKeys[key] && !matchesDynamicMapKeyPattern(key, strictMetadata.dynamicMapKeyPatterns) {
				unknownKeyErrors = append(unknownKeyErrors, FieldError{
					FieldPath: key,
					Code:      ErrCodeUnknownKey,
					Message:   "unknown configuration key (strict mode)",
				})
				continue
			}

			if fieldType, ok := strictMetadata.validKeyTypes[key]; ok {
				unknownKeyErrors = append(unknownKeyErrors, collectStructuredUnknownKeyErrors(entry.value, fieldType, key, strictMetadataCache)...)
			}
		}

		if len(unknownKeyErrors) > 0 {
			return nil, nil, &ValidationError{FieldErrors: unknownKeyErrors}
		}
	}

	// Step 4: Create zero instance of T
	cfg := new(T)
	cfgValue := reflect.ValueOf(cfg).Elem()

	// Step 5: Bind struct fields from merged data
	var provenanceFields []FieldProvenance
	presentFields := make(map[string]bool)
	bindErrors := bindStructWithPresence(cfgValue, mergedData, &provenanceFields, presentFields, "", "")

	allErrors := append([]FieldError{}, bindErrors...)

	// Step 6: Run typed transforms after successful binding so transformers operate on converted values.
	if len(bindErrors) == 0 {
		for i, transformer := range l.transformers {
			err := transformer.Transform(ctx, cfg)
			if err != nil {
				if valErr, ok := err.(*ValidationError); ok {
					allErrors = append(allErrors, valErr.FieldErrors...)
				} else {
					return nil, nil, fmt.Errorf("transformer %d failed: %w", i, err)
				}
			}
		}
	}

	// Step 7: Validate struct (tag-based validation)
	// Required checks use presence data captured before conversion.
	validationErrors := validateStructWithPresence(cfgValue, presentFields)
	allErrors = append(allErrors, validationErrors...)

	// Step 8: Run custom validators
	for i, validator := range l.validators {
		err := validator.Validate(ctx, cfg)
		if err != nil {
			// Check if it's a ValidationError
			if valErr, ok := err.(*ValidationError); ok {
				allErrors = append(allErrors, valErr.FieldErrors...)
			} else {
				// Wrap other errors as validation errors
				return nil, nil, fmt.Errorf("validator %d failed: %w", i, err)
			}
		}
	}

	// Step 9: Return error if any validation failed
	if len(allErrors) > 0 {
		return nil, nil, &ValidationError{FieldErrors: allErrors}
	}

	// Step 10: Build provenance for the config instance
	prov := &Provenance{Fields: provenanceFields}
	if store {
		storeProvenance(cfg, prov)
	}

	// Step 11: Return the loaded configuration
	return cfg, prov, nil
}

// Watch monitors sources for changes and auto-reloads configuration.
// Returns: snapshots channel, errors channel, initial load error.
// Changes are debounced (100ms). Built-in sources don't support watching yet.
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

	// Walk through cached exported field metadata.
	for _, meta := range getStructFieldMeta(t) {
		field := meta.field
		tagCfg := meta.tagCfg

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

func collectValidKeyTypes(t reflect.Type, prefix string) map[string]reflect.Type {
	validKeyTypes := make(map[string]reflect.Type)

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return validKeyTypes
	}

	for _, meta := range getStructFieldMeta(t) {
		field := meta.field
		tagCfg := meta.tagCfg
		keyPath := determineKeyPath(field.Name, tagCfg, prefix)
		fieldType := field.Type

		validKeyTypes[keyPath] = fieldType

		if isOptionalType(fieldType) {
			innerType := fieldType.Field(0).Type
			if innerType.Kind() == reflect.Struct {
				for nestedKey, nestedType := range collectValidKeyTypes(innerType, keyPath) {
					validKeyTypes[nestedKey] = nestedType
				}
			}
			continue
		}

		if fieldType.Kind() != reflect.Struct || fieldType.PkgPath() == "time" {
			continue
		}

		nestedPrefix := keyPath
		if tagCfg.prefix != "" {
			nestedPrefix = tagCfg.prefix
		}
		for nestedKey, nestedType := range collectValidKeyTypes(fieldType, nestedPrefix) {
			validKeyTypes[nestedKey] = nestedType
		}
	}

	return validKeyTypes
}

type strictValidationMetadata struct {
	validKeys             map[string]bool
	validKeyTypes         map[string]reflect.Type
	dynamicMapKeyPatterns []string
}

type strictValidationMetadataKey struct {
	targetType reflect.Type
	prefix     string
}

func getStrictValidationMetadata(
	targetType reflect.Type,
	prefix string,
	cache map[strictValidationMetadataKey]strictValidationMetadata,
) strictValidationMetadata {
	for targetType.Kind() == reflect.Ptr {
		targetType = targetType.Elem()
	}

	cacheKey := strictValidationMetadataKey{
		targetType: targetType,
		prefix:     prefix,
	}
	if metadata, ok := cache[cacheKey]; ok {
		return metadata
	}

	metadata := strictValidationMetadata{
		validKeys:             collectValidKeys(targetType, prefix),
		validKeyTypes:         collectValidKeyTypes(targetType, prefix),
		dynamicMapKeyPatterns: collectDynamicMapKeyPatterns(targetType, prefix),
	}
	cache[cacheKey] = metadata
	return metadata
}

func collectDynamicMapKeyPatterns(t reflect.Type, prefix string) []string {
	patternSet := make(map[string]struct{})

	var walk func(reflect.Type, string)
	walk = func(currType reflect.Type, currPrefix string) {
		if currType.Kind() == reflect.Ptr {
			currType = currType.Elem()
		}
		if currType.Kind() != reflect.Struct {
			return
		}

		for _, meta := range getStructFieldMeta(currType) {
			field := meta.field
			tagCfg := meta.tagCfg
			keyPath := determineKeyPath(field.Name, tagCfg, currPrefix)

			fieldType := field.Type
			unwrappedType := fieldType
			if isOptionalType(unwrappedType) {
				unwrappedType = unwrappedType.Field(0).Type
			}

			if unwrappedType.Kind() == reflect.Map && unwrappedType.Key().Kind() == reflect.String {
				for _, pattern := range collectMapValueKeyPatterns(keyPath, unwrappedType.Elem()) {
					patternSet[pattern] = struct{}{}
				}
			}

			if isOptionalType(fieldType) {
				innerType := fieldType.Field(0).Type
				if innerType.Kind() == reflect.Struct {
					walk(innerType, keyPath)
				}
				continue
			}

			if fieldType.Kind() != reflect.Struct || fieldType.PkgPath() == "time" {
				continue
			}

			nestedPrefix := keyPath
			if tagCfg.prefix != "" {
				nestedPrefix = tagCfg.prefix
			}
			walk(fieldType, nestedPrefix)
		}
	}

	walk(t, prefix)

	patterns := make([]string, 0, len(patternSet))
	for pattern := range patternSet {
		patterns = append(patterns, pattern)
	}

	return patterns
}

func collectStructuredUnknownKeyErrors(
	rawValue any,
	targetType reflect.Type,
	keyPath string,
	metadataCache map[strictValidationMetadataKey]strictValidationMetadata,
) []FieldError {
	if rawValue == nil {
		return nil
	}

	targetType = unwrapStrictType(targetType)

	switch targetType.Kind() {
	case reflect.Struct:
		if targetType == timeType || targetType == durationType {
			return nil
		}

		rawMap, ok := rawStringMap(rawValue)
		if !ok {
			return nil
		}

		return collectUnknownStructMapKeys(rawMap, targetType, keyPath, metadataCache)

	case reflect.Slice, reflect.Array:
		elemType := unwrapStrictType(targetType.Elem())
		if elemType.Kind() != reflect.Struct || elemType == timeType || elemType == durationType {
			return nil
		}

		rawRef := reflect.ValueOf(rawValue)
		if rawRef.Kind() != reflect.Slice && rawRef.Kind() != reflect.Array {
			return nil
		}

		var fieldErrors []FieldError
		for i := 0; i < rawRef.Len(); i++ {
			elemMap, ok := rawStringMap(rawRef.Index(i).Interface())
			if !ok {
				continue
			}

			fieldErrors = append(fieldErrors, collectUnknownStructMapKeys(elemMap, elemType, fmt.Sprintf("%s.%d", keyPath, i), metadataCache)...)
		}
		return fieldErrors

	case reflect.Map:
		if targetType.Key().Kind() != reflect.String {
			return nil
		}

		elemType := unwrapStrictType(targetType.Elem())
		if elemType.Kind() != reflect.Struct || elemType == timeType || elemType == durationType {
			return nil
		}

		rawMapRef := reflect.ValueOf(rawValue)
		if rawMapRef.Kind() != reflect.Map {
			return nil
		}

		var fieldErrors []FieldError
		for _, rawKey := range rawMapRef.MapKeys() {
			if rawKey.Kind() != reflect.String {
				continue
			}

			elemMap, ok := rawStringMap(rawMapRef.MapIndex(rawKey).Interface())
			if !ok {
				continue
			}

			fieldErrors = append(fieldErrors, collectUnknownStructMapKeys(elemMap, elemType, keyPath+"."+strings.ToLower(rawKey.String()), metadataCache)...)
		}
		return fieldErrors
	}

	return nil
}

func collectUnknownStructMapKeys(
	rawMap map[string]any,
	targetType reflect.Type,
	keyPath string,
	metadataCache map[strictValidationMetadataKey]strictValidationMetadata,
) []FieldError {
	strictMetadata := getStrictValidationMetadata(targetType, "", metadataCache)

	var fieldErrors []FieldError
	for rawKey, rawValue := range rawMap {
		normalizedKey := strings.ToLower(rawKey)
		if !strictMetadata.validKeys[normalizedKey] && !matchesDynamicMapKeyPattern(normalizedKey, strictMetadata.dynamicMapKeyPatterns) {
			fieldErrors = append(fieldErrors, FieldError{
				FieldPath: keyPath + "." + normalizedKey,
				Code:      ErrCodeUnknownKey,
				Message:   "unknown configuration key (strict mode)",
			})
			continue
		}

		if nestedType, ok := strictMetadata.validKeyTypes[normalizedKey]; ok {
			fieldErrors = append(fieldErrors, collectStructuredUnknownKeyErrors(rawValue, nestedType, keyPath+"."+normalizedKey, metadataCache)...)
		}
	}

	return fieldErrors
}

func unwrapStrictType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if isOptionalType(t) {
		return unwrapStrictType(t.Field(0).Type)
	}
	return t
}

func rawStringMap(rawValue any) (map[string]any, bool) {
	rawMap, ok := rawValue.(map[string]any)
	if ok {
		return rawMap, true
	}

	rawRef := reflect.ValueOf(rawValue)
	if !rawRef.IsValid() || rawRef.Kind() != reflect.Map {
		return nil, false
	}

	result := make(map[string]any, rawRef.Len())
	for _, rawKey := range rawRef.MapKeys() {
		if rawKey.Kind() != reflect.String {
			return nil, false
		}
		result[rawKey.String()] = rawRef.MapIndex(rawKey).Interface()
	}

	return result, true
}

func collectMapValueKeyPatterns(baseKey string, valueType reflect.Type) []string {
	if valueType.Kind() == reflect.Ptr {
		valueType = valueType.Elem()
	}

	if isOptionalType(valueType) {
		valueType = valueType.Field(0).Type
	}

	if valueType.Kind() == reflect.Struct && valueType.PkgPath() != "time" {
		patterns := make([]string, 0)
		for key := range collectValidKeys(valueType, baseKey+".*") {
			patterns = append(patterns, key)
		}
		return patterns
	}

	return []string{baseKey + ".*"}
}

func matchesDynamicMapKeyPattern(key string, patterns []string) bool {
	for _, pattern := range patterns {
		if matchDotKeyPattern(pattern, key) {
			return true
		}
	}

	return false
}

func matchDotKeyPattern(pattern string, key string) bool {
	patternParts := strings.Split(pattern, ".")
	keyParts := strings.Split(key, ".")
	if len(patternParts) != len(keyParts) {
		return false
	}

	for i := range patternParts {
		if patternParts[i] == "*" {
			continue
		}
		if patternParts[i] != keyParts[i] {
			return false
		}
	}

	return true
}

// watchLoop is the main goroutine that monitors sources for changes and reloads configuration.
// It handles debouncing, thread-safe snapshot emission, and cleanup.
func (l *Loader[T]) watchLoop(ctx context.Context, initialCfg *T, snapshotCh chan<- Snapshot[T], errorCh chan<- error) {
	defer close(snapshotCh)
	defer close(errorCh)

	// Emit initial snapshot
	currentVersion := int64(1)
	currentCfg := initialCfg
	snapshotCh <- Snapshot[T]{
		Config:   initialCfg,
		Version:  currentVersion,
		LoadedAt: time.Now(),
		Source:   "initial",
	}

	// Start watching all sources
	changeChannels := make([]<-chan ChangeEvent, 0, len(l.sources))
	cancelFuncs := make([]context.CancelFunc, 0, len(l.sources))

	for _, source := range l.sources {
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
			case errorCh <- fmt.Errorf("watch source %s: %w", source.Name(), err):
			case <-ctx.Done():
				cancel()
				return
			}
			cancel()
			continue
		}

		changeChannels = append(changeChannels, changeCh)
	}
	defer func() {
		for _, cancel := range cancelFuncs {
			cancel()
		}
	}()

	// If no sources support watching, we're done
	if len(changeChannels) == 0 {
		return
	}

	// Create a debounce timer
	var debounceTimer *time.Timer
	var debounceCh <-chan time.Time
	pendingReload := false
	pendingCause := ""
	const debounceDelay = 100 * time.Millisecond
	defer func() {
		if debounceTimer == nil {
			return
		}
		if !debounceTimer.Stop() {
			select {
			case <-debounceTimer.C:
			default:
			}
		}
	}()

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
			event, ok := value.Interface().(ChangeEvent)
			if !ok {
				continue
			}

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
		if mergedChanges == nil && !pendingReload {
			return
		}

		select {
		case <-ctx.Done():
			return

		case event, ok := <-mergedChanges:
			if !ok {
				// All change channels closed; process any pending debounced reload first.
				mergedChanges = nil
				if !pendingReload {
					return
				}
				continue
			}

			// Debounce: reset timer on each event
			pendingCause = event.Cause
			pendingReload = true
			if debounceTimer == nil {
				debounceTimer = time.NewTimer(debounceDelay)
				debounceCh = debounceTimer.C
				continue
			}

			if !debounceTimer.Stop() {
				select {
				case <-debounceTimer.C:
				default:
				}
			}
			debounceTimer.Reset(debounceDelay)
			debounceCh = debounceTimer.C

		case <-debounceCh:
			debounceCh = nil
			if !pendingReload {
				continue
			}

			// Reload configuration from the latest debounced event.
			cause := pendingCause
			pendingReload = false
			newCfg, err := l.Load(ctx)
			if err != nil {
				// Send error, keep previous config.
				select {
				case errorCh <- fmt.Errorf("reload failed: %w", err):
				case <-ctx.Done():
					return
				}
				continue
			}

			// Emit snapshot first; only then release the superseded provenance.
			snapshot := Snapshot[T]{
				Config:   newCfg,
				Version:  currentVersion + 1,
				LoadedAt: time.Now(),
				Source:   cause,
			}
			select {
			case snapshotCh <- snapshot:
				currentVersion++
				ReleaseProvenance(currentCfg)
				currentCfg = newCfg
			case <-ctx.Done():
				ReleaseProvenance(newCfg)
				return
			}
		}
	}
}
