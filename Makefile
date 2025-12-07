.PHONY: help build install test test-unit test-all clean run lint fmt vet

# Default target
.DEFAULT_GOAL := help

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS := -X 'github.com/rpgo/retirement-calculator/internal/cli.Version=$(VERSION)' \
           -X 'github.com/rpgo/retirement-calculator/internal/cli.BuildDate=$(BUILD_DATE)' \
           -X 'github.com/rpgo/retirement-calculator/internal/cli.GitCommit=$(GIT_COMMIT)'

# Binary name
BINARY := fers-calc

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the CLI binary
	@echo "Building $(BINARY) version $(VERSION)..."
	go build -ldflags="$(LDFLAGS)" -o $(BINARY) ./cmd/fers-calc
	@echo "Build complete: ./$(BINARY)"

build-all: ## Build binaries for multiple platforms
	@echo "Building for multiple platforms..."
	GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BINARY)-linux-amd64 ./cmd/fers-calc
	GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BINARY)-darwin-amd64 ./cmd/fers-calc
	GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(BINARY)-darwin-arm64 ./cmd/fers-calc
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BINARY)-windows-amd64.exe ./cmd/fers-calc
	@echo "Multi-platform build complete"

install: build ## Install the binary to $GOPATH/bin
	@echo "Installing $(BINARY)..."
	go install -ldflags="$(LDFLAGS)" ./cmd/fers-calc
	@echo "Installed to $(shell go env GOPATH)/bin/$(BINARY)"

test: ## Run all tests
	@echo "Running tests..."
	go test -v ./...

test-unit: ## Run tests including files behind the 'unit' build tag (short)
	@echo "Running unit-tagged short tests..."
	go test -v -short -tags=unit ./...

test-all: ## Run all tests including 'unit' build tag (may be slow)
	@echo "Running full test suite with unit-tagged tests..."
	go test -v -tags=unit ./...

test-short: ## Run tests without long-running tests
	@echo "Running short tests..."
	go test -v -short ./...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

clean: ## Remove build artifacts
	@echo "Cleaning..."
	rm -f $(BINARY) $(BINARY)-*
	rm -f coverage.out coverage.html
	rm -f retirement_report_*.* montecarlo_report_*.*
	@echo "Clean complete"

clean-local: ## Remove local generated reports, logs, and stray binaries
	@echo "Cleaning local generated files..."
	rm -f $(BINARY)
	rm -f *.html *.csv *.json *.xlsx
	rm -f *.log *.err
	@echo "Local cleanup complete"

run: build ## Build and run with example config
	@echo "Running $(BINARY)..."
	./$(BINARY) calculate example_config.yaml

fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

lint: ## Run golint (requires golint to be installed)
	@echo "Running golint..."
	@command -v golint >/dev/null 2>&1 || { echo "golint not installed. Run: go install golang.org/x/lint/golint@latest"; exit 1; }
	golint ./...

tidy: ## Tidy go.mod
	@echo "Tidying go.mod..."
	go mod tidy

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download

verify: fmt vet test ## Format, vet, and test

.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t fers-calc:$(VERSION) .

.PHONY: docker-run
docker-run: ## Run in Docker container
	@echo "Running in Docker..."
	docker run --rm -v $(PWD):/data fers-calc:$(VERSION) calculate /data/example_config.yaml
