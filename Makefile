.PHONY: help test fmt vet lint ci clean bench bench-compare

BASE_REF ?= main
BENCH ?= BenchmarkCreateSnapshot_SmallConfig$$
BENCH_COUNT ?= 5
BENCH_PKG ?= .

# Default target
help:
	@echo "Rigging - Development Makefile"
	@echo ""
	@echo "  make ci      Run all CI checks (fmt, vet, test, lint)"
	@echo "  make test    Run tests"
	@echo "  make fmt     Format code"
	@echo "  make vet     Run go vet"
	@echo "  make lint    Run golangci-lint"
	@echo "  make bench   Run configured benchmark on current tree"
	@echo "  make bench-compare Compare benchmark vs base ref using benchstat"
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

# Run benchmark on current tree
bench:
	@go test $(BENCH_PKG) -run '^$$' -bench "$(BENCH)" -benchmem -count $(BENCH_COUNT)

# Compare benchmark against base ref (default: main)
bench-compare:
	@./scripts/bench_compare.sh "$(BASE_REF)" "$(BENCH)" "$(BENCH_COUNT)" "$(BENCH_PKG)"

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
