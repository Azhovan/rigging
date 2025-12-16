package rigging

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFlattenConfig_NestedStructs(t *testing.T) {
	type Database struct {
		Host     string `conf:"name:host"`
		Port     int    `conf:"name:port"`
		Password string `conf:"name:password,secret"`
	}

	type Config struct {
		AppName  string   `conf:"name:app.name"`
		Database Database `conf:"prefix:database"`
	}

	cfg := &Config{
		AppName: "myapp",
		Database: Database{
			Host:     "db.example.com",
			Port:     5432,
			Password: "dbpass",
		},
	}

	prov := &Provenance{
		Fields: []FieldProvenance{
			{FieldPath: "AppName", KeyPath: "app.name", SourceName: "env", Secret: false},
			{FieldPath: "Database.Host", KeyPath: "database.host", SourceName: "file", Secret: false},
			{FieldPath: "Database.Port", KeyPath: "database.port", SourceName: "file", Secret: false},
			{FieldPath: "Database.Password", KeyPath: "database.password", SourceName: "env", Secret: true},
		},
	}
	storeProvenance(cfg, prov)
	defer deleteProvenance(cfg)

	result := flattenConfig(cfg)

	// Check nested fields are flattened with dot notation
	if result["app.name"] != "myapp" {
		t.Errorf("Expected app.name=myapp, got: %v", result["app.name"])
	}
	if result["database.host"] != "db.example.com" {
		t.Errorf("Expected database.host=db.example.com, got: %v", result["database.host"])
	}
	if result["database.port"] != int64(5432) {
		t.Errorf("Expected database.port=5432, got: %v (type: %T)", result["database.port"], result["database.port"])
	}
}

func TestFlattenConfig_OptionalHandling(t *testing.T) {
	type Config struct {
		Required string           `conf:"name:required"`
		Optional Optional[string] `conf:"name:optional"`
		NotSet   Optional[int]    `conf:"name:notset"`
	}

	cfg := &Config{
		Required: "value",
		Optional: Optional[string]{Value: "set", Set: true},
		NotSet:   Optional[int]{Value: 0, Set: false},
	}

	prov := &Provenance{
		Fields: []FieldProvenance{
			{FieldPath: "Required", KeyPath: "required", SourceName: "env", Secret: false},
			{FieldPath: "Optional", KeyPath: "optional", SourceName: "file", Secret: false},
		},
	}
	storeProvenance(cfg, prov)
	defer deleteProvenance(cfg)

	result := flattenConfig(cfg)

	// Check required field is present
	if result["required"] != "value" {
		t.Errorf("Expected required=value, got: %v", result["required"])
	}

	// Check set optional is present
	if result["optional"] != "set" {
		t.Errorf("Expected optional=set, got: %v", result["optional"])
	}

	// Check unset optional is NOT present (omitted)
	if _, exists := result["notset"]; exists {
		t.Errorf("Expected notset to be omitted, but it exists: %v", result["notset"])
	}
}

func TestFlattenConfig_SecretRedaction(t *testing.T) {
	type Config struct {
		Host     string `conf:"name:host"`
		Password string `conf:"name:password,secret"`
		APIKey   string `conf:"name:api_key,secret"`
	}

	cfg := &Config{
		Host:     "localhost",
		Password: "secret123",
		APIKey:   "key-abc-123",
	}

	prov := &Provenance{
		Fields: []FieldProvenance{
			{FieldPath: "Host", KeyPath: "host", SourceName: "env", Secret: false},
			{FieldPath: "Password", KeyPath: "password", SourceName: "env", Secret: true},
			{FieldPath: "APIKey", KeyPath: "api_key", SourceName: "env", Secret: true},
		},
	}
	storeProvenance(cfg, prov)
	defer deleteProvenance(cfg)

	result := flattenConfig(cfg)

	// Check non-secret field is not redacted
	if result["host"] != "localhost" {
		t.Errorf("Expected host=localhost, got: %v", result["host"])
	}

	// Check secret fields are redacted
	if result["password"] != "***redacted***" {
		t.Errorf("Expected password to be redacted, got: %v", result["password"])
	}
	if result["api_key"] != "***redacted***" {
		t.Errorf("Expected api_key to be redacted, got: %v", result["api_key"])
	}

	// Ensure actual secrets are not in result
	for key, val := range result {
		if strVal, ok := val.(string); ok {
			if strVal == "secret123" || strVal == "key-abc-123" {
				t.Errorf("Secret value found in result at key %s: %v", key, val)
			}
		}
	}
}

func TestFlattenConfig_EmptyConfig(t *testing.T) {
	type Config struct {
		Host string `conf:"name:host"`
		Port int    `conf:"name:port"`
	}

	cfg := &Config{} // Zero values

	result := flattenConfig(cfg)

	// Empty config should still produce a map with zero values
	if result["host"] != "" {
		t.Errorf("Expected host to be empty string, got: %v", result["host"])
	}
	if result["port"] != int64(0) {
		t.Errorf("Expected port to be 0, got: %v", result["port"])
	}
}

func TestFlattenConfig_NilConfig(t *testing.T) {
	var cfg *struct{}

	result := flattenConfig(cfg)

	// Nil config should return empty map
	if result == nil {
		t.Error("Expected empty map, got nil")
	}
	if len(result) != 0 {
		t.Errorf("Expected empty map, got: %v", result)
	}
}

func TestFlattenConfig_DifferentTypes(t *testing.T) {
	type Config struct {
		StringVal   string        `conf:"name:string_val"`
		IntVal      int           `conf:"name:int_val"`
		FloatVal    float64       `conf:"name:float_val"`
		BoolVal     bool          `conf:"name:bool_val"`
		DurationVal time.Duration `conf:"name:duration_val"`
		SliceVal    []string      `conf:"name:slice_val"`
		TimeVal     time.Time     `conf:"name:time_val"`
	}

	testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	cfg := &Config{
		StringVal:   "hello",
		IntVal:      42,
		FloatVal:    3.14,
		BoolVal:     true,
		DurationVal: 5 * time.Second,
		SliceVal:    []string{"a", "b", "c"},
		TimeVal:     testTime,
	}

	result := flattenConfig(cfg)

	// Check all types are correctly flattened
	if result["string_val"] != "hello" {
		t.Errorf("Expected string_val=hello, got: %v", result["string_val"])
	}
	if result["int_val"] != int64(42) {
		t.Errorf("Expected int_val=42, got: %v (type: %T)", result["int_val"], result["int_val"])
	}
	if result["float_val"] != 3.14 {
		t.Errorf("Expected float_val=3.14, got: %v", result["float_val"])
	}
	if result["bool_val"] != true {
		t.Errorf("Expected bool_val=true, got: %v", result["bool_val"])
	}
	if result["duration_val"] != "5s" {
		t.Errorf("Expected duration_val=5s, got: %v", result["duration_val"])
	}

	// Check slice
	sliceVal, ok := result["slice_val"].([]string)
	if !ok {
		t.Errorf("Expected slice_val to be []string, got: %T", result["slice_val"])
	} else if len(sliceVal) != 3 || sliceVal[0] != "a" || sliceVal[1] != "b" || sliceVal[2] != "c" {
		t.Errorf("Expected slice_val=[a,b,c], got: %v", sliceVal)
	}

	// Check time is formatted as RFC3339
	if result["time_val"] != "2024-01-15T10:30:00Z" {
		t.Errorf("Expected time_val=2024-01-15T10:30:00Z, got: %v", result["time_val"])
	}
}

func TestFlattenConfig_NoProvenance(t *testing.T) {
	type Config struct {
		Host string `conf:"name:host"`
		Port int    `conf:"name:port"`
	}

	cfg := &Config{
		Host: "localhost",
		Port: 8080,
	}

	// Don't store provenance - should still work

	result := flattenConfig(cfg)

	// Should still flatten correctly without provenance
	if result["host"] != "localhost" {
		t.Errorf("Expected host=localhost, got: %v", result["host"])
	}
	if result["port"] != int64(8080) {
		t.Errorf("Expected port=8080, got: %v", result["port"])
	}
}

func TestFlattenConfig_DeeplyNested(t *testing.T) {
	type Inner struct {
		Value string `conf:"name:value"`
	}

	type Middle struct {
		Inner Inner `conf:"prefix:inner"`
	}

	type Config struct {
		Middle Middle `conf:"prefix:middle"`
	}

	cfg := &Config{
		Middle: Middle{
			Inner: Inner{
				Value: "deep",
			},
		},
	}

	prov := &Provenance{
		Fields: []FieldProvenance{
			{FieldPath: "Middle.Inner.Value", KeyPath: "middle.inner.value", SourceName: "file", Secret: false},
		},
	}
	storeProvenance(cfg, prov)
	defer deleteProvenance(cfg)

	result := flattenConfig(cfg)

	// Check deeply nested field
	if result["middle.inner.value"] != "deep" {
		t.Errorf("Expected middle.inner.value=deep, got: %v", result["middle.inner.value"])
	}
}

func TestApplyExclusions_ExactPathMatching(t *testing.T) {
	config := map[string]any{
		"database.host":     "localhost",
		"database.port":     5432,
		"database.password": "secret",
		"api.key":           "apikey123",
	}

	exclude := []string{"database.password", "api.key"}

	result := applyExclusions(config, exclude)

	// Check excluded fields are removed
	if _, exists := result["database.password"]; exists {
		t.Error("Expected database.password to be excluded")
	}
	if _, exists := result["api.key"]; exists {
		t.Error("Expected api.key to be excluded")
	}

	// Check non-excluded fields are preserved
	if result["database.host"] != "localhost" {
		t.Errorf("Expected database.host=localhost, got: %v", result["database.host"])
	}
	if result["database.port"] != 5432 {
		t.Errorf("Expected database.port=5432, got: %v", result["database.port"])
	}
}

func TestApplyExclusions_CaseInsensitiveMatching(t *testing.T) {
	config := map[string]any{
		"Database.Host":     "localhost",
		"database.port":     5432,
		"DATABASE.PASSWORD": "secret",
	}

	// Exclude with different case
	exclude := []string{"database.host", "DATABASE.PORT", "Database.Password"}

	result := applyExclusions(config, exclude)

	// All should be excluded regardless of case
	if _, exists := result["Database.Host"]; exists {
		t.Error("Expected Database.Host to be excluded (case-insensitive)")
	}
	if _, exists := result["database.port"]; exists {
		t.Error("Expected database.port to be excluded (case-insensitive)")
	}
	if _, exists := result["DATABASE.PASSWORD"]; exists {
		t.Error("Expected DATABASE.PASSWORD to be excluded (case-insensitive)")
	}

	// Result should be empty
	if len(result) != 0 {
		t.Errorf("Expected empty result, got: %v", result)
	}
}

func TestApplyExclusions_EmptyExclusionList(t *testing.T) {
	config := map[string]any{
		"database.host": "localhost",
		"database.port": 5432,
	}

	result := applyExclusions(config, []string{})

	// All fields should be preserved
	if len(result) != len(config) {
		t.Errorf("Expected %d fields, got %d", len(config), len(result))
	}
	if result["database.host"] != "localhost" {
		t.Errorf("Expected database.host=localhost, got: %v", result["database.host"])
	}
	if result["database.port"] != 5432 {
		t.Errorf("Expected database.port=5432, got: %v", result["database.port"])
	}
}

func TestApplyExclusions_NilExclusionList(t *testing.T) {
	config := map[string]any{
		"database.host": "localhost",
		"database.port": 5432,
	}

	result := applyExclusions(config, nil)

	// All fields should be preserved
	if len(result) != len(config) {
		t.Errorf("Expected %d fields, got %d", len(config), len(result))
	}
}

func TestApplyExclusions_NonExistentPaths(t *testing.T) {
	config := map[string]any{
		"database.host": "localhost",
		"database.port": 5432,
	}

	// Exclude paths that don't exist
	exclude := []string{"nonexistent.field", "another.missing"}

	result := applyExclusions(config, exclude)

	// All original fields should be preserved
	if len(result) != len(config) {
		t.Errorf("Expected %d fields, got %d", len(config), len(result))
	}
	if result["database.host"] != "localhost" {
		t.Errorf("Expected database.host=localhost, got: %v", result["database.host"])
	}
	if result["database.port"] != 5432 {
		t.Errorf("Expected database.port=5432, got: %v", result["database.port"])
	}
}

func TestApplyExclusions_EmptyConfig(t *testing.T) {
	config := map[string]any{}

	exclude := []string{"database.password"}

	result := applyExclusions(config, exclude)

	// Result should be empty
	if len(result) != 0 {
		t.Errorf("Expected empty result, got: %v", result)
	}
}

func TestApplyExclusions_PreservesOriginalConfig(t *testing.T) {
	config := map[string]any{
		"database.host":     "localhost",
		"database.password": "secret",
	}

	exclude := []string{"database.password"}

	_ = applyExclusions(config, exclude)

	// Original config should not be modified
	if _, exists := config["database.password"]; !exists {
		t.Error("Original config should not be modified")
	}
	if config["database.password"] != "secret" {
		t.Errorf("Original config value should be preserved, got: %v", config["database.password"])
	}
}

// CreateSnapshot unit tests

func TestCreateSnapshot_NilConfig(t *testing.T) {
	var cfg *struct{}

	snapshot, err := CreateSnapshot(cfg)

	if err != ErrNilConfig {
		t.Errorf("Expected ErrNilConfig, got: %v", err)
	}
	if snapshot != nil {
		t.Errorf("Expected nil snapshot, got: %v", snapshot)
	}
}

func TestCreateSnapshot_WithoutProvenance(t *testing.T) {
	type Config struct {
		Host string `conf:"name:host"`
		Port int    `conf:"name:port"`
	}

	cfg := &Config{
		Host: "localhost",
		Port: 8080,
	}

	// Don't store provenance - should still work

	snapshot, err := CreateSnapshot(cfg)

	if err != nil {
		t.Fatalf("CreateSnapshot failed: %v", err)
	}
	if snapshot == nil {
		t.Fatal("Expected snapshot, got nil")
	}

	// Check basic fields
	if snapshot.Version != SnapshotVersion {
		t.Errorf("Expected version=%s, got: %s", SnapshotVersion, snapshot.Version)
	}
	if snapshot.Timestamp.IsZero() {
		t.Error("Expected non-zero timestamp")
	}
	if snapshot.Config["host"] != "localhost" {
		t.Errorf("Expected host=localhost, got: %v", snapshot.Config["host"])
	}
}

func TestCreateSnapshot_EmptyConfig(t *testing.T) {
	type Config struct{}

	cfg := &Config{}

	snapshot, err := CreateSnapshot(cfg)

	if err != nil {
		t.Fatalf("CreateSnapshot failed: %v", err)
	}
	if snapshot == nil {
		t.Fatal("Expected snapshot, got nil")
	}

	// Empty config should produce valid snapshot with empty config map
	if snapshot.Version != SnapshotVersion {
		t.Errorf("Expected version=%s, got: %s", SnapshotVersion, snapshot.Version)
	}
	if snapshot.Config == nil {
		t.Error("Expected non-nil config map")
	}
}

func TestCreateSnapshot_VersionAndTimestamp(t *testing.T) {
	type Config struct {
		Host string `conf:"name:host"`
	}

	cfg := &Config{Host: "localhost"}

	before := time.Now().UTC()
	snapshot, err := CreateSnapshot(cfg)
	after := time.Now().UTC()

	if err != nil {
		t.Fatalf("CreateSnapshot failed: %v", err)
	}

	// Check version
	if snapshot.Version != "1.0" {
		t.Errorf("Expected version=1.0, got: %s", snapshot.Version)
	}

	// Check timestamp is within expected range
	if snapshot.Timestamp.Before(before) || snapshot.Timestamp.After(after) {
		t.Errorf("Timestamp %v not in expected range [%v, %v]", snapshot.Timestamp, before, after)
	}

	// Check timestamp is UTC
	if snapshot.Timestamp.Location() != time.UTC {
		t.Errorf("Expected UTC timestamp, got: %v", snapshot.Timestamp.Location())
	}
}

func TestCreateSnapshot_WithProvenance(t *testing.T) {
	type Config struct {
		Host     string `conf:"name:host"`
		Password string `conf:"name:password,secret"`
	}

	cfg := &Config{
		Host:     "localhost",
		Password: "secret123",
	}

	prov := &Provenance{
		Fields: []FieldProvenance{
			{FieldPath: "Host", KeyPath: "host", SourceName: "env:HOST", Secret: false},
			{FieldPath: "Password", KeyPath: "password", SourceName: "env:PASSWORD", Secret: true},
		},
	}
	storeProvenance(cfg, prov)
	defer deleteProvenance(cfg)

	snapshot, err := CreateSnapshot(cfg)

	if err != nil {
		t.Fatalf("CreateSnapshot failed: %v", err)
	}

	// Check provenance is included
	if len(snapshot.Provenance) != 2 {
		t.Errorf("Expected 2 provenance entries, got: %d", len(snapshot.Provenance))
	}

	// Check secrets are redacted in config
	if snapshot.Config["password"] != "***redacted***" {
		t.Errorf("Expected password to be redacted, got: %v", snapshot.Config["password"])
	}

	// Check non-secret is not redacted
	if snapshot.Config["host"] != "localhost" {
		t.Errorf("Expected host=localhost, got: %v", snapshot.Config["host"])
	}
}

func TestCreateSnapshot_WithExclusions(t *testing.T) {
	type Config struct {
		Host     string `conf:"name:host"`
		Port     int    `conf:"name:port"`
		Password string `conf:"name:password"`
	}

	cfg := &Config{
		Host:     "localhost",
		Port:     8080,
		Password: "secret",
	}

	snapshot, err := CreateSnapshot(cfg, WithExcludeFields("password", "port"))

	if err != nil {
		t.Fatalf("CreateSnapshot failed: %v", err)
	}

	// Check excluded fields are not present
	if _, exists := snapshot.Config["password"]; exists {
		t.Error("Expected password to be excluded")
	}
	if _, exists := snapshot.Config["port"]; exists {
		t.Error("Expected port to be excluded")
	}

	// Check non-excluded field is present
	if snapshot.Config["host"] != "localhost" {
		t.Errorf("Expected host=localhost, got: %v", snapshot.Config["host"])
	}
}

// Property-based tests for CreateSnapshot

func TestCreateSnapshotProperties_SecretRedaction(t *testing.T) {
	// **Feature: snapshot-core, Property 2: Secret Redaction Completeness**
	// **Validates: Requirements 1.5**
	// For any configuration with fields marked as secret, the snapshot config
	// SHALL contain "***redacted***" for all secret field paths.

	type Config struct {
		Host     string `conf:"name:host"`
		Password string `conf:"name:password,secret"`
		APIKey   string `conf:"name:api_key,secret"`
		Token    string `conf:"name:token,secret"`
	}

	// Test with various secret values
	testCases := []struct {
		password string
		apiKey   string
		token    string
	}{
		{"secret1", "key1", "tok1"},
		{"", "", ""},
		{"very-long-secret-value-that-should-still-be-redacted", "another-key", "another-token"},
		{"special!@#$%^&*()", "key with spaces", "token\nwith\nnewlines"},
	}

	for _, tc := range testCases {
		cfg := &Config{
			Host:     "localhost",
			Password: tc.password,
			APIKey:   tc.apiKey,
			Token:    tc.token,
		}

		prov := &Provenance{
			Fields: []FieldProvenance{
				{FieldPath: "Host", KeyPath: "host", SourceName: "env", Secret: false},
				{FieldPath: "Password", KeyPath: "password", SourceName: "env", Secret: true},
				{FieldPath: "APIKey", KeyPath: "api_key", SourceName: "env", Secret: true},
				{FieldPath: "Token", KeyPath: "token", SourceName: "env", Secret: true},
			},
		}
		storeProvenance(cfg, prov)

		snapshot, err := CreateSnapshot(cfg)
		deleteProvenance(cfg)

		if err != nil {
			t.Fatalf("CreateSnapshot failed: %v", err)
		}

		// Property: ALL secret fields must be redacted
		secretFields := []string{"password", "api_key", "token"}
		for _, field := range secretFields {
			if snapshot.Config[field] != "***redacted***" {
				t.Errorf("Secret field %s not redacted, got: %v", field, snapshot.Config[field])
			}
		}

		// Property: Non-secret fields must NOT be redacted
		if snapshot.Config["host"] != "localhost" {
			t.Errorf("Non-secret field host should not be redacted, got: %v", snapshot.Config["host"])
		}
	}
}

func TestCreateSnapshotProperties_FieldExclusion(t *testing.T) {
	// **Feature: snapshot-core, Property 3: Field Exclusion Correctness**
	// **Validates: Requirements 4.1**
	// For any configuration and exclusion list, the snapshot config
	// SHALL NOT contain any field paths that match the exclusion list.

	type Config struct {
		Host     string `conf:"name:host"`
		Port     int    `conf:"name:port"`
		Password string `conf:"name:password"`
		APIKey   string `conf:"name:api_key"`
		Debug    bool   `conf:"name:debug"`
	}

	cfg := &Config{
		Host:     "localhost",
		Port:     8080,
		Password: "secret",
		APIKey:   "key123",
		Debug:    true,
	}

	// Test various exclusion combinations
	exclusionTests := []struct {
		exclude  []string
		expected map[string]bool // fields that should be present
	}{
		{
			exclude:  []string{},
			expected: map[string]bool{"host": true, "port": true, "password": true, "api_key": true, "debug": true},
		},
		{
			exclude:  []string{"password"},
			expected: map[string]bool{"host": true, "port": true, "api_key": true, "debug": true},
		},
		{
			exclude:  []string{"password", "api_key"},
			expected: map[string]bool{"host": true, "port": true, "debug": true},
		},
		{
			exclude:  []string{"host", "port", "password", "api_key", "debug"},
			expected: map[string]bool{},
		},
		{
			exclude:  []string{"PASSWORD", "API_KEY"}, // case-insensitive
			expected: map[string]bool{"host": true, "port": true, "debug": true},
		},
	}

	for _, tc := range exclusionTests {
		snapshot, err := CreateSnapshot(cfg, WithExcludeFields(tc.exclude...))
		if err != nil {
			t.Fatalf("CreateSnapshot failed: %v", err)
		}

		// Property: Excluded fields must NOT appear
		for _, excluded := range tc.exclude {
			normalizedKey := strings.ToLower(excluded)
			if _, exists := snapshot.Config[normalizedKey]; exists {
				t.Errorf("Excluded field %s should not appear in snapshot", excluded)
			}
		}

		// Property: Non-excluded fields must appear
		for field := range tc.expected {
			if _, exists := snapshot.Config[field]; !exists {
				t.Errorf("Non-excluded field %s should appear in snapshot", field)
			}
		}
	}
}

func TestCreateSnapshotProperties_ProvenancePreservation(t *testing.T) {
	// **Feature: snapshot-core, Property 6: Provenance Preservation**
	// **Validates: Requirements 1.2**
	// For any configuration with provenance data, the snapshot's Provenance field
	// SHALL contain entries matching the provenance returned by GetProvenance.

	type Config struct {
		Host     string `conf:"name:host"`
		Port     int    `conf:"name:port"`
		Password string `conf:"name:password,secret"`
	}

	cfg := &Config{
		Host:     "localhost",
		Port:     8080,
		Password: "secret",
	}

	originalProv := &Provenance{
		Fields: []FieldProvenance{
			{FieldPath: "Host", KeyPath: "host", SourceName: "env:HOST", Secret: false},
			{FieldPath: "Port", KeyPath: "port", SourceName: "file:config.yaml", Secret: false},
			{FieldPath: "Password", KeyPath: "password", SourceName: "env:PASSWORD", Secret: true},
		},
	}
	storeProvenance(cfg, originalProv)
	defer deleteProvenance(cfg)

	snapshot, err := CreateSnapshot(cfg)
	if err != nil {
		t.Fatalf("CreateSnapshot failed: %v", err)
	}

	// Property: Provenance count must match
	if len(snapshot.Provenance) != len(originalProv.Fields) {
		t.Errorf("Expected %d provenance entries, got %d", len(originalProv.Fields), len(snapshot.Provenance))
	}

	// Property: Each provenance entry must be preserved
	provMap := make(map[string]FieldProvenance)
	for _, p := range snapshot.Provenance {
		provMap[p.FieldPath] = p
	}

	for _, orig := range originalProv.Fields {
		snapshotProv, exists := provMap[orig.FieldPath]
		if !exists {
			t.Errorf("Provenance for %s not found in snapshot", orig.FieldPath)
			continue
		}

		if snapshotProv.KeyPath != orig.KeyPath {
			t.Errorf("KeyPath mismatch for %s: expected %s, got %s", orig.FieldPath, orig.KeyPath, snapshotProv.KeyPath)
		}
		if snapshotProv.SourceName != orig.SourceName {
			t.Errorf("SourceName mismatch for %s: expected %s, got %s", orig.FieldPath, orig.SourceName, snapshotProv.SourceName)
		}
		if snapshotProv.Secret != orig.Secret {
			t.Errorf("Secret mismatch for %s: expected %v, got %v", orig.FieldPath, orig.Secret, snapshotProv.Secret)
		}
	}
}

// ExpandPath and ExpandPathWithTime unit tests

func TestExpandPathWithTime_SingleTimestamp(t *testing.T) {
	// Test template with single {{timestamp}}
	testTime := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)
	template := "config-{{timestamp}}.json"

	result := ExpandPathWithTime(template, testTime)

	expected := "config-20240115-103045.json"
	if result != expected {
		t.Errorf("Expected %s, got: %s", expected, result)
	}
}

func TestExpandPathWithTime_MultipleTimestamps(t *testing.T) {
	// Test template with multiple {{timestamp}} occurrences
	testTime := time.Date(2024, 6, 20, 14, 5, 9, 0, time.UTC)
	template := "{{timestamp}}/config-{{timestamp}}.json"

	result := ExpandPathWithTime(template, testTime)

	expected := "20240620-140509/config-20240620-140509.json"
	if result != expected {
		t.Errorf("Expected %s, got: %s", expected, result)
	}
}

func TestExpandPathWithTime_NoVariables(t *testing.T) {
	// Test template with no variables (unchanged)
	testTime := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)
	template := "config/snapshot.json"

	result := ExpandPathWithTime(template, testTime)

	if result != template {
		t.Errorf("Expected unchanged path %s, got: %s", template, result)
	}
}

func TestExpandPathWithTime_EmptyPath(t *testing.T) {
	testTime := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)
	template := ""

	result := ExpandPathWithTime(template, testTime)

	if result != "" {
		t.Errorf("Expected empty string, got: %s", result)
	}
}

func TestExpandPathWithTime_TimestampOnly(t *testing.T) {
	testTime := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
	template := "{{timestamp}}"

	result := ExpandPathWithTime(template, testTime)

	expected := "20241231-235959"
	if result != expected {
		t.Errorf("Expected %s, got: %s", expected, result)
	}
}

func TestExpandPathWithTime_NonUTCTime(t *testing.T) {
	// Test that non-UTC times are converted to UTC for formatting
	loc, _ := time.LoadLocation("America/New_York")
	testTime := time.Date(2024, 1, 15, 10, 30, 45, 0, loc) // EST
	template := "config-{{timestamp}}.json"

	result := ExpandPathWithTime(template, testTime)

	// 10:30:45 EST = 15:30:45 UTC
	expected := "config-20240115-153045.json"
	if result != expected {
		t.Errorf("Expected %s, got: %s", expected, result)
	}
}

func TestExpandPath_UsesCurrentTime(t *testing.T) {
	// Test ExpandPath vs ExpandPathWithTime consistency
	template := "config-{{timestamp}}.json"

	before := time.Now().UTC()
	result := ExpandPath(template)
	after := time.Now().UTC()

	// The result should contain a timestamp between before and after
	// We can't check exact value, but we can verify format
	if !strings.HasPrefix(result, "config-") || !strings.HasSuffix(result, ".json") {
		t.Errorf("Unexpected format: %s", result)
	}

	// Extract timestamp from result
	timestampStr := strings.TrimPrefix(result, "config-")
	timestampStr = strings.TrimSuffix(timestampStr, ".json")

	// Verify it's a valid timestamp format (YYYYMMDD-HHMMSS)
	if len(timestampStr) != 15 { // 8 + 1 + 6
		t.Errorf("Expected timestamp length 15, got %d: %s", len(timestampStr), timestampStr)
	}

	// Parse the timestamp to verify it's in the expected range
	parsedTime, err := time.Parse("20060102-150405", timestampStr)
	if err != nil {
		t.Errorf("Failed to parse timestamp %s: %v", timestampStr, err)
	}

	// Allow 1 second tolerance for test execution time
	if parsedTime.Before(before.Add(-time.Second)) || parsedTime.After(after.Add(time.Second)) {
		t.Errorf("Timestamp %v not in expected range [%v, %v]", parsedTime, before, after)
	}
}

func TestExpandPath_EquivalentToExpandPathWithTime(t *testing.T) {
	// Verify that ExpandPath produces the same result as ExpandPathWithTime
	// when called with the same time
	template := "snapshots/{{timestamp}}/config.json"

	// Get current time and call both functions
	now := time.Now()
	expectedResult := ExpandPathWithTime(template, now)

	// ExpandPath uses time.Now() internally, so we can't get exact match
	// but we can verify the format is consistent
	result := ExpandPath(template)

	// Both should have the same structure
	if !strings.HasPrefix(result, "snapshots/") || !strings.HasSuffix(result, "/config.json") {
		t.Errorf("Unexpected format from ExpandPath: %s", result)
	}
	if !strings.HasPrefix(expectedResult, "snapshots/") || !strings.HasSuffix(expectedResult, "/config.json") {
		t.Errorf("Unexpected format from ExpandPathWithTime: %s", expectedResult)
	}
}

// Property-based tests for ExpandPath

func TestExpandPathProperties_TemplateExpansionConsistency(t *testing.T) {
	// **Feature: snapshot-core, Property 4: Template Expansion Consistency**
	// **Validates: Requirements 3.1, 3.2, 3.3**
	// For any path template containing {{timestamp}}, expanding with a given time
	// SHALL replace all occurrences with the same formatted timestamp string,
	// and paths without templates SHALL remain unchanged.

	// Test with various times across different edge cases
	testTimes := []time.Time{
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),      // New Year midnight
		time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC), // End of year
		time.Date(2024, 6, 15, 12, 30, 45, 0, time.UTC),  // Mid-year
		time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),      // Y2K
		time.Date(2099, 12, 31, 23, 59, 59, 0, time.UTC), // Far future
		time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),      // Unix epoch
		time.Date(2024, 2, 29, 12, 0, 0, 0, time.UTC),    // Leap year
	}

	// Test templates with various patterns
	templates := []string{
		"config-{{timestamp}}.json",
		"{{timestamp}}/snapshot.json",
		"snapshots/{{timestamp}}/config-{{timestamp}}.json",
		"{{timestamp}}",
		"/var/log/app/{{timestamp}}/{{timestamp}}/data.json",
		"no-template-here.json",
		"",
		"prefix-{{timestamp}}-suffix-{{timestamp}}-end",
	}

	for _, testTime := range testTimes {
		expectedTimestamp := testTime.UTC().Format("20060102-150405")

		for _, template := range templates {
			result := ExpandPathWithTime(template, testTime)

			// Property 1: Same time produces same output (deterministic)
			result2 := ExpandPathWithTime(template, testTime)
			if result != result2 {
				t.Errorf("Non-deterministic expansion for template %q with time %v: got %q and %q",
					template, testTime, result, result2)
			}

			// Property 2: All {{timestamp}} occurrences are replaced with the same value
			if strings.Contains(template, "{{timestamp}}") {
				// Result should not contain any {{timestamp}}
				if strings.Contains(result, "{{timestamp}}") {
					t.Errorf("Template %q not fully expanded: %q", template, result)
				}

				// Count occurrences in template and verify they're all replaced with same timestamp
				templateCount := strings.Count(template, "{{timestamp}}")
				resultCount := strings.Count(result, expectedTimestamp)
				if templateCount != resultCount {
					t.Errorf("Template %q has %d {{timestamp}} but result has %d occurrences of %s: %q",
						template, templateCount, resultCount, expectedTimestamp, result)
				}
			}

			// Property 3: Paths without templates remain unchanged
			if !strings.Contains(template, "{{timestamp}}") {
				if result != template {
					t.Errorf("Template without variables should be unchanged: %q -> %q", template, result)
				}
			}

			// Property 4: The timestamp format is always YYYYMMDD-HHMMSS (15 chars)
			if strings.Contains(template, "{{timestamp}}") {
				// Verify the timestamp in result matches expected format
				if !strings.Contains(result, expectedTimestamp) {
					t.Errorf("Result %q does not contain expected timestamp %s", result, expectedTimestamp)
				}
			}
		}
	}
}

func TestExpandPathProperties_TimezoneNormalization(t *testing.T) {
	// **Feature: snapshot-core, Property 4: Template Expansion Consistency**
	// **Validates: Requirements 3.1**
	// For any time in any timezone, the expansion SHALL use UTC.

	// Same instant in different timezones should produce same result
	utcTime := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)

	// Load various timezones
	timezones := []string{
		"America/New_York",
		"Europe/London",
		"Asia/Tokyo",
		"Australia/Sydney",
		"Pacific/Auckland",
	}

	template := "config-{{timestamp}}.json"
	expectedResult := ExpandPathWithTime(template, utcTime)

	for _, tzName := range timezones {
		loc, err := time.LoadLocation(tzName)
		if err != nil {
			t.Logf("Skipping timezone %s: %v", tzName, err)
			continue
		}

		// Convert UTC time to local timezone
		localTime := utcTime.In(loc)

		result := ExpandPathWithTime(template, localTime)

		// Property: Same instant in any timezone produces same result
		if result != expectedResult {
			t.Errorf("Timezone %s produced different result: expected %q, got %q",
				tzName, expectedResult, result)
		}
	}
}

// generateTempFileName unit tests

func TestGenerateTempFileName_UniqueNames(t *testing.T) {
	targetPath := "/path/to/config.json"

	// Generate multiple temp file names and verify they're unique
	names := make(map[string]bool)
	for i := 0; i < 100; i++ {
		name, err := generateTempFileName(targetPath)
		if err != nil {
			t.Fatalf("generateTempFileName failed: %v", err)
		}

		if names[name] {
			t.Errorf("Duplicate temp file name generated: %s", name)
		}
		names[name] = true
	}
}

func TestGenerateTempFileName_SameDirectory(t *testing.T) {
	testCases := []struct {
		targetPath  string
		expectedDir string
	}{
		{"/path/to/config.json", "/path/to/config.json.tmp."},
		{"config.json", "config.json.tmp."},
		{"/var/log/app/snapshot.json", "/var/log/app/snapshot.json.tmp."},
		{"./relative/path/file.json", "./relative/path/file.json.tmp."},
	}

	for _, tc := range testCases {
		name, err := generateTempFileName(tc.targetPath)
		if err != nil {
			t.Fatalf("generateTempFileName failed for %s: %v", tc.targetPath, err)
		}

		// Verify temp file starts with target path + ".tmp."
		if !strings.HasPrefix(name, tc.expectedDir) {
			t.Errorf("Expected temp file to start with %s, got: %s", tc.expectedDir, name)
		}

		// Verify there's a random suffix after ".tmp."
		suffix := strings.TrimPrefix(name, tc.expectedDir)
		if len(suffix) != 16 { // 8 bytes = 16 hex chars
			t.Errorf("Expected 16-char hex suffix, got %d chars: %s", len(suffix), suffix)
		}

		// Verify suffix is valid hex
		for _, c := range suffix {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Errorf("Invalid hex character in suffix: %c", c)
			}
		}
	}
}

func TestGenerateTempFileName_Format(t *testing.T) {
	targetPath := "snapshot.json"

	name, err := generateTempFileName(targetPath)
	if err != nil {
		t.Fatalf("generateTempFileName failed: %v", err)
	}

	// Format should be: targetPath + ".tmp." + 16hexchars
	expectedPrefix := "snapshot.json.tmp."
	if !strings.HasPrefix(name, expectedPrefix) {
		t.Errorf("Expected prefix %s, got: %s", expectedPrefix, name)
	}

	// Total length: len(targetPath) + 5 (".tmp.") + 16 (hex)
	expectedLen := len(targetPath) + 5 + 16
	if len(name) != expectedLen {
		t.Errorf("Expected length %d, got %d: %s", expectedLen, len(name), name)
	}
}

// WriteSnapshot unit tests

func TestWriteSnapshot_WritesValidSnapshot(t *testing.T) {
	// Create a temp directory for test files
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "snapshot.json")

	snapshot := &ConfigSnapshot{
		Version:   SnapshotVersion,
		Timestamp: time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC),
		Config: map[string]any{
			"host": "localhost",
			"port": 8080,
		},
		Provenance: []FieldProvenance{
			{FieldPath: "Host", KeyPath: "host", SourceName: "env", Secret: false},
		},
	}

	err := WriteSnapshot(snapshot, targetPath)
	if err != nil {
		t.Fatalf("WriteSnapshot failed: %v", err)
	}

	// Verify file exists
	if _, statErr := os.Stat(targetPath); os.IsNotExist(statErr) {
		t.Fatal("Snapshot file was not created")
	}

	// Verify content is valid JSON
	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("Failed to read snapshot file: %v", err)
	}

	var readSnapshot ConfigSnapshot
	if err := json.Unmarshal(data, &readSnapshot); err != nil {
		t.Fatalf("Failed to unmarshal snapshot: %v", err)
	}

	// Verify content matches
	if readSnapshot.Version != snapshot.Version {
		t.Errorf("Version mismatch: expected %s, got %s", snapshot.Version, readSnapshot.Version)
	}
	if !readSnapshot.Timestamp.Equal(snapshot.Timestamp) {
		t.Errorf("Timestamp mismatch: expected %v, got %v", snapshot.Timestamp, readSnapshot.Timestamp)
	}
}

func TestWriteSnapshot_CreatesParentDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a path with nested directories that don't exist
	targetPath := filepath.Join(tmpDir, "nested", "dirs", "snapshot.json")

	snapshot := &ConfigSnapshot{
		Version:   SnapshotVersion,
		Timestamp: time.Now().UTC(),
		Config:    map[string]any{"key": "value"},
	}

	err := WriteSnapshot(snapshot, targetPath)
	if err != nil {
		t.Fatalf("WriteSnapshot failed: %v", err)
	}

	// Verify file exists
	if _, statErr := os.Stat(targetPath); os.IsNotExist(statErr) {
		t.Fatal("Snapshot file was not created")
	}

	// Verify parent directories were created
	parentDir := filepath.Dir(targetPath)
	info, err := os.Stat(parentDir)
	if err != nil {
		t.Fatalf("Parent directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("Parent path is not a directory")
	}
}

func TestWriteSnapshot_SetsCorrectFilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "snapshot.json")

	snapshot := &ConfigSnapshot{
		Version:   SnapshotVersion,
		Timestamp: time.Now().UTC(),
		Config:    map[string]any{"key": "value"},
	}

	err := WriteSnapshot(snapshot, targetPath)
	if err != nil {
		t.Fatalf("WriteSnapshot failed: %v", err)
	}

	// Check file permissions
	info, err := os.Stat(targetPath)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	// File should have 0600 permissions (owner read/write only)
	expectedPerm := os.FileMode(0600)
	actualPerm := info.Mode().Perm()
	if actualPerm != expectedPerm {
		t.Errorf("Expected permissions %o, got %o", expectedPerm, actualPerm)
	}
}

func TestWriteSnapshot_ExpandsTimestampTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "config-{{timestamp}}.json")

	snapshotTime := time.Date(2024, 6, 15, 14, 30, 45, 0, time.UTC)
	snapshot := &ConfigSnapshot{
		Version:   SnapshotVersion,
		Timestamp: snapshotTime,
		Config:    map[string]any{"key": "value"},
	}

	err := WriteSnapshot(snapshot, templatePath)
	if err != nil {
		t.Fatalf("WriteSnapshot failed: %v", err)
	}

	// Verify file was created with expanded timestamp
	expectedPath := filepath.Join(tmpDir, "config-20240615-143045.json")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected file at %s was not created", expectedPath)
	}

	// Verify template path file does NOT exist (it should be expanded)
	if _, err := os.Stat(templatePath); !os.IsNotExist(err) {
		t.Error("Template path should not exist as a file")
	}
}

func TestWriteSnapshot_ReturnsErrSnapshotTooLarge(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "snapshot.json")

	// Create a snapshot with a very large config that exceeds MaxSnapshotSize
	// We'll create a config with many large string values
	largeConfig := make(map[string]any)
	// Each entry is about 1MB, we need > 100MB
	largeValue := strings.Repeat("x", 1024*1024) // 1MB string
	for i := 0; i < 110; i++ {
		largeConfig[fmt.Sprintf("key%d", i)] = largeValue
	}

	snapshot := &ConfigSnapshot{
		Version:   SnapshotVersion,
		Timestamp: time.Now().UTC(),
		Config:    largeConfig,
	}

	err := WriteSnapshot(snapshot, targetPath)
	if err != ErrSnapshotTooLarge {
		t.Errorf("Expected ErrSnapshotTooLarge, got: %v", err)
	}

	// Verify no file was created
	if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
		t.Error("File should not be created for oversized snapshot")
	}
}

func TestWriteSnapshot_NilSnapshot(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "snapshot.json")

	err := WriteSnapshot(nil, targetPath)
	if err != ErrNilConfig {
		t.Errorf("Expected ErrNilConfig, got: %v", err)
	}
}

func TestWriteSnapshot_TempFileCleanupOnError(t *testing.T) {
	// This test verifies that temp files are cleaned up when an error occurs
	// We'll test by trying to write to a read-only directory (after creating it)
	tmpDir := t.TempDir()

	// Create a subdirectory and make it read-only
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(readOnlyDir, 0700); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Write a file first, then make directory read-only
	targetPath := filepath.Join(readOnlyDir, "snapshot.json")
	snapshot := &ConfigSnapshot{
		Version:   SnapshotVersion,
		Timestamp: time.Now().UTC(),
		Config:    map[string]any{"key": "value"},
	}

	// First write should succeed
	if err := WriteSnapshot(snapshot, targetPath); err != nil {
		t.Fatalf("First write failed: %v", err)
	}

	// Now make the directory read-only
	if err := os.Chmod(readOnlyDir, 0500); err != nil {
		t.Fatalf("Failed to chmod directory: %v", err)
	}
	defer os.Chmod(readOnlyDir, 0700) // Restore permissions for cleanup

	// Try to write again - should fail because we can't create temp file
	err := WriteSnapshot(snapshot, targetPath)
	if err == nil {
		t.Error("Expected error when writing to read-only directory")
	}

	// Verify no temp files are left behind
	entries, err := os.ReadDir(readOnlyDir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".tmp.") {
			t.Errorf("Temp file not cleaned up: %s", entry.Name())
		}
	}
}

func TestWriteSnapshot_OverwritesExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "snapshot.json")

	// Write first snapshot
	snapshot1 := &ConfigSnapshot{
		Version:   SnapshotVersion,
		Timestamp: time.Now().UTC(),
		Config:    map[string]any{"version": "1"},
	}
	if err := WriteSnapshot(snapshot1, targetPath); err != nil {
		t.Fatalf("First write failed: %v", err)
	}

	// Write second snapshot to same path
	snapshot2 := &ConfigSnapshot{
		Version:   SnapshotVersion,
		Timestamp: time.Now().UTC(),
		Config:    map[string]any{"version": "2"},
	}
	if err := WriteSnapshot(snapshot2, targetPath); err != nil {
		t.Fatalf("Second write failed: %v", err)
	}

	// Verify second snapshot overwrote first
	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	var readSnapshot ConfigSnapshot
	if err := json.Unmarshal(data, &readSnapshot); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if readSnapshot.Config["version"] != "2" {
		t.Errorf("Expected version=2, got: %v", readSnapshot.Config["version"])
	}
}

// Property-based tests for WriteSnapshot

func TestWriteSnapshotProperties_TimestampFilenameConsistency(t *testing.T) {
	// **Feature: snapshot-core, Property 5: Timestamp Filename Consistency**
	// **Validates: Requirements 7.2, 7.3**
	// For any snapshot written with a {{timestamp}} template path, the timestamp
	// in the filename SHALL match the snapshot's internal Timestamp field
	// (formatted as 20060102-150405).

	tmpDir := t.TempDir()

	// Test with various timestamps across different edge cases
	testTimes := []time.Time{
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),      // New Year midnight
		time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC), // End of year
		time.Date(2024, 6, 15, 12, 30, 45, 0, time.UTC),  // Mid-year
		time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),      // Y2K
		time.Date(2024, 2, 29, 12, 0, 0, 0, time.UTC),    // Leap year
		time.Date(1999, 12, 31, 23, 59, 59, 0, time.UTC), // Pre-Y2K
		time.Date(2030, 7, 4, 15, 30, 0, 0, time.UTC),    // Future date
	}

	for i, snapshotTime := range testTimes {
		// Create unique subdirectory for each test
		testDir := filepath.Join(tmpDir, fmt.Sprintf("test%d", i))
		templatePath := filepath.Join(testDir, "config-{{timestamp}}.json")

		snapshot := &ConfigSnapshot{
			Version:   SnapshotVersion,
			Timestamp: snapshotTime,
			Config:    map[string]any{"test": i},
		}

		err := WriteSnapshot(snapshot, templatePath)
		if err != nil {
			t.Fatalf("WriteSnapshot failed for time %v: %v", snapshotTime, err)
		}

		// Property: The filename timestamp must match the snapshot's internal timestamp
		expectedTimestamp := snapshotTime.UTC().Format("20060102-150405")
		expectedPath := filepath.Join(testDir, fmt.Sprintf("config-%s.json", expectedTimestamp))

		// Verify the file exists at the expected path
		if _, statErr := os.Stat(expectedPath); os.IsNotExist(statErr) {
			t.Errorf("Expected file at %s for snapshot time %v, but file does not exist",
				expectedPath, snapshotTime)
		}

		// Read the file and verify the internal timestamp matches the filename
		data, err := os.ReadFile(expectedPath)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		var readSnapshot ConfigSnapshot
		if err := json.Unmarshal(data, &readSnapshot); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		// Property: Internal timestamp must match filename timestamp
		internalTimestamp := readSnapshot.Timestamp.UTC().Format("20060102-150405")
		if internalTimestamp != expectedTimestamp {
			t.Errorf("Timestamp mismatch: filename has %s but internal timestamp is %s",
				expectedTimestamp, internalTimestamp)
		}
	}
}

func TestWriteSnapshotProperties_TimestampNotCurrentTime(t *testing.T) {
	// **Feature: snapshot-core, Property 5: Timestamp Filename Consistency**
	// **Validates: Requirements 7.2, 7.3**
	// Verify that WriteSnapshot uses the snapshot's timestamp, NOT the current time.

	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "config-{{timestamp}}.json")

	// Create a snapshot with a timestamp in the past
	pastTime := time.Date(2020, 1, 1, 12, 0, 0, 0, time.UTC)
	snapshot := &ConfigSnapshot{
		Version:   SnapshotVersion,
		Timestamp: pastTime,
		Config:    map[string]any{"key": "value"},
	}

	// Write the snapshot (current time is 2024+, but filename should use 2020)
	err := WriteSnapshot(snapshot, templatePath)
	if err != nil {
		t.Fatalf("WriteSnapshot failed: %v", err)
	}

	// The file should be named with the snapshot's timestamp (2020), not current time
	expectedPath := filepath.Join(tmpDir, "config-20200101-120000.json")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected file at %s (using snapshot timestamp), but file does not exist", expectedPath)

		// List files in directory to see what was actually created
		entries, _ := os.ReadDir(tmpDir)
		for _, entry := range entries {
			t.Logf("Found file: %s", entry.Name())
		}
	}

	// Verify no file was created with current timestamp
	// (We can't check all possible current timestamps, but we can verify the expected one exists)
}

func TestWriteSnapshotProperties_MultipleTimestampOccurrences(t *testing.T) {
	// **Feature: snapshot-core, Property 5: Timestamp Filename Consistency**
	// **Validates: Requirements 7.2, 7.3**
	// Verify that multiple {{timestamp}} occurrences all use the same snapshot timestamp.

	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "{{timestamp}}", "config-{{timestamp}}.json")

	snapshotTime := time.Date(2024, 3, 15, 9, 45, 30, 0, time.UTC)
	snapshot := &ConfigSnapshot{
		Version:   SnapshotVersion,
		Timestamp: snapshotTime,
		Config:    map[string]any{"key": "value"},
	}

	err := WriteSnapshot(snapshot, templatePath)
	if err != nil {
		t.Fatalf("WriteSnapshot failed: %v", err)
	}

	// Both occurrences should use the same timestamp
	expectedTimestamp := "20240315-094530"
	expectedPath := filepath.Join(tmpDir, expectedTimestamp, fmt.Sprintf("config-%s.json", expectedTimestamp))

	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected file at %s, but file does not exist", expectedPath)

		// List what was created
		filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
			if err == nil {
				t.Logf("Found: %s", path)
			}
			return nil
		})
	}
}
