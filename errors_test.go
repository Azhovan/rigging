package rigging

import (
	"strings"
	"testing"
)

func TestValidationError_Error_SingleError(t *testing.T) {
	ve := &ValidationError{
		FieldErrors: []FieldError{
			{
				FieldPath: "Database.Host",
				Code:      ErrCodeRequired,
				Message:   "field is required",
			},
		},
	}

	got := ve.Error()
	want := "config validation failed: 1 error\n  - Database.Host: required (field is required)"

	if got != want {
		t.Errorf("ValidationError.Error() with single error\ngot:  %q\nwant: %q", got, want)
	}
}

func TestValidationError_Error_MultipleErrors(t *testing.T) {
	ve := &ValidationError{
		FieldErrors: []FieldError{
			{
				FieldPath: "Database.Host",
				Code:      ErrCodeRequired,
				Message:   "field is required",
			},
			{
				FieldPath: "Database.Port",
				Code:      ErrCodeMin,
				Message:   "value must be at least 1",
			},
			{
				FieldPath: "Server.Mode",
				Code:      ErrCodeOneOf,
				Message:   "must be one of: dev, prod",
			},
		},
	}

	got := ve.Error()

	// Check header
	if !strings.HasPrefix(got, "config validation failed: 3 errors\n") {
		t.Errorf("ValidationError.Error() header incorrect\ngot: %q", got)
	}

	// Check each error is present
	expectedErrors := []string{
		"  - Database.Host: required (field is required)",
		"  - Database.Port: min (value must be at least 1)",
		"  - Server.Mode: oneof (must be one of: dev, prod)",
	}

	for _, expected := range expectedErrors {
		if !strings.Contains(got, expected) {
			t.Errorf("ValidationError.Error() missing expected error\ngot:  %q\nwant to contain: %q", got, expected)
		}
	}
}

func TestValidationError_Error_NoErrors(t *testing.T) {
	ve := &ValidationError{
		FieldErrors: []FieldError{},
	}

	got := ve.Error()
	want := "config validation failed: no errors"

	if got != want {
		t.Errorf("ValidationError.Error() with no errors\ngot:  %q\nwant: %q", got, want)
	}
}

func TestErrorCodes(t *testing.T) {
	tests := []struct {
		name string
		code string
		want string
	}{
		{"required code", ErrCodeRequired, "required"},
		{"min code", ErrCodeMin, "min"},
		{"max code", ErrCodeMax, "max"},
		{"oneof code", ErrCodeOneOf, "oneof"},
		{"invalid_type code", ErrCodeInvalidType, "invalid_type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.code != tt.want {
				t.Errorf("error code = %q, want %q", tt.code, tt.want)
			}
		})
	}
}

func TestFieldError_Structure(t *testing.T) {
	fe := FieldError{
		FieldPath: "Config.Value",
		Code:      ErrCodeInvalidType,
		Message:   "expected string, got int",
	}

	if fe.FieldPath != "Config.Value" {
		t.Errorf("FieldError.FieldPath = %q, want %q", fe.FieldPath, "Config.Value")
	}
	if fe.Code != ErrCodeInvalidType {
		t.Errorf("FieldError.Code = %q, want %q", fe.Code, ErrCodeInvalidType)
	}
	if fe.Message != "expected string, got int" {
		t.Errorf("FieldError.Message = %q, want %q", fe.Message, "expected string, got int")
	}
}

func TestValidationError_ErrorFormatting(t *testing.T) {
	ve := &ValidationError{
		FieldErrors: []FieldError{
			{
				FieldPath: "API.Timeout",
				Code:      ErrCodeMax,
				Message:   "value must be at most 300",
			},
		},
	}

	got := ve.Error()

	// Verify no trailing newline
	if strings.HasSuffix(got, "\n\n") {
		t.Error("ValidationError.Error() should not have trailing double newline")
	}

	// Verify proper line structure
	lines := strings.Split(got, "\n")
	if len(lines) != 2 {
		t.Errorf("ValidationError.Error() should have 2 lines, got %d", len(lines))
	}

	// Verify indentation
	if !strings.HasPrefix(lines[1], "  - ") {
		t.Errorf("ValidationError.Error() field error should be indented with '  - ', got: %q", lines[1])
	}
}
