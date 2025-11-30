.PHONY: help test fmt vet lint ci clean

# Default target
help:
	@echo "Rigging - Development Makefile"
	@echo ""
	@echo "  make ci      Run all CI checks (fmt, vet, test, lint)"
	@echo "  make test    Run tests"
	@echo "  make fmt     Format code"
	@echo "  make vet     Run go vet"
	@echo "  make lint    Run golangci-lint"
	@echo "  make clean   Clean artifacts"

# Run tests
test:
	@go test -race ./...

# Format code
fmt:
	@gofmt -s -w .

# Run go vet
vet:
	@go vet ./...

# Run linter
lint:
	@if [ -f $(HOME)/go/bin/golangci-lint ]; then \
		$(HOME)/go/bin/golangci-lint run --timeout=5m; \
	elif command -v golangci-lint > /dev/null; then \
		golangci-lint run --timeout=5m; \
	else \
		echo "golangci-lint not installed. Install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

# Run all CI checks
ci:
	@echo "=== Formatting ==="
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "Code not formatted. Run: make fmt"; \
		exit 1; \
	fi
	@echo "PASS: Formatted"
	@echo ""
	@echo "=== Vetting ==="
	@go vet ./...
	@echo "PASS: Vet"
	@echo ""
	@echo "=== Testing ==="
	@go test -race ./...
	@echo "PASS: Tests"
	@echo ""
	@echo "=== Linting ==="
	@$(MAKE) lint
	@echo "PASS: Lint"
	@echo ""
	@echo "=== Coverage ==="
	@go test -coverprofile=coverage.out ./... > /dev/null 2>&1
	@COVERAGE=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	echo "Coverage: $$COVERAGE%"; \
	if [ $$(echo "$$COVERAGE < 70" | bc -l) -eq 1 ]; then \
		echo "FAIL: Coverage below 70%"; \
		exit 1; \
	fi
	@echo ""
	@echo "============================"
	@echo "All CI checks passed"
	@echo "============================"

# Clean artifacts
clean:
	@rm -f coverage.out
	@go clean ./...
