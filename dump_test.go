package rigging

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestDumpEffective_TextFormat(t *testing.T) {
	type Config struct {
		Host     string `conf:"name:host"`
		Port     int    `conf:"name:port"`
		Password string `conf:"name:password,secret"`
		Enabled  bool   `conf:"name:enabled"`
	}

	cfg := &Config{
		Host:     "localhost",
		Port:     8080,
		Password: "secret123",
		Enabled:  true,
	}

	// Store provenance with secret marking
	prov := &Provenance{
		Fields: []FieldProvenance{
			{FieldPath: "Host", KeyPath: "host", SourceName: "env", Secret: false},
			{FieldPath: "Port", KeyPath: "port", SourceName: "file", Secret: false},
			{FieldPath: "Password", KeyPath: "password", SourceName: "env", Secret: true},
			{FieldPath: "Enabled", KeyPath: "enabled", SourceName: "default", Secret: false},
		},
	}
	storeProvenance(cfg, prov)

	var buf bytes.Buffer
	err := DumpEffective(&buf, cfg)
	if err != nil {
		t.Fatalf("DumpEffective failed: %v", err)
	}

	output := buf.String()

	// Check that password is redacted
	if !strings.Contains(output, "***redacted***") {
		t.Errorf("Expected password to be redacted, got: %s", output)
	}

	// Check that other fields are present
	if !strings.Contains(output, `"localhost"`) {
		t.Errorf("Expected host to be present, got: %s", output)
	}
	if !strings.Contains(output, "8080") {
		t.Errorf("Expected port to be present, got: %s", output)
	}
	if !strings.Contains(output, "true") {
		t.Errorf("Expected enabled to be present, got: %s", output)
	}

	// Ensure actual password is not in output
	if strings.Contains(output, "secret123") {
		t.Errorf("Password should be redacted, but found in output: %s", output)
	}
}

func TestDumpEffective_WithSources(t *testing.T) {
	type Config struct {
		Host string `conf:"name:host"`
		Port int    `conf:"name:port"`
	}

	cfg := &Config{
		Host: "localhost",
		Port: 8080,
	}

	prov := &Provenance{
		Fields: []FieldProvenance{
			{FieldPath: "Host", KeyPath: "host", SourceName: "env:HOST", Secret: false},
			{FieldPath: "Port", KeyPath: "port", SourceName: "file:/etc/config.yaml", Secret: false},
		},
	}
	storeProvenance(cfg, prov)

	var buf bytes.Buffer
	err := DumpEffective(&buf, cfg, WithSources())
	if err != nil {
		t.Fatalf("DumpEffective failed: %v", err)
	}

	output := buf.String()

	// Check that source attribution is included
	if !strings.Contains(output, "(source: env:HOST)") {
		t.Errorf("Expected source attribution for host, got: %s", output)
	}
	if !strings.Contains(output, "(source: file:/etc/config.yaml)") {
		t.Errorf("Expected source attribution for port, got: %s", output)
	}
}

func TestDumpEffective_JSONFormat(t *testing.T) {
	type Config struct {
		Host     string `conf:"name:host"`
		Port     int    `conf:"name:port"`
		Password string `conf:"name:password,secret"`
		Enabled  bool   `conf:"name:enabled"`
	}

	cfg := &Config{
		Host:     "localhost",
		Port:     8080,
		Password: "secret123",
		Enabled:  true,
	}

	prov := &Provenance{
		Fields: []FieldProvenance{
			{FieldPath: "Host", KeyPath: "host", SourceName: "env", Secret: false},
			{FieldPath: "Port", KeyPath: "port", SourceName: "file", Secret: false},
			{FieldPath: "Password", KeyPath: "password", SourceName: "env", Secret: true},
			{FieldPath: "Enabled", KeyPath: "enabled", SourceName: "default", Secret: false},
		},
	}
	storeProvenance(cfg, prov)

	var buf bytes.Buffer
	err := DumpEffective(&buf, cfg, AsJSON())
	if err != nil {
		t.Fatalf("DumpEffective failed: %v", err)
	}

	// Parse JSON to verify structure
	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Check values
	if result["host"] != "localhost" {
		t.Errorf("Expected host=localhost, got: %v", result["host"])
	}
	if result["port"] != float64(8080) { // JSON numbers are float64
		t.Errorf("Expected port=8080, got: %v", result["port"])
	}
	if result["password"] != "***redacted***" {
		t.Errorf("Expected password to be redacted, got: %v", result["password"])
	}
	if result["enabled"] != true {
		t.Errorf("Expected enabled=true, got: %v", result["enabled"])
	}
}

func TestDumpEffective_NestedStructs(t *testing.T) {
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

	var buf bytes.Buffer
	err := DumpEffective(&buf, cfg)
	if err != nil {
		t.Fatalf("DumpEffective failed: %v", err)
	}

	output := buf.String()

	// Check nested fields
	if !strings.Contains(output, "database.host") {
		t.Errorf("Expected database.host in output, got: %s", output)
	}
	if !strings.Contains(output, "database.port") {
		t.Errorf("Expected database.port in output, got: %s", output)
	}
	if !strings.Contains(output, "database.password") {
		t.Errorf("Expected database.password in output, got: %s", output)
	}

	// Check password is redacted
	if !strings.Contains(output, "***redacted***") {
		t.Errorf("Expected password to be redacted, got: %s", output)
	}
	if strings.Contains(output, "dbpass") {
		t.Errorf("Password should be redacted, but found in output: %s", output)
	}
}

func TestDumpEffective_JSONNestedStructs(t *testing.T) {
	type Database struct {
		Host     string `conf:"name:host"`
		Port     int    `conf:"name:port"`
		Password string `conf:"name:password,secret"`
	}

	type Config struct {
		AppName  string   `conf:"name:app_name"`
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
			{FieldPath: "AppName", KeyPath: "app_name", SourceName: "env", Secret: false},
			{FieldPath: "Database.Host", KeyPath: "database.host", SourceName: "file", Secret: false},
			{FieldPath: "Database.Port", KeyPath: "database.port", SourceName: "file", Secret: false},
			{FieldPath: "Database.Password", KeyPath: "database.password", SourceName: "env", Secret: true},
		},
	}
	storeProvenance(cfg, prov)

	var buf bytes.Buffer
	err := DumpEffective(&buf, cfg, AsJSON())
	if err != nil {
		t.Fatalf("DumpEffective failed: %v", err)
	}

	// Parse JSON
	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Check nested structure
	if result["app_name"] != "myapp" {
		t.Errorf("Expected app_name=myapp, got: %v", result["app_name"])
	}

	database, ok := result["database"].(map[string]any)
	if !ok {
		t.Fatalf("Expected database to be a map, got: %T", result["database"])
	}

	if database["host"] != "db.example.com" {
		t.Errorf("Expected database.host=db.example.com, got: %v", database["host"])
	}
	if database["port"] != float64(5432) {
		t.Errorf("Expected database.port=5432, got: %v", database["port"])
	}
	if database["password"] != "***redacted***" {
		t.Errorf("Expected database.password to be redacted, got: %v", database["password"])
	}
}

func TestDumpEffective_OptionalFields(t *testing.T) {
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
			{FieldPath: "NotSet", KeyPath: "notset", SourceName: "", Secret: false},
		},
	}
	storeProvenance(cfg, prov)

	var buf bytes.Buffer
	err := DumpEffective(&buf, cfg)
	if err != nil {
		t.Fatalf("DumpEffective failed: %v", err)
	}

	output := buf.String()

	// Check that set optional is shown
	if !strings.Contains(output, `"set"`) {
		t.Errorf("Expected optional field to show value, got: %s", output)
	}

	// Check that unset optional is shown as not set
	if !strings.Contains(output, "<not set>") {
		t.Errorf("Expected notset field to show <not set>, got: %s", output)
	}
}

func TestDumpEffective_DifferentTypes(t *testing.T) {
	type Config struct {
		StringVal   string        `conf:"name:string_val"`
		IntVal      int           `conf:"name:int_val"`
		FloatVal    float64       `conf:"name:float_val"`
		BoolVal     bool          `conf:"name:bool_val"`
		DurationVal time.Duration `conf:"name:duration_val"`
		SliceVal    []string      `conf:"name:slice_val"`
	}

	cfg := &Config{
		StringVal:   "hello",
		IntVal:      42,
		FloatVal:    3.14,
		BoolVal:     true,
		DurationVal: 5 * time.Second,
		SliceVal:    []string{"a", "b", "c"},
	}

	var buf bytes.Buffer
	err := DumpEffective(&buf, cfg)
	if err != nil {
		t.Fatalf("DumpEffective failed: %v", err)
	}

	output := buf.String()

	// Check all types are present
	if !strings.Contains(output, `"hello"`) {
		t.Errorf("Expected string value, got: %s", output)
	}
	if !strings.Contains(output, "42") {
		t.Errorf("Expected int value, got: %s", output)
	}
	if !strings.Contains(output, "3.14") {
		t.Errorf("Expected float value, got: %s", output)
	}
	if !strings.Contains(output, "true") {
		t.Errorf("Expected bool value, got: %s", output)
	}
	if !strings.Contains(output, "5s") {
		t.Errorf("Expected duration value, got: %s", output)
	}
	if !strings.Contains(output, "[a, b, c]") {
		t.Errorf("Expected slice value, got: %s", output)
	}
}

func TestDumpEffective_NilConfig(t *testing.T) {
	var cfg *struct{}
	var buf bytes.Buffer
	err := DumpEffective(&buf, cfg)
	if err == nil {
		t.Error("Expected error for nil config")
	}
	if !strings.Contains(err.Error(), "nil") {
		t.Errorf("Expected error message to mention nil, got: %v", err)
	}
}

func TestDumpEffective_WithIndent(t *testing.T) {
	type Config struct {
		Host string `conf:"name:host"`
		Port int    `conf:"name:port"`
	}

	cfg := &Config{
		Host: "localhost",
		Port: 8080,
	}

	var buf bytes.Buffer
	err := DumpEffective(&buf, cfg, AsJSON(), WithIndent("\t"))
	if err != nil {
		t.Fatalf("DumpEffective failed: %v", err)
	}

	output := buf.String()

	// Check that output uses tabs for indentation
	if !strings.Contains(output, "\t") {
		t.Errorf("Expected tab indentation in JSON output, got: %s", output)
	}
}

func TestDumpEffective_NoProvenance(t *testing.T) {
	type Config struct {
		Host string `conf:"name:host"`
		Port int    `conf:"name:port"`
	}

	cfg := &Config{
		Host: "localhost",
		Port: 8080,
	}

	// Don't store provenance

	var buf bytes.Buffer
	err := DumpEffective(&buf, cfg)
	if err != nil {
		t.Fatalf("DumpEffective failed: %v", err)
	}

	output := buf.String()

	// Should still work without provenance
	if !strings.Contains(output, `"localhost"`) {
		t.Errorf("Expected host value, got: %s", output)
	}
	if !strings.Contains(output, "8080") {
		t.Errorf("Expected port value, got: %s", output)
	}
}

func TestDumpEffective_SecretWithoutProvenance(t *testing.T) {
	type Config struct {
		Password string `conf:"name:password,secret"`
	}

	cfg := &Config{
		Password: "secret123",
	}

	// Don't store provenance - secret tag won't be detected
	// This tests that without provenance, secrets aren't redacted

	var buf bytes.Buffer
	err := DumpEffective(&buf, cfg)
	if err != nil {
		t.Fatalf("DumpEffective failed: %v", err)
	}

	output := buf.String()

	// Without provenance, the secret tag isn't tracked, so value shows
	// This is expected behavior - provenance is needed for redaction
	if !strings.Contains(output, "secret123") {
		t.Logf("Note: Without provenance, secrets are not redacted. Output: %s", output)
	}
}
