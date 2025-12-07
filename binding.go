package rigging

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

// tagConfig holds parsed directives from a struct field's `conf` tag.
type tagConfig struct {
	env        string   // Environment variable name (env:VAR_NAME)
	name       string   // Custom key path (name:custom.path)
	prefix     string   // Prefix for nested structs (prefix:foo)
	defValue   string   // Default value (default:value)
	min        string   // Minimum constraint (min:N)
	max        string   // Maximum constraint (max:M)
	oneof      []string // Allowed values (oneof:a,b,c)
	required   bool     // Field is required (required or required:true)
	secret     bool     // Field is secret (secret or secret:true)
	hasDefault bool     // Whether a default directive was present
}

// parseTag parses a `conf` struct tag into a structured tagConfig.
// Tag format: "directive1:value1,directive2:value2,..."
// Boolean directives can omit `:true` (e.g., "required" == "required:true")
func parseTag(tag string) tagConfig {
	cfg := tagConfig{}

	if tag == "" {
		return cfg
	}

	directives := extractTagDirectives(tag)

	for _, directive := range directives {
		// remove empty/invalid tags
		directive = strings.TrimSpace(directive)
		if directive == "" {
			continue
		}

		// Split by colon to separate directive name from value
		parts := strings.SplitN(directive, ":", 2)
		name := strings.TrimSpace(parts[0])

		var value string
		if len(parts) > 1 {
			value = parts[1] // Don't trim value - empty strings may be intentional
		}

		switch name {
		case "env":
			cfg.env = value
		case "name":
			cfg.name = value
		case "prefix":
			cfg.prefix = value
		case "default":
			cfg.defValue = value
			cfg.hasDefault = true
		case "min":
			cfg.min = value
		case "max":
			cfg.max = value
		case "oneof":
			// Empty or duplicated values are ignored.
			// The final result is sorted.
			if value != "" {
				parts := strings.Split(value, ",")
				seen := make(map[string]bool)
				for _, v := range parts {
					trimmed := strings.TrimSpace(v)
					if trimmed == "" || seen[trimmed] {
						continue
					}

					cfg.oneof = append(cfg.oneof, trimmed)
					seen[trimmed] = true
				}

				sort.Strings(cfg.oneof)
			}
		case "required":
			// No value or explicit "true" means true
			if value == "" || value == "true" {
				cfg.required = true
			} else if value == "false" {
				cfg.required = false
			} else {
				// Invalid value, default to true for safety
				cfg.required = true
			}
		case "secret":
			// No value or explicit "true" means true
			if value == "" || value == "true" {
				cfg.secret = true
			} else if value == "false" {
				cfg.secret = false
			} else {
				// Invalid value, default to true for safety
				cfg.secret = true
			}
		}
	}

	return cfg
}

// extractTagDirectives extracts individual directives from a tag string.
// It handles the special case where oneof values contain commas.
// It doesn't validate the tags, validation happens in parseTag().
func extractTagDirectives(tag string) []string {
	var directives []string
	var current strings.Builder
	inOneof := false

	for i := 0; i < len(tag); i++ {
		ch := tag[i]

		// Check if we're entering an oneof directive
		if !inOneof && i+6 <= len(tag) && tag[i:i+6] == "oneof:" {
			inOneof = true
			current.WriteString("oneof:")
			i += 5 // Skip past "oneof:"
			continue
		}

		if ch == ',' {
			if inOneof {
				// Check if the next directive starts after this comma
				// Look ahead to see if we have a known directive name
				remaining := tag[i+1:]
				if startsWithDirective(remaining) {
					// This comma ends the oneof directive
					inOneof = false
					directives = append(directives, current.String())
					current.Reset()
					continue
				} else {
					// This comma is part of oneof values
					current.WriteByte(ch)
				}
			} else {
				// Regular comma separator between directives
				directives = append(directives, current.String())
				current.Reset()
			}
		} else {
			current.WriteByte(ch)
		}
	}

	// Add the last directive
	if current.Len() > 0 {
		directives = append(directives, current.String())
	}

	return directives
}

// startsWithDirective checks if a string starts with a known directive name.
func startsWithDirective(s string) bool {
	s = strings.TrimSpace(s)
	directives := []string{"env:", "name:", "prefix:", "default:", "min:", "max:", "oneof:", "required", "secret"}
	for _, d := range directives {
		if strings.HasPrefix(s, d) {
			return true
		}
	}
	return false
}

// convertValue converts a raw value to the target type using reflection.
// It supports:
// - string, bool
// - int, int8, int16, int32, int64
// - uint, uint8, uint16, uint32, uint64
// - float32, float64
// - time.Duration (parsed from strings like "5s", "10m", "1h")
// - time.Time (parsed from RFC3339, RFC3339Nano, and common date formats)
// - []string (from comma-separated strings or arrays)
// - nested structs (returned as-is for recursive binding)
// - Optional[T] types
//
// Returns an error with type information if conversion fails.
func convertValue(rawValue any, targetType reflect.Type) (any, error) {
	// Handle nil values
	if rawValue == nil {
		return reflect.Zero(targetType).Interface(), nil
	}

	// Check if target is Optional[T]
	if isOptionalType(targetType) {
		// Extract the inner type T from Optional[T]
		innerType := targetType.Field(0).Type
		innerValue, err := convertValue(rawValue, innerType)
		if err != nil {
			return nil, err
		}

		// Create Optional[T] with Set=true
		optionalVal := reflect.New(targetType).Elem()
		optionalVal.Field(0).Set(reflect.ValueOf(innerValue))
		optionalVal.Field(1).SetBool(true) // Set field
		return optionalVal.Interface(), nil
	}

	// If rawValue is already the target type, return as-is
	rawType := reflect.TypeOf(rawValue)
	if rawType == targetType {
		return rawValue, nil
	}

	// Handle time.Time specially before generic struct handling
	if targetType == reflect.TypeOf(time.Time{}) {
		switch v := rawValue.(type) {
		case string:
			// Try multiple common time formats
			formats := []string{
				time.RFC3339,
				time.RFC3339Nano,
				"2006-01-02T15:04:05Z07:00",
				"2006-01-02 15:04:05",
				"2006-01-02",
			}
			for _, format := range formats {
				if t, err := time.Parse(format, v); err == nil {
					return t, nil
				}
			}
			return nil, fmt.Errorf("cannot parse %q as time.Time (tried RFC3339, RFC3339Nano, and common formats)", v)
		case time.Time:
			return v, nil
		default:
			return nil, fmt.Errorf("cannot convert %T to time.Time", rawValue)
		}
	}

	// Handle nested structs - return as-is for recursive binding
	if targetType.Kind() == reflect.Struct {
		// If rawValue is a map, it will be handled by recursive binding
		if rawType.Kind() == reflect.Map {
			return rawValue, nil
		}
		// If rawValue is already a struct, return as-is
		if rawType.Kind() == reflect.Struct {
			return rawValue, nil
		}
	}

	// Convert to string first for easier parsing
	var strValue string
	switch v := rawValue.(type) {
	case string:
		strValue = v
	case []byte:
		strValue = string(v)
	default:
		// For non-string types, use fmt.Sprint
		strValue = fmt.Sprint(rawValue)
	}

	// Handle target type conversion
	switch targetType.Kind() {
	case reflect.String:
		return strValue, nil

	case reflect.Bool:
		return parseBool(strValue)

	case reflect.Int:
		val, err := strconv.ParseInt(strValue, 10, 0)
		if err != nil {
			return nil, fmt.Errorf("cannot convert %q to int: %w", strValue, err)
		}
		return int(val), nil

	case reflect.Int8:
		val, err := strconv.ParseInt(strValue, 10, 8)
		if err != nil {
			return nil, fmt.Errorf("cannot convert %q to int8: %w", strValue, err)
		}
		return int8(val), nil

	case reflect.Int16:
		val, err := strconv.ParseInt(strValue, 10, 16)
		if err != nil {
			return nil, fmt.Errorf("cannot convert %q to int16: %w", strValue, err)
		}
		return int16(val), nil

	case reflect.Int32:
		val, err := strconv.ParseInt(strValue, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("cannot convert %q to int32: %w", strValue, err)
		}
		return int32(val), nil

	case reflect.Int64:
		// Special case: time.Duration is an int64
		if targetType == reflect.TypeOf(time.Duration(0)) {
			duration, err := time.ParseDuration(strValue)
			if err != nil {
				return nil, fmt.Errorf("cannot convert %q to time.Duration: %w", strValue, err)
			}
			return duration, nil
		}

		val, err := strconv.ParseInt(strValue, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("cannot convert %q to int64: %w", strValue, err)
		}
		return val, nil

	case reflect.Uint:
		val, err := strconv.ParseUint(strValue, 10, 0)
		if err != nil {
			return nil, fmt.Errorf("cannot convert %q to uint: %w", strValue, err)
		}
		return uint(val), nil

	case reflect.Uint8:
		val, err := strconv.ParseUint(strValue, 10, 8)
		if err != nil {
			return nil, fmt.Errorf("cannot convert %q to uint8: %w", strValue, err)
		}
		return uint8(val), nil

	case reflect.Uint16:
		val, err := strconv.ParseUint(strValue, 10, 16)
		if err != nil {
			return nil, fmt.Errorf("cannot convert %q to uint16: %w", strValue, err)
		}
		return uint16(val), nil

	case reflect.Uint32:
		val, err := strconv.ParseUint(strValue, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("cannot convert %q to uint32: %w", strValue, err)
		}
		return uint32(val), nil

	case reflect.Uint64:
		val, err := strconv.ParseUint(strValue, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("cannot convert %q to uint64: %w", strValue, err)
		}
		return val, nil

	case reflect.Float32:
		val, err := strconv.ParseFloat(strValue, 32)
		if err != nil {
			return nil, fmt.Errorf("cannot convert %q to float32: %w", strValue, err)
		}
		return float32(val), nil

	case reflect.Float64:
		val, err := strconv.ParseFloat(strValue, 64)
		if err != nil {
			return nil, fmt.Errorf("cannot convert %q to float64: %w", strValue, err)
		}
		return val, nil

	case reflect.Slice:
		// Handle []string
		if targetType.Elem().Kind() == reflect.String {
			return parseStringSlice(rawValue)
		}
		return nil, fmt.Errorf("unsupported slice type: %s", targetType)

	default:
		return nil, fmt.Errorf("unsupported target type: %s", targetType)
	}
}

// parseBool parses a boolean value from a string.
// Accepts: "true", "false", "1", "0", "yes", "no" (case-insensitive)
func parseBool(s string) (bool, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "true", "1", "yes":
		return true, nil
	case "false", "0", "no":
		return false, nil
	default:
		return false, fmt.Errorf("cannot convert %q to bool", s)
	}
}

// parseStringSlice converts a value to []string.
// Handles:
// - []string: return as-is
// - []any: convert each element to string
// - string: split by comma
func parseStringSlice(rawValue any) ([]string, error) {
	switch v := rawValue.(type) {
	case []string:
		return v, nil
	case []any:
		result := make([]string, len(v))
		for i, item := range v {
			result[i] = fmt.Sprint(item)
		}
		return result, nil
	case string:
		// Split by comma and trim whitespace
		if v == "" {
			return []string{}, nil
		}
		parts := strings.Split(v, ",")
		result := make([]string, len(parts))
		for i, part := range parts {
			result[i] = strings.TrimSpace(part)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("cannot convert %T to []string", rawValue)
	}
}

// mergedEntry represents a configuration value with its source information.
type mergedEntry struct {
	value      any
	sourceName string
	sourceKey  string // Original key from the source (e.g., "API_DATABASE__PASSWORD")
}

// bindStruct binds configuration data to a struct using reflection.
// It walks struct fields recursively, parses tags, looks up values in the data map,
// applies defaults, converts types, and records provenance.
// All errors are collected and returned together rather than failing fast.
func bindStruct(target reflect.Value, data map[string]mergedEntry, provenanceFields *[]FieldProvenance, parentPrefix string, parentFieldPath string) []FieldError {
	var fieldErrors []FieldError

	// Ensure the target is a struct
	if target.Kind() == reflect.Ptr {
		target = target.Elem()
	}
	if target.Kind() != reflect.Struct {
		return fieldErrors
	}

	targetType := target.Type()

	// Walk through all fields
	for i := 0; i < target.NumField(); i++ {
		field := targetType.Field(i)
		fieldValue := target.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Parse struct tag
		tag := field.Tag.Get("conf")
		tagCfg := parseTag(tag)

		// Determine the field path for provenance (e.g., "Database.Host")
		fieldPath := field.Name
		if parentFieldPath != "" {
			fieldPath = parentFieldPath + "." + field.Name
		}

		// Determine the key path for lookup
		keyPath := determineKeyPath(field.Name, tagCfg, parentPrefix)

		// Handle nested structs with prefix
		if fieldValue.Kind() == reflect.Struct && tagCfg.prefix != "" {
			// Recursively bind nested struct with new prefix
			nestedErrors := bindStruct(fieldValue, data, provenanceFields, tagCfg.prefix, fieldPath)
			fieldErrors = append(fieldErrors, nestedErrors...)
			continue
		}

		// Handle nested structs (non-prefix case) - check this before looking up values
		// because nested structs might not have a direct value in the data map
		if fieldValue.Kind() == reflect.Struct && !isOptionalType(fieldValue.Type()) && fieldValue.Type() != reflect.TypeOf(time.Time{}) && fieldValue.Type() != reflect.TypeOf(time.Duration(0)) {
			// Look up value in data map to see if there's a direct map value
			entry, found := data[keyPath]

			// Check if rawValue is a map (from file sources)
			if found && entry.value != nil {
				if rawMap, ok := entry.value.(map[string]any); ok {
					// Convert map entries to mergedEntry format
					nestedData := make(map[string]mergedEntry)
					for k, v := range rawMap {
						nestedData[k] = mergedEntry{value: v, sourceName: entry.sourceName}
					}
					nestedErrors := bindStruct(fieldValue, nestedData, provenanceFields, "", fieldPath)
					fieldErrors = append(fieldErrors, nestedErrors...)
					continue
				}
			}
			// Otherwise, try recursive binding with current data and prefix
			// This handles the case where nested fields are flattened with dot notation
			nestedErrors := bindStruct(fieldValue, data, provenanceFields, keyPath, fieldPath)
			fieldErrors = append(fieldErrors, nestedErrors...)
			continue
		}

		// Look up value in data map
		entry, found := data[keyPath]
		var rawValue any
		var sourceName string

		if found {
			rawValue = entry.value
			sourceName = entry.sourceName
		} else if tagCfg.hasDefault {
			// Apply default value
			rawValue = tagCfg.defValue
			sourceName = "default"
		}

		// If no value found and no default, leave as zero value
		// The validation phase will check if the field is required
		if !found && !tagCfg.hasDefault {
			continue
		}

		// Convert value to target type
		convertedValue, err := convertValue(rawValue, fieldValue.Type())
		if err != nil {
			fieldErrors = append(fieldErrors, FieldError{
				FieldPath: fieldPath,
				Code:      ErrCodeInvalidType,
				Message:   fmt.Sprintf("type conversion failed: %v", err),
			})
			continue
		}

		// Set field value
		if fieldValue.CanSet() {
			fieldValue.Set(reflect.ValueOf(convertedValue))

			// Record provenance
			if provenanceFields != nil {
				// Use sourceKey from entry if available, otherwise use sourceName
				sourceInfo := sourceName
				if found && entry.sourceKey != "" {
					sourceInfo = entry.sourceKey
				}

				*provenanceFields = append(*provenanceFields, FieldProvenance{
					FieldPath:  fieldPath,
					KeyPath:    keyPath,
					SourceName: sourceInfo,
					Secret:     tagCfg.secret,
				})
			}
		}
	}

	return fieldErrors
}

// determineKeyPath determines the configuration key path for a field.
// Priority: name tag > prefix + derived > derived
// All keys are normalized to lowercase for consistent matching.
func determineKeyPath(fieldName string, tagCfg tagConfig, parentPrefix string) string {
	// If the name tag is specified, use it directly (ignores prefix)
	if tagCfg.name != "" {
		return strings.ToLower(tagCfg.name)
	}

	// Derive key from field name (fully lowercase)
	derived := deriveFieldKey(fieldName)

	// Apply parent prefix if present (normalize prefix too)
	if parentPrefix != "" {
		return strings.ToLower(parentPrefix) + "." + derived
	}

	return derived
}

// deriveFieldKey derives a configuration key from a field name.
// It fully lowercases the field name to match source key normalization.
func deriveFieldKey(fieldName string) string {
	if fieldName == "" {
		return ""
	}

	return strings.ToLower(fieldName)
}

// isOptionalType checks if a type is an Optional[T] type.
func isOptionalType(t reflect.Type) bool {
	if t.Kind() != reflect.Struct {
		return false
	}
	if t.NumField() != 2 {
		return false
	}
	if t.Field(0).Name != "Value" {
		return false
	}
	if t.Field(1).Name != "Set" || t.Field(1).Type.Kind() != reflect.Bool {
		return false
	}
	return true
}
