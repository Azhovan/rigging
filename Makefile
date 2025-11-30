.PHONY: help test test-race test-coverage fmt vet lint build clean all ci install-tools

# Default target
help:
	@echo "Rigging - Development Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make test          Run tests"
	@echo "  make test-race     Run tests with race detection"
	@echo "  make test-coverage Run tests with coverage report"
	@echo "  make fmt           Format code"
	@echo "  make vet           Run go vet"
	@echo "  make lint          Run golangci-lint"
	@echo "  make build         Build all packages"
	@echo "  make ci            Run all CI checks locally"
	@echo "  make install-tools Install development tools"
	@echo "  make clean         Clean build artifacts"
	@echo "  make all           Format, vet, test, and build"

# Run tests
test:
	@echo "Running tests..."
	go test ./...

# Run tests with race detection
test-race:
	@echo "Running tests with race detection..."
	go test -race ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | grep total
	@echo ""
	@echo "To view coverage in browser: go tool cover -html=coverage.out"

# Format code
fmt:
	@echo "Formatting code..."
	gofmt -s -w .
	@echo "✓ Code formatted"

# Check formatting
fmt-check:
	@echo "Checking code formatting..."
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "✗ Code is not formatted. Run 'make fmt'"; \
		gofmt -l .; \
		exit 1; \
	fi
	@echo "✓ Code is properly formatted"

# Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...
	@echo "✓ No vet warnings"

# Run linter
lint:
	@echo "Running golangci-lint..."
	@if [ ! -f $(HOME)/go/bin/golangci-lint ] && ! command -v golangci-lint > /dev/null; then \
		echo "golangci-lint not installed. Run 'make install-tools'"; \
		exit 1; \
	fi
	@if [ -f $(HOME)/go/bin/golangci-lint ]; then \
		$(HOME)/go/bin/golangci-lint run --timeout=5m; \
	else \
		golangci-lint run --timeout=5m; \
	fi
	@echo "✓ Linting passed"

# Build all packages
build:
	@echo "Building packages..."
	go build -v ./...
	@echo ""
	@echo "Building examples..."
	cd examples/basic && go build -v .
	@echo "✓ Build successful"

# Run all CI checks locally
ci: fmt-check vet test-race lint
	@echo ""
	@echo "Running coverage check..."
	@go test -coverprofile=coverage.out ./... > /dev/null 2>&1
	@COVERAGE=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	echo "Coverage: $$COVERAGE%"; \
	if [ $$(echo "$$COVERAGE < 70" | bc -l) -eq 1 ]; then \
		echo "✗ Coverage below 70%"; \
		exit 1; \
	fi
	@echo ""
	@echo "==================================="
	@echo "✓ All CI checks passed!"
	@echo "==================================="

# Install development tools
install-tools:
	@echo "Installing golangci-lint..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "✓ Tools installed"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f coverage.out
	rm -f examples/basic/basic
	go clean ./...
	@echo "✓ Clean complete"

# Format, vet, test, and build
all: fmt vet test build
	@echo ""
	@echo "✓ All checks passed"

# Quick check before commit
pre-commit: fmt vet test
	@echo ""
	@echo "✓ Ready to commit"
