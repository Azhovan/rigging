package rigging

import (
	"reflect"
	"testing"
)

func TestGetStructFieldMeta_ReturnsDefensiveCopy(t *testing.T) {
	type testConfig struct {
		Host string
		Port int
	}

	typ := reflect.TypeOf(testConfig{})

	first := getStructFieldMeta(typ)
	if len(first) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(first))
	}

	// Mutate the returned slice to verify cache isolation.
	first[0] = structFieldMeta{}

	second := getStructFieldMeta(typ)
	if len(second) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(second))
	}

	if second[0].field.Name != "Host" {
		t.Fatalf("expected cached field[0] to remain Host, got %q", second[0].field.Name)
	}
	if second[1].field.Name != "Port" {
		t.Fatalf("expected cached field[1] to remain Port, got %q", second[1].field.Name)
	}
}
