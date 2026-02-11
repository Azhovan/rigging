package rigging

import (
	"bytes"
	"strings"
	"testing"
)

func TestDumpAndSnapshot_ConsistentKeyPathAndInheritedSecretRedaction(t *testing.T) {
	type Auth struct {
		User  string `conf:"name:user"`
		Token string `conf:"name:token,secret"`
	}

	type Service struct {
		Endpoint string `conf:"name:endpoint"`
		Auth     Auth   `conf:"prefix:auth,secret"`
	}

	type Config struct {
		AppName string  `conf:"name:app.name"`
		Service Service `conf:"prefix:service"`
	}

	cfg := &Config{
		AppName: "billing",
		Service: Service{
			Endpoint: "https://api.example.com",
			Auth: Auth{
				User:  "alice",
				Token: "top-secret-token",
			},
		},
	}

	storeProvenance(cfg, &Provenance{
		Fields: []FieldProvenance{
			{FieldPath: "AppName", KeyPath: "app.name", SourceName: "file", Secret: false},
			{FieldPath: "Service.Endpoint", KeyPath: "service.endpoint", SourceName: "file", Secret: false},
			{FieldPath: "Service.Auth.User", KeyPath: "service.auth.user", SourceName: "env", Secret: false},
			{FieldPath: "Service.Auth.Token", KeyPath: "service.auth.token", SourceName: "env", Secret: true},
		},
	})
	defer deleteProvenance(cfg)

	var buf bytes.Buffer
	if err := DumpEffective(&buf, cfg); err != nil {
		t.Fatalf("DumpEffective failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `app.name: "billing"`) {
		t.Fatalf("expected app.name in dump output, got: %s", output)
	}
	if !strings.Contains(output, `service.endpoint: "https://api.example.com"`) {
		t.Fatalf("expected service.endpoint in dump output, got: %s", output)
	}
	if !strings.Contains(output, "service.auth.user: ***redacted***") {
		t.Fatalf("expected inherited redaction for service.auth.user, got: %s", output)
	}
	if !strings.Contains(output, "service.auth.token: ***redacted***") {
		t.Fatalf("expected redaction for service.auth.token, got: %s", output)
	}
	if strings.Contains(output, "alice") || strings.Contains(output, "top-secret-token") {
		t.Fatalf("secret values leaked in dump output: %s", output)
	}

	snapshot, err := CreateSnapshot(cfg)
	if err != nil {
		t.Fatalf("CreateSnapshot failed: %v", err)
	}

	if snapshot.Config["app.name"] != "billing" {
		t.Fatalf("expected app.name=billing in snapshot, got: %v", snapshot.Config["app.name"])
	}
	if snapshot.Config["service.endpoint"] != "https://api.example.com" {
		t.Fatalf("expected service.endpoint in snapshot, got: %v", snapshot.Config["service.endpoint"])
	}
	if snapshot.Config["service.auth.user"] != "***redacted***" {
		t.Fatalf("expected inherited redaction for snapshot service.auth.user, got: %v", snapshot.Config["service.auth.user"])
	}
	if snapshot.Config["service.auth.token"] != "***redacted***" {
		t.Fatalf("expected redaction for snapshot service.auth.token, got: %v", snapshot.Config["service.auth.token"])
	}
}

func TestDumpAndSnapshot_OptionalHandlingSplitRegression(t *testing.T) {
	type Config struct {
		Required string           `conf:"name:required"`
		SetOpt   Optional[string] `conf:"name:set_opt"`
		UnsetOpt Optional[int]    `conf:"name:unset_opt"`
	}

	cfg := &Config{
		Required: "ok",
		SetOpt:   Optional[string]{Value: "set", Set: true},
		UnsetOpt: Optional[int]{Set: false},
	}

	storeProvenance(cfg, &Provenance{
		Fields: []FieldProvenance{
			{FieldPath: "Required", KeyPath: "required", SourceName: "file", Secret: false},
			{FieldPath: "SetOpt", KeyPath: "set_opt", SourceName: "file", Secret: false},
		},
	})
	defer deleteProvenance(cfg)

	var buf bytes.Buffer
	if err := DumpEffective(&buf, cfg); err != nil {
		t.Fatalf("DumpEffective failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `set_opt: "set"`) {
		t.Fatalf("expected set_opt in dump output, got: %s", output)
	}
	if !strings.Contains(output, "unset_opt: <not set>") {
		t.Fatalf("expected unset_opt as <not set> in dump output, got: %s", output)
	}

	snapshot, err := CreateSnapshot(cfg)
	if err != nil {
		t.Fatalf("CreateSnapshot failed: %v", err)
	}

	if snapshot.Config["set_opt"] != "set" {
		t.Fatalf("expected set_opt in snapshot, got: %v", snapshot.Config["set_opt"])
	}
	if _, ok := snapshot.Config["unset_opt"]; ok {
		t.Fatalf("expected unset_opt to be omitted from snapshot, got: %v", snapshot.Config["unset_opt"])
	}
}
