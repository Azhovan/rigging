package rigging

import (
	"fmt"
	"strings"
)

// Standard error codes for validation failures
const (
	ErrCodeRequired    = "required"
	ErrCodeMin         = "min"
	ErrCodeMax         = "max"
	ErrCodeOneOf       = "oneof"
	ErrCodeInvalidType = "invalid_type"
)

// ValidationError represents one or more validation failures.
// It contains all field-level errors encountered during configuration
// loading and validation.
type ValidationError struct {
	FieldErrors []FieldError
}

// Error formats the validation error as a multi-line message listing
// all field validation failures.
func (e *ValidationError) Error() string {
	if len(e.FieldErrors) == 0 {
		return "config validation failed: no errors"
	}

	var b strings.Builder
	if len(e.FieldErrors) == 1 {
		b.WriteString("config validation failed: 1 error\n")
	} else {
		fmt.Fprintf(&b, "config validation failed: %d errors\n", len(e.FieldErrors))
	}

	for _, fe := range e.FieldErrors {
		fmt.Fprintf(&b, "  - %s: %s (%s)\n", fe.FieldPath, fe.Code, fe.Message)
	}

	return strings.TrimRight(b.String(), "\n")
}

// FieldError represents a single field validation failure.
type FieldError struct {
	// FieldPath is the logical path to the field (e.g., "Database.Host")
	FieldPath string

	// Code is the error code (e.g., "required", "min", "max", "oneof", "invalid_type")
	Code string

	// Message is a human-readable description of the error
	Message string
}
