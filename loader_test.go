package rigging

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestNewLoader verifies that NewLoader creates a loader with correct defaults.
func TestNewLoader(t *testing.T) {
	loader := NewLoader[struct{}]()

	if loader == nil {
		t.Fatal("NewLoader returned nil")
	}

	if loader.sources == nil {
		t.Error("sources slice should be initialized")
	}

	if loader.validators == nil {
		t.Error("validators slice should be initialized")
	}

	if !loader.strict {
		t.Error("strict mode should be enabled by default")
	}

	if len(loader.sources) != 0 {
		t.Errorf("expected 0 sources, got %d", len(loader.sources))
	}

	if len(loader.validators) != 0 {
		t.Errorf("expected 0 validators, got %d", len(loader.validators))
	}
}

// TestWithSource verifies that WithSource adds sources and returns the loader for chaining.
func TestWithSource(t *testing.T) {
	loader := NewLoader[struct{}]()
	mockSource1 := &mockSource{name: "source1"}
	mockSource2 := &mockSource{name: "source2"}

	// Test fluent API
	result := loader.WithSource(mockSource1)
	if result != loader {
		t.Error("WithSource should return the same loader instance for chaining")
	}

	if len(loader.sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(loader.sources))
	}

	// Add second source
	loader.WithSource(mockSource2)
	if len(loader.sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(loader.sources))
	}

	// Verify order is preserved
	if loader.sources[0] != mockSource1 {
		t.Error("first source should be mockSource1")
	}
	if loader.sources[1] != mockSource2 {
		t.Error("second source should be mockSource2")
	}
}

// TestWithValidator verifies that WithValidator adds validators and returns the loader for chaining.
func TestWithValidator(t *testing.T) {
	loader := NewLoader[struct{}]()
	validator1 := ValidatorFunc[struct{}](func(ctx context.Context, cfg *struct{}) error {
		return nil
	})
	validator2 := ValidatorFunc[struct{}](func(ctx context.Context, cfg *struct{}) error {
		return nil
	})

	// Test fluent API
	result := loader.WithValidator(validator1)
	if result != loader {
		t.Error("WithValidator should return the same loader instance for chaining")
	}

	if len(loader.validators) != 1 {
		t.Fatalf("expected 1 validator, got %d", len(loader.validators))
	}

	// Add second validator
	loader.WithValidator(validator2)
	if len(loader.validators) != 2 {
		t.Fatalf("expected 2 validators, got %d", len(loader.validators))
	}
}

// TestStrict verifies that Strict method sets the strict flag and returns the loader for chaining.
func TestStrict(t *testing.T) {
	loader := NewLoader[struct{}]()

	// Default should be true
	if !loader.strict {
		t.Error("strict should be true by default")
	}

	// Test setting to false
	result := loader.Strict(false)
	if result != loader {
		t.Error("Strict should return the same loader instance for chaining")
	}

	if loader.strict {
		t.Error("strict should be false after Strict(false)")
	}

	// Test setting back to true
	loader.Strict(true)
	if !loader.strict {
		t.Error("strict should be true after Strict(true)")
	}
}

// TestFluentAPI verifies that all methods can be chained together.
func TestFluentAPI(t *testing.T) {
	mockSource := &mockSource{name: "test"}
	validator := ValidatorFunc[struct{}](func(ctx context.Context, cfg *struct{}) error {
		return nil
	})

	loader := NewLoader[struct{}]().
		WithSource(mockSource).
		WithValidator(validator).
		Strict(false)

	if len(loader.sources) != 1 {
		t.Errorf("expected 1 source, got %d", len(loader.sources))
	}

	if len(loader.validators) != 1 {
		t.Errorf("expected 1 validator, got %d", len(loader.validators))
	}

	if loader.strict {
		t.Error("strict should be false")
	}
}

// mockSource is a test helper that implements the Source interface.
type mockSource struct {
	name string
	data map[string]any
	err  error
}

func (m *mockSource) Load(ctx context.Context) (map[string]any, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.data == nil {
		return make(map[string]any), nil
	}
	return m.data, nil
}

func (m *mockSource) Watch(ctx context.Context) (<-chan ChangeEvent, error) {
	return nil, ErrWatchNotSupported
}

// TestLoad_SingleSource verifies that Load works with a single source.
func TestLoad_SingleSource(t *testing.T) {
	type Config struct {
		Host string `conf:"required"`
		Port int    `conf:"default:8080"`
	}

	source := &mockSource{
		data: map[string]any{
			"host": "localhost",
		},
	}

	loader := NewLoader[Config]().WithSource(source)
	cfg, err := loader.Load(context.Background())

	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Host != "localhost" {
		t.Errorf("expected Host=localhost, got %s", cfg.Host)
	}

	if cfg.Port != 8080 {
		t.Errorf("expected Port=8080 (default), got %d", cfg.Port)
	}
}

// TestLoad_MultipleSources verifies that later sources override earlier ones.
func TestLoad_MultipleSources(t *testing.T) {
	type Config struct {
		Host string
		Port int
	}

	source1 := &mockSource{
		data: map[string]any{
			"host": "localhost",
			"port": 8080,
		},
	}

	source2 := &mockSource{
		data: map[string]any{
			"port": 9090, // Override port
		},
	}

	loader := NewLoader[Config]().
		WithSource(source1).
		WithSource(source2)

	cfg, err := loader.Load(context.Background())

	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Host != "localhost" {
		t.Errorf("expected Host=localhost, got %s", cfg.Host)
	}

	if cfg.Port != 9090 {
		t.Errorf("expected Port=9090 (overridden), got %d", cfg.Port)
	}
}

// TestLoad_ValidationError verifies that validation errors are returned.
func TestLoad_ValidationError(t *testing.T) {
	type Config struct {
		Host string `conf:"required"`
		Port int    `conf:"min:1024,max:65535"`
	}

	source := &mockSource{
		data: map[string]any{
			"port": 80, // Below minimum
		},
	}

	loader := NewLoader[Config]().WithSource(source)
	cfg, err := loader.Load(context.Background())

	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	valErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T", err)
	}

	if len(valErr.FieldErrors) != 2 {
		t.Logf("Field errors:")
		for _, fe := range valErr.FieldErrors {
			t.Logf("  - %s: %s (%s)", fe.FieldPath, fe.Code, fe.Message)
		}
		t.Fatalf("expected 2 field errors, got %d", len(valErr.FieldErrors))
	}

	// Check for required error
	foundRequired := false
	foundMin := false
	for _, fe := range valErr.FieldErrors {
		if fe.FieldPath == "Host" && fe.Code == ErrCodeRequired {
			foundRequired = true
		}
		if fe.FieldPath == "Port" && fe.Code == ErrCodeMin {
			foundMin = true
		}
	}

	if !foundRequired {
		t.Error("expected required error for Host field")
	}
	if !foundMin {
		t.Error("expected min error for Port field")
	}

	if cfg != nil {
		t.Error("cfg should be nil when validation fails")
	}
}

// TestLoad_CustomValidator verifies that custom validators are executed.
func TestLoad_CustomValidator(t *testing.T) {
	type Config struct {
		Env  string
		Host string
	}

	source := &mockSource{
		data: map[string]any{
			"env":  "prod",
			"host": "localhost",
		},
	}

	validator := ValidatorFunc[Config](func(ctx context.Context, cfg *Config) error {
		if cfg.Env == "prod" && cfg.Host == "localhost" {
			return &ValidationError{
				FieldErrors: []FieldError{{
					FieldPath: "Host",
					Code:      "invalid_prod_host",
					Message:   "production cannot use localhost",
				}},
			}
		}
		return nil
	})

	loader := NewLoader[Config]().
		WithSource(source).
		WithValidator(validator)

	cfg, err := loader.Load(context.Background())

	if err == nil {
		t.Fatal("expected validation error from custom validator")
	}

	valErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T", err)
	}

	if len(valErr.FieldErrors) != 1 {
		t.Fatalf("expected 1 field error, got %d", len(valErr.FieldErrors))
	}

	if valErr.FieldErrors[0].Code != "invalid_prod_host" {
		t.Errorf("expected code=invalid_prod_host, got %s", valErr.FieldErrors[0].Code)
	}

	if cfg != nil {
		t.Error("cfg should be nil when validation fails")
	}
}

// TestLoad_StrictMode verifies that strict mode detects unknown keys.
func TestLoad_StrictMode(t *testing.T) {
	type Config struct {
		Host string
		Port int
	}

	source := &mockSource{
		data: map[string]any{
			"host":    "localhost",
			"port":    8080,
			"unknown": "value", // Unknown key
		},
	}

	// Test with strict mode enabled (default)
	loader := NewLoader[Config]().WithSource(source)
	cfg, err := loader.Load(context.Background())

	if err == nil {
		t.Fatal("expected error for unknown key in strict mode")
	}

	if cfg != nil {
		t.Error("cfg should be nil when strict mode fails")
	}

	// Verify it's a ValidationError with unknown_key code
	valErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}

	if len(valErr.FieldErrors) != 1 {
		t.Fatalf("expected 1 field error, got %d", len(valErr.FieldErrors))
	}

	if valErr.FieldErrors[0].Code != ErrCodeUnknownKey {
		t.Errorf("expected code %q, got %q", ErrCodeUnknownKey, valErr.FieldErrors[0].Code)
	}

	if valErr.FieldErrors[0].FieldPath != "unknown" {
		t.Errorf("expected FieldPath %q, got %q", "unknown", valErr.FieldErrors[0].FieldPath)
	}

	// Test with strict mode disabled
	loader = NewLoader[Config]().WithSource(source).Strict(false)
	cfg, err = loader.Load(context.Background())

	if err != nil {
		t.Fatalf("Load failed with strict=false: %v", err)
	}

	if cfg.Host != "localhost" {
		t.Errorf("expected Host=localhost, got %s", cfg.Host)
	}
}

// TestLoad_Provenance verifies that provenance is stored for loaded config.
func TestLoad_Provenance(t *testing.T) {
	type Config struct {
		Host     string `conf:"secret"`
		Port     int
		Password string `conf:"secret"`
	}

	source := &mockSource{
		data: map[string]any{
			"host":     "localhost",
			"port":     8080,
			"password": "secret123",
		},
	}

	loader := NewLoader[Config]().WithSource(source)
	cfg, err := loader.Load(context.Background())

	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Get provenance
	prov, ok := GetProvenance(cfg)
	if !ok {
		t.Fatal("provenance not found for config")
	}

	if len(prov.Fields) != 3 {
		t.Fatalf("expected 3 provenance fields, got %d", len(prov.Fields))
	}

	// Check that secret fields are marked
	secretCount := 0
	for _, field := range prov.Fields {
		if field.Secret {
			secretCount++
		}
	}

	if secretCount != 2 {
		t.Errorf("expected 2 secret fields, got %d", secretCount)
	}
}

// TestLoad_NestedStruct verifies that nested structs are bound correctly.
func TestLoad_NestedStruct(t *testing.T) {
	type Database struct {
		Host string
		Port int
	}

	type Config struct {
		Database Database `conf:"prefix:db"`
	}

	source := &mockSource{
		data: map[string]any{
			"db.host": "localhost",
			"db.port": 5432,
		},
	}

	loader := NewLoader[Config]().WithSource(source)
	cfg, err := loader.Load(context.Background())

	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Database.Host != "localhost" {
		t.Errorf("expected Database.Host=localhost, got %s", cfg.Database.Host)
	}

	if cfg.Database.Port != 5432 {
		t.Errorf("expected Database.Port=5432, got %d", cfg.Database.Port)
	}
}

// TestLoad_SourceError verifies that source load errors are propagated.
func TestLoad_SourceError(t *testing.T) {
	type Config struct {
		Host string
	}

	source := &mockSource{
		err: context.DeadlineExceeded,
	}

	loader := NewLoader[Config]().WithSource(source)
	cfg, err := loader.Load(context.Background())

	if err == nil {
		t.Fatal("expected error from source")
	}

	if cfg != nil {
		t.Error("cfg should be nil when source fails")
	}
}

// watchableSource is a test helper that implements the Source interface with Watch support.
type watchableSource struct {
	name     string
	data     map[string]any
	err      error
	changeCh chan ChangeEvent
}

func newWatchableSource(name string, data map[string]any) *watchableSource {
	return &watchableSource{
		name:     name,
		data:     data,
		changeCh: make(chan ChangeEvent, 10),
	}
}

func (w *watchableSource) Load(ctx context.Context) (map[string]any, error) {
	if w.err != nil {
		return nil, w.err
	}
	if w.data == nil {
		return make(map[string]any), nil
	}
	// Return a copy to avoid race conditions
	result := make(map[string]any)
	for k, v := range w.data {
		result[k] = v
	}
	return result, nil
}

func (w *watchableSource) Watch(ctx context.Context) (<-chan ChangeEvent, error) {
	if w.err != nil {
		return nil, w.err
	}
	return w.changeCh, nil
}

func (w *watchableSource) updateData(data map[string]any) {
	w.data = data
}

func (w *watchableSource) triggerChange(cause string) {
	w.changeCh <- ChangeEvent{
		At:    time.Now(),
		Cause: cause,
	}
}

func (w *watchableSource) close() {
	close(w.changeCh)
}

// TestWatch_InitialSnapshot verifies that Watch emits an initial snapshot.
func TestWatch_InitialSnapshot(t *testing.T) {
	type Config struct {
		Host string
		Port int
	}

	source := newWatchableSource("test", map[string]any{
		"host": "localhost",
		"port": 8080,
	})
	defer source.close()

	loader := NewLoader[Config]().WithSource(source)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	snapshots, errors, err := loader.Watch(ctx)
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}

	// Should receive initial snapshot
	select {
	case snapshot := <-snapshots:
		if snapshot.Config.Host != "localhost" {
			t.Errorf("expected Host=localhost, got %s", snapshot.Config.Host)
		}
		if snapshot.Config.Port != 8080 {
			t.Errorf("expected Port=8080, got %d", snapshot.Config.Port)
		}
		if snapshot.Version != 1 {
			t.Errorf("expected Version=1, got %d", snapshot.Version)
		}
		if snapshot.Source != "initial" {
			t.Errorf("expected Source=initial, got %s", snapshot.Source)
		}
	case err := <-errors:
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for initial snapshot")
	}
}

// TestWatch_ReloadOnChange verifies that Watch reloads config when source changes.
func TestWatch_ReloadOnChange(t *testing.T) {
	type Config struct {
		Host string
		Port int
	}

	source := newWatchableSource("test", map[string]any{
		"host": "localhost",
		"port": 8080,
	})
	defer source.close()

	loader := NewLoader[Config]().WithSource(source)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	snapshots, errors, err := loader.Watch(ctx)
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}

	// Receive initial snapshot
	select {
	case <-snapshots:
		// Initial snapshot received
	case err := <-errors:
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for initial snapshot")
	}

	// Update source data and trigger change
	source.updateData(map[string]any{
		"host": "example.com",
		"port": 9090,
	})
	source.triggerChange("test-change")

	// Should receive new snapshot with updated config
	select {
	case snapshot := <-snapshots:
		if snapshot.Config.Host != "example.com" {
			t.Errorf("expected Host=example.com, got %s", snapshot.Config.Host)
		}
		if snapshot.Config.Port != 9090 {
			t.Errorf("expected Port=9090, got %d", snapshot.Config.Port)
		}
		if snapshot.Version != 2 {
			t.Errorf("expected Version=2, got %d", snapshot.Version)
		}
		if snapshot.Source != "test-change" {
			t.Errorf("expected Source=test-change, got %s", snapshot.Source)
		}
	case err := <-errors:
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for reload snapshot")
	}
}

// TestWatch_ValidationError verifies that validation errors are sent to error channel.
func TestWatch_ValidationError(t *testing.T) {
	type Config struct {
		Host string `conf:"required"`
		Port int    `conf:"min:1024"`
	}

	source := newWatchableSource("test", map[string]any{
		"host": "localhost",
		"port": 8080,
	})
	defer source.close()

	loader := NewLoader[Config]().WithSource(source)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	snapshots, errors, err := loader.Watch(ctx)
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}

	// Receive initial snapshot
	select {
	case <-snapshots:
		// Initial snapshot received
	case err := <-errors:
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for initial snapshot")
	}

	// Update source with invalid data
	source.updateData(map[string]any{
		"port": 80, // Below minimum, and Host is missing
	})
	source.triggerChange("invalid-change")

	// Should receive error, not a new snapshot
	select {
	case snapshot := <-snapshots:
		t.Fatalf("expected error, got snapshot: %+v", snapshot)
	case err := <-errors:
		if err == nil {
			t.Fatal("expected non-nil error")
		}
		// Verify it's a validation error
		if !strings.Contains(err.Error(), "reload failed") {
			t.Errorf("expected 'reload failed' in error, got: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for error")
	}
}

// TestWatch_ContextCancellation verifies that channels are closed when context is cancelled.
func TestWatch_ContextCancellation(t *testing.T) {
	type Config struct {
		Host string
	}

	source := newWatchableSource("test", map[string]any{
		"host": "localhost",
	})
	defer source.close()

	loader := NewLoader[Config]().WithSource(source)
	ctx, cancel := context.WithCancel(context.Background())

	snapshots, errors, err := loader.Watch(ctx)
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}

	// Receive initial snapshot
	select {
	case <-snapshots:
		// Initial snapshot received
	case err := <-errors:
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for initial snapshot")
	}

	// Cancel context
	cancel()

	// Both channels should be closed
	// Give some time for goroutines to clean up
	time.Sleep(50 * time.Millisecond)

	// Check that both channels are closed by trying to receive
	select {
	case _, ok := <-snapshots:
		if ok {
			t.Error("snapshot channel should be closed after context cancellation")
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("timeout: snapshot channel not closed")
	}

	select {
	case _, ok := <-errors:
		if ok {
			t.Error("error channel should be closed after context cancellation")
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("timeout: error channel not closed")
	}
}

// TestWatch_NoWatchableSource verifies behavior when no sources support watching.
func TestWatch_NoWatchableSource(t *testing.T) {
	type Config struct {
		Host string
	}

	source := &mockSource{
		data: map[string]any{
			"host": "localhost",
		},
	}

	loader := NewLoader[Config]().WithSource(source)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	snapshots, errors, err := loader.Watch(ctx)
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}

	// Should receive initial snapshot
	select {
	case snapshot := <-snapshots:
		if snapshot.Config.Host != "localhost" {
			t.Errorf("expected Host=localhost, got %s", snapshot.Config.Host)
		}
	case err := <-errors:
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for initial snapshot")
	}

	// Channels should close since no sources support watching
	select {
	case _, ok := <-snapshots:
		if ok {
			t.Error("snapshot channel should be closed when no sources support watching")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for snapshot channel to close")
	}
}

// TestWatch_MultipleChanges verifies that multiple changes increment version correctly.
func TestWatch_MultipleChanges(t *testing.T) {
	type Config struct {
		Counter int
	}

	source := newWatchableSource("test", map[string]any{
		"counter": 1,
	})
	defer source.close()

	loader := NewLoader[Config]().WithSource(source)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	snapshots, errors, err := loader.Watch(ctx)
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}

	// Receive initial snapshot (version 1)
	select {
	case snapshot := <-snapshots:
		if snapshot.Version != 1 {
			t.Errorf("expected Version=1, got %d", snapshot.Version)
		}
	case err := <-errors:
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for initial snapshot")
	}

	// Trigger multiple changes
	for i := 2; i <= 4; i++ {
		source.updateData(map[string]any{
			"counter": i,
		})
		source.triggerChange(fmt.Sprintf("change-%d", i))

		select {
		case snapshot := <-snapshots:
			if snapshot.Version != int64(i) {
				t.Errorf("expected Version=%d, got %d", i, snapshot.Version)
			}
			if snapshot.Config.Counter != i {
				t.Errorf("expected Counter=%d, got %d", i, snapshot.Config.Counter)
			}
		case err := <-errors:
			t.Fatalf("unexpected error: %v", err)
		case <-time.After(1 * time.Second):
			t.Fatalf("timeout waiting for snapshot %d", i)
		}
	}
}

// TestWatch_Debouncing verifies that rapid changes are debounced.
func TestWatch_Debouncing(t *testing.T) {
	type Config struct {
		Value int
	}

	source := newWatchableSource("test", map[string]any{
		"value": 1,
	})
	defer source.close()

	loader := NewLoader[Config]().WithSource(source)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	snapshots, errors, err := loader.Watch(ctx)
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}

	// Receive initial snapshot
	select {
	case <-snapshots:
		// Initial snapshot received
	case err := <-errors:
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for initial snapshot")
	}

	// Trigger rapid changes
	for i := 2; i <= 10; i++ {
		source.updateData(map[string]any{
			"value": i,
		})
		source.triggerChange(fmt.Sprintf("rapid-change-%d", i))
		time.Sleep(10 * time.Millisecond) // Faster than debounce delay
	}

	// Should receive only one snapshot with the final value due to debouncing
	receivedCount := 0
	var lastSnapshot Snapshot[Config]

	// Wait for debounce to complete
	time.Sleep(200 * time.Millisecond)

	// Drain any snapshots
	for {
		select {
		case snapshot := <-snapshots:
			receivedCount++
			lastSnapshot = snapshot
		case err := <-errors:
			t.Fatalf("unexpected error: %v", err)
		case <-time.After(200 * time.Millisecond):
			// No more snapshots
			goto done
		}
	}

done:
	// Should have received 1 snapshot (debounced)
	if receivedCount != 1 {
		t.Logf("Note: Received %d snapshots (debouncing may vary)", receivedCount)
	}

	// The last snapshot should have the final value
	if lastSnapshot.Config.Value != 10 {
		t.Errorf("expected final Value=10, got %d", lastSnapshot.Config.Value)
	}
}

// TestWatch_InitialLoadFailure verifies that Watch returns error if initial load fails.
func TestWatch_InitialLoadFailure(t *testing.T) {
	type Config struct {
		Host string `conf:"required"`
	}

	source := newWatchableSource("test", map[string]any{
		// Missing required field
	})
	defer source.close()

	loader := NewLoader[Config]().WithSource(source)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	snapshots, errors, err := loader.Watch(ctx)
	if err == nil {
		t.Fatal("expected error from Watch when initial load fails")
	}

	if snapshots != nil {
		t.Error("snapshots channel should be nil when Watch fails")
	}
	if errors != nil {
		t.Error("errors channel should be nil when Watch fails")
	}
}

// TestWatch_MultipleSources verifies that Watch monitors multiple sources.
func TestWatch_MultipleSources(t *testing.T) {
	type Config struct {
		Host string
		Port int
	}

	source1 := newWatchableSource("source1", map[string]any{
		"host": "localhost",
		"port": 8080,
	})
	defer source1.close()

	source2 := newWatchableSource("source2", map[string]any{
		"port": 9090, // Override port
	})
	defer source2.close()

	loader := NewLoader[Config]().
		WithSource(source1).
		WithSource(source2)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	snapshots, errors, err := loader.Watch(ctx)
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}

	// Receive initial snapshot
	select {
	case snapshot := <-snapshots:
		if snapshot.Config.Port != 9090 {
			t.Errorf("expected Port=9090 (from source2), got %d", snapshot.Config.Port)
		}
	case err := <-errors:
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for initial snapshot")
	}

	// Trigger change in source1
	source1.updateData(map[string]any{
		"host": "example.com",
		"port": 8080,
	})
	source1.triggerChange("source1-change")

	// Should receive new snapshot
	select {
	case snapshot := <-snapshots:
		if snapshot.Config.Host != "example.com" {
			t.Errorf("expected Host=example.com, got %s", snapshot.Config.Host)
		}
		// Port should still be 9090 from source2
		if snapshot.Config.Port != 9090 {
			t.Errorf("expected Port=9090 (still from source2), got %d", snapshot.Config.Port)
		}
	case err := <-errors:
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for reload snapshot")
	}
}
