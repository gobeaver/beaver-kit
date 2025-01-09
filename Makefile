# Makefile for beaver-kit

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOFMT=gofmt
GOVET=$(GOCMD) vet
GOMOD=$(GOCMD) mod

# All packages
ALL_PACKAGES=./...

# Default target
.DEFAULT_GOAL := help

.PHONY: fmt
fmt: ## Format all Go files
	@echo "Formatting code..."
	@$(GOFMT) -s -w .
	@$(GOCMD) fmt $(ALL_PACKAGES)

.PHONY: vet
vet: ## Run go vet on all packages
	@echo "Running go vet..."
	@$(GOVET) $(ALL_PACKAGES)

.PHONY: lint
lint: fmt vet ## Run formatting and vetting

.PHONY: test
test: ## Run all tests
	@echo "Running tests..."
	@$(GOTEST) -v $(ALL_PACKAGES)

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@$(GOTEST) -v -coverprofile=coverage.out $(ALL_PACKAGES)
	@$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: test-short
test-short: ## Run short tests only
	@echo "Running short tests..."
	@$(GOTEST) -short $(ALL_PACKAGES)

.PHONY: test-race
test-race: ## Run tests with race detector
	@echo "Running tests with race detector..."
	@$(GOTEST) -race $(ALL_PACKAGES)

.PHONY: test-package
test-package: ## Test specific package (usage: make test-package PKG=./krypto)
	@echo "Testing package $(PKG)..."
	@$(GOTEST) -v $(PKG)

.PHONY: bench
bench: ## Run benchmarks
	@echo "Running benchmarks..."
	@$(GOTEST) -bench=. -benchmem $(ALL_PACKAGES)

.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning..."
	@$(GOCLEAN)
	@rm -f coverage.out coverage.html

.PHONY: deps
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@$(GOMOD) download

.PHONY: tidy
tidy: ## Tidy go.mod
	@echo "Tidying go.mod..."
	@$(GOMOD) tidy

.PHONY: verify
verify: ## Verify dependencies
	@echo "Verifying dependencies..."
	@$(GOMOD) verify

.PHONY: update
update: ## Update dependencies
	@echo "Updating dependencies..."
	@$(GOCMD) get -u $(ALL_PACKAGES)
	@$(GOMOD) tidy

.PHONY: check
check: lint test ## Run lint and tests

.PHONY: ci
ci: deps lint test-race ## Run CI pipeline (deps, lint, test with race)

.PHONY: install-tools
install-tools: ## Install development tools
	@echo "Installing development tools..."
	@$(GOCMD) install golang.org/x/tools/cmd/goimports@latest
	@$(GOCMD) install golang.org/x/lint/golint@latest
	@$(GOCMD) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

.PHONY: imports
imports: ## Fix imports
	@echo "Fixing imports..."
	@goimports -w .

.PHONY: all
all: fmt vet test ## Run fmt, vet, and test

.PHONY: help
help: ## Display this help message
	@echo "beaver-kit Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'