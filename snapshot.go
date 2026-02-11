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

	provenanceMap := buildProvenanceMap(cfg)
	v, ok := getStructRootValue(cfg)
	if !ok {
		return make(map[string]any)
	}

	result := make(map[string]any)
	walkFlatFields(v, "", "", provenanceMap, false, func(w walkedField) {
		// Snapshots omit unset Optional[T] fields.
		if w.optional && !w.set {
			return
		}
		result[w.keyPath] = formatFlatValue(w.value, w.secret)
	})
	return result
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

// ReadSnapshot loads a snapshot from disk.
// Returns ErrUnsupportedVersion if snapshot version is not supported.
// Returns appropriate errors for missing file or invalid JSON.
func ReadSnapshot(path string) (*ConfigSnapshot, error) {
	// Read file contents
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Unmarshal JSON to ConfigSnapshot
	var snapshot ConfigSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, err
	}

	// Validate version field is present
	if snapshot.Version == "" {
		return nil, ErrUnsupportedVersion
	}

	// Check version against supportedVersions map
	if !supportedVersions[snapshot.Version] {
		return nil, ErrUnsupportedVersion
	}

	return &snapshot, nil
}

// formatFlatValue formats a field value for the flattened config map.
// Secrets are redacted, other values are returned in their natural types.
func formatFlatValue(v reflect.Value, secret bool) any {
	return formatStructuredValue(v, secret)
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
