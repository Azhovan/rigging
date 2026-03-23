package rigging

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func requireValidationError(t *testing.T, err error) *ValidationError {
	t.Helper()

	valErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T", err)
	}

	return valErr
}

func requireSingleFieldError(t *testing.T, err error) FieldError {
	t.Helper()

	valErr := requireValidationError(t, err)
	if len(valErr.FieldErrors) != 1 {
		t.Fatalf("expected 1 field error, got %d: %#v", len(valErr.FieldErrors), valErr.FieldErrors)
	}

	return valErr.FieldErrors[0]
}

func TestLoad_InvalidTag_UnknownDirective(t *testing.T) {
	type Config struct {
		Host string `conf:"unknown:value"`
	}

	_, err := NewLoader[Config]().Load(context.Background())
	fieldErr := requireSingleFieldError(t, err)

	if fieldErr.FieldPath != "Host" {
		t.Fatalf("expected FieldPath=Host, got %s", fieldErr.FieldPath)
	}
	if fieldErr.Code != ErrCodeInvalidTag {
		t.Fatalf("expected Code=%s, got %s", ErrCodeInvalidTag, fieldErr.Code)
	}
	if fieldErr.Message != `unknown directive "unknown"` {
		t.Fatalf("unexpected message: %s", fieldErr.Message)
	}
}

func TestLoad_InvalidTag_InvalidRequiredValue(t *testing.T) {
	type Config struct {
		Host string `conf:"required:maybe"`
	}

	_, err := NewLoader[Config]().Load(context.Background())
	fieldErr := requireSingleFieldError(t, err)

	if fieldErr.Code != ErrCodeInvalidTag {
		t.Fatalf("expected Code=%s, got %s", ErrCodeInvalidTag, fieldErr.Code)
	}
	if fieldErr.Message != `invalid required value "maybe": use true or false` {
		t.Fatalf("unexpected message: %s", fieldErr.Message)
	}
}

func TestLoad_InvalidTag_InvalidSecretValue(t *testing.T) {
	type Config struct {
		Host string `conf:"secret:maybe"`
	}

	_, err := NewLoader[Config]().Load(context.Background())
	fieldErr := requireSingleFieldError(t, err)

	if fieldErr.Code != ErrCodeInvalidTag {
		t.Fatalf("expected Code=%s, got %s", ErrCodeInvalidTag, fieldErr.Code)
	}
	if fieldErr.Message != `invalid secret value "maybe": use true or false` {
		t.Fatalf("unexpected message: %s", fieldErr.Message)
	}
}

func TestLoad_InvalidTag_MinMaxConstraints(t *testing.T) {
	t.Run("int min", func(t *testing.T) {
		type Config struct {
			Port int `conf:"min:abc"`
		}

		_, err := NewLoader[Config]().Load(context.Background())
		fieldErr := requireSingleFieldError(t, err)
		if fieldErr.Message != `invalid min constraint "abc": must be a valid integer` {
			t.Fatalf("unexpected message: %s", fieldErr.Message)
		}
	})

	t.Run("uint max", func(t *testing.T) {
		type Config struct {
			Port uint `conf:"max:abc"`
		}

		_, err := NewLoader[Config]().Load(context.Background())
		fieldErr := requireSingleFieldError(t, err)
		if fieldErr.Message != `invalid max constraint "abc": must be a valid unsigned integer` {
			t.Fatalf("unexpected message: %s", fieldErr.Message)
		}
	})

	t.Run("float min", func(t *testing.T) {
		type Config struct {
			Port float64 `conf:"min:abc"`
		}

		_, err := NewLoader[Config]().Load(context.Background())
		fieldErr := requireSingleFieldError(t, err)
		if fieldErr.Message != `invalid min constraint "abc": must be a valid number` {
			t.Fatalf("unexpected message: %s", fieldErr.Message)
		}
	})

	t.Run("string max", func(t *testing.T) {
		type Config struct {
			Name string `conf:"max:abc"`
		}

		_, err := NewLoader[Config]().Load(context.Background())
		fieldErr := requireSingleFieldError(t, err)
		if fieldErr.Message != `invalid max constraint "abc": must be a valid integer` {
			t.Fatalf("unexpected message: %s", fieldErr.Message)
		}
	})
}

func TestLoad_InvalidTag_NestedStructField(t *testing.T) {
	type Database struct {
		Host string
	}

	type Config struct {
		Database Database `conf:"secret:maybe"`
	}

	_, err := NewLoader[Config]().Load(context.Background())
	fieldErr := requireSingleFieldError(t, err)

	if fieldErr.FieldPath != "Database" {
		t.Fatalf("expected FieldPath=Database, got %s", fieldErr.FieldPath)
	}
	if fieldErr.Code != ErrCodeInvalidTag {
		t.Fatalf("expected Code=%s, got %s", ErrCodeInvalidTag, fieldErr.Code)
	}
}

func TestLoad_InvalidTag_UnsetOptionalField(t *testing.T) {
	type Config struct {
		Timeout Optional[int] `conf:"required:maybe"`
	}

	_, err := NewLoader[Config]().Load(context.Background())
	fieldErr := requireSingleFieldError(t, err)

	if fieldErr.FieldPath != "Timeout" {
		t.Fatalf("expected FieldPath=Timeout, got %s", fieldErr.FieldPath)
	}
	if fieldErr.Code != ErrCodeInvalidTag {
		t.Fatalf("expected Code=%s, got %s", ErrCodeInvalidTag, fieldErr.Code)
	}
}

func TestLoad_InvalidTag_OptionalNestedStructField(t *testing.T) {
	type TLSConfig struct {
		Cert string `conf:"secret:maybe"`
	}

	type Config struct {
		TLS Optional[TLSConfig]
	}

	_, err := NewLoader[Config]().Load(context.Background())
	fieldErr := requireSingleFieldError(t, err)

	if fieldErr.FieldPath != "TLS.Cert" {
		t.Fatalf("expected FieldPath=TLS.Cert, got %s", fieldErr.FieldPath)
	}
	if fieldErr.Code != ErrCodeInvalidTag {
		t.Fatalf("expected Code=%s, got %s", ErrCodeInvalidTag, fieldErr.Code)
	}
}

func TestLoad_InvalidTag_PrecedesSourceErrors(t *testing.T) {
	type Config struct {
		Host string `conf:"required:maybe"`
	}

	loader := NewLoader[Config]().WithSource(&mockSource{err: context.DeadlineExceeded})
	cfg, err := loader.Load(context.Background())
	if cfg != nil {
		t.Fatal("expected cfg to be nil")
	}

	fieldErr := requireSingleFieldError(t, err)
	if fieldErr.Code != ErrCodeInvalidTag {
		t.Fatalf("expected Code=%s, got %s", ErrCodeInvalidTag, fieldErr.Code)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("schema validation error should take precedence over source errors")
	}
}

func TestLoad_InvalidTag_RepeatedLoadsStable(t *testing.T) {
	type Config struct {
		Host string `conf:"required:maybe"`
	}

	loader := NewLoader[Config]()

	_, err := loader.Load(context.Background())
	first := requireSingleFieldError(t, err)

	_, err = loader.Load(context.Background())
	second := requireSingleFieldError(t, err)

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("expected repeated loads to return stable errors:\nfirst: %#v\nsecond: %#v", first, second)
	}
}

func TestGetSchemaValidationErrors_ReturnsDefensiveCopy(t *testing.T) {
	type Config struct {
		Host string `conf:"required:maybe"`
	}

	rootType := reflect.TypeOf(Config{})

	first := getSchemaValidationErrors(rootType)
	if len(first) != 1 {
		t.Fatalf("expected 1 cached error, got %d: %#v", len(first), first)
	}

	first[0].Message = "mutated"

	second := getSchemaValidationErrors(rootType)
	if len(second) != 1 {
		t.Fatalf("expected 1 cached error, got %d: %#v", len(second), second)
	}
	if second[0].Message != `invalid required value "maybe": use true or false` {
		t.Fatalf("expected defensive copy, got message %q", second[0].Message)
	}
}
