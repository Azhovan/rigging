package rigging

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestBinding_ParseTag(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		expected tagConfig
	}{
		// Basic directives
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
			name: "env with underscores",
			tag:  "env:DB__HOST__NAME",
			expected: tagConfig{
				env: "DB__HOST__NAME",
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
			name: "name with dots",
			tag:  "name:database.connection.host",
			expected: tagConfig{
				name: "database.connection.host",
			},
		},
		{
			name: "prefix directive",
			tag:  "prefix:database",
			expected: tagConfig{
				prefix: "database",
			},
		},

		// Default directive
		{
			name: "default directive",
			tag:  "default:5432",
			expected: tagConfig{
				defValue:   "5432",
				hasDefault: true,
			},
		},
		{
			name: "default with empty value",
			tag:  "default:",
			expected: tagConfig{
				defValue:   "",
				hasDefault: true,
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
			name: "default with URL",
			tag:  "default:https://example.com:8080/path?query=value",
			expected: tagConfig{
				defValue:   "https://example.com:8080/path?query=value",
				hasDefault: true,
			},
		},
		{
			name: "default with JSON-like value",
			tag:  "default:{\"key\":\"value\"}",
			expected: tagConfig{
				defValue:   "{\"key\":\"value\"}",
				hasDefault: true,
			},
		},
		{
			name: "default with special characters",
			tag:  "default:!@#$%^&*()",
			expected: tagConfig{
				defValue:   "!@#$%^&*()",
				hasDefault: true,
			},
		},
		{
			name: "default with spaces",
			tag:  "default:  value with spaces  ",
			expected: tagConfig{
				defValue:   "  value with spaces",
				hasDefault: true,
			},
		},
		{
			name: "default with comma terminates directive",
			tag:  "default:a,b,c",
			expected: tagConfig{
				defValue:   "a",
				hasDefault: true,
			},
		},
		{
			name: "default with negative number",
			tag:  "default:-123",
			expected: tagConfig{
				defValue:   "-123",
				hasDefault: true,
			},
		},
		{
			name: "default with float",
			tag:  "default:3.14159",
			expected: tagConfig{
				defValue:   "3.14159",
				hasDefault: true,
			},
		},

		// Min/Max directives
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
			name: "min and max constraints",
			tag:  "min:10,max:100",
			expected: tagConfig{
				min: "10",
				max: "100",
			},
		},
		{
			name: "min with negative value",
			tag:  "min:-100",
			expected: tagConfig{
				min: "-100",
			},
		},
		{
			name: "max with negative value",
			tag:  "max:-10",
			expected: tagConfig{
				max: "-10",
			},
		},
		{
			name: "min with float",
			tag:  "min:3.14",
			expected: tagConfig{
				min: "3.14",
			},
		},
		{
			name: "max with float",
			tag:  "max:99.99",
			expected: tagConfig{
				max: "99.99",
			},
		},
		{
			name: "min with zero",
			tag:  "min:0",
			expected: tagConfig{
				min: "0",
			},
		},
		{
			name: "max with zero",
			tag:  "max:0",
			expected: tagConfig{
				max: "0",
			},
		},
		{
			name: "min and max with same value",
			tag:  "min:10,max:10",
			expected: tagConfig{
				min: "10",
				max: "10",
			},
		},
		{
			name: "min empty value",
			tag:  "min:",
			expected: tagConfig{
				min: "",
			},
		},
		{
			name: "max empty value",
			tag:  "max:",
			expected: tagConfig{
				max: "",
			},
		},

		// Oneof directive
		{
			name: "oneof directive",
			tag:  "oneof:prod,staging,dev",
			expected: tagConfig{
				oneof: []string{"dev", "prod", "staging"},
			},
		},
		{
			name: "oneof with spaces",
			tag:  "oneof:prod, staging, dev",
			expected: tagConfig{
				oneof: []string{"dev", "prod", "staging"},
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
			name: "oneof with duplicated values",
			tag:  "oneof:dev,prod,staging,dev,prod",
			expected: tagConfig{
				oneof: []string{"dev", "prod", "staging"},
			},
		},
		{
			name: "oneof with empty value",
			tag:  "oneof:",
			expected: tagConfig{
				oneof: nil,
			},
		},
		{
			name: "oneof with only commas",
			tag:  "oneof:,,,",
			expected: tagConfig{
				oneof: nil,
			},
		},
		{
			name: "oneof with trailing comma",
			tag:  "oneof:a,b,c,",
			expected: tagConfig{
				oneof: []string{"a", "b", "c"},
			},
		},
		{
			name: "oneof with leading comma",
			tag:  "oneof:,a,b,c",
			expected: tagConfig{
				oneof: []string{"a", "b", "c"},
			},
		},
		{
			name: "oneof with excessive spaces",
			tag:  "oneof:  a  ,  b  ,  c  ",
			expected: tagConfig{
				oneof: []string{"a", "b", "c"},
			},
		},
		{
			name: "oneof with special characters",
			tag:  "oneof:prod-1,staging_2,dev.3",
			expected: tagConfig{
				oneof: []string{"dev.3", "prod-1", "staging_2"},
			},
		},
		{
			name: "oneof with numbers",
			tag:  "oneof:1,2,3,4,5",
			expected: tagConfig{
				oneof: []string{"1", "2", "3", "4", "5"},
			},
		},
		{
			name: "oneof with URLs",
			tag:  "oneof:http://localhost:8080,https://example.com:443",
			expected: tagConfig{
				oneof: []string{"http://localhost:8080", "https://example.com:443"},
			},
		},
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
				oneof:    []string{"bar", "baz", "foo"},
				required: true,
				secret:   true,
			},
		},

		// Boolean directives (required/secret)
		{
			name: "required without value",
			tag:  "required",
			expected: tagConfig{
				required: true,
			},
		},
		{
			name: "required with true",
			tag:  "required:true",
			expected: tagConfig{
				required: true,
			},
		},
		{
			name: "required with false",
			tag:  "required:false",
			expected: tagConfig{
				required: false,
			},
		},
		{
			name: "required with invalid value defaults to true",
			tag:  "required:invalid",
			expected: tagConfig{
				required: true,
			},
		},
		{
			name: "required with numeric value defaults to true",
			tag:  "required:1",
			expected: tagConfig{
				required: true,
			},
		},
		{
			name: "secret without value",
			tag:  "secret",
			expected: tagConfig{
				secret: true,
			},
		},
		{
			name: "secret with true",
			tag:  "secret:true",
			expected: tagConfig{
				secret: true,
			},
		},
		{
			name: "secret with false",
			tag:  "secret:false",
			expected: tagConfig{
				secret: false,
			},
		},
		{
			name: "secret with invalid value defaults to true",
			tag:  "secret:invalid",
			expected: tagConfig{
				secret: true,
			},
		},
		{
			name: "secret with yes defaults to true",
			tag:  "secret:yes",
			expected: tagConfig{
				secret: true,
			},
		},
		{
			name: "both required and secret false",
			tag:  "required:false,secret:false",
			expected: tagConfig{
				required: false,
				secret:   false,
			},
		},
		{
			name: "required true and secret false",
			tag:  "required:true,secret:false",
			expected: tagConfig{
				required: true,
				secret:   false,
			},
		},
		{
			name: "required false and secret true",
			tag:  "required:false,secret:true",
			expected: tagConfig{
				required: false,
				secret:   true,
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

		// Multiple directives
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
		{
			name: "all directives with edge case values",
			tag:  "env:DB__HOST,name:db.host,prefix:database,default:localhost:5432,min:-1,max:65535,oneof:localhost:5432,remote:5432,required:false,secret:true",
			expected: tagConfig{
				env:        "DB__HOST",
				name:       "db.host",
				prefix:     "database",
				defValue:   "localhost:5432",
				hasDefault: true,
				min:        "-1",
				max:        "65535",
				oneof:      []string{"localhost:5432", "remote:5432"},
				required:   false,
				secret:     true,
			},
		},
		{
			name: "oneof with URLs and other directives",
			tag:  "oneof:http://localhost:8080,https://prod.com:443,required,default:http://localhost:8080",
			expected: tagConfig{
				oneof:      []string{"http://localhost:8080", "https://prod.com:443"},
				required:   true,
				defValue:   "http://localhost:8080",
				hasDefault: true,
			},
		},

		// Whitespace handling
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
			name: "spaces around directive names",
			tag:  " env : VAR , required , secret ",
			expected: tagConfig{
				env:      " VAR",
				required: true,
				secret:   true,
			},
		},
		{
			name: "tabs and spaces",
			tag:  "env:VAR\t,\trequired\t,\tsecret",
			expected: tagConfig{
				env:      "VAR",
				required: true,
				secret:   true,
			},
		},
		{
			name: "multiple spaces between directives",
			tag:  "env:VAR  ,  required  ,  secret",
			expected: tagConfig{
				env:      "VAR",
				required: true,
				secret:   true,
			},
		},
		{
			name: "empty directives from multiple commas",
			tag:  "env:VAR,,required",
			expected: tagConfig{
				env:      "VAR",
				required: true,
			},
		},
		{
			name: "trailing comma",
			tag:  "env:VAR,required,",
			expected: tagConfig{
				env:      "VAR",
				required: true,
			},
		},
		{
			name: "leading comma",
			tag:  ",env:VAR,required",
			expected: tagConfig{
				env:      "VAR",
				required: true,
			},
		},

		// Unknown directives
		{
			name: "unknown directive ignored",
			tag:  "unknown:value,env:VAR",
			expected: tagConfig{
				env: "VAR",
			},
		},
		{
			name: "multiple unknown directives",
			tag:  "foo:bar,env:VAR,baz:qux,required",
			expected: tagConfig{
				env:      "VAR",
				required: true,
			},
		},
		{
			name:     "only unknown directives",
			tag:      "unknown:value,another:thing",
			expected: tagConfig{},
		},
		{
			name:     "typo in directive name",
			tag:      "envv:VAR,requiired:true", // intentional typos to test silent ignore
			expected: tagConfig{},
		},

		// Edge cases
		{
			name: "duplicate directives - last one wins",
			tag:  "env:VAR1,env:VAR2,required:false,required:true",
			expected: tagConfig{
				env:      "VAR2",
				required: true,
			},
		},
		{
			name: "empty values for multiple directives",
			tag:  "env:,name:,prefix:,default:",
			expected: tagConfig{
				env:        "",
				name:       "",
				prefix:     "",
				defValue:   "",
				hasDefault: true,
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

func TestBinding_ConvertValue(t *testing.T) {
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

func TestBinding_ConvertValue_Optional(t *testing.T) {
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

func TestBinding_ParseBool(t *testing.T) {
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

func TestBinding_ParseStringSlice(t *testing.T) {
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

func TestBinding_DetermineKeyPath(t *testing.T) {
	tests := []struct {
		name         string
		fieldName    string
		tagCfg       tagConfig
		parentPrefix string
		expected     string
	}{
		// Basic behavior - no prefix, no name tag
		{
			name:         "simple field name without prefix",
			fieldName:    "Host",
			tagCfg:       tagConfig{},
			parentPrefix: "",
			expected:     "host",
		},
		{
			name:         "single letter field name",
			fieldName:    "X",
			tagCfg:       tagConfig{},
			parentPrefix: "",
			expected:     "x",
		},
		{
			name:         "field with underscores",
			fieldName:    "Max_Connections",
			tagCfg:       tagConfig{},
			parentPrefix: "",
			expected:     "max_connections",
		},
		{
			name:         "field with numbers",
			fieldName:    "Port8080",
			tagCfg:       tagConfig{},
			parentPrefix: "",
			expected:     "port8080",
		},
		{
			name:         "very long field name",
			fieldName:    "VeryLongFieldNameThatShouldStillWork",
			tagCfg:       tagConfig{},
			parentPrefix: "",
			expected:     "verylongfieldnamethatshouldstillwork",
		},

		// With parent prefix
		{
			name:         "field name with parent prefix",
			fieldName:    "Host",
			tagCfg:       tagConfig{},
			parentPrefix: "database",
			expected:     "database.host",
		},
		{
			name:         "single letter with prefix",
			fieldName:    "Y",
			tagCfg:       tagConfig{},
			parentPrefix: "coord",
			expected:     "coord.y",
		},
		{
			name:         "parent prefix with dots",
			fieldName:    "Host",
			tagCfg:       tagConfig{},
			parentPrefix: "app.server.db",
			expected:     "app.server.db.host",
		},
		{
			name:         "very long prefix",
			fieldName:    "Host",
			tagCfg:       tagConfig{},
			parentPrefix: "application.configuration.database.primary",
			expected:     "application.configuration.database.primary.host",
		},

		// Name tag behavior (takes precedence)
		{
			name:      "name tag takes precedence over parent prefix",
			fieldName: "Host",
			tagCfg: tagConfig{
				name: "custom_host",
			},
			parentPrefix: "database",
			expected:     "custom_host",
		},
		{
			name:      "name tag ignores parent prefix",
			fieldName: "Port",
			tagCfg: tagConfig{
				name: "server_port",
			},
			parentPrefix: "config",
			expected:     "server_port",
		},
		{
			name:      "name tag with dots",
			fieldName: "Host",
			tagCfg: tagConfig{
				name: "db.primary.host",
			},
			parentPrefix: "",
			expected:     "db.primary.host",
		},
		{
			name:      "name tag with special characters",
			fieldName: "Host",
			tagCfg: tagConfig{
				name: "host-name_v2",
			},
			parentPrefix: "",
			expected:     "host-name_v2",
		},
		{
			name:      "name tag with spaces",
			fieldName: "Host",
			tagCfg: tagConfig{
				name: "host name",
			},
			parentPrefix: "",
			expected:     "host name",
		},
		{
			name:      "name tag with prefix tag and parent prefix",
			fieldName: "Host",
			tagCfg: tagConfig{
				name:   "override",
				prefix: "ignored_prefix",
			},
			parentPrefix: "ignored_parent",
			expected:     "override",
		},
		{
			name:      "name tag with all other tags",
			fieldName: "Host",
			tagCfg: tagConfig{
				name:       "custom_key",
				prefix:     "ignored",
				env:        "ENV_VAR",
				defValue:   "default",
				hasDefault: true,
			},
			parentPrefix: "parent",
			expected:     "custom_key",
		},

		// Case normalization
		{
			name:         "mixed case field name normalized to lowercase",
			fieldName:    "HTTPPort",
			tagCfg:       tagConfig{},
			parentPrefix: "",
			expected:     "httpport",
		},
		{
			name:         "all caps field name",
			fieldName:    "URL",
			tagCfg:       tagConfig{},
			parentPrefix: "",
			expected:     "url",
		},
		{
			name:         "mixed case field with prefix",
			fieldName:    "APIKey",
			tagCfg:       tagConfig{},
			parentPrefix: "auth",
			expected:     "auth.apikey",
		},
		{
			name:         "all caps with prefix",
			fieldName:    "API",
			tagCfg:       tagConfig{},
			parentPrefix: "server",
			expected:     "server.api",
		},
		{
			name:         "mixed case parent prefix normalized",
			fieldName:    "Host",
			tagCfg:       tagConfig{},
			parentPrefix: "Database",
			expected:     "database.host",
		},
		{
			name:      "mixed case name tag normalized",
			fieldName: "Host",
			tagCfg: tagConfig{
				name: "CustomHost",
			},
			parentPrefix: "",
			expected:     "customhost",
		},
		{
			name:      "name tag with uppercase and dots",
			fieldName: "Host",
			tagCfg: tagConfig{
				name: "DB.Primary.HOST",
			},
			parentPrefix: "",
			expected:     "db.primary.host",
		},

		// Edge cases
		{
			name:         "empty field name",
			fieldName:    "",
			tagCfg:       tagConfig{},
			parentPrefix: "",
			expected:     "",
		},
		{
			name:         "empty field name with prefix",
			fieldName:    "",
			tagCfg:       tagConfig{},
			parentPrefix: "database",
			expected:     "database.",
		},
		{
			name:      "empty name tag falls back to derived",
			fieldName: "Host",
			tagCfg: tagConfig{
				name: "",
			},
			parentPrefix: "",
			expected:     "host",
		},
		{
			name:      "prefix in tagCfg is ignored(individual field)",
			fieldName: "Host",
			tagCfg: tagConfig{
				prefix: "ignored",
			},
			parentPrefix: "database",
			expected:     "database.host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineKeyPath(tt.fieldName, tt.tagCfg, tt.parentPrefix)
			if result != tt.expected {
				t.Errorf("determineKeyPath(%q, tagCfg, %q) = %q, want %q",
					tt.fieldName, tt.parentPrefix, result, tt.expected)
			}
		})
	}
}

func TestBinding_ExtractTagDirectives(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		expected []string
	}{
		{
			name:     "random tags(no validation)",
			tag:      "random:value",
			expected: []string{"random:value"},
		},
		{
			name:     "empty tag",
			tag:      "",
			expected: []string{},
		},
		{
			name:     "single directive",
			tag:      "env:DB_HOST",
			expected: []string{"env:DB_HOST"},
		},
		{
			name:     "multiple simple directives",
			tag:      "env:DB_HOST,required",
			expected: []string{"env:DB_HOST", "required"},
		},
		{
			name:     "multiple directives with values",
			tag:      "env:DB_HOST,default:localhost,required",
			expected: []string{"env:DB_HOST", "default:localhost", "required"},
		},
		{
			name:     "oneof with single value",
			tag:      "oneof:dev",
			expected: []string{"oneof:dev"},
		},
		{
			name:     "oneof with multiple values",
			tag:      "oneof:dev,staging,prod",
			expected: []string{"oneof:dev,staging,prod"},
		},
		{
			name:     "oneof with multiple values and other directives",
			tag:      "env:ENV,oneof:dev,staging,prod,required",
			expected: []string{"env:ENV", "oneof:dev,staging,prod", "required"},
		},
		{
			name:     "oneof at the end",
			tag:      "env:ENV,required,oneof:dev,staging,prod",
			expected: []string{"env:ENV", "required", "oneof:dev,staging,prod"},
		},
		{
			name:     "oneof in the middle",
			tag:      "required,oneof:dev,staging,prod,default:dev",
			expected: []string{"required", "oneof:dev,staging,prod", "default:dev"},
		},
		{
			name:     "complex tag with all directive types",
			tag:      "env:LOG_LEVEL,name:logging.level,oneof:debug,info,warn,error,default:info,required",
			expected: []string{"env:LOG_LEVEL", "name:logging.level", "oneof:debug,info,warn,error", "default:info", "required"},
		},
		{
			name:     "oneof with values containing special characters",
			tag:      "oneof:us-east-1,us-west-2,eu-central-1",
			expected: []string{"oneof:us-east-1,us-west-2,eu-central-1"},
		},
		{
			name:     "multiple oneof directives",
			tag:      "oneof:a,b,c,env:TEST,oneof:x,y,z",
			expected: []string{"oneof:a,b,c", "env:TEST", "oneof:x,y,z"},
		},
		{
			name:     "min and max directives",
			tag:      "min:1,max:100,default:50",
			expected: []string{"min:1", "max:100", "default:50"},
		},
		{
			name:     "prefix directive",
			tag:      "prefix:database,required",
			expected: []string{"prefix:database", "required"},
		},
		{
			name:     "secret directive",
			tag:      "env:API_KEY,secret,required",
			expected: []string{"env:API_KEY", "secret", "required"},
		},
		{
			name:     "oneof with empty values",
			tag:      "oneof:,,empty",
			expected: []string{"oneof:,,empty"},
		},
		{
			name:     "consecutive commas outside oneof",
			tag:      "env:TEST,,required",
			expected: []string{"env:TEST", "", "required"},
		},
		{
			name:     "directive with no value",
			tag:      "env:,required",
			expected: []string{"env:", "required"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTagDirectives(tt.tag)
			// Handle nil vs empty slice comparison
			if len(result) != 0 && len(tt.expected) != 0 {
				if !reflect.DeepEqual(result, tt.expected) {
					t.Errorf("extractTagDirectives(%q) = %v, want %v", tt.tag, result, tt.expected)
				}
			}
		})
	}
}

func TestBinding_StartsWithDirective(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "env directive",
			input:    "env:DB_HOST",
			expected: true,
		},
		{
			name:     "name directive",
			input:    "name:custom.path",
			expected: true,
		},
		{
			name:     "prefix directive",
			input:    "prefix:database",
			expected: true,
		},
		{
			name:     "default directive",
			input:    "default:localhost",
			expected: true,
		},
		{
			name:     "min directive",
			input:    "min:1",
			expected: true,
		},
		{
			name:     "max directive",
			input:    "max:100",
			expected: true,
		},
		{
			name:     "oneof directive",
			input:    "oneof:dev,staging,prod",
			expected: true,
		},
		{
			name:     "required directive",
			input:    "required",
			expected: true,
		},
		{
			name:     "secret directive",
			input:    "secret",
			expected: true,
		},
		{
			name:     "with leading whitespace",
			input:    "  env:TEST",
			expected: true,
		},
		{
			name:     "with leading whitespace required",
			input:    "  required",
			expected: true,
		},
		{
			name:     "not a directive",
			input:    "random_text",
			expected: false,
		},
		{
			name:     "partial match",
			input:    "environment:TEST",
			expected: false,
		},
		{
			name:     "directive in middle",
			input:    "some env:TEST",
			expected: false,
		},
		{
			name:     "only comma at the end",
			input:    " ,",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := startsWithDirective(tt.input)
			if result != tt.expected {
				t.Errorf("startsWithDirective(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
