package rigging

import (
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
