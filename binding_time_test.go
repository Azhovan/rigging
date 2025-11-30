package rigging

import (
	"reflect"
	"testing"
	"time"
)

// TestConvertValue_TimeTime tests time.Time conversion from various formats.
func TestConvertValue_TimeTime(t *testing.T) {
	targetType := reflect.TypeOf(time.Time{})

	tests := []struct {
		name      string
		input     any
		wantError bool
	}{
		{
			name:      "RFC3339 format",
			input:     "2025-11-30T12:00:00Z",
			wantError: false,
		},
		{
			name:      "RFC3339Nano format",
			input:     "2025-11-30T12:00:00.123456789Z",
			wantError: false,
		},
		{
			name:      "RFC3339 with timezone",
			input:     "2025-11-30T12:00:00+05:30",
			wantError: false,
		},
		{
			name:      "Date and time without timezone",
			input:     "2025-11-30 12:00:00",
			wantError: false,
		},
		{
			name:      "Date only",
			input:     "2025-11-30",
			wantError: false,
		},
		{
			name:      "time.Time value",
			input:     time.Date(2025, 11, 30, 12, 0, 0, 0, time.UTC),
			wantError: false,
		},
		{
			name:      "Invalid format",
			input:     "not a time",
			wantError: true,
		},
		{
			name:      "Invalid type",
			input:     12345,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertValue(tt.input, targetType)

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if _, ok := result.(time.Time); !ok {
				t.Errorf("expected time.Time, got %T", result)
			}
		})
	}
}

// TestBindStruct_TimeTimeField tests binding time.Time fields from string values.
func TestBindStruct_TimeTimeField(t *testing.T) {
	type Config struct {
		CreatedAt time.Time
		UpdatedAt time.Time
		Date      time.Time
	}

	data := map[string]mergedEntry{
		"createdat": {value: "2025-11-30T12:00:00Z", sourceName: "file"},
		"updatedat": {value: "2025-12-01T15:30:00+05:30", sourceName: "env"},
		"date":      {value: "2025-11-30", sourceName: "default"},
	}

	var cfg Config
	var provFields []FieldProvenance
	errors := bindStruct(reflect.ValueOf(&cfg), data, &provFields, "", "")

	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	// Verify CreatedAt
	expectedCreatedAt := time.Date(2025, 11, 30, 12, 0, 0, 0, time.UTC)
	if !cfg.CreatedAt.Equal(expectedCreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", cfg.CreatedAt, expectedCreatedAt)
	}

	// Verify UpdatedAt (with timezone)
	if cfg.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should not be zero")
	}

	// Verify Date (date only)
	expectedDate := time.Date(2025, 11, 30, 0, 0, 0, 0, time.UTC)
	if !cfg.Date.Equal(expectedDate) {
		t.Errorf("Date = %v, want %v", cfg.Date, expectedDate)
	}
}

// TestBindStruct_TimeTimeInvalidFormat tests error handling for invalid time formats.
func TestBindStruct_TimeTimeInvalidFormat(t *testing.T) {
	type Config struct {
		Timestamp time.Time
	}

	data := map[string]mergedEntry{
		"timestamp": {value: "not a valid time", sourceName: "file"},
	}

	var cfg Config
	var provFields []FieldProvenance
	errors := bindStruct(reflect.ValueOf(&cfg), data, &provFields, "", "")

	if len(errors) == 0 {
		t.Fatal("expected error for invalid time format")
	}

	if errors[0].Code != ErrCodeInvalidType {
		t.Errorf("expected code %q, got %q", ErrCodeInvalidType, errors[0].Code)
	}
}

// TestBindStruct_TimeDurationAndTimeTime tests both time types together.
func TestBindStruct_TimeDurationAndTimeTime(t *testing.T) {
	type Config struct {
		Timeout   time.Duration
		CreatedAt time.Time
	}

	data := map[string]mergedEntry{
		"timeout":   {value: "30s", sourceName: "file"},
		"createdat": {value: "2025-11-30T12:00:00Z", sourceName: "file"},
	}

	var cfg Config
	var provFields []FieldProvenance
	errors := bindStruct(reflect.ValueOf(&cfg), data, &provFields, "", "")

	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	if cfg.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want %v", cfg.Timeout, 30*time.Second)
	}

	expectedTime := time.Date(2025, 11, 30, 12, 0, 0, 0, time.UTC)
	if !cfg.CreatedAt.Equal(expectedTime) {
		t.Errorf("CreatedAt = %v, want %v", cfg.CreatedAt, expectedTime)
	}
}

// TestConvertValue_TimeDuration ensures time.Duration still works correctly.
func TestConvertValue_TimeDuration(t *testing.T) {
	targetType := reflect.TypeOf(time.Duration(0))

	tests := []struct {
		name      string
		input     string
		want      time.Duration
		wantError bool
	}{
		{"seconds", "30s", 30 * time.Second, false},
		{"minutes", "5m", 5 * time.Minute, false},
		{"hours", "2h", 2 * time.Hour, false},
		{"combined", "1h30m", 90 * time.Minute, false},
		{"milliseconds", "100ms", 100 * time.Millisecond, false},
		{"invalid", "not a duration", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertValue(tt.input, targetType)

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			duration, ok := result.(time.Duration)
			if !ok {
				t.Fatalf("expected time.Duration, got %T", result)
			}

			if duration != tt.want {
				t.Errorf("got %v, want %v", duration, tt.want)
			}
		})
	}
}
