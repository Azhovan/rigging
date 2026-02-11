package rigging

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"
)

// DumpOption configures dump behavior.
type DumpOption func(*dumpConfig)

// dumpConfig holds options for DumpEffective.
type dumpConfig struct {
	withSources bool   // Include source attribution for each field
	asJSON      bool   // Output as JSON instead of text format
	indent      string // Indentation for JSON output (default: "  ")
}

// WithSources includes source attribution in output.
func WithSources() DumpOption {
	return func(cfg *dumpConfig) {
		cfg.withSources = true
	}
}

// AsJSON outputs configuration as JSON. Secrets are still redacted.
func AsJSON() DumpOption {
	return func(cfg *dumpConfig) {
		cfg.asJSON = true
	}
}

// WithIndent sets JSON indentation (default: "  "). No effect for text output.
func WithIndent(indent string) DumpOption {
	return func(cfg *dumpConfig) {
		cfg.indent = indent
	}
}

// DumpEffective writes configuration with automatic secret redaction.
// Supports text or JSON format. Use WithSources(), AsJSON(), WithIndent() options.
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

	provenanceMap := buildProvenanceMap(cfg)
	v, ok := getStructRootValue(cfg)
	if !ok {
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
	result := buildJSONStructure(v, "", provenanceMap, config.withSources)

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

// collectFields walks a struct and collects flattened display fields.
// keyPathPrefix is used for display-key composition in recursive calls.
func collectFields(v reflect.Value, keyPathPrefix string, provenanceMap map[string]*FieldProvenance) []fieldData {
	var fields []fieldData

	walkFlatFields(v, "", keyPathPrefix, provenanceMap, false, func(w walkedField) {
		displayValue := "<not set>"
		if w.set {
			displayValue = formatValue(w.value, w.secret)
		}

		fields = append(fields, fieldData{
			keyPath:      w.keyPath,
			displayValue: displayValue,
			sourceName:   getSourceName(w.provenance),
		})
	})

	return fields
}

// buildJSONStructure recursively builds a nested map for JSON output.
func buildJSONStructure(v reflect.Value, prefix string, provenanceMap map[string]*FieldProvenance, withSources bool) map[string]any {
	return buildJSONStructureWithSecret(v, prefix, provenanceMap, withSources, false)
}

func buildJSONStructureWithSecret(v reflect.Value, prefix string, provenanceMap map[string]*FieldProvenance, withSources bool, inheritedSecret bool) map[string]any {
	result := make(map[string]any)

	t := v.Type()
	for _, meta := range getStructFieldMeta(t) {
		field := meta.field
		fieldValue := v.Field(meta.index)
		tagCfg := meta.tagCfg

		// Determine field path for provenance lookup
		fieldPath := field.Name
		if prefix != "" {
			fieldPath = prefix + "." + field.Name
		}

		// Determine JSON key
		jsonKey := deriveFieldKey(field.Name)
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
		isSecret := shouldRedactField(tagCfg, prov, inheritedSecret)

		// Handle nested structs recursively
		if fieldValue.Kind() == reflect.Struct && field.Type != timeType {
			// Check if this is an Optional type
			if isOptionalType(field.Type) {
				// Handle Optional[T]
				setField := fieldValue.FieldByName("Set")
				valueField := fieldValue.FieldByName("Value")
				if setField.IsValid() && setField.Bool() && valueField.IsValid() {
					result[jsonKey] = buildJSONFieldValue(formatValueForJSON(valueField, isSecret), prov, withSources)
				} else {
					result[jsonKey] = nil
				}
			} else {
				// Regular nested struct
				nestedPrefix := fieldPath
				result[jsonKey] = buildJSONStructureWithSecret(fieldValue, nestedPrefix, provenanceMap, withSources, isSecret)
			}
			continue
		}

		// Format value for JSON
		result[jsonKey] = buildJSONFieldValue(formatValueForJSON(fieldValue, isSecret), prov, withSources)
	}

	return result
}

// buildJSONFieldValue wraps a value with source information if requested.
func buildJSONFieldValue(value any, prov *FieldProvenance, withSources bool) any {
	if !withSources || prov == nil || prov.SourceName == "" {
		return value
	}

	// When sources are requested, return an object with value and source
	return map[string]any{
		"value":  value,
		"source": prov.SourceName,
	}
}

// formatValue formats a field value as a string, redacting secrets.
func formatValue(v reflect.Value, secret bool) string {
	if secret {
		return "***redacted***"
	}

	return formatValueAsString(v)
}

// formatValueForJSON formats a field value for JSON output, redacting secrets.
func formatValueForJSON(v reflect.Value, secret bool) any {
	return formatStructuredValue(v, secret)
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
		if v.Type() == durationType {
			if dur, ok := v.Interface().(time.Duration); ok {
				return dur.String()
			}
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
		if v.Type() == timeType {
			if t, ok := v.Interface().(time.Time); ok {
				return t.Format(time.RFC3339)
			}
		}
		return fmt.Sprintf("%v", v.Interface())
	default:
		return fmt.Sprintf("%v", v.Interface())
	}
}

// getSourceName extracts the source name from provenance, or returns empty string.
func getSourceName(prov *FieldProvenance) string {
	if prov == nil {
		return ""
	}
	return prov.SourceName
}
