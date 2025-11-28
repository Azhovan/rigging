package rigging

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// validateField validates a single field value against tag-based constraints.
// It checks required, min, max, and oneof constraints based on the field's type.
// Returns a slice of FieldError for any validation failures.
func validateField(fieldValue reflect.Value, fieldPath string, tags tagConfig) []FieldError {
	var errors []FieldError

	// Check required constraint
	if tags.required {
		if isZeroValue(fieldValue) {
			errors = append(errors, FieldError{
				FieldPath: fieldPath,
				Code:      ErrCodeRequired,
				Message:   "field is required but not provided",
			})
			// If required and zero, skip other validations
			return errors
		}
	}

	// Skip other validations if value is zero (for non-required fields)
	if isZeroValue(fieldValue) {
		return errors
	}

	// Validate min/max constraints based on type
	switch fieldValue.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		errors = append(errors, validateIntMinMax(fieldValue, fieldPath, tags)...)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		errors = append(errors, validateUintMinMax(fieldValue, fieldPath, tags)...)
	case reflect.Float32, reflect.Float64:
		errors = append(errors, validateFloatMinMax(fieldValue, fieldPath, tags)...)
	case reflect.String:
		errors = append(errors, validateStringMinMax(fieldValue, fieldPath, tags)...)
	}

	// Validate oneof constraint
	if len(tags.oneof) > 0 {
		errors = append(errors, validateOneof(fieldValue, fieldPath, tags)...)
	}

	return errors
}

// validateStruct walks a struct and validates all fields according to their tags.
// It recursively validates nested structs.
// Returns a slice of all FieldError encountered.
func validateStruct(cfg reflect.Value) []FieldError {
	return validateStructRecursive(cfg, "")
}

// validateStructRecursive is the internal recursive implementation of validateStruct.
func validateStructRecursive(cfg reflect.Value, parentFieldPath string) []FieldError {
	var fieldErrors []FieldError

	// Dereference pointer if needed
	if cfg.Kind() == reflect.Ptr {
		if cfg.IsNil() {
			return fieldErrors
		}
		cfg = cfg.Elem()
	}

	// Ensure we have a struct
	if cfg.Kind() != reflect.Struct {
		return fieldErrors
	}

	cfgType := cfg.Type()

	// Walk through all fields
	for i := 0; i < cfg.NumField(); i++ {
		field := cfgType.Field(i)
		fieldValue := cfg.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Build field path
		fieldPath := field.Name
		if parentFieldPath != "" {
			fieldPath = parentFieldPath + "." + field.Name
		}

		// Parse struct tag
		tag := field.Tag.Get("conf")
		tagCfg := parseTag(tag)

		// Handle Optional[T] types - validate the inner value if set
		if isOptionalType(fieldValue.Type()) {
			setField := fieldValue.Field(1) // Set field
			if setField.Bool() {
				valueField := fieldValue.Field(0) // Value field
				// Validate the inner value
				errors := validateField(valueField, fieldPath, tagCfg)
				fieldErrors = append(fieldErrors, errors...)
			}
			continue
		}

		// Handle nested structs recursively
		if fieldValue.Kind() == reflect.Struct {
			// Skip time.Time and time.Duration (they're structs but should be treated as primitives)
			if fieldValue.Type().PkgPath() == "time" {
				// Validate as a regular field
				errors := validateField(fieldValue, fieldPath, tagCfg)
				fieldErrors = append(fieldErrors, errors...)
				continue
			}

			// Recursively validate nested struct
			nestedErrors := validateStructRecursive(fieldValue, fieldPath)
			fieldErrors = append(fieldErrors, nestedErrors...)
			continue
		}

		// Validate the field
		errors := validateField(fieldValue, fieldPath, tagCfg)
		fieldErrors = append(fieldErrors, errors...)
	}

	return fieldErrors
}

// isZeroValue checks if a reflect.Value is the zero value for its type.
func isZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	case reflect.Struct:
		// For structs, check if all fields are zero
		return v.IsZero()
	default:
		return v.IsZero()
	}
}

// validateIntMinMax validates min/max constraints for signed integer types.
func validateIntMinMax(fieldValue reflect.Value, fieldPath string, tags tagConfig) []FieldError {
	var errors []FieldError
	value := fieldValue.Int()

	if tags.min != "" {
		minVal, err := strconv.ParseInt(tags.min, 10, 64)
		if err == nil && value < minVal {
			errors = append(errors, FieldError{
				FieldPath: fieldPath,
				Code:      ErrCodeMin,
				Message:   fmt.Sprintf("value %d is below minimum %d", value, minVal),
			})
		}
	}

	if tags.max != "" {
		maxVal, err := strconv.ParseInt(tags.max, 10, 64)
		if err == nil && value > maxVal {
			errors = append(errors, FieldError{
				FieldPath: fieldPath,
				Code:      ErrCodeMax,
				Message:   fmt.Sprintf("value %d exceeds maximum %d", value, maxVal),
			})
		}
	}

	return errors
}

// validateUintMinMax validates min/max constraints for unsigned integer types.
func validateUintMinMax(fieldValue reflect.Value, fieldPath string, tags tagConfig) []FieldError {
	var errors []FieldError
	value := fieldValue.Uint()

	if tags.min != "" {
		minVal, err := strconv.ParseUint(tags.min, 10, 64)
		if err == nil && value < minVal {
			errors = append(errors, FieldError{
				FieldPath: fieldPath,
				Code:      ErrCodeMin,
				Message:   fmt.Sprintf("value %d is below minimum %d", value, minVal),
			})
		}
	}

	if tags.max != "" {
		maxVal, err := strconv.ParseUint(tags.max, 10, 64)
		if err == nil && value > maxVal {
			errors = append(errors, FieldError{
				FieldPath: fieldPath,
				Code:      ErrCodeMax,
				Message:   fmt.Sprintf("value %d exceeds maximum %d", value, maxVal),
			})
		}
	}

	return errors
}

// validateFloatMinMax validates min/max constraints for floating-point types.
func validateFloatMinMax(fieldValue reflect.Value, fieldPath string, tags tagConfig) []FieldError {
	var errors []FieldError
	value := fieldValue.Float()

	if tags.min != "" {
		minVal, err := strconv.ParseFloat(tags.min, 64)
		if err == nil && value < minVal {
			errors = append(errors, FieldError{
				FieldPath: fieldPath,
				Code:      ErrCodeMin,
				Message:   fmt.Sprintf("value %g is below minimum %g", value, minVal),
			})
		}
	}

	if tags.max != "" {
		maxVal, err := strconv.ParseFloat(tags.max, 64)
		if err == nil && value > maxVal {
			errors = append(errors, FieldError{
				FieldPath: fieldPath,
				Code:      ErrCodeMax,
				Message:   fmt.Sprintf("value %g exceeds maximum %g", value, maxVal),
			})
		}
	}

	return errors
}

// validateStringMinMax validates min/max constraints for string length.
func validateStringMinMax(fieldValue reflect.Value, fieldPath string, tags tagConfig) []FieldError {
	var errors []FieldError
	value := fieldValue.String()
	length := len(value)

	if tags.min != "" {
		minLen, err := strconv.Atoi(tags.min)
		if err == nil && length < minLen {
			errors = append(errors, FieldError{
				FieldPath: fieldPath,
				Code:      ErrCodeMin,
				Message:   fmt.Sprintf("string length %d is below minimum %d", length, minLen),
			})
		}
	}

	if tags.max != "" {
		maxLen, err := strconv.Atoi(tags.max)
		if err == nil && length > maxLen {
			errors = append(errors, FieldError{
				FieldPath: fieldPath,
				Code:      ErrCodeMax,
				Message:   fmt.Sprintf("string length %d exceeds maximum %d", length, maxLen),
			})
		}
	}

	return errors
}

// validateOneof validates that a field value is one of the allowed options.
func validateOneof(fieldValue reflect.Value, fieldPath string, tags tagConfig) []FieldError {
	var errors []FieldError

	// Convert field value to string for comparison
	var valueStr string
	switch fieldValue.Kind() {
	case reflect.String:
		valueStr = fieldValue.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		valueStr = strconv.FormatInt(fieldValue.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		valueStr = strconv.FormatUint(fieldValue.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		valueStr = strconv.FormatFloat(fieldValue.Float(), 'f', -1, 64)
	case reflect.Bool:
		valueStr = strconv.FormatBool(fieldValue.Bool())
	default:
		// For unsupported types, skip oneof validation
		return errors
	}

	// Check if value is in the allowed set
	found := false
	for _, allowed := range tags.oneof {
		if valueStr == allowed {
			found = true
			break
		}
	}

	if !found {
		errors = append(errors, FieldError{
			FieldPath: fieldPath,
			Code:      ErrCodeOneOf,
			Message:   fmt.Sprintf("value %q must be one of: %s", valueStr, strings.Join(tags.oneof, ", ")),
		})
	}

	return errors
}
