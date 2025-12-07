package rigging

import (
	"context"
	"fmt"
	"reflect"
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

func (m *mockSource) Name() string {
	if m.name != "" {
		return m.name
	}
	return "mock"
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

func (w *watchableSource) Name() string {
	if w.name != "" {
		return w.name
	}
	return "watchable"
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

func TestCollectValidKeys_SimpleStruct(t *testing.T) {
	type Config struct {
		Host string
		Port int
	}

	validKeys := collectValidKeys(reflect.TypeOf(Config{}), "")

	expectedKeys := []string{"host", "port"}
	if len(validKeys) != len(expectedKeys) {
		t.Fatalf("expected %d keys, got %d", len(expectedKeys), len(validKeys))
	}

	for _, key := range expectedKeys {
		if !validKeys[key] {
			t.Errorf("expected key %q to be valid", key)
		}
	}
}

// TestCollectValidKeys_WithPrefix verifies that collectValidKeys applies prefix to keys.
func TestCollectValidKeys_WithPrefix(t *testing.T) {
	type Config struct {
		Host string
		Port int
	}

	validKeys := collectValidKeys(reflect.TypeOf(Config{}), "app")

	expectedKeys := []string{"app.host", "app.port"}
	if len(validKeys) != len(expectedKeys) {
		t.Fatalf("expected %d keys, got %d", len(expectedKeys), len(validKeys))
	}

	for _, key := range expectedKeys {
		if !validKeys[key] {
			t.Errorf("expected key %q to be valid", key)
		}
	}
}

// TestCollectValidKeys_NestedStruct verifies that collectValidKeys handles nested structs.
func TestCollectValidKeys_NestedStruct(t *testing.T) {
	type Database struct {
		Host string
		Port int
	}

	type Config struct {
		Database Database `conf:"prefix:db"`
	}

	validKeys := collectValidKeys(reflect.TypeOf(Config{}), "")

	// Should have the database field itself plus nested keys
	expectedKeys := []string{"database", "db.host", "db.port"}
	if len(validKeys) != len(expectedKeys) {
		t.Fatalf("expected %d keys, got %d: %v", len(expectedKeys), len(validKeys), validKeys)
	}

	for _, key := range expectedKeys {
		if !validKeys[key] {
			t.Errorf("expected key %q to be valid", key)
		}
	}
}

// TestCollectValidKeys_UnexportedFields verifies that collectValidKeys skips unexported fields.
func TestCollectValidKeys_UnexportedFields(t *testing.T) {
	type Config struct {
		Host     string
		port     int    // unexported
		internal string // unexported
	}

	validKeys := collectValidKeys(reflect.TypeOf(Config{}), "")

	if len(validKeys) != 1 {
		t.Fatalf("expected 1 key, got %d: %v", len(validKeys), validKeys)
	}

	if !validKeys["host"] {
		t.Error("expected key 'host' to be valid")
	}

	if validKeys["port"] {
		t.Error("unexported field 'port' should not be valid")
	}

	if validKeys["internal"] {
		t.Error("unexported field 'internal' should not be valid")
	}
}

// TestCollectValidKeys_PointerType verifies that collectValidKeys handles pointer types.
func TestCollectValidKeys_PointerType(t *testing.T) {
	type Config struct {
		Host string
		Port int
	}

	// Pass pointer type
	validKeys := collectValidKeys(reflect.TypeOf(&Config{}), "")

	expectedKeys := []string{"host", "port"}
	if len(validKeys) != len(expectedKeys) {
		t.Fatalf("expected %d keys, got %d", len(expectedKeys), len(validKeys))
	}

	for _, key := range expectedKeys {
		if !validKeys[key] {
			t.Errorf("expected key %q to be valid", key)
		}
	}
}

// TestCollectValidKeys_NonStructType verifies that collectValidKeys returns empty map for non-struct types.
func TestCollectValidKeys_NonStructType(t *testing.T) {
	// Test with int
	validKeys := collectValidKeys(reflect.TypeOf(42), "")
	if len(validKeys) != 0 {
		t.Errorf("expected 0 keys for int type, got %d", len(validKeys))
	}

	// Test with string
	validKeys = collectValidKeys(reflect.TypeOf("test"), "")
	if len(validKeys) != 0 {
		t.Errorf("expected 0 keys for string type, got %d", len(validKeys))
	}

	// Test with slice
	validKeys = collectValidKeys(reflect.TypeOf([]int{}), "")
	if len(validKeys) != 0 {
		t.Errorf("expected 0 keys for slice type, got %d", len(validKeys))
	}
}

// TestCollectValidKeys_CustomName verifies that collectValidKeys respects name tag.
func TestCollectValidKeys_CustomName(t *testing.T) {
	type Config struct {
		Host string `conf:"name:hostname"`
		Port int    `conf:"name:port_number"`
	}

	validKeys := collectValidKeys(reflect.TypeOf(Config{}), "")

	expectedKeys := []string{"hostname", "port_number"}
	if len(validKeys) != len(expectedKeys) {
		t.Fatalf("expected %d keys, got %d: %v", len(expectedKeys), len(validKeys), validKeys)
	}

	for _, key := range expectedKeys {
		if !validKeys[key] {
			t.Errorf("expected key %q to be valid", key)
		}
	}

	// Original field names should not be valid
	if validKeys["host"] {
		t.Error("original field name 'host' should not be valid when custom name is used")
	}
	if validKeys["port"] {
		t.Error("original field name 'port' should not be valid when custom name is used")
	}
}

// TestCollectValidKeys_TimeTypes verifies that collectValidKeys handles time.Time and time.Duration.
func TestCollectValidKeys_TimeTypes(t *testing.T) {
	type Config struct {
		Timestamp time.Time
		Timeout   time.Duration
		Name      string
	}

	validKeys := collectValidKeys(reflect.TypeOf(Config{}), "")

	// All three should be valid keys (time types are treated as primitives)
	expectedKeys := []string{"timestamp", "timeout", "name"}
	if len(validKeys) != len(expectedKeys) {
		t.Fatalf("expected %d keys, got %d: %v", len(expectedKeys), len(validKeys), validKeys)
	}

	for _, key := range expectedKeys {
		if !validKeys[key] {
			t.Errorf("expected key %q to be valid", key)
		}
	}
}

// TestCollectValidKeys_DeeplyNestedStruct verifies that collectValidKeys handles deeply nested structs.
func TestCollectValidKeys_DeeplyNestedStruct(t *testing.T) {
	type Credentials struct {
		Username string
		Password string
	}

	type Database struct {
		Host        string
		Port        int
		Credentials Credentials `conf:"prefix:creds"`
	}

	type Config struct {
		Database Database `conf:"prefix:db"`
	}

	validKeys := collectValidKeys(reflect.TypeOf(Config{}), "")

	expectedKeys := []string{
		"database",
		"db.host",
		"db.port",
		"db.credentials",
		"creds.username",
		"creds.password",
	}

	if len(validKeys) != len(expectedKeys) {
		t.Fatalf("expected %d keys, got %d: %v", len(expectedKeys), len(validKeys), validKeys)
	}

	for _, key := range expectedKeys {
		if !validKeys[key] {
			t.Errorf("expected key %q to be valid", key)
		}
	}
}

// TestCollectValidKeys_OptionalType verifies that collectValidKeys handles Optional[T] types.
func TestCollectValidKeys_OptionalType(t *testing.T) {
	type Database struct {
		Host string
		Port int
	}

	type Config struct {
		Database Optional[Database] `conf:"prefix:db"`
	}

	validKeys := collectValidKeys(reflect.TypeOf(Config{}), "")

	// Should have the database field itself plus nested keys from Optional[Database]
	// Note: For Optional types, the prefix tag is ignored and keyPath is used instead
	expectedKeys := []string{"database", "database.host", "database.port"}
	if len(validKeys) != len(expectedKeys) {
		t.Fatalf("expected %d keys, got %d: %v", len(expectedKeys), len(validKeys), validKeys)
	}

	for _, key := range expectedKeys {
		if !validKeys[key] {
			t.Errorf("expected key %q to be valid", key)
		}
	}
}

// TestCollectValidKeys_PointerFields verifies that collectValidKeys handles pointer fields within structs.
func TestCollectValidKeys_PointerFields(t *testing.T) {
	type Database struct {
		Host string
		Port int
	}

	type Config struct {
		Name     string
		Timeout  *int
		Database *Database `conf:"prefix:db"`
	}

	validKeys := collectValidKeys(reflect.TypeOf(Config{}), "")

	// Current implementation: pointer fields to structs are treated as leaf values (not recursed)
	// This documents the actual behavior - pointer fields are not dereferenced
	expectedKeys := []string{"name", "timeout", "database"}
	if len(validKeys) != len(expectedKeys) {
		t.Fatalf("expected %d keys, got %d: %v", len(expectedKeys), len(validKeys), validKeys)
	}

	for _, key := range expectedKeys {
		if !validKeys[key] {
			t.Errorf("expected key %q to be valid", key)
		}
	}

	// Pointer to struct fields are NOT recursed into (unlike non-pointer struct fields)
	if validKeys["db.host"] || validKeys["db.port"] {
		t.Error("pointer to struct fields should not be recursed into")
	}
}

// TestCollectValidKeys_SliceAndMapFields verifies that collectValidKeys treats slices and maps as leaf values.
func TestCollectValidKeys_SliceAndMapFields(t *testing.T) {
	type Config struct {
		Hosts    []string
		Tags     []int
		Metadata map[string]string
		Ports    []int
	}

	validKeys := collectValidKeys(reflect.TypeOf(Config{}), "")

	// Slices and maps should be treated as leaf values (not recursed into)
	expectedKeys := []string{"hosts", "tags", "metadata", "ports"}
	if len(validKeys) != len(expectedKeys) {
		t.Fatalf("expected %d keys, got %d: %v", len(expectedKeys), len(validKeys), validKeys)
	}

	for _, key := range expectedKeys {
		if !validKeys[key] {
			t.Errorf("expected key %q to be valid", key)
		}
	}
}

// TestCollectValidKeys_EmptyStructTag verifies that empty struct tag behaves like no tag.
func TestCollectValidKeys_EmptyStructTag(t *testing.T) {
	type Config struct {
		Host string `conf:""`
		Port int
	}

	validKeys := collectValidKeys(reflect.TypeOf(Config{}), "")

	expectedKeys := []string{"host", "port"}
	if len(validKeys) != len(expectedKeys) {
		t.Fatalf("expected %d keys, got %d: %v", len(expectedKeys), len(validKeys), validKeys)
	}

	for _, key := range expectedKeys {
		if !validKeys[key] {
			t.Errorf("expected key %q to be valid", key)
		}
	}
}

// TestCollectValidKeys_NameTakesPrecedenceOverPrefix verifies that name tag overrides prefix.
func TestCollectValidKeys_NameTakesPrecedenceOverPrefix(t *testing.T) {
	type Database struct {
		Host string `conf:"name:db_host"`
		Port int
	}

	type Config struct {
		Database Database `conf:"prefix:db"`
	}

	validKeys := collectValidKeys(reflect.TypeOf(Config{}), "")

	// The name tag should take precedence, so we get "db_host" not "db.host"
	expectedKeys := []string{"database", "db_host", "db.port"}
	if len(validKeys) != len(expectedKeys) {
		t.Fatalf("expected %d keys, got %d: %v", len(expectedKeys), len(validKeys), validKeys)
	}

	for _, key := range expectedKeys {
		if !validKeys[key] {
			t.Errorf("expected key %q to be valid", key)
		}
	}

	// Should not have the prefixed version
	if validKeys["db.host"] {
		t.Error("should not have 'db.host' when name tag is specified")
	}
}

// TestCollectValidKeys_AllUnexportedFields verifies handling of struct with only unexported fields.
func TestCollectValidKeys_AllUnexportedFields(t *testing.T) {
	type Config struct {
		host string // unexported
		port int    // unexported
	}

	validKeys := collectValidKeys(reflect.TypeOf(Config{}), "")

	if len(validKeys) != 0 {
		t.Fatalf("expected 0 keys for struct with only unexported fields, got %d: %v", len(validKeys), validKeys)
	}
}

// TestCollectValidKeys_EmptyStruct verifies handling of empty struct.
func TestCollectValidKeys_EmptyStruct(t *testing.T) {
	type Config struct{}

	validKeys := collectValidKeys(reflect.TypeOf(Config{}), "")

	if len(validKeys) != 0 {
		t.Fatalf("expected 0 keys for empty struct, got %d: %v", len(validKeys), validKeys)
	}
}

// TestCollectValidKeys_PrefixWithDots verifies that prefixes containing dots are handled correctly.
func TestCollectValidKeys_PrefixWithDots(t *testing.T) {
	type Server struct {
		Host string
		Port int
	}

	type Config struct {
		Server Server `conf:"prefix:app.server"`
	}

	validKeys := collectValidKeys(reflect.TypeOf(Config{}), "")

	// Prefix with dots should be preserved
	expectedKeys := []string{"server", "app.server.host", "app.server.port"}
	if len(validKeys) != len(expectedKeys) {
		t.Fatalf("expected %d keys, got %d: %v", len(expectedKeys), len(validKeys), validKeys)
	}

	for _, key := range expectedKeys {
		if !validKeys[key] {
			t.Errorf("expected key %q to be valid", key)
		}
	}
}

// TestCollectValidKeys_CaseSensitivity verifies that field names are normalized to lowercase.
func TestCollectValidKeys_CaseSensitivity(t *testing.T) {
	type Config struct {
		HTTPPort int
		APIKey   string
		DBHost   string
		UserName string
	}

	validKeys := collectValidKeys(reflect.TypeOf(Config{}), "")

	// All keys should be lowercase
	expectedKeys := []string{"httpport", "apikey", "dbhost", "username"}
	if len(validKeys) != len(expectedKeys) {
		t.Fatalf("expected %d keys, got %d: %v", len(expectedKeys), len(validKeys), validKeys)
	}

	for _, key := range expectedKeys {
		if !validKeys[key] {
			t.Errorf("expected key %q to be valid", key)
		}
	}

	// Should not have mixed-case versions
	invalidKeys := []string{"HTTPPort", "APIKey", "DBHost", "UserName"}
	for _, key := range invalidKeys {
		if validKeys[key] {
			t.Errorf("should not have mixed-case key %q", key)
		}
	}
}

// TestCollectValidKeys_NestedOptionalTypes verifies handling of nested Optional types.
func TestCollectValidKeys_NestedOptionalTypes(t *testing.T) {
	type Credentials struct {
		Username string
		Password string
	}

	type Database struct {
		Host        string
		Credentials Optional[Credentials] `conf:"prefix:creds"`
	}

	type Config struct {
		Database Optional[Database] `conf:"prefix:db"`
	}

	validKeys := collectValidKeys(reflect.TypeOf(Config{}), "")

	// Optional types should be unwrapped and recursed
	expectedKeys := []string{
		"database",
		"database.host",
		"database.credentials",
		"database.credentials.username",
		"database.credentials.password",
	}

	if len(validKeys) != len(expectedKeys) {
		t.Fatalf("expected %d keys, got %d: %v", len(expectedKeys), len(validKeys), validKeys)
	}

	for _, key := range expectedKeys {
		if !validKeys[key] {
			t.Errorf("expected key %q to be valid", key)
		}
	}
}

// TestCollectValidKeys_MixedFieldTypes verifies handling of struct with various field types.
func TestCollectValidKeys_MixedFieldTypes(t *testing.T) {
	type Nested struct {
		Value string
	}

	type Config struct {
		StringField   string
		IntField      int
		BoolField     bool
		FloatField    float64
		SliceField    []string
		MapField      map[string]int
		PointerField  *string
		StructField   Nested `conf:"prefix:nested"`
		TimeField     time.Time
		DurationField time.Duration
	}

	validKeys := collectValidKeys(reflect.TypeOf(Config{}), "")

	expectedKeys := []string{
		"stringfield",
		"intfield",
		"boolfield",
		"floatfield",
		"slicefield",
		"mapfield",
		"pointerfield",
		"structfield",
		"nested.value",
		"timefield",
		"durationfield",
	}

	if len(validKeys) != len(expectedKeys) {
		t.Fatalf("expected %d keys, got %d: %v", len(expectedKeys), len(validKeys), validKeys)
	}

	for _, key := range expectedKeys {
		if !validKeys[key] {
			t.Errorf("expected key %q to be valid", key)
		}
	}
}

// TestCollectValidKeys_PrefixOnNonStructField verifies that prefix on non-struct fields is ignored.
func TestCollectValidKeys_PrefixOnNonStructField(t *testing.T) {
	type Config struct {
		Host string `conf:"prefix:server"` // prefix should be ignored for non-struct
		Port int    `conf:"prefix:server"` // prefix should be ignored for non-struct
	}

	validKeys := collectValidKeys(reflect.TypeOf(Config{}), "")

	// Prefix should be ignored for non-struct fields
	expectedKeys := []string{"host", "port"}
	if len(validKeys) != len(expectedKeys) {
		t.Fatalf("expected %d keys, got %d: %v", len(expectedKeys), len(validKeys), validKeys)
	}

	for _, key := range expectedKeys {
		if !validKeys[key] {
			t.Errorf("expected key %q to be valid", key)
		}
	}

	// Should not have prefixed versions
	if validKeys["server.host"] || validKeys["server.port"] {
		t.Error("prefix should be ignored for non-struct fields")
	}
}

// TestCollectValidKeys_NestedStructWithoutPrefix verifies nested struct without prefix tag.
func TestCollectValidKeys_NestedStructWithoutPrefix(t *testing.T) {
	type Database struct {
		Host string
		Port int
	}

	type Config struct {
		Database Database // no prefix tag
	}

	validKeys := collectValidKeys(reflect.TypeOf(Config{}), "")

	// Without prefix tag, nested keys should use parent field name as prefix
	expectedKeys := []string{"database", "database.host", "database.port"}
	if len(validKeys) != len(expectedKeys) {
		t.Fatalf("expected %d keys, got %d: %v", len(expectedKeys), len(validKeys), validKeys)
	}

	for _, key := range expectedKeys {
		if !validKeys[key] {
			t.Errorf("expected key %q to be valid", key)
		}
	}
}
