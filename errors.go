package rigging

import (
	"fmt"
	"strings"
)

// Error codes for validation failures.
const (
	ErrCodeRequired    = "required"     // Field is required but not provided
	ErrCodeMin         = "min"          // Value is below minimum constraint
	ErrCodeMax         = "max"          // Value exceeds maximum constraint
	ErrCodeOneOf       = "oneof"        // Value is not in the allowed set
	ErrCodeInvalidType = "invalid_type" // Type conversion failed
	ErrCodeUnknownKey  = "unknown_key"  // Configuration key doesn't map to any field (strict mode)
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
