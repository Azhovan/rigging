package rigging

import (
	"fmt"
	"strings"
)

// Error codes for validation failures.
const (
	ErrCodeRequired    = "required"
	ErrCodeMin         = "min"
	ErrCodeMax         = "max"
	ErrCodeOneOf       = "oneof"
	ErrCodeInvalidType = "invalid_type"
)

// ValidationError aggregates field-level validation failures.
type ValidationError struct {
	FieldErrors []FieldError
}

// Error formats validation errors as a multi-line message.
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
	FieldPath string // Dot notation (e.g., "Database.Host")
	Code      string // Error code (e.g., "required", "min")
	Message   string // Human-readable description
}
