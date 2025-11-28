package rigging

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestParseTag(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		expected tagConfig
	}{
		{
			name:     "empty tag",
			tag:      "",
			expected: tagConfig{},
		},
		{
			name: "env directive",
			tag:  "env:DB_HOST",
			expected: tagConfig{
				env: "DB_HOST",
			},
		},
		{
			name: "name directive",
			tag:  "name:custom.path",
			expected: tagConfig{
				name: "custom.path",
			},
		},
		{
			name: "prefix directive",
			tag:  "prefix:database",
			expected: tagConfig{
				prefix: "database",
			},
		},
		{
			name: "default directive",
			tag:  "default:5432",
			expected: tagConfig{
				defValue:   "5432",
				hasDefault: true,
			},
		},
		{
			name: "default directive with empty value",
			tag:  "default:",
			expected: tagConfig{
				defValue:   "",
				hasDefault: true,
			},
		},
		{
			name: "min directive",
			tag:  "min:1024",
			expected: tagConfig{
				min: "1024",
			},
		},
		{
			name: "max directive",
			tag:  "max:65535",
			expected: tagConfig{
				max: "65535",
			},
		},
		{
			name: "oneof directive",
			tag:  "oneof:prod,staging,dev",
			expected: tagConfig{
				oneof: []string{"prod", "staging", "dev"},
			},
		},
		{
			name: "oneof directive with spaces",
			tag:  "oneof:prod, staging, dev",
			expected: tagConfig{
				oneof: []string{"prod", "staging", "dev"},
			},
		},
		{
			name: "required directive without value",
			tag:  "required",
			expected: tagConfig{
				required: true,
			},
		},
		{
			name: "required directive with true",
			tag:  "required:true",
			expected: tagConfig{
				required: true,
			},
		},
		{
			name: "required directive with false",
			tag:  "required:false",
			expected: tagConfig{
				required: false,
			},
		},
		{
			name: "secret directive without value",
			tag:  "secret",
			expected: tagConfig{
				secret: true,
			},
		},
		{
			name: "secret directive with true",
			tag:  "secret:true",
			expected: tagConfig{
				secret: true,
			},
		},
		{
			name: "secret directive with false",
			tag:  "secret:false",
			expected: tagConfig{
				secret: false,
			},
		},
		{
			name: "multiple directives",
			tag:  "env:DB_HOST,required,default:localhost",
			expected: tagConfig{
				env:        "DB_HOST",
				defValue:   "localhost",
				hasDefault: true,
				required:   true,
			},
		},
		{
			name: "complex tag with all directives",
			tag:  "env:DB_PORT,name:database.port,default:5432,required,min:1024,max:65535,secret",
			expected: tagConfig{
				env:        "DB_PORT",
				name:       "database.port",
				defValue:   "5432",
				hasDefault: true,
				min:        "1024",
				max:        "65535",
				required:   true,
				secret:     true,
			},
		},
		{
			name: "tag with spaces around commas",
			tag:  "env:VAR, required, default:value",
			expected: tagConfig{
				env:        "VAR",
				defValue:   "value",
				hasDefault: true,
				required:   true,
			},
		},
		{
			name: "prefix with nested struct",
			tag:  "prefix:database",
			expected: tagConfig{
				prefix: "database",
			},
		},
		{
			name: "oneof with single value",
			tag:  "oneof:prod",
			expected: tagConfig{
				oneof: []string{"prod"},
			},
		},
		{
			name: "min and max constraints",
			tag:  "min:10,max:100",
			expected: tagConfig{
				min: "10",
				max: "100",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTag(tt.tag)

			if result.env != tt.expected.env {
				t.Errorf("env: got %q, want %q", result.env, tt.expected.env)
			}
			if result.name != tt.expected.name {
				t.Errorf("name: got %q, want %q", result.name, tt.expected.name)
			}
			if result.prefix != tt.expected.prefix {
				t.Errorf("prefix: got %q, want %q", result.prefix, tt.expected.prefix)
			}
			if result.defValue != tt.expected.defValue {
				t.Errorf("defValue: got %q, want %q", result.defValue, tt.expected.defValue)
			}
			if result.hasDefault != tt.expected.hasDefault {
				t.Errorf("hasDefault: got %v, want %v", result.hasDefault, tt.expected.hasDefault)
			}
			if result.min != tt.expected.min {
				t.Errorf("min: got %q, want %q", result.min, tt.expected.min)
			}
			if result.max != tt.expected.max {
				t.Errorf("max: got %q, want %q", result.max, tt.expected.max)
			}
			if !reflect.DeepEqual(result.oneof, tt.expected.oneof) {
				t.Errorf("oneof: got %v, want %v", result.oneof, tt.expected.oneof)
			}
			if result.required != tt.expected.required {
				t.Errorf("required: got %v, want %v", result.required, tt.expected.required)
			}
			if result.secret != tt.expected.secret {
				t.Errorf("secret: got %v, want %v", result.secret, tt.expected.secret)
			}
		})
	}
}

func TestParseTagEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		expected tagConfig
	}{
		{
			name: "oneof followed by other directives",
			tag:  "oneof:a,b,c,required,secret",
			expected: tagConfig{
				oneof:    []string{"a", "b", "c"},
				required: true,
				secret:   true,
			},
		},
		{
			name: "oneof in middle of tag",
			tag:  "required,oneof:x,y,z,secret",
			expected: tagConfig{
				oneof:    []string{"x", "y", "z"},
				required: true,
				secret:   true,
			},
		},
		{
			name: "oneof at end of tag",
			tag:  "required,secret,oneof:foo,bar,baz",
			expected: tagConfig{
				oneof:    []string{"foo", "bar", "baz"},
				required: true,
				secret:   true,
			},
		},
		{
			name: "default with colon",
			tag:  "default:http://localhost:8080",
			expected: tagConfig{
				defValue:   "http://localhost:8080",
				hasDefault: true,
			},
		},
		{
			name: "multiple boolean directives",
			tag:  "required,secret",
			expected: tagConfig{
				required: true,
				secret:   true,
			},
		},
		{
			name: "env with underscores",
			tag:  "env:DB__HOST__NAME",
			expected: tagConfig{
				env: "DB__HOST__NAME",
			},
		},
		{
			name: "name with dots",
			tag:  "name:database.connection.host",
			expected: tagConfig{
				name: "database.connection.host",
			},
		},
		{
			name: "oneof with empty string option",
			tag:  "oneof:,a,b",
			expected: tagConfig{
				oneof: []string{"", "a", "b"},
			},
		},
		{
			name: "all directives in realistic order",
			tag:  "env:APP_PORT,name:server.port,prefix:server,default:8080,required,min:1024,max:65535,oneof:8080,8443,9000",
			expected: tagConfig{
				env:        "APP_PORT",
				name:       "server.port",
				prefix:     "server",
				defValue:   "8080",
				hasDefault: true,
				min:        "1024",
				max:        "65535",
				oneof:      []string{"8080", "8443", "9000"},
				required:   true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTag(tt.tag)

			if result.env != tt.expected.env {
				t.Errorf("env: got %q, want %q", result.env, tt.expected.env)
			}
			if result.name != tt.expected.name {
				t.Errorf("name: got %q, want %q", result.name, tt.expected.name)
			}
			if result.prefix != tt.expected.prefix {
				t.Errorf("prefix: got %q, want %q", result.prefix, tt.expected.prefix)
			}
			if result.defValue != tt.expected.defValue {
				t.Errorf("defValue: got %q, want %q", result.defValue, tt.expected.defValue)
			}
			if result.hasDefault != tt.expected.hasDefault {
				t.Errorf("hasDefault: got %v, want %v", result.hasDefault, tt.expected.hasDefault)
			}
			if result.min != tt.expected.min {
				t.Errorf("min: got %q, want %q", result.min, tt.expected.min)
			}
			if result.max != tt.expected.max {
				t.Errorf("max: got %q, want %q", result.max, tt.expected.max)
			}
			if !reflect.DeepEqual(result.oneof, tt.expected.oneof) {
				t.Errorf("oneof: got %v, want %v", result.oneof, tt.expected.oneof)
			}
			if result.required != tt.expected.required {
				t.Errorf("required: got %v, want %v", result.required, tt.expected.required)
			}
			if result.secret != tt.expected.secret {
				t.Errorf("secret: got %v, want %v", result.secret, tt.expected.secret)
			}
		})
	}
}

func TestConvertValue(t *testing.T) {
	tests := []struct {
		name        string
		rawValue    any
		targetType  reflect.Type
		want        any
		wantErr     bool
		errContains string
	}{
		// String conversions
		{
			name:       "string to string",
			rawValue:   "hello",
			targetType: reflect.TypeOf(""),
			want:       "hello",
		},
		{
			name:       "int to string",
			rawValue:   42,
			targetType: reflect.TypeOf(""),
			want:       "42",
		},
		{
			name:       "nil to string",
			rawValue:   nil,
			targetType: reflect.TypeOf(""),
			want:       "",
		},

		// Bool conversions
		{
			name:       "string 'true' to bool",
			rawValue:   "true",
			targetType: reflect.TypeOf(false),
			want:       true,
		},
		{
			name:       "string 'false' to bool",
			rawValue:   "false",
			targetType: reflect.TypeOf(false),
			want:       false,
		},
		{
			name:       "string '1' to bool",
			rawValue:   "1",
			targetType: reflect.TypeOf(false),
			want:       true,
		},
		{
			name:       "string '0' to bool",
			rawValue:   "0",
			targetType: reflect.TypeOf(false),
			want:       false,
		},
		{
			name:       "string 'yes' to bool",
			rawValue:   "yes",
			targetType: reflect.TypeOf(false),
			want:       true,
		},
		{
			name:       "string 'no' to bool",
			rawValue:   "no",
			targetType: reflect.TypeOf(false),
			want:       false,
		},
		{
			name:       "string 'YES' to bool (case insensitive)",
			rawValue:   "YES",
			targetType: reflect.TypeOf(false),
			want:       true,
		},
		{
			name:        "invalid string to bool",
			rawValue:    "maybe",
			targetType:  reflect.TypeOf(false),
			wantErr:     true,
			errContains: "cannot convert",
		},

		// Int conversions
		{
			name:       "string to int",
			rawValue:   "42",
			targetType: reflect.TypeOf(0),
			want:       42,
		},
		{
			name:       "negative string to int",
			rawValue:   "-123",
			targetType: reflect.TypeOf(0),
			want:       -123,
		},
		{
			name:        "invalid string to int",
			rawValue:    "not a number",
			targetType:  reflect.TypeOf(0),
			wantErr:     true,
			errContains: "cannot convert",
		},

		// Int8 conversions
		{
			name:       "string to int8",
			rawValue:   "127",
			targetType: reflect.TypeOf(int8(0)),
			want:       int8(127),
		},
		{
			name:        "overflow string to int8",
			rawValue:    "128",
			targetType:  reflect.TypeOf(int8(0)),
			wantErr:     true,
			errContains: "cannot convert",
		},

		// Int16 conversions
		{
			name:       "string to int16",
			rawValue:   "32767",
			targetType: reflect.TypeOf(int16(0)),
			want:       int16(32767),
		},

		// Int32 conversions
		{
			name:       "string to int32",
			rawValue:   "2147483647",
			targetType: reflect.TypeOf(int32(0)),
			want:       int32(2147483647),
		},

		// Int64 conversions
		{
			name:       "string to int64",
			rawValue:   "9223372036854775807",
			targetType: reflect.TypeOf(int64(0)),
			want:       int64(9223372036854775807),
		},

		// Uint conversions
		{
			name:       "string to uint",
			rawValue:   "42",
			targetType: reflect.TypeOf(uint(0)),
			want:       uint(42),
		},
		{
			name:        "negative string to uint",
			rawValue:    "-1",
			targetType:  reflect.TypeOf(uint(0)),
			wantErr:     true,
			errContains: "cannot convert",
		},

		// Uint8 conversions
		{
			name:       "string to uint8",
			rawValue:   "255",
			targetType: reflect.TypeOf(uint8(0)),
			want:       uint8(255),
		},

		// Uint16 conversions
		{
			name:       "string to uint16",
			rawValue:   "65535",
			targetType: reflect.TypeOf(uint16(0)),
			want:       uint16(65535),
		},

		// Uint32 conversions
		{
			name:       "string to uint32",
			rawValue:   "4294967295",
			targetType: reflect.TypeOf(uint32(0)),
			want:       uint32(4294967295),
		},

		// Uint64 conversions
		{
			name:       "string to uint64",
			rawValue:   "18446744073709551615",
			targetType: reflect.TypeOf(uint64(0)),
			want:       uint64(18446744073709551615),
		},

		// Float32 conversions
		{
			name:       "string to float32",
			rawValue:   "3.14",
			targetType: reflect.TypeOf(float32(0)),
			want:       float32(3.14),
		},
		{
			name:        "invalid string to float32",
			rawValue:    "not a float",
			targetType:  reflect.TypeOf(float32(0)),
			wantErr:     true,
			errContains: "cannot convert",
		},

		// Float64 conversions
		{
			name:       "string to float64",
			rawValue:   "3.141592653589793",
			targetType: reflect.TypeOf(float64(0)),
			want:       3.141592653589793,
		},

		// time.Duration conversions
		{
			name:       "string to time.Duration",
			rawValue:   "5s",
			targetType: reflect.TypeOf(time.Duration(0)),
			want:       5 * time.Second,
		},
		{
			name:       "string to time.Duration (minutes)",
			rawValue:   "10m",
			targetType: reflect.TypeOf(time.Duration(0)),
			want:       10 * time.Minute,
		},
		{
			name:       "string to time.Duration (hours)",
			rawValue:   "2h",
			targetType: reflect.TypeOf(time.Duration(0)),
			want:       2 * time.Hour,
		},
		{
			name:        "invalid string to time.Duration",
			rawValue:    "not a duration",
			targetType:  reflect.TypeOf(time.Duration(0)),
			wantErr:     true,
			errContains: "cannot convert",
		},

		// []string conversions
		{
			name:       "[]string to []string",
			rawValue:   []string{"a", "b", "c"},
			targetType: reflect.TypeOf([]string{}),
			want:       []string{"a", "b", "c"},
		},
		{
			name:       "comma-separated string to []string",
			rawValue:   "a,b,c",
			targetType: reflect.TypeOf([]string{}),
			want:       []string{"a", "b", "c"},
		},
		{
			name:       "comma-separated string with spaces to []string",
			rawValue:   "a, b, c",
			targetType: reflect.TypeOf([]string{}),
			want:       []string{"a", "b", "c"},
		},
		{
			name:       "empty string to []string",
			rawValue:   "",
			targetType: reflect.TypeOf([]string{}),
			want:       []string{},
		},
		{
			name:       "[]any to []string",
			rawValue:   []any{"a", 1, true},
			targetType: reflect.TypeOf([]string{}),
			want:       []string{"a", "1", "true"},
		},

		// Nested struct (map) - should return as-is
		{
			name:       "map to struct",
			rawValue:   map[string]any{"key": "value"},
			targetType: reflect.TypeOf(struct{ Key string }{}),
			want:       map[string]any{"key": "value"},
		},

		// Same type - return as-is
		{
			name:       "same type int",
			rawValue:   42,
			targetType: reflect.TypeOf(42),
			want:       42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertValue(tt.rawValue, tt.targetType)
			if tt.wantErr {
				if err == nil {
					t.Errorf("convertValue() expected error but got none")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("convertValue() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("convertValue() unexpected error = %v", err)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("convertValue() = %v (%T), want %v (%T)", got, got, tt.want, tt.want)
			}
		})
	}
}

func TestConvertValue_Optional(t *testing.T) {
	// Test Optional[int]
	t.Run("string to Optional[int]", func(t *testing.T) {
		targetType := reflect.TypeOf(Optional[int]{})
		got, err := convertValue("42", targetType)
		if err != nil {
			t.Fatalf("convertValue() unexpected error = %v", err)
		}

		optVal, ok := got.(Optional[int])
		if !ok {
			t.Fatalf("convertValue() returned %T, want Optional[int]", got)
		}

		if !optVal.Set {
			t.Errorf("Optional.Set = false, want true")
		}
		if optVal.Value != 42 {
			t.Errorf("Optional.Value = %v, want 42", optVal.Value)
		}
	})

	// Test Optional[string]
	t.Run("string to Optional[string]", func(t *testing.T) {
		targetType := reflect.TypeOf(Optional[string]{})
		got, err := convertValue("hello", targetType)
		if err != nil {
			t.Fatalf("convertValue() unexpected error = %v", err)
		}

		optVal, ok := got.(Optional[string])
		if !ok {
			t.Fatalf("convertValue() returned %T, want Optional[string]", got)
		}

		if !optVal.Set {
			t.Errorf("Optional.Set = false, want true")
		}
		if optVal.Value != "hello" {
			t.Errorf("Optional.Value = %v, want 'hello'", optVal.Value)
		}
	})

	// Test Optional[bool]
	t.Run("string to Optional[bool]", func(t *testing.T) {
		targetType := reflect.TypeOf(Optional[bool]{})
		got, err := convertValue("true", targetType)
		if err != nil {
			t.Fatalf("convertValue() unexpected error = %v", err)
		}

		optVal, ok := got.(Optional[bool])
		if !ok {
			t.Fatalf("convertValue() returned %T, want Optional[bool]", got)
		}

		if !optVal.Set {
			t.Errorf("Optional.Set = false, want true")
		}
		if optVal.Value != true {
			t.Errorf("Optional.Value = %v, want true", optVal.Value)
		}
	})

	// Test nil to Optional
	t.Run("nil to Optional[int]", func(t *testing.T) {
		targetType := reflect.TypeOf(Optional[int]{})
		got, err := convertValue(nil, targetType)
		if err != nil {
			t.Fatalf("convertValue() unexpected error = %v", err)
		}

		optVal, ok := got.(Optional[int])
		if !ok {
			t.Fatalf("convertValue() returned %T, want Optional[int]", got)
		}

		// nil should result in zero value with Set=false
		if optVal.Set {
			t.Errorf("Optional.Set = true, want false for nil value")
		}
		if optVal.Value != 0 {
			t.Errorf("Optional.Value = %v, want 0", optVal.Value)
		}
	})
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		input   string
		want    bool
		wantErr bool
	}{
		{"true", true, false},
		{"True", true, false},
		{"TRUE", true, false},
		{"false", false, false},
		{"False", false, false},
		{"FALSE", false, false},
		{"1", true, false},
		{"0", false, false},
		{"yes", true, false},
		{"Yes", true, false},
		{"YES", true, false},
		{"no", false, false},
		{"No", false, false},
		{"NO", false, false},
		{"  true  ", true, false},
		{"  false  ", false, false},
		{"maybe", false, true},
		{"", false, true},
		{"2", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseBool(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseBool(%q) expected error but got none", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("parseBool(%q) unexpected error = %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("parseBool(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseStringSlice(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		want    []string
		wantErr bool
	}{
		{
			name:  "[]string",
			input: []string{"a", "b", "c"},
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "comma-separated string",
			input: "a,b,c",
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "comma-separated with spaces",
			input: "a, b, c",
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "empty string",
			input: "",
			want:  []string{},
		},
		{
			name:  "single value",
			input: "single",
			want:  []string{"single"},
		},
		{
			name:  "[]any",
			input: []any{"a", 1, true, 3.14},
			want:  []string{"a", "1", "true", "3.14"},
		},
		{
			name:    "unsupported type",
			input:   42,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseStringSlice(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseStringSlice() expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("parseStringSlice() unexpected error = %v", err)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseStringSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}
