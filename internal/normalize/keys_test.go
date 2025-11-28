package normalize

import (
	"testing"
)

func TestToLowerDotPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "double underscore to dot",
			input:    "FOO__BAR",
			expected: "foo.bar",
		},
		{
			name:     "single underscore preserved",
			input:    "DB_MAX_CONNECTIONS",
			expected: "db_max_connections",
		},
		{
			name:     "mixed double and single underscores",
			input:    "API__RATE_LIMIT",
			expected: "api.rate_limit",
		},
		{
			name:     "multiple levels",
			input:    "APP__DATABASE__HOST",
			expected: "app.database.host",
		},
		{
			name:     "already lowercase",
			input:    "simple",
			expected: "simple",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only underscores",
			input:    "____",
			expected: "..",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToLowerDotPath(tt.input)
			if result != tt.expected {
				t.Errorf("ToLowerDotPath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDeriveFieldPath(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		expected  string
	}{
		{
			name:      "simple field",
			fieldName: "Host",
			expected:  "host",
		},
		{
			name:      "single letter",
			fieldName: "P",
			expected:  "p",
		},
		{
			name:      "camelCase field",
			fieldName: "APIKey",
			expected:  "aPIKey",
		},
		{
			name:      "already lowercase first letter",
			fieldName: "port",
			expected:  "port",
		},
		{
			name:      "empty string",
			fieldName: "",
			expected:  "",
		},
		{
			name:      "multi-word field",
			fieldName: "MaxConnections",
			expected:  "maxConnections",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DeriveFieldPath(tt.fieldName)
			if result != tt.expected {
				t.Errorf("DeriveFieldPath(%q) = %q, want %q", tt.fieldName, result, tt.expected)
			}
		})
	}
}

func TestApplyPrefix(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		key      string
		expected string
	}{
		{
			name:     "with prefix",
			prefix:   "database",
			key:      "host",
			expected: "database.host",
		},
		{
			name:     "empty prefix",
			prefix:   "",
			key:      "host",
			expected: "host",
		},
		{
			name:     "empty key",
			prefix:   "database",
			key:      "",
			expected: "database",
		},
		{
			name:     "both empty",
			prefix:   "",
			key:      "",
			expected: "",
		},
		{
			name:     "nested prefix",
			prefix:   "api.v1",
			key:      "endpoint",
			expected: "api.v1.endpoint",
		},
		{
			name:     "key with underscore",
			prefix:   "api",
			key:      "rate_limit",
			expected: "api.rate_limit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ApplyPrefix(tt.prefix, tt.key)
			if result != tt.expected {
				t.Errorf("ApplyPrefix(%q, %q) = %q, want %q", tt.prefix, tt.key, result, tt.expected)
			}
		})
	}
}
