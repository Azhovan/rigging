# Contributing to Rigging

Thanks for considering contributing to Rigging! This is a focused Go library for type-safe configuration management.

## Quick Start

**Prerequisites**: Go 1.21+

```bash
# Fork and clone
git clone https://github.com/Azhovan/rigging.git
cd rigging

# Run CI checks (what GitHub Actions runs)
make ci

# Or run individual checks
make test
make test-race
make fmt
```

## What We Value

**Simplicity**: Rigging is intentionally minimal. New features must justify their complexity.

**Zero Dependencies**: The core package has no external dependencies. Keep it that way.

**Type Safety**: We leverage Go's type system. Avoid reflection where possible.

**Clear Documentation**: Code should be self-explanatory. Comments explain *why*, not *what*.

## Making Changes

### Branch Naming
- `fix/validation-panic` - Bug fixes
- `feat/etcd-source` - New features
- `docs/clarify-tags` - Documentation
- `perf/reduce-allocs` - Performance

### Testing Requirements
- All exported functions must have tests
- Aim for >80% coverage
- Use table-driven tests
- Test error paths, not just happy paths

```go
func TestLoad_WithMissingRequiredField_ReturnsError(t *testing.T) {
    // Clear, specific test names
}
```

### Code Style
- Run `gofmt`, `go vet` before committing
- Keep functions under 50 lines when possible
- Avoid complex nested logic
- Use early returns to reduce nesting

### Documentation
- **Package docs**: Minimal. See `doc.go` for our style.
- **Exported types**: One-line comment stating purpose.
- **Complex logic**: Brief inline comments explaining *why*.

Example:
```go
// Loader loads configuration from multiple sources.
// Sources are processed in order (later override earlier).
type Loader[T any] struct { ... }
```

## What to Contribute

### High Priority
- **Bug fixes**: Always welcome
- **Test improvements**: Increase coverage, add edge cases
- **Performance**: Reduce allocations, optimize hot paths
- **Source implementations**: etcd, Consul, Vault (separate packages)

### Medium Priority
- **Documentation**: Fix unclear docs, add examples
- **Validation**: New validators that are broadly useful
- **File watching**: fsnotify integration for sourcefile

### Low Priority / Need Discussion
- **Breaking changes**: Must have strong justification
- **New core features**: May increase complexity too much
- **Dependencies**: Core must stay dependency-free

## Pull Requests

1. Open an issue first for significant changes
2. Keep PRs small and focused
3. Include tests
4. Update relevant docs
5. Ensure CI passes

**PR Title Format**:
```
fix: prevent panic in strict mode with nested structs
feat: add etcd source in sourceecrd package
docs: clarify oneof tag behavior
```

**PR Description**:
- What problem does this solve?
- How did you test it?
- Any breaking changes?

## Project Philosophy

Rigging is designed for:
- **Production use**: Reliability over features
- **Type safety**: Compile-time guarantees where possible
- **Explicit configuration**: No magic, no surprises
- **Minimal API surface**: Easy to understand completely

We reject:
- **Magic**: Auto-discovery, reflection-heavy solutions
- **Bloat**: Features that serve narrow use cases
- **Complexity**: Clever code that's hard to maintain

## Development Commands

**Using Make (recommended):**
```bash
make ci              # Run all CI checks locally
make test            # Run tests
make test-coverage   # Run tests with coverage report
make fmt             # Format code
make vet             # Run go vet
make lint            # Run golangci-lint
make build           # Build packages
make all             # Format, vet, test, and build
make help            # Show all commands
```

**Manual commands:**
```bash
go test ./...                           # Tests
go test -race ./...                     # Race detection
go test -coverprofile=coverage.out ./... # Coverage
go tool cover -html=coverage.out        # View coverage
```

## Getting Help

- **Questions**: Open a discussion
- **Bugs**: Open an issue with minimal reproduction
- **Security**: Email maintainer directly (see README)

## License

By contributing, you agree your code is licensed under MIT.

---

Keep it simple. Make it fast. Test everything.
