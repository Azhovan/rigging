package rigging

import (
	"reflect"
	"testing"
)

func TestParseTag(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		expected tagConfig
	}{
		{
			name: "empty tag",
			tag:  "",
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
