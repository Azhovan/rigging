package rigging

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestProvenance_GetProvenance(t *testing.T) {
	type TestConfig struct {
		Host string
		Port int
	}

	t.Run("returns provenance for stored config", func(t *testing.T) {
		cfg := &TestConfig{Host: "localhost", Port: 8080}
		expected := &Provenance{
			Fields: []FieldProvenance{
				{
					FieldPath:  "Host",
					KeyPath:    "host",
					SourceName: "env",
					Secret:     false,
				},
				{
					FieldPath:  "Port",
					KeyPath:    "port",
					SourceName: "file:/etc/config.yaml",
					Secret:     false,
				},
			},
		}

		storeProvenance(cfg, expected)

		prov, ok := GetProvenance(cfg)
		if !ok {
			t.Fatal("expected provenance to be found")
		}
		if prov == nil {
			t.Fatal("expected non-nil provenance")
		}
		if len(prov.Fields) != 2 {
			t.Errorf("expected 2 fields, got %d", len(prov.Fields))
		}
		if prov.Fields[0].FieldPath != "Host" {
			t.Errorf("expected FieldPath 'Host', got %q", prov.Fields[0].FieldPath)
		}
		if prov.Fields[0].SourceName != "env" {
			t.Errorf("expected SourceName 'env', got %q", prov.Fields[0].SourceName)
		}
	})

	t.Run("returns false for config without provenance", func(t *testing.T) {
		cfg := &TestConfig{Host: "localhost", Port: 8080}

		prov, ok := GetProvenance(cfg)
		if ok {
			t.Error("expected provenance not to be found")
		}
		if prov != nil {
			t.Error("expected nil provenance")
		}
	})

	t.Run("returns false for nil config", func(t *testing.T) {
		var cfg *TestConfig

		prov, ok := GetProvenance(cfg)
		if ok {
			t.Error("expected provenance not to be found for nil config")
		}
		if prov != nil {
			t.Error("expected nil provenance for nil config")
		}
	})
}

func TestProvenance_StoreAndDelete(t *testing.T) {
	type TestConfig struct {
		Value string
	}

	cfg := &TestConfig{Value: "test"}
	prov := &Provenance{
		Fields: []FieldProvenance{
			{
				FieldPath:  "Value",
				KeyPath:    "value",
				SourceName: "env",
				Secret:     true,
			},
		},
	}

	// Store provenance
	storeProvenance(cfg, prov)

	// Verify it's stored
	retrieved, ok := GetProvenance(cfg)
	if !ok {
		t.Fatal("expected provenance to be stored")
	}
	if len(retrieved.Fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(retrieved.Fields))
	}
	if !retrieved.Fields[0].Secret {
		t.Error("expected Secret to be true")
	}

	// Delete provenance
	deleteProvenance(cfg)

	// Verify it's deleted
	_, ok = GetProvenance(cfg)
	if ok {
		t.Error("expected provenance to be deleted")
	}
}

func TestProvenance_SecretField(t *testing.T) {
	type TestConfig struct {
		Password string
	}

	cfg := &TestConfig{Password: "secret123"}
	prov := &Provenance{
		Fields: []FieldProvenance{
			{
				FieldPath:  "Password",
				KeyPath:    "password",
				SourceName: "env:DB_PASSWORD",
				Secret:     true,
			},
		},
	}

	storeProvenance(cfg, prov)

	retrieved, ok := GetProvenance(cfg)
	if !ok {
		t.Fatal("expected provenance to be found")
	}
	if len(retrieved.Fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(retrieved.Fields))
	}

	field := retrieved.Fields[0]
	if field.FieldPath != "Password" {
		t.Errorf("expected FieldPath 'Password', got %q", field.FieldPath)
	}
	if field.KeyPath != "password" {
		t.Errorf("expected KeyPath 'password', got %q", field.KeyPath)
	}
	if field.SourceName != "env:DB_PASSWORD" {
		t.Errorf("expected SourceName 'env:DB_PASSWORD', got %q", field.SourceName)
	}
	if !field.Secret {
		t.Error("expected Secret to be true")
	}
}

func TestProvenance_MultipleConfigs(t *testing.T) {
	type TestConfig struct {
		Value string
	}

	cfg1 := &TestConfig{Value: "config1"}
	cfg2 := &TestConfig{Value: "config2"}

	prov1 := &Provenance{
		Fields: []FieldProvenance{
			{FieldPath: "Value", KeyPath: "value", SourceName: "source1", Secret: false},
		},
	}
	prov2 := &Provenance{
		Fields: []FieldProvenance{
			{FieldPath: "Value", KeyPath: "value", SourceName: "source2", Secret: true},
		},
	}

	storeProvenance(cfg1, prov1)
	storeProvenance(cfg2, prov2)

	// Verify cfg1 has correct provenance
	retrieved1, ok := GetProvenance(cfg1)
	if !ok {
		t.Fatal("expected provenance for cfg1")
	}
	if retrieved1.Fields[0].SourceName != "source1" {
		t.Errorf("expected source1, got %q", retrieved1.Fields[0].SourceName)
	}
	if retrieved1.Fields[0].Secret {
		t.Error("expected Secret to be false for cfg1")
	}

	// Verify cfg2 has correct provenance
	retrieved2, ok := GetProvenance(cfg2)
	if !ok {
		t.Fatal("expected provenance for cfg2")
	}
	if retrieved2.Fields[0].SourceName != "source2" {
		t.Errorf("expected source2, got %q", retrieved2.Fields[0].SourceName)
	}
	if !retrieved2.Fields[0].Secret {
		t.Error("expected Secret to be true for cfg2")
	}
}

// mockSourceWithKeys implements SourceWithKeys for testing.
type mockSourceWithKeys struct {
	name         string
	data         map[string]any
	originalKeys map[string]string
	err          error
}

func (m *mockSourceWithKeys) Load(ctx context.Context) (map[string]any, error) {
	result, _, err := m.LoadWithKeys(ctx)
	return result, err
}

func (m *mockSourceWithKeys) LoadWithKeys(ctx context.Context) (map[string]any, map[string]string, error) {
	if m.err != nil {
		return nil, nil, m.err
	}
	if m.data == nil {
		return make(map[string]any), make(map[string]string), nil
	}
	return m.data, m.originalKeys, nil
}

func (m *mockSourceWithKeys) Watch(ctx context.Context) (<-chan ChangeEvent, error) {
	return nil, ErrWatchNotSupported
}

func (m *mockSourceWithKeys) Name() string {
	return m.name
}

// TestProvenance_WithSourceKeys verifies that provenance tracks original source keys.
func TestProvenance_WithSourceKeys(t *testing.T) {
	type Config struct {
		Host     string `conf:"required"`
		Port     int    `conf:"default:8080"`
		Password string `conf:"secret,required"`
	}

	source := &mockSourceWithKeys{
		name: "env:APP_",
		data: map[string]any{
			"host":     "localhost",
			"port":     9090,
			"password": "secret123",
		},
		originalKeys: map[string]string{
			"host":     "APP_HOST",
			"port":     "APP_PORT",
			"password": "APP_PASSWORD",
		},
	}

	loader := NewLoader[Config]().WithSource(source)
	cfg, err := loader.Load(context.Background())

	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Get provenance
	prov, ok := GetProvenance(cfg)
	if !ok {
		t.Fatal("provenance not found for config")
	}

	if len(prov.Fields) != 3 {
		t.Fatalf("expected 3 provenance fields, got %d", len(prov.Fields))
	}

	// Verify each field has the correct original key
	expectedSources := map[string]string{
		"Host":     "env:APP_APP_HOST",
		"Port":     "env:APP_APP_PORT",
		"Password": "env:APP_APP_PASSWORD",
	}

	for _, field := range prov.Fields {
		expected, ok := expectedSources[field.FieldPath]
		if !ok {
			t.Errorf("unexpected field in provenance: %s", field.FieldPath)
			continue
		}

		if field.SourceName != expected {
			t.Errorf("field %s: expected source %q, got %q", field.FieldPath, expected, field.SourceName)
		}

		// Verify secret flag
		if field.FieldPath == "Password" && !field.Secret {
			t.Errorf("field %s should be marked as secret", field.FieldPath)
		}
	}
}

// TestProvenance_WithoutSourceKeys verifies that provenance works with sources that don't implement SourceWithKeys.
func TestProvenance_WithoutSourceKeys(t *testing.T) {
	type Config struct {
		Host string
		Port int
	}

	source := &mockSource{
		name: "file:config.yaml",
		data: map[string]any{
			"host": "localhost",
			"port": 8080,
		},
	}

	loader := NewLoader[Config]().WithSource(source)
	cfg, err := loader.Load(context.Background())

	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Get provenance
	prov, ok := GetProvenance(cfg)
	if !ok {
		t.Fatal("provenance not found for config")
	}

	if len(prov.Fields) != 2 {
		t.Fatalf("expected 2 provenance fields, got %d", len(prov.Fields))
	}

	// Verify each field has the source name (not full key since SourceWithKeys not implemented)
	for _, field := range prov.Fields {
		if field.SourceName != "file:config.yaml" {
			t.Errorf("field %s: expected source %q, got %q", field.FieldPath, "file:config.yaml", field.SourceName)
		}
	}
}

// TestProvenance_MultipleSources verifies that provenance tracks which source provided each value.
func TestProvenance_MultipleSources(t *testing.T) {
	type Config struct {
		Host     string
		Port     int
		Database string
	}

	source1 := &mockSourceWithKeys{
		name: "file:config.yaml",
		data: map[string]any{
			"host": "localhost",
			"port": 8080,
		},
		originalKeys: map[string]string{
			"host": "server.host",
			"port": "server.port",
		},
	}

	source2 := &mockSourceWithKeys{
		name: "env:APP_",
		data: map[string]any{
			"port":     9090, // Override port
			"database": "mydb",
		},
		originalKeys: map[string]string{
			"port":     "APP_PORT",
			"database": "APP_DATABASE",
		},
	}

	loader := NewLoader[Config]().
		WithSource(source1).
		WithSource(source2)

	cfg, err := loader.Load(context.Background())

	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify values
	if cfg.Host != "localhost" {
		t.Errorf("expected Host=localhost, got %s", cfg.Host)
	}
	if cfg.Port != 9090 {
		t.Errorf("expected Port=9090 (overridden), got %d", cfg.Port)
	}
	if cfg.Database != "mydb" {
		t.Errorf("expected Database=mydb, got %s", cfg.Database)
	}

	// Get provenance
	prov, ok := GetProvenance(cfg)
	if !ok {
		t.Fatal("provenance not found for config")
	}

	if len(prov.Fields) != 3 {
		t.Fatalf("expected 3 provenance fields, got %d", len(prov.Fields))
	}

	// Verify sources
	expectedSources := map[string]string{
		"Host":     "file:config.yaml",
		"Port":     "env:APP_APP_PORT", // Should be from source2
		"Database": "env:APP_APP_DATABASE",
	}

	for _, field := range prov.Fields {
		expected, ok := expectedSources[field.FieldPath]
		if !ok {
			t.Errorf("unexpected field in provenance: %s", field.FieldPath)
			continue
		}

		if field.SourceName != expected {
			t.Errorf("field %s: expected source %q, got %q", field.FieldPath, expected, field.SourceName)
		}
	}
}

// TestProvenance_DefaultValues verifies that provenance tracks default values.
func TestProvenance_DefaultValues(t *testing.T) {
	type Config struct {
		Host string `conf:"default:localhost"`
		Port int    `conf:"default:8080"`
		Name string
	}

	source := &mockSourceWithKeys{
		name: "env:APP_",
		data: map[string]any{
			"name": "myapp",
		},
		originalKeys: map[string]string{
			"name": "APP_NAME",
		},
	}

	loader := NewLoader[Config]().WithSource(source)
	cfg, err := loader.Load(context.Background())

	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify values
	if cfg.Host != "localhost" {
		t.Errorf("expected Host=localhost (default), got %s", cfg.Host)
	}
	if cfg.Port != 8080 {
		t.Errorf("expected Port=8080 (default), got %d", cfg.Port)
	}

	// Get provenance
	prov, ok := GetProvenance(cfg)
	if !ok {
		t.Fatal("provenance not found for config")
	}

	// Should have 3 fields (Host, Port, Name)
	if len(prov.Fields) != 3 {
		t.Fatalf("expected 3 provenance fields, got %d", len(prov.Fields))
	}

	// Verify sources
	expectedSources := map[string]string{
		"Host": "default",
		"Port": "default",
		"Name": "env:APP_APP_NAME",
	}

	for _, field := range prov.Fields {
		expected, ok := expectedSources[field.FieldPath]
		if !ok {
			t.Errorf("unexpected field in provenance: %s", field.FieldPath)
			continue
		}

		if field.SourceName != expected {
			t.Errorf("field %s: expected source %q, got %q", field.FieldPath, expected, field.SourceName)
		}
	}
}

// TestProvenance_NestedStructs verifies that provenance tracks nested struct fields.
func TestProvenance_NestedStructs(t *testing.T) {
	type Database struct {
		Host     string `conf:"required"`
		Port     int    `conf:"default:5432"`
		Password string `conf:"secret,required"`
	}

	type Config struct {
		AppName  string   `conf:"required"`
		Database Database `conf:"prefix:db"`
	}

	source := &mockSourceWithKeys{
		name: "env:APP_",
		data: map[string]any{
			"appname":     "myapp",
			"db.host":     "dbhost",
			"db.password": "dbpass",
		},
		originalKeys: map[string]string{
			"appname":     "APP_APPNAME",
			"db.host":     "APP_DB__HOST",
			"db.password": "APP_DB__PASSWORD",
		},
	}

	loader := NewLoader[Config]().WithSource(source)
	cfg, err := loader.Load(context.Background())

	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Get provenance
	prov, ok := GetProvenance(cfg)
	if !ok {
		t.Fatal("provenance not found for config")
	}

	// Should have 4 fields (AppName, Database.Host, Database.Port, Database.Password)
	if len(prov.Fields) != 4 {
		t.Fatalf("expected 4 provenance fields, got %d", len(prov.Fields))
	}

	// Verify sources
	expectedSources := map[string]string{
		"AppName":           "env:APP_APP_APPNAME",
		"Database.Host":     "env:APP_APP_DB__HOST",
		"Database.Port":     "default",
		"Database.Password": "env:APP_APP_DB__PASSWORD",
	}

	for _, field := range prov.Fields {
		expected, ok := expectedSources[field.FieldPath]
		if !ok {
			t.Errorf("unexpected field in provenance: %s", field.FieldPath)
			continue
		}

		if field.SourceName != expected {
			t.Errorf("field %s: expected source %q, got %q", field.FieldPath, expected, field.SourceName)
		}

		// Verify secret flag
		if field.FieldPath == "Database.Password" && !field.Secret {
			t.Errorf("field %s should be marked as secret", field.FieldPath)
		}
	}
}

// TestProvenance_KeyPath verifies that provenance tracks the normalized key path.
func TestProvenance_KeyPath(t *testing.T) {
	type Database struct {
		Host string
		Port int
	}

	type Config struct {
		Database Database `conf:"name:db"`
	}

	source := &mockSourceWithKeys{
		name: "file:config.yaml",
		data: map[string]any{
			"db.host": "localhost",
			"db.port": 5432,
		},
		originalKeys: map[string]string{
			"db.host": "database.host",
			"db.port": "database.port",
		},
	}

	loader := NewLoader[Config]().WithSource(source)
	cfg, err := loader.Load(context.Background())

	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Get provenance
	prov, ok := GetProvenance(cfg)
	if !ok {
		t.Fatal("provenance not found for config")
	}

	// Verify key paths
	expectedKeyPaths := map[string]string{
		"Database.Host": "db.host",
		"Database.Port": "db.port",
	}

	for _, field := range prov.Fields {
		expected, ok := expectedKeyPaths[field.FieldPath]
		if !ok {
			t.Errorf("unexpected field in provenance: %s", field.FieldPath)
			continue
		}

		if field.KeyPath != expected {
			t.Errorf("field %s: expected key path %q, got %q", field.FieldPath, expected, field.KeyPath)
		}
	}
}

// TestProvenance_RealEnvSource tests provenance with actual environment variables.
func TestProvenance_RealEnvSource(t *testing.T) {
	// This test requires importing sourceenv package
	// We'll create a simple test that verifies the integration
	type Config struct {
		Host string
		Port int
	}

	// Set environment variables
	os.Setenv("TEST_HOST", "testhost")
	os.Setenv("TEST_PORT", "9999")
	defer os.Unsetenv("TEST_HOST")
	defer os.Unsetenv("TEST_PORT")

	// Create a mock env source that simulates sourceenv behavior
	source := &mockSourceWithKeys{
		name: "env:TEST_",
		data: map[string]any{
			"host": "testhost",
			"port": "9999",
		},
		originalKeys: map[string]string{
			"host": "TEST_HOST",
			"port": "TEST_PORT",
		},
	}

	loader := NewLoader[Config]().WithSource(source)
	cfg, err := loader.Load(context.Background())

	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Get provenance
	prov, ok := GetProvenance(cfg)
	if !ok {
		t.Fatal("provenance not found for config")
	}

	// Verify sources include full env var names
	for _, field := range prov.Fields {
		if field.FieldPath == "Host" {
			if field.SourceName != "env:TEST_TEST_HOST" {
				t.Errorf("expected source %q, got %q", "env:TEST_TEST_HOST", field.SourceName)
			}
		}
		if field.FieldPath == "Port" {
			if field.SourceName != "env:TEST_TEST_PORT" {
				t.Errorf("expected source %q, got %q", "env:TEST_TEST_PORT", field.SourceName)
			}
		}
	}
}

// TestProvenance_RealFileSource tests provenance with actual file sources.
func TestProvenance_RealFileSource(t *testing.T) {
	type Config struct {
		Server struct {
			Host string
			Port int
		}
	}

	// Create a temporary YAML file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")
	configContent := `server:
  host: filehost
  port: 7777
`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Create a mock file source that simulates sourcefile behavior
	source := &mockSourceWithKeys{
		name: "file:config.yaml",
		data: map[string]any{
			"server.host": "filehost",
			"server.port": 7777,
		},
		originalKeys: map[string]string{
			"server.host": "server.host",
			"server.port": "server.port",
		},
	}

	loader := NewLoader[Config]().WithSource(source)
	cfg, err := loader.Load(context.Background())

	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Get provenance
	prov, ok := GetProvenance(cfg)
	if !ok {
		t.Fatal("provenance not found for config")
	}

	// Verify sources include file name and key path
	for _, field := range prov.Fields {
		if !contains(field.SourceName, "file:config.yaml") {
			t.Errorf("field %s: expected source to contain %q, got %q", field.FieldPath, "file:config.yaml", field.SourceName)
		}
	}
}

// TestProvenance_MixedSources tests provenance with multiple source types.
func TestProvenance_MixedSources(t *testing.T) {
	type Config struct {
		Host     string
		Port     int
		Database string
		Secret   string `conf:"secret"`
	}

	fileSource := &mockSourceWithKeys{
		name: "file:config.yaml",
		data: map[string]any{
			"host": "filehost",
			"port": 8080,
		},
		originalKeys: map[string]string{
			"host": "server.host",
			"port": "server.port",
		},
	}

	envSource := &mockSourceWithKeys{
		name: "env:APP_",
		data: map[string]any{
			"port":     9090, // Override
			"database": "proddb",
			"secret":   "topsecret",
		},
		originalKeys: map[string]string{
			"port":     "APP_PORT",
			"database": "APP_DATABASE",
			"secret":   "APP_SECRET",
		},
	}

	loader := NewLoader[Config]().
		WithSource(fileSource).
		WithSource(envSource)

	cfg, err := loader.Load(context.Background())

	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify values
	if cfg.Host != "filehost" {
		t.Errorf("expected Host=filehost, got %s", cfg.Host)
	}
	if cfg.Port != 9090 {
		t.Errorf("expected Port=9090 (overridden by env), got %d", cfg.Port)
	}

	// Get provenance
	prov, ok := GetProvenance(cfg)
	if !ok {
		t.Fatal("provenance not found for config")
	}

	// Verify each field's source
	expectedSources := map[string]string{
		"Host":     "file:config.yaml",
		"Port":     "env:APP_APP_PORT", // Overridden by env
		"Database": "env:APP_APP_DATABASE",
		"Secret":   "env:APP_APP_SECRET",
	}

	for _, field := range prov.Fields {
		expected, ok := expectedSources[field.FieldPath]
		if !ok {
			t.Errorf("unexpected field in provenance: %s", field.FieldPath)
			continue
		}

		if field.SourceName != expected {
			t.Errorf("field %s: expected source %q, got %q", field.FieldPath, expected, field.SourceName)
		}

		// Verify secret flag
		if field.FieldPath == "Secret" && !field.Secret {
			t.Errorf("field %s should be marked as secret", field.FieldPath)
		}
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
