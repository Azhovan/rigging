package rigging

import (
	"context"
	"testing"
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
