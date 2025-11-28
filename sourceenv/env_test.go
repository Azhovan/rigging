package sourceenv

import (
	"context"
	"os"
	"testing"

	"github.com/Azhovan/rigging"
)

func TestEnvSource_Load(t *testing.T) {
	tests := []struct {
		name     string
		opts     Options
		envVars  map[string]string
		expected map[string]any
	}{
		{
			name: "basic environment variables",
			opts: Options{},
			envVars: map[string]string{
				"HOST": "localhost",
				"PORT": "8080",
			},
			expected: map[string]any{
				"host": "localhost",
				"port": "8080",
			},
		},
		{
			name: "double underscore as level separator",
			opts: Options{},
			envVars: map[string]string{
				"DATABASE__HOST": "db.example.com",
				"DATABASE__PORT": "5432",
			},
			expected: map[string]any{
				"database.host": "db.example.com",
				"database.port": "5432",
			},
		},
		{
			name: "single underscore preserved",
			opts: Options{},
			envVars: map[string]string{
				"DB_MAX_CONNECTIONS": "100",
				"API__RATE_LIMIT":    "1000",
			},
			expected: map[string]any{
				"db_max_connections": "100",
				"api.rate_limit":     "1000",
			},
		},
		{
			name: "with prefix filtering",
			opts: Options{Prefix: "APP_"},
			envVars: map[string]string{
				"APP_HOST":     "localhost",
				"APP_PORT":     "8080",
				"OTHER_VAR":    "ignored",
				"APP_DB__HOST": "db.local",
			},
			expected: map[string]any{
				"host":    "localhost",
				"port":    "8080",
				"db.host": "db.local",
			},
		},
		{
			name: "prefix case insensitive matching",
			opts: Options{Prefix: "app_"},
			envVars: map[string]string{
				"APP_HOST": "localhost",
				"app_PORT": "8080",
				"App_NAME": "myapp",
			},
			expected: map[string]any{
				"host": "localhost",
				"port": "8080",
				"name": "myapp",
			},
		},
		{
			name: "empty prefix processes all variables",
			opts: Options{Prefix: ""},
			envVars: map[string]string{
				"VAR1": "value1",
				"VAR2": "value2",
			},
			expected: map[string]any{
				"var1": "value1",
				"var2": "value2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			source := New(tt.opts)
			ctx := context.Background()

			result, err := source.Load(ctx)
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}

			// Check that all expected keys are present with correct values
			for key, expectedValue := range tt.expected {
				actualValue, ok := result[key]
				if !ok {
					t.Errorf("expected key %q not found in result", key)
					continue
				}
				if actualValue != expectedValue {
					t.Errorf("key %q: got %v, want %v", key, actualValue, expectedValue)
				}
			}
		})
	}
}

func TestEnvSource_Watch(t *testing.T) {
	source := New(Options{})
	ctx := context.Background()

	ch, err := source.Watch(ctx)
	if err != rigging.ErrWatchNotSupported {
		t.Errorf("Watch() error = %v, want %v", err, rigging.ErrWatchNotSupported)
	}
	if ch != nil {
		t.Errorf("Watch() channel = %v, want nil", ch)
	}
}

func TestEnvSource_EmptyValues(t *testing.T) {
	os.Setenv("EMPTY_VAR", "")
	defer os.Unsetenv("EMPTY_VAR")

	source := New(Options{})
	ctx := context.Background()

	result, err := source.Load(ctx)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Empty values should still be included
	if val, ok := result["empty_var"]; !ok {
		t.Error("expected empty_var to be present")
	} else if val != "" {
		t.Errorf("empty_var = %v, want empty string", val)
	}
}

func TestEnvSource_ComplexNesting(t *testing.T) {
	envVars := map[string]string{
		"APP__DATABASE__CONNECTION__HOST":     "db.example.com",
		"APP__DATABASE__CONNECTION__PORT":     "5432",
		"APP__DATABASE__CONNECTION__USER":     "admin",
		"APP__DATABASE__CONNECTION__PASSWORD": "secret",
	}

	for k, v := range envVars {
		os.Setenv(k, v)
		defer os.Unsetenv(k)
	}

	source := New(Options{})
	ctx := context.Background()

	result, err := source.Load(ctx)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	expected := map[string]any{
		"app.database.connection.host":     "db.example.com",
		"app.database.connection.port":     "5432",
		"app.database.connection.user":     "admin",
		"app.database.connection.password": "secret",
	}

	for key, expectedValue := range expected {
		actualValue, ok := result[key]
		if !ok {
			t.Errorf("expected key %q not found in result", key)
			continue
		}
		if actualValue != expectedValue {
			t.Errorf("key %q: got %v, want %v", key, actualValue, expectedValue)
		}
	}
}

// Helper function for case-insensitive prefix checking
func hasPrefix(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if toLower(s[i]) != toLower(prefix[i]) {
			return false
		}
	}
	return true
}

func toLower(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}
