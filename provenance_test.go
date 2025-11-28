package rigging

import (
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
