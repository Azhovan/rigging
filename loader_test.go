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
