package rigging

import (
	"reflect"
	"testing"
	"time"
)

func TestBindStruct_SimpleFields(t *testing.T) {
	type Config struct {
		Host string
		Port int
	}

	data := map[string]mergedEntry{
		"host": {value: "localhost", sourceName: "env"},
		"port": {value: "8080", sourceName: "env"},
	}

	var cfg Config
	var provFields []FieldProvenance
	errors := bindStruct(reflect.ValueOf(&cfg), data, &provFields, "", "")

	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	if cfg.Host != "localhost" {
		t.Errorf("Host = %q, want %q", cfg.Host, "localhost")
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want %d", cfg.Port, 8080)
	}

	// Check provenance
	if len(provFields) != 2 {
		t.Fatalf("provenance fields = %d, want 2", len(provFields))
	}
}

func TestBindStruct_WithDefaults(t *testing.T) {
	type Config struct {
		Host string `conf:"default:localhost"`
		Port int    `conf:"default:8080"`
	}

	data := map[string]mergedEntry{
		"host": {value: "example.com", sourceName: "env"},
		// port not provided, should use default
	}

	var cfg Config
	var provFields []FieldProvenance
	errors := bindStruct(reflect.ValueOf(&cfg), data, &provFields, "", "")

	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	if cfg.Host != "example.com" {
		t.Errorf("Host = %q, want %q", cfg.Host, "example.com")
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want %d", cfg.Port, 8080)
	}

	// Check that default source is recorded
	portProv := findProvenance(provFields, "Port")
	if portProv == nil {
		t.Fatal("Port provenance not found")
	}
	if portProv.SourceName != "default" {
		t.Errorf("Port source = %q, want %q", portProv.SourceName, "default")
	}
}

func TestBindStruct_RequiredField(t *testing.T) {
	type Config struct {
		Host string `conf:"required"`
		Port int
	}

	data := map[string]mergedEntry{
		"port": {value: "8080", sourceName: "env"},
		// host not provided but required
	}

	var cfg Config
	var provFields []FieldProvenance
	errors := bindStruct(reflect.ValueOf(&cfg), data, &provFields, "", "")

	// Binding phase should not check for required fields - that's validation's job
	// So we expect 0 errors from binding
	if len(errors) != 0 {
		t.Fatalf("errors = %d, want 0 (required check is done in validation phase)", len(errors))
	}

	// Verify that Port was set correctly
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want 8080", cfg.Port)
	}

	// Verify that Host is zero value (empty string)
	if cfg.Host != "" {
		t.Errorf("Host = %q, want empty string", cfg.Host)
	}
}

func TestBindStruct_TypeConversionError(t *testing.T) {
	type Config struct {
		Port int
	}

	data := map[string]mergedEntry{
		"port": {value: "not-a-number", sourceName: "env"},
	}

	var cfg Config
	var provFields []FieldProvenance
	errors := bindStruct(reflect.ValueOf(&cfg), data, &provFields, "", "")

	if len(errors) != 1 {
		t.Fatalf("errors = %d, want 1", len(errors))
	}

	if errors[0].Code != ErrCodeInvalidType {
		t.Errorf("error code = %q, want %q", errors[0].Code, ErrCodeInvalidType)
	}
	if errors[0].FieldPath != "Port" {
		t.Errorf("error field path = %q, want %q", errors[0].FieldPath, "Port")
	}
}

func TestBindStruct_NestedStruct(t *testing.T) {
	type Database struct {
		Host string
		Port int
	}
	type Config struct {
		Database Database
	}

	data := map[string]mergedEntry{
		"database.host": {value: "db.example.com", sourceName: "file"},
		"database.port": {value: "5432", sourceName: "file"},
	}

	var cfg Config
	var provFields []FieldProvenance
	errors := bindStruct(reflect.ValueOf(&cfg), data, &provFields, "", "")

	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	if cfg.Database.Host != "db.example.com" {
		t.Errorf("Database.Host = %q, want %q", cfg.Database.Host, "db.example.com")
	}
	if cfg.Database.Port != 5432 {
		t.Errorf("Database.Port = %d, want %d", cfg.Database.Port, 5432)
	}

	// Check provenance includes nested field paths
	hostProv := findProvenance(provFields, "Database.Host")
	if hostProv == nil {
		t.Fatal("Database.Host provenance not found")
	}
	if hostProv.KeyPath != "database.host" {
		t.Errorf("Database.Host key path = %q, want %q", hostProv.KeyPath, "database.host")
	}
}

func TestBindStruct_NestedStructWithPrefix(t *testing.T) {
	type Database struct {
		Host string
		Port int
	}
	type Config struct {
		Database Database `conf:"prefix:db"`
	}

	data := map[string]mergedEntry{
		"db.host": {value: "db.example.com", sourceName: "env"},
		"db.port": {value: "5432", sourceName: "env"},
	}

	var cfg Config
	var provFields []FieldProvenance
	errors := bindStruct(reflect.ValueOf(&cfg), data, &provFields, "", "")

	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	if cfg.Database.Host != "db.example.com" {
		t.Errorf("Database.Host = %q, want %q", cfg.Database.Host, "db.example.com")
	}
	if cfg.Database.Port != 5432 {
		t.Errorf("Database.Port = %d, want %d", cfg.Database.Port, 5432)
	}

	// Check provenance uses prefix
	hostProv := findProvenance(provFields, "Database.Host")
	if hostProv == nil {
		t.Fatal("Database.Host provenance not found")
	}
	if hostProv.KeyPath != "db.host" {
		t.Errorf("Database.Host key path = %q, want %q", hostProv.KeyPath, "db.host")
	}
}

func TestBindStruct_CustomName(t *testing.T) {
	type Config struct {
		APIKey string `conf:"name:api.key"`
	}

	data := map[string]mergedEntry{
		"api.key": {value: "secret123", sourceName: "env"},
	}

	var cfg Config
	var provFields []FieldProvenance
	errors := bindStruct(reflect.ValueOf(&cfg), data, &provFields, "", "")

	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	if cfg.APIKey != "secret123" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "secret123")
	}

	// Check provenance uses custom name
	apiProv := findProvenance(provFields, "APIKey")
	if apiProv == nil {
		t.Fatal("APIKey provenance not found")
	}
	if apiProv.KeyPath != "api.key" {
		t.Errorf("APIKey key path = %q, want %q", apiProv.KeyPath, "api.key")
	}
}

func TestBindStruct_SecretField(t *testing.T) {
	type Config struct {
		Password string `conf:"secret"`
	}

	data := map[string]mergedEntry{
		"password": {value: "secret123", sourceName: "env"},
	}

	var cfg Config
	var provFields []FieldProvenance
	errors := bindStruct(reflect.ValueOf(&cfg), data, &provFields, "", "")

	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	if cfg.Password != "secret123" {
		t.Errorf("Password = %q, want %q", cfg.Password, "secret123")
	}

	// Check provenance marks field as secret
	passProv := findProvenance(provFields, "Password")
	if passProv == nil {
		t.Fatal("Password provenance not found")
	}
	if !passProv.Secret {
		t.Error("Password should be marked as secret")
	}
}

func TestBindStruct_OptionalField(t *testing.T) {
	type Config struct {
		Timeout Optional[time.Duration]
	}

	t.Run("value provided", func(t *testing.T) {
		data := map[string]mergedEntry{
			"timeout": {value: "5s", sourceName: "env"},
		}

		var cfg Config
		var provFields []FieldProvenance
		errors := bindStruct(reflect.ValueOf(&cfg), data, &provFields, "", "")

		if len(errors) > 0 {
			t.Fatalf("unexpected errors: %v", errors)
		}

		val, set := cfg.Timeout.Get()
		if !set {
			t.Error("Timeout should be set")
		}
		if val != 5*time.Second {
			t.Errorf("Timeout = %v, want %v", val, 5*time.Second)
		}
	})

	t.Run("value not provided", func(t *testing.T) {
		data := map[string]mergedEntry{}

		var cfg Config
		var provFields []FieldProvenance
		errors := bindStruct(reflect.ValueOf(&cfg), data, &provFields, "", "")

		if len(errors) > 0 {
			t.Fatalf("unexpected errors: %v", errors)
		}

		_, set := cfg.Timeout.Get()
		if set {
			t.Error("Timeout should not be set")
		}
	})

	t.Run("default value", func(t *testing.T) {
		type ConfigWithDefault struct {
			Timeout Optional[time.Duration] `conf:"default:10s"`
		}

		data := map[string]mergedEntry{}

		var cfg ConfigWithDefault
		var provFields []FieldProvenance
		errors := bindStruct(reflect.ValueOf(&cfg), data, &provFields, "", "")

		if len(errors) > 0 {
			t.Fatalf("unexpected errors: %v", errors)
		}

		val, set := cfg.Timeout.Get()
		if !set {
			t.Error("Timeout should be set from default")
		}
		if val != 10*time.Second {
			t.Errorf("Timeout = %v, want %v", val, 10*time.Second)
		}
	})
}

func TestBindStruct_MultipleErrors(t *testing.T) {
	type Config struct {
		Host string `conf:"required"`
		Port int    `conf:"required"`
		Max  int
	}

	data := map[string]mergedEntry{
		"max": {value: "not-a-number", sourceName: "env"},
		// host and port missing
	}

	var cfg Config
	var provFields []FieldProvenance
	errors := bindStruct(reflect.ValueOf(&cfg), data, &provFields, "", "")

	// Binding phase only checks type conversion errors, not required fields
	// Should have 1 error: 1 type conversion (required checks are in validation phase)
	if len(errors) != 1 {
		t.Fatalf("errors = %d, want 1 (only type conversion error)", len(errors))
	}

	// Check we have the type conversion error
	if errors[0].Code != ErrCodeInvalidType {
		t.Errorf("error code = %q, want %q", errors[0].Code, ErrCodeInvalidType)
	}
	if errors[0].FieldPath != "Max" {
		t.Errorf("error field path = %q, want %q", errors[0].FieldPath, "Max")
	}
}

func TestBindStruct_AllTypes(t *testing.T) {
	type Config struct {
		Str      string
		Bool     bool
		Int      int
		Int8     int8
		Int16    int16
		Int32    int32
		Int64    int64
		Uint     uint
		Uint8    uint8
		Uint16   uint16
		Uint32   uint32
		Uint64   uint64
		Float32  float32
		Float64  float64
		Duration time.Duration
		Strings  []string
	}

	data := map[string]mergedEntry{
		"str":      {value: "hello", sourceName: "env"},
		"bool":     {value: "true", sourceName: "env"},
		"int":      {value: "42", sourceName: "env"},
		"int8":     {value: "8", sourceName: "env"},
		"int16":    {value: "16", sourceName: "env"},
		"int32":    {value: "32", sourceName: "env"},
		"int64":    {value: "64", sourceName: "env"},
		"uint":     {value: "42", sourceName: "env"},
		"uint8":    {value: "8", sourceName: "env"},
		"uint16":   {value: "16", sourceName: "env"},
		"uint32":   {value: "32", sourceName: "env"},
		"uint64":   {value: "64", sourceName: "env"},
		"float32":  {value: "3.14", sourceName: "env"},
		"float64":  {value: "2.718", sourceName: "env"},
		"duration": {value: "5m", sourceName: "env"},
		"strings":  {value: "a,b,c", sourceName: "env"},
	}

	var cfg Config
	var provFields []FieldProvenance
	errors := bindStruct(reflect.ValueOf(&cfg), data, &provFields, "", "")

	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	if cfg.Str != "hello" {
		t.Errorf("Str = %q, want %q", cfg.Str, "hello")
	}
	if cfg.Bool != true {
		t.Errorf("Bool = %v, want %v", cfg.Bool, true)
	}
	if cfg.Int != 42 {
		t.Errorf("Int = %d, want %d", cfg.Int, 42)
	}
	if cfg.Int8 != 8 {
		t.Errorf("Int8 = %d, want %d", cfg.Int8, 8)
	}
	if cfg.Int16 != 16 {
		t.Errorf("Int16 = %d, want %d", cfg.Int16, 16)
	}
	if cfg.Int32 != 32 {
		t.Errorf("Int32 = %d, want %d", cfg.Int32, 32)
	}
	if cfg.Int64 != 64 {
		t.Errorf("Int64 = %d, want %d", cfg.Int64, 64)
	}
	if cfg.Uint != 42 {
		t.Errorf("Uint = %d, want %d", cfg.Uint, 42)
	}
	if cfg.Uint8 != 8 {
		t.Errorf("Uint8 = %d, want %d", cfg.Uint8, 8)
	}
	if cfg.Uint16 != 16 {
		t.Errorf("Uint16 = %d, want %d", cfg.Uint16, 16)
	}
	if cfg.Uint32 != 32 {
		t.Errorf("Uint32 = %d, want %d", cfg.Uint32, 32)
	}
	if cfg.Uint64 != 64 {
		t.Errorf("Uint64 = %d, want %d", cfg.Uint64, 64)
	}
	if cfg.Float32 != 3.14 {
		t.Errorf("Float32 = %f, want %f", cfg.Float32, 3.14)
	}
	if cfg.Float64 != 2.718 {
		t.Errorf("Float64 = %f, want %f", cfg.Float64, 2.718)
	}
	if cfg.Duration != 5*time.Minute {
		t.Errorf("Duration = %v, want %v", cfg.Duration, 5*time.Minute)
	}
	if !reflect.DeepEqual(cfg.Strings, []string{"a", "b", "c"}) {
		t.Errorf("Strings = %v, want %v", cfg.Strings, []string{"a", "b", "c"})
	}
}

func TestBindStruct_NestedStructFromMap(t *testing.T) {
	type Database struct {
		Host string
		Port int
	}
	type Config struct {
		Database Database
	}

	// Simulate file source that provides nested map
	data := map[string]mergedEntry{
		"database": {
			value: map[string]any{
				"host": "db.example.com",
				"port": 5432,
			},
			sourceName: "file",
		},
	}

	var cfg Config
	var provFields []FieldProvenance
	errors := bindStruct(reflect.ValueOf(&cfg), data, &provFields, "", "")

	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	if cfg.Database.Host != "db.example.com" {
		t.Errorf("Database.Host = %q, want %q", cfg.Database.Host, "db.example.com")
	}
	if cfg.Database.Port != 5432 {
		t.Errorf("Database.Port = %d, want %d", cfg.Database.Port, 5432)
	}
}

// Helper function to find provenance by field path
func findProvenance(fields []FieldProvenance, fieldPath string) *FieldProvenance {
	for i := range fields {
		if fields[i].FieldPath == fieldPath {
			return &fields[i]
		}
	}
	return nil
}
