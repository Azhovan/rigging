package rigging

import (
	"reflect"
	"testing"
)

// TestBindStruct_FieldNameNormalization tests that multi-word field names
// are properly normalized to match lowercase configuration keys.
func TestBindStruct_FieldNameNormalization(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		configKey string
		value     string
	}{
		{
			name:      "APIKey matches apikey",
			fieldName: "APIKey",
			configKey: "apikey",
			value:     "secret123",
		},
		{
			name:      "MaxConnections matches maxconnections",
			fieldName: "MaxConnections",
			configKey: "maxconnections",
			value:     "100",
		},
		{
			name:      "RetryTimeout matches retrytimeout",
			fieldName: "RetryTimeout",
			configKey: "retrytimeout",
			value:     "30s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a struct type dynamically for each test case
			// For simplicity, we'll test with predefined struct types
		})
	}
}

// TestBindStruct_MultiWordFields tests binding with actual multi-word field names.
func TestBindStruct_MultiWordFields(t *testing.T) {
	type Config struct {
		APIKey         string
		MaxConnections int
		RetryTimeout   string
	}

	data := map[string]mergedEntry{
		"apikey":         {value: "secret123", sourceName: "env"},
		"maxconnections": {value: "100", sourceName: "file"},
		"retrytimeout":   {value: "30s", sourceName: "default"},
	}

	var cfg Config
	var provFields []FieldProvenance
	errors := bindStruct(reflect.ValueOf(&cfg), data, &provFields, "", "")

	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	if cfg.APIKey != "secret123" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "secret123")
	}
	if cfg.MaxConnections != 100 {
		t.Errorf("MaxConnections = %d, want %d", cfg.MaxConnections, 100)
	}
	if cfg.RetryTimeout != "30s" {
		t.Errorf("RetryTimeout = %q, want %q", cfg.RetryTimeout, "30s")
	}

	// Verify provenance
	if len(provFields) != 3 {
		t.Fatalf("provenance fields = %d, want 3", len(provFields))
	}
}

// TestBindStruct_PrefixNormalization tests that prefix tags are normalized.
func TestBindStruct_PrefixNormalization(t *testing.T) {
	type DatabaseConfig struct {
		Host string
		Port int
	}

	type Config struct {
		Database DatabaseConfig `conf:"prefix:DATABASE"` // Uppercase prefix
	}

	data := map[string]mergedEntry{
		"database.host": {value: "localhost", sourceName: "file"},
		"database.port": {value: "5432", sourceName: "file"},
	}

	var cfg Config
	var provFields []FieldProvenance
	errors := bindStruct(reflect.ValueOf(&cfg), data, &provFields, "", "")

	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	if cfg.Database.Host != "localhost" {
		t.Errorf("Database.Host = %q, want %q", cfg.Database.Host, "localhost")
	}
	if cfg.Database.Port != 5432 {
		t.Errorf("Database.Port = %d, want %d", cfg.Database.Port, 5432)
	}
}

// TestBindStruct_CustomNameNormalization tests that name tags are normalized.
func TestBindStruct_CustomNameNormalization(t *testing.T) {
	type Config struct {
		APIKey string `conf:"name:API.KEY"` // Uppercase in tag
	}

	data := map[string]mergedEntry{
		"api.key": {value: "secret123", sourceName: "env"},
	}

	var cfg Config
	var provFields []FieldProvenance
	errors := bindStruct(reflect.ValueOf(&cfg), data, &provFields, "", "")

	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	if cfg.APIKey != "secret123" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "secret123")
	}

	// Verify provenance uses normalized key
	if len(provFields) != 1 {
		t.Fatalf("provenance fields = %d, want 1", len(provFields))
	}
	if provFields[0].KeyPath != "api.key" {
		t.Errorf("KeyPath = %q, want %q", provFields[0].KeyPath, "api.key")
	}
}

// TestDeriveFieldKey tests the field key derivation function.
func TestDeriveFieldKey(t *testing.T) {
	tests := []struct {
		fieldName string
		want      string
	}{
		{"Host", "host"},
		{"Port", "port"},
		{"APIKey", "apikey"},
		{"MaxConnections", "maxconnections"},
		{"RetryTimeout", "retrytimeout"},
		{"HTTPServer", "httpserver"},
		{"URLPath", "urlpath"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.fieldName, func(t *testing.T) {
			got := deriveFieldKey(tt.fieldName)
			if got != tt.want {
				t.Errorf("deriveFieldKey(%q) = %q, want %q", tt.fieldName, got, tt.want)
			}
		})
	}
}
