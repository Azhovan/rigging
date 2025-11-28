package rigging

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"
)

// DumpOption configures dump behavior using the functional options pattern.
type DumpOption func(*dumpConfig)

// dumpConfig holds options for DumpEffective.
type dumpConfig struct {
	withSources bool   // Include source attribution for each field
	asJSON      bool   // Output as JSON instead of text format
	indent      string // Indentation for JSON output (default: "  ")
}

// WithSources includes source attribution for each field in the output.
func WithSources() DumpOption {
	return func(cfg *dumpConfig) {
		cfg.withSources = true
	}
}

// AsJSON outputs configuration as JSON instead of text format.
func AsJSON() DumpOption {
	return func(cfg *dumpConfig) {
		cfg.asJSON = true
	}
}

// WithIndent sets the indentation for JSON output.
// Default is two spaces ("  ").
func WithIndent(indent string) DumpOption {
	return func(cfg *dumpConfig) {
		cfg.indent = indent
	}
}

// DumpEffective writes a human-readable representation of the configuration.
// Secret fields are automatically redacted as "***redacted***".
// Returns an error if writing to the writer fails.
func DumpEffective[T any](w io.Writer, cfg *T, opts ...DumpOption) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	// Apply options
	config := dumpConfig{
		indent: "  ", // Default indent
	}
	for _, opt := range opts {
		opt(&config)
	}

	// Get provenance for secret detection and source attribution
	prov, _ := GetProvenance(cfg)

	// Build a map of field paths to provenance info for quick lookup
	provenanceMap := make(map[string]*FieldProvenance)
	if prov != nil {
		for i := range prov.Fields {
			provenanceMap[prov.Fields[i].FieldPath] = &prov.Fields[i]
		}
	}

	// Walk the struct and collect field data
	v := reflect.ValueOf(cfg)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return fmt.Errorf("config must be a struct or pointer to struct")
	}

	if config.asJSON {
		return dumpAsJSON(w, v, provenanceMap, config)
	}
	return dumpAsText(w, v, provenanceMap, config)
}

// dumpAsText outputs configuration in text format (key: value).
func dumpAsText(w io.Writer, v reflect.Value, provenanceMap map[string]*FieldProvenance, config dumpConfig) error {
	fields := collectFields(v, "", provenanceMap)

	for _, field := range fields {
		line := fmt.Sprintf("%s: %s", field.keyPath, field.displayValue)
		if config.withSources && field.sourceName != "" {
			line += fmt.Sprintf(" (source: %s)", field.sourceName)
		}
		line += "\n"

		if _, err := w.Write([]byte(line)); err != nil {
			return fmt.Errorf("write error: %w", err)
		}
	}

	return nil
}

// dumpAsJSON outputs configuration as JSON with secret redaction.
func dumpAsJSON(w io.Writer, v reflect.Value, provenanceMap map[string]*FieldProvenance, config dumpConfig) error {
	// Build a nested map structure for JSON output
	result := buildJSONStructure(v, "", provenanceMap)

	// Marshal to JSON
	var data []byte
	var err error
	if config.indent != "" {
		data, err = json.MarshalIndent(result, "", config.indent)
	} else {
		data, err = json.Marshal(result)
	}

	if err != nil {
		return fmt.Errorf("json marshal error: %w", err)
	}

	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("write error: %w", err)
	}

	// Add newline for better formatting
	if _, err := w.Write([]byte("\n")); err != nil {
		return fmt.Errorf("write error: %w", err)
	}

	return nil
}

// fieldData holds information about a single field for dumping.
type fieldData struct {
	keyPath      string // Dot-separated key path (e.g., "database.host")
	displayValue string // Value to display (redacted if secret)
	sourceName   string // Source attribution
}

// collectFields recursively walks a struct and collects field data.
// fieldPathPrefix is used for provenance lookup, keyPathPrefix is used for display
func collectFields(v reflect.Value, keyPathPrefix string, provenanceMap map[string]*FieldProvenance) []fieldData {
	return collectFieldsWithPath(v, "", keyPathPrefix, provenanceMap)
}

// collectFieldsWithPath is the internal recursive function that tracks both field path and key path
func collectFieldsWithPath(v reflect.Value, fieldPathPrefix string, keyPathPrefix string, provenanceMap map[string]*FieldProvenance) []fieldData {
	var fields []fieldData

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Determine field path for provenance lookup
		fieldPath := field.Name
		if fieldPathPrefix != "" {
			fieldPath = fieldPathPrefix + "." + field.Name
		}

		// Parse tag to get custom name or prefix
		tag := field.Tag.Get("conf")
		tagCfg := parseTag(tag)

		// Get provenance info first
		var prov *FieldProvenance
		if p, ok := provenanceMap[fieldPath]; ok {
			prov = p
		}

		// Determine key path for display
		// Prefer the key path from provenance if available, otherwise derive it
		var keyPath string
		if prov != nil && prov.KeyPath != "" {
			keyPath = prov.KeyPath
		} else if tagCfg.name != "" {
			keyPath = tagCfg.name
		} else {
			// Derive from field name (lowercase first letter)
			keyPath = deriveKeyPath(field.Name)
			if keyPathPrefix != "" {
				keyPath = keyPathPrefix + "." + keyPath
			}
		}

		// Handle nested structs recursively
		if fieldValue.Kind() == reflect.Struct && field.Type.String() != "time.Time" {
			// Check if this is an Optional type
			if strings.HasPrefix(field.Type.String(), "rigging.Optional[") {
				// Handle Optional[T] - extract the value if set
				setField := fieldValue.FieldByName("Set")
				valueField := fieldValue.FieldByName("Value")
				if setField.IsValid() && setField.Bool() && valueField.IsValid() {
					displayValue := formatValue(valueField, prov)
					fields = append(fields, fieldData{
						keyPath:      keyPath,
						displayValue: displayValue,
						sourceName:   getSourceName(prov),
					})
				} else {
					// Not set, show as empty or skip
					fields = append(fields, fieldData{
						keyPath:      keyPath,
						displayValue: "<not set>",
						sourceName:   getSourceName(prov),
					})
				}
			} else {
				// Regular nested struct - recurse
				// For nested structs, use the prefix tag if present, otherwise use the key path
				var nestedKeyPrefix string
				if tagCfg.prefix != "" {
					// Use the prefix tag value
					nestedKeyPrefix = tagCfg.prefix
				} else {
					// Use the derived key path
					nestedKeyPrefix = keyPath
				}
				nestedFields := collectFieldsWithPath(fieldValue, fieldPath, nestedKeyPrefix, provenanceMap)
				fields = append(fields, nestedFields...)
			}
			continue
		}

		// Format the value (with redaction if secret)
		displayValue := formatValue(fieldValue, prov)

		fields = append(fields, fieldData{
			keyPath:      keyPath,
			displayValue: displayValue,
			sourceName:   getSourceName(prov),
		})
	}

	return fields
}

// buildJSONStructure recursively builds a nested map for JSON output.
func buildJSONStructure(v reflect.Value, prefix string, provenanceMap map[string]*FieldProvenance) map[string]any {
	result := make(map[string]any)

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Determine field path for provenance lookup
		fieldPath := field.Name
		if prefix != "" {
			fieldPath = prefix + "." + field.Name
		}

		// Parse tag
		tag := field.Tag.Get("conf")
		tagCfg := parseTag(tag)

		// Determine JSON key
		jsonKey := deriveKeyPath(field.Name)
		if tagCfg.name != "" {
			// Use custom name, but only the last component for JSON
			parts := strings.Split(tagCfg.name, ".")
			jsonKey = parts[len(parts)-1]
		}

		// Get provenance info
		var prov *FieldProvenance
		if p, ok := provenanceMap[fieldPath]; ok {
			prov = p
		}

		// Handle nested structs recursively
		if fieldValue.Kind() == reflect.Struct && field.Type.String() != "time.Time" {
			// Check if this is an Optional type
			if strings.HasPrefix(field.Type.String(), "rigging.Optional[") {
				// Handle Optional[T]
				setField := fieldValue.FieldByName("Set")
				valueField := fieldValue.FieldByName("Value")
				if setField.IsValid() && setField.Bool() && valueField.IsValid() {
					result[jsonKey] = formatValueForJSON(valueField, prov)
				} else {
					result[jsonKey] = nil
				}
			} else {
				// Regular nested struct
				nestedPrefix := fieldPath
				result[jsonKey] = buildJSONStructure(fieldValue, nestedPrefix, provenanceMap)
			}
			continue
		}

		// Format value for JSON
		result[jsonKey] = formatValueForJSON(fieldValue, prov)
	}

	return result
}

// formatValue formats a field value as a string, redacting secrets.
func formatValue(v reflect.Value, prov *FieldProvenance) string {
	// Check if this field is secret
	if prov != nil && prov.Secret {
		return "***redacted***"
	}

	return formatValueAsString(v)
}

// formatValueForJSON formats a field value for JSON output, redacting secrets.
func formatValueForJSON(v reflect.Value, prov *FieldProvenance) any {
	// Check if this field is secret
	if prov != nil && prov.Secret {
		return "***redacted***"
	}

	// Return the actual value for JSON marshaling
	if !v.IsValid() || (v.Kind() == reflect.Ptr && v.IsNil()) {
		return nil
	}

	// Handle different types
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Bool:
		return v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Special handling for time.Duration
		if v.Type().String() == "time.Duration" {
			return v.Interface().(time.Duration).String()
		}
		return v.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint()
	case reflect.Float32, reflect.Float64:
		return v.Float()
	case reflect.Slice:
		// Handle slices
		if v.Type().Elem().Kind() == reflect.String {
			slice := make([]string, v.Len())
			for i := 0; i < v.Len(); i++ {
				slice[i] = v.Index(i).String()
			}
			return slice
		}
		// For other slice types, convert to []any
		slice := make([]any, v.Len())
		for i := 0; i < v.Len(); i++ {
			slice[i] = v.Index(i).Interface()
		}
		return slice
	case reflect.Struct:
		if v.Type().String() == "time.Time" {
			return v.Interface().(time.Time).Format(time.RFC3339)
		}
		return v.Interface()
	default:
		return v.Interface()
	}
}

// formatValueAsString formats a field value as a string for text output.
func formatValueAsString(v reflect.Value) string {
	if !v.IsValid() || (v.Kind() == reflect.Ptr && v.IsNil()) {
		return "<nil>"
	}

	switch v.Kind() {
	case reflect.String:
		return fmt.Sprintf("%q", v.String())
	case reflect.Bool:
		return fmt.Sprintf("%t", v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Special handling for time.Duration
		if v.Type().String() == "time.Duration" {
			return v.Interface().(time.Duration).String()
		}
		return fmt.Sprintf("%d", v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", v.Uint())
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%g", v.Float())
	case reflect.Slice:
		// Handle slices
		if v.Type().Elem().Kind() == reflect.String {
			strs := make([]string, v.Len())
			for i := 0; i < v.Len(); i++ {
				strs[i] = v.Index(i).String()
			}
			return fmt.Sprintf("[%s]", strings.Join(strs, ", "))
		}
		return fmt.Sprintf("%v", v.Interface())
	case reflect.Struct:
		if v.Type().String() == "time.Time" {
			return v.Interface().(time.Time).Format(time.RFC3339)
		}
		return fmt.Sprintf("%v", v.Interface())
	default:
		return fmt.Sprintf("%v", v.Interface())
	}
}

// deriveKeyPath derives a key path from a field name (lowercase first letter).
func deriveKeyPath(fieldName string) string {
	if fieldName == "" {
		return ""
	}
	return strings.ToLower(fieldName[:1]) + fieldName[1:]
}

// getSourceName extracts the source name from provenance, or returns empty string.
func getSourceName(prov *FieldProvenance) string {
	if prov == nil {
		return ""
	}
	return prov.SourceName
}
