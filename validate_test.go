package rigging

import (
	"reflect"
	"testing"
)

func TestValidateField_Required(t *testing.T) {
	tests := []struct {
		name      string
		value     any
		tags      tagConfig
		wantError bool
	}{
		{
			name:      "required field with value",
			value:     "hello",
			tags:      tagConfig{required: true},
			wantError: false,
		},
		{
			name:      "required field without value (empty string)",
			value:     "",
			tags:      tagConfig{required: true},
			wantError: true,
		},
		{
			name:      "required field without value (zero int)",
			value:     0,
			tags:      tagConfig{required: true},
			wantError: true,
		},
		{
			name:      "non-required field without value",
			value:     "",
			tags:      tagConfig{required: false},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fieldValue := reflect.ValueOf(tt.value)
			errors := validateField(fieldValue, "TestField", tt.tags)

			if tt.wantError && len(errors) == 0 {
				t.Errorf("expected validation error, got none")
			}
			if !tt.wantError && len(errors) > 0 {
				t.Errorf("expected no validation error, got: %v", errors)
			}
			if tt.wantError && len(errors) > 0 {
				if errors[0].Code != ErrCodeRequired {
					t.Errorf("expected error code %q, got %q", ErrCodeRequired, errors[0].Code)
				}
			}
		})
	}
}

func TestValidateField_IntMinMax(t *testing.T) {
	tests := []struct {
		name      string
		value     int
		tags      tagConfig
		wantError bool
		wantCode  string
	}{
		{
			name:      "int within range",
			value:     5000,
			tags:      tagConfig{min: "1024", max: "65535"},
			wantError: false,
		},
		{
			name:      "int below minimum",
			value:     500,
			tags:      tagConfig{min: "1024"},
			wantError: true,
			wantCode:  ErrCodeMin,
		},
		{
			name:      "int above maximum",
			value:     70000,
			tags:      tagConfig{max: "65535"},
			wantError: true,
			wantCode:  ErrCodeMax,
		},
		{
			name:      "int at minimum boundary",
			value:     1024,
			tags:      tagConfig{min: "1024"},
			wantError: false,
		},
		{
			name:      "int at maximum boundary",
			value:     65535,
			tags:      tagConfig{max: "65535"},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fieldValue := reflect.ValueOf(tt.value)
			errors := validateField(fieldValue, "TestField", tt.tags)

			if tt.wantError && len(errors) == 0 {
				t.Errorf("expected validation error, got none")
			}
			if !tt.wantError && len(errors) > 0 {
				t.Errorf("expected no validation error, got: %v", errors)
			}
			if tt.wantError && len(errors) > 0 {
				if errors[0].Code != tt.wantCode {
					t.Errorf("expected error code %q, got %q", tt.wantCode, errors[0].Code)
				}
			}
		})
	}
}

func TestValidateField_FloatMinMax(t *testing.T) {
	tests := []struct {
		name      string
		value     float64
		tags      tagConfig
		wantError bool
		wantCode  string
	}{
		{
			name:      "float within range",
			value:     5.5,
			tags:      tagConfig{min: "1.0", max: "10.0"},
			wantError: false,
		},
		{
			name:      "float below minimum",
			value:     0.5,
			tags:      tagConfig{min: "1.0"},
			wantError: true,
			wantCode:  ErrCodeMin,
		},
		{
			name:      "float above maximum",
			value:     15.0,
			tags:      tagConfig{max: "10.0"},
			wantError: true,
			wantCode:  ErrCodeMax,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fieldValue := reflect.ValueOf(tt.value)
			errors := validateField(fieldValue, "TestField", tt.tags)

			if tt.wantError && len(errors) == 0 {
				t.Errorf("expected validation error, got none")
			}
			if !tt.wantError && len(errors) > 0 {
				t.Errorf("expected no validation error, got: %v", errors)
			}
			if tt.wantError && len(errors) > 0 {
				if errors[0].Code != tt.wantCode {
					t.Errorf("expected error code %q, got %q", tt.wantCode, errors[0].Code)
				}
			}
		})
	}
}

func TestValidateField_StringMinMax(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		tags      tagConfig
		wantError bool
		wantCode  string
	}{
		{
			name:      "string within length range",
			value:     "hello",
			tags:      tagConfig{min: "3", max: "10"},
			wantError: false,
		},
		{
			name:      "string below minimum length",
			value:     "hi",
			tags:      tagConfig{min: "3"},
			wantError: true,
			wantCode:  ErrCodeMin,
		},
		{
			name:      "string above maximum length",
			value:     "this is a very long string",
			tags:      tagConfig{max: "10"},
			wantError: true,
			wantCode:  ErrCodeMax,
		},
		{
			name:      "empty string with min constraint",
			value:     "",
			tags:      tagConfig{min: "1"},
			wantError: false, // Zero values skip validation unless required
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fieldValue := reflect.ValueOf(tt.value)
			errors := validateField(fieldValue, "TestField", tt.tags)

			if tt.wantError && len(errors) == 0 {
				t.Errorf("expected validation error, got none")
			}
			if !tt.wantError && len(errors) > 0 {
				t.Errorf("expected no validation error, got: %v", errors)
			}
			if tt.wantError && len(errors) > 0 {
				if errors[0].Code != tt.wantCode {
					t.Errorf("expected error code %q, got %q", tt.wantCode, errors[0].Code)
				}
			}
		})
	}
}

func TestValidateField_Oneof(t *testing.T) {
	tests := []struct {
		name      string
		value     any
		tags      tagConfig
		wantError bool
	}{
		{
			name:      "string in allowed set",
			value:     "prod",
			tags:      tagConfig{oneof: []string{"prod", "staging", "dev"}},
			wantError: false,
		},
		{
			name:      "string not in allowed set",
			value:     "production",
			tags:      tagConfig{oneof: []string{"prod", "staging", "dev"}},
			wantError: true,
		},
		{
			name:      "int in allowed set",
			value:     2,
			tags:      tagConfig{oneof: []string{"1", "2", "3"}},
			wantError: false,
		},
		{
			name:      "int not in allowed set",
			value:     5,
			tags:      tagConfig{oneof: []string{"1", "2", "3"}},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fieldValue := reflect.ValueOf(tt.value)
			errors := validateField(fieldValue, "TestField", tt.tags)

			if tt.wantError && len(errors) == 0 {
				t.Errorf("expected validation error, got none")
			}
			if !tt.wantError && len(errors) > 0 {
				t.Errorf("expected no validation error, got: %v", errors)
			}
			if tt.wantError && len(errors) > 0 {
				if errors[0].Code != ErrCodeOneOf {
					t.Errorf("expected error code %q, got %q", ErrCodeOneOf, errors[0].Code)
				}
			}
		})
	}
}

func TestValidateStruct(t *testing.T) {
	type Config struct {
		Name     string `conf:"required"`
		Port     int    `conf:"min:1024,max:65535"`
		Env      string `conf:"oneof:prod,staging,dev"`
		Optional string `conf:""`
	}

	tests := []struct {
		name       string
		config     Config
		wantErrors int
	}{
		{
			name: "valid config",
			config: Config{
				Name: "myapp",
				Port: 8080,
				Env:  "prod",
			},
			wantErrors: 0,
		},
		{
			name: "missing required field",
			config: Config{
				Name: "",
				Port: 8080,
				Env:  "prod",
			},
			wantErrors: 1,
		},
		{
			name: "port below minimum",
			config: Config{
				Name: "myapp",
				Port: 80,
				Env:  "prod",
			},
			wantErrors: 1,
		},
		{
			name: "invalid env value",
			config: Config{
				Name: "myapp",
				Port: 8080,
				Env:  "production",
			},
			wantErrors: 1,
		},
		{
			name: "multiple validation errors",
			config: Config{
				Name: "",
				Port: 80,
				Env:  "production",
			},
			wantErrors: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgValue := reflect.ValueOf(tt.config)
			errors := validateStruct(cfgValue)

			if len(errors) != tt.wantErrors {
				t.Errorf("expected %d validation errors, got %d: %v", tt.wantErrors, len(errors), errors)
			}
		})
	}
}

func TestValidateStruct_NestedStructs(t *testing.T) {
	type Database struct {
		Host string `conf:"required"`
		Port int    `conf:"min:1024,max:65535"`
	}

	type Config struct {
		AppName  string   `conf:"required"`
		Database Database `conf:"prefix:database"`
	}

	tests := []struct {
		name       string
		config     Config
		wantErrors int
	}{
		{
			name: "valid nested config",
			config: Config{
				AppName: "myapp",
				Database: Database{
					Host: "localhost",
					Port: 5432,
				},
			},
			wantErrors: 0,
		},
		{
			name: "missing nested required field",
			config: Config{
				AppName: "myapp",
				Database: Database{
					Host: "",
					Port: 5432,
				},
			},
			wantErrors: 1,
		},
		{
			name: "nested port validation error",
			config: Config{
				AppName: "myapp",
				Database: Database{
					Host: "localhost",
					Port: 80,
				},
			},
			wantErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgValue := reflect.ValueOf(tt.config)
			errors := validateStruct(cfgValue)

			if len(errors) != tt.wantErrors {
				t.Errorf("expected %d validation errors, got %d: %v", tt.wantErrors, len(errors), errors)
			}
		})
	}
}

func TestValidateStruct_OptionalFields(t *testing.T) {
	type Config struct {
		Required string           `conf:"required"`
		Optional Optional[string] `conf:"min:3"`
	}

	tests := []struct {
		name       string
		config     Config
		wantErrors int
	}{
		{
			name: "optional not set",
			config: Config{
				Required: "value",
				Optional: Optional[string]{Set: false},
			},
			wantErrors: 0,
		},
		{
			name: "optional set with valid value",
			config: Config{
				Required: "value",
				Optional: Optional[string]{Value: "hello", Set: true},
			},
			wantErrors: 0,
		},
		{
			name: "optional set with invalid value",
			config: Config{
				Required: "value",
				Optional: Optional[string]{Value: "hi", Set: true},
			},
			wantErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgValue := reflect.ValueOf(tt.config)
			errors := validateStruct(cfgValue)

			if len(errors) != tt.wantErrors {
				t.Errorf("expected %d validation errors, got %d: %v", tt.wantErrors, len(errors), errors)
			}
		})
	}
}

func TestIsZeroValue(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		wantZero bool
	}{
		{"empty string", "", true},
		{"non-empty string", "hello", false},
		{"zero int", 0, true},
		{"non-zero int", 42, false},
		{"false bool", false, true},
		{"true bool", true, false},
		{"zero float", 0.0, true},
		{"non-zero float", 3.14, false},
		{"empty slice", []string{}, true},
		{"non-empty slice", []string{"a"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := reflect.ValueOf(tt.value)
			got := isZeroValue(v)
			if got != tt.wantZero {
				t.Errorf("isZeroValue(%v) = %v, want %v", tt.value, got, tt.wantZero)
			}
		})
	}
}
