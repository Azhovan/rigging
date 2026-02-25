# Examples Index

Use this page to quickly find the right example format.

## Runnable Examples

- [basic](basic/) - End-to-end app config loading with file + env layering, validation, provenance, and redaction.
- [transformer](transformer/) - Typed `WithTransformer(...)` canonicalization before tag validation.

Run it from the repository root:

```bash
go run ./examples/basic
go run ./examples/transformer
```

## GoDoc Examples (API-Focused)

Rigging also includes executable examples in [`example_test.go`](../example_test.go).
These are surfaced on pkg.go.dev and can be run locally with:

```bash
go test -run '^Example' ./...
```

Useful starting points:

- `Example` - Basic multi-source load.
- `ExampleLoader_Load` - Typed load with validation.
- `ExampleLoader_WithValidator` - Custom validation.
- `ExampleDumpEffective` - Redacted effective config output.
- `ExampleDumpEffective_withSources` - Output with source attribution.
- `ExampleDumpEffective_asJSON` - JSON output mode.
- `ExampleGetProvenance` - Field-level provenance inspection.
- `ExampleLoader_Strict` - Strict mode behavior.
- `ExampleValidationError` - Handling aggregated validation errors.
- `ExampleSource` - Implementing a custom source.
- `ExampleLoader_Watch` - Watch API behavior.
- `ExampleCreateSnapshot` - Snapshot creation.
- `ExampleWriteSnapshot` - Snapshot persistence.
- `Example_snapshotRoundTrip` - End-to-end snapshot lifecycle.

## Picking the Right Path

- Need a quick end-to-end run: use [basic](basic/).
- Need API behavior for one method: use GoDoc examples in `example_test.go`.
- Need guided learning: start at [docs/quick-start.md](../docs/quick-start.md), then return here for targeted examples.
