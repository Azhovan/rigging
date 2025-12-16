package rigging

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

// MaxSnapshotSize is the maximum allowed snapshot size (100MB).
const MaxSnapshotSize = 100 * 1024 * 1024

// SnapshotVersion is the current snapshot format version.
const SnapshotVersion = "1.0"

// Snapshot errors.
var (
	// ErrSnapshotTooLarge is returned when a snapshot exceeds MaxSnapshotSize.
	ErrSnapshotTooLarge = errors.New("rigging: snapshot exceeds 100MB size limit")

	// ErrNilConfig is returned when CreateSnapshot receives a nil config.
	ErrNilConfig = errors.New("rigging: config is nil")

	// ErrUnsupportedVersion is returned when reading a snapshot with unknown version.
	ErrUnsupportedVersion = errors.New("rigging: unsupported snapshot version")
)

// supportedVersions lists snapshot format versions that can be read.
// Used by ReadSnapshot in Phase 5.
//
//nolint:unused // Will be used by ReadSnapshot implementation
var supportedVersions = map[string]bool{
	"1.0": true,
}

// ConfigSnapshot represents a point-in-time configuration capture.
type ConfigSnapshot struct {
	// Version is the snapshot format version (currently "1.0")
	Version string `json:"version"`

	// Timestamp is when the snapshot was created
	Timestamp time.Time `json:"timestamp"`

	// Config contains flattened configuration values with secrets redacted.
	// Keys are dot-notation paths (e.g., "database.host").
	Config map[string]any `json:"config"`

	// Provenance tracks the source of each configuration field.
	Provenance []FieldProvenance `json:"provenance"`
}

// SnapshotOption configures snapshot creation behavior.
type SnapshotOption func(*snapshotConfig)

// snapshotConfig holds internal configuration for snapshot creation.
type snapshotConfig struct {
	excludeFields []string // Field paths to exclude
}

// WithExcludeFields excludes specified field paths from the snapshot.
// Paths use dot notation (e.g., "database.password", "cache.redis.url").
func WithExcludeFields(paths ...string) SnapshotOption {
	return func(cfg *snapshotConfig) {
		cfg.excludeFields = append(cfg.excludeFields, paths...)
	}
}

// CreateSnapshot captures the current configuration state.
// Returns a snapshot with flattened config, provenance, and metadata.
// Secrets are automatically redacted using existing provenance data.
// The snapshot's Timestamp is captured at creation time.
func CreateSnapshot[T any](cfg *T, opts ...SnapshotOption) (*ConfigSnapshot, error) {
	if cfg == nil {
		return nil, ErrNilConfig
	}

	// Apply options
	snapCfg := &snapshotConfig{}
	for _, opt := range opts {
		opt(snapCfg)
	}

	// Capture timestamp at creation time
	timestamp := time.Now().UTC()

	// Get provenance data
	var provFields []FieldProvenance
	if prov, ok := GetProvenance(cfg); ok && prov != nil {
		provFields = prov.Fields
	}

	// Flatten config (handles secret redaction internally)
	flatConfig := flattenConfig(cfg)

	// Apply field exclusions
	flatConfig = applyExclusions(flatConfig, snapCfg.excludeFields)

	return &ConfigSnapshot{
		Version:    SnapshotVersion,
		Timestamp:  timestamp,
		Config:     flatConfig,
		Provenance: provFields,
	}, nil
}

// flattenConfig walks a configuration struct and returns a flat map of key paths to values.
// It handles nested structs, Optional[T] types, and time.Time.
// Secret fields are redacted using provenance information.
func flattenConfig[T any](cfg *T) map[string]any {
	if cfg == nil {
		return make(map[string]any)
	}

	// Get provenance for secret detection
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
		return make(map[string]any)
	}

	result := make(map[string]any)
	flattenStructFields(v, "", "", provenanceMap, result)
	return result
}

// flattenStructFields recursively walks struct fields and populates the result map.
// fieldPathPrefix is used for provenance lookup, keyPathPrefix is used for the output keys.
func flattenStructFields(v reflect.Value, fieldPathPrefix string, keyPathPrefix string, provenanceMap map[string]*FieldProvenance, result map[string]any) {
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

		// Get provenance info
		var prov *FieldProvenance
		if p, ok := provenanceMap[fieldPath]; ok {
			prov = p
		}

		// Determine key path for output
		var keyPath string
		if prov != nil && prov.KeyPath != "" {
			keyPath = prov.KeyPath
		} else if tagCfg.name != "" {
			keyPath = tagCfg.name
		} else {
			// Derive from field name (lowercase)
			keyPath = strings.ToLower(field.Name)
			if keyPathPrefix != "" {
				keyPath = keyPathPrefix + "." + keyPath
			}
		}

		// Handle nested structs recursively
		if fieldValue.Kind() == reflect.Struct && field.Type.String() != "time.Time" {
			// Check if this is an Optional type
			if isOptionalType(field.Type) {
				// Handle Optional[T] - extract the value if set
				setField := fieldValue.FieldByName("Set")
				valueField := fieldValue.FieldByName("Value")
				if setField.IsValid() && setField.Bool() && valueField.IsValid() {
					result[keyPath] = formatFlatValue(valueField, prov)
				}
				// If not set, omit from result (don't include unset optionals)
			} else {
				// Regular nested struct - recurse
				var nestedKeyPrefix string
				if tagCfg.prefix != "" {
					nestedKeyPrefix = tagCfg.prefix
				} else {
					nestedKeyPrefix = keyPath
				}
				flattenStructFields(fieldValue, fieldPath, nestedKeyPrefix, provenanceMap, result)
			}
			continue
		}

		// Format the value (with redaction if secret)
		result[keyPath] = formatFlatValue(fieldValue, prov)
	}
}

// applyExclusions filters out excluded field paths from the config map.
// Matching is case-insensitive.
func applyExclusions(config map[string]any, exclude []string) map[string]any {
	if len(exclude) == 0 {
		return config
	}

	// Build a set of lowercase exclusion paths for case-insensitive matching
	excludeSet := make(map[string]bool)
	for _, path := range exclude {
		excludeSet[strings.ToLower(path)] = true
	}

	result := make(map[string]any)
	for key, value := range config {
		if !excludeSet[strings.ToLower(key)] {
			result[key] = value
		}
	}
	return result
}

// ExpandPath expands template variables using current time.
// For consistency with snapshot metadata, prefer WriteSnapshot which
// uses the snapshot's internal timestamp for expansion.
func ExpandPath(template string) string {
	return ExpandPathWithTime(template, time.Now())
}

// ExpandPathWithTime expands template variables using the provided timestamp.
// Replaces all {{timestamp}} occurrences with the time formatted as 20060102-150405.
// Returns the path unchanged if no template variables are present.
func ExpandPathWithTime(template string, t time.Time) string {
	timestamp := t.UTC().Format("20060102-150405")
	return strings.ReplaceAll(template, "{{timestamp}}", timestamp)
}

// WriteSnapshot persists a snapshot to disk with atomic write semantics.
// Supports {{timestamp}} template variable in path - uses snapshot.Timestamp
// (not current time) to ensure filename matches internal metadata.
// Returns ErrSnapshotTooLarge if serialized size exceeds 100MB.
func WriteSnapshot(snapshot *ConfigSnapshot, pathTemplate string) error {
	if snapshot == nil {
		return ErrNilConfig
	}

	// Expand path template using snapshot's timestamp for consistency
	targetPath := ExpandPathWithTime(pathTemplate, snapshot.Timestamp)

	// Marshal snapshot to indented JSON
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}

	// Check size against MaxSnapshotSize
	if len(data) > MaxSnapshotSize {
		return ErrSnapshotTooLarge
	}

	// Create parent directories with 0700 permissions
	dir := filepath.Dir(targetPath)
	if dir != "" && dir != "." {
		if mkdirErr := os.MkdirAll(dir, 0700); mkdirErr != nil {
			return mkdirErr
		}
	}

	// Generate temp file name in same directory for atomic rename
	tempPath, err := generateTempFileName(targetPath)
	if err != nil {
		return err
	}

	// Ensure temp file is cleaned up on any error
	var tempFileCreated bool
	defer func() {
		if tempFileCreated {
			_ = os.Remove(tempPath)
		}
	}()

	// Write to temp file
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return err
	}
	tempFileCreated = true

	// Set file permissions explicitly (WriteFile should set them, but be explicit)
	if err := os.Chmod(tempPath, 0600); err != nil {
		return err
	}

	// Atomic rename temp file to target path
	if err := os.Rename(tempPath, targetPath); err != nil {
		return err
	}

	// Rename succeeded, don't clean up temp file (it's now the target)
	tempFileCreated = false

	return nil
}

// formatFlatValue formats a field value for the flattened config map.
// Secrets are redacted, other values are returned in their natural types.
func formatFlatValue(v reflect.Value, prov *FieldProvenance) any {
	// Check if this field is secret
	if prov != nil && prov.Secret {
		return "***redacted***"
	}

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
			if dur, ok := v.Interface().(time.Duration); ok {
				return dur.String()
			}
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
			if t, ok := v.Interface().(time.Time); ok {
				return t.Format(time.RFC3339)
			}
		}
		return v.Interface()
	default:
		return v.Interface()
	}
}

// generateTempFileName generates a unique temporary file name for atomic writes.
// The temp file is placed in the same directory as the target to ensure
// atomic rename works (same filesystem).
// Format: targetPath + ".tmp." + randomHex
func generateTempFileName(targetPath string) (string, error) {
	// Generate 8 random bytes (16 hex chars)
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	suffix := hex.EncodeToString(randomBytes)
	return targetPath + ".tmp." + suffix, nil
}
