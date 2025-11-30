.PHONY: all build test test-coverage lint fmt vet clean help install-tools docs

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOVET=$(GOCMD) vet

# Binary name
BINARY_NAME=fluentsql

# Directories
SRC_DIR=./...
COVERAGE_DIR=./coverage

# Colors for terminal output
GREEN=\033[0;32m
YELLOW=\033[0;33m
RED=\033[0;31m
NC=\033[0m # No Color

all: lint test build

## build: Build the application
build:
	@echo "$(GREEN)Building...$(NC)"
	$(GOBUILD) -v $(SRC_DIR)

## test: Run all tests
test:
	@echo "$(GREEN)Running tests...$(NC)"
	$(GOTEST) -v -race $(SRC_DIR)

## test-short: Run tests without race detector (faster)
test-short:
	@echo "$(GREEN)Running short tests...$(NC)"
	$(GOTEST) -v -short $(SRC_DIR)

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "$(GREEN)Running tests with coverage...$(NC)"
	@mkdir -p $(COVERAGE_DIR)
	$(GOTEST) -v -race -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic $(SRC_DIR)
	$(GOCMD) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "$(GREEN)Coverage report: $(COVERAGE_DIR)/coverage.html$(NC)"

## test-coverage-func: Show coverage by function
test-coverage-func: test-coverage
	$(GOCMD) tool cover -func=$(COVERAGE_DIR)/coverage.out

## bench: Run benchmarks
bench:
	@echo "$(GREEN)Running benchmarks...$(NC)"
	$(GOTEST) -bench=. -benchmem $(SRC_DIR)

## bench-save: Run benchmarks and save results
bench-save:
	@echo "$(GREEN)Running benchmarks and saving results...$(NC)"
	$(GOTEST) -bench=. -benchmem $(SRC_DIR) > benchmark_$(shell date +%Y%m%d_%H%M%S).txt

## lint: Run golangci-lint
lint:
	@echo "$(GREEN)Running linter...$(NC)"
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run $(SRC_DIR); \
	else \
		echo "$(YELLOW)golangci-lint not installed. Run 'make install-tools'$(NC)"; \
	fi

## fmt: Format code
fmt:
	@echo "$(GREEN)Formatting code...$(NC)"
	$(GOFMT) -s -w .

## fmt-check: Check if code is formatted
fmt-check:
	@echo "$(GREEN)Checking code format...$(NC)"
	@test -z "$$($(GOFMT) -l .)" || (echo "$(RED)Code is not formatted. Run 'make fmt'$(NC)" && exit 1)

## vet: Run go vet
vet:
	@echo "$(GREEN)Running go vet...$(NC)"
	$(GOVET) $(SRC_DIR)

## tidy: Tidy and verify module dependencies
tidy:
	@echo "$(GREEN)Tidying modules...$(NC)"
	$(GOMOD) tidy
	$(GOMOD) verify

## deps: Download dependencies
deps:
	@echo "$(GREEN)Downloading dependencies...$(NC)"
	$(GOMOD) download

## update-deps: Update all dependencies
update-deps:
	@echo "$(GREEN)Updating dependencies...$(NC)"
	$(GOGET) -u $(SRC_DIR)
	$(GOMOD) tidy

## clean: Clean build artifacts
clean:
	@echo "$(GREEN)Cleaning...$(NC)"
	$(GOCMD) clean
	rm -rf $(COVERAGE_DIR)
	rm -f $(BINARY_NAME)
	rm -f *.out *.test *.prof

## install-tools: Install development tools
install-tools:
	@echo "$(GREEN)Installing development tools...$(NC)"
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest

## security: Run security scan
security:
	@echo "$(GREEN)Running security scan...$(NC)"
	@if command -v gosec > /dev/null; then \
		gosec -quiet $(SRC_DIR); \
	else \
		echo "$(YELLOW)gosec not installed. Run 'make install-tools'$(NC)"; \
	fi

## docs: Generate documentation
docs:
	@echo "$(GREEN)Generating documentation...$(NC)"
	@echo "View docs at: https://pkg.go.dev/github.com/biyonik/go-fluent-sql"

## ci: Run all CI checks
ci: fmt-check vet lint test-coverage security
	@echo "$(GREEN)All CI checks passed!$(NC)"

## pre-commit: Run before committing
pre-commit: fmt tidy vet lint test
	@echo "$(GREEN)Pre-commit checks passed!$(NC)"

## version: Show Go version
version:
	@$(GOCMD) version

## env: Show Go environment
env:
	@$(GOCMD) env

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

# Default target
.DEFAULT_GOAL := help
