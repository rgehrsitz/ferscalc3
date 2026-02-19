.PHONY: help build build-gui build-all install test test-unit test-all clean run serve lint fmt vet

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

build-gui: ## Build standalone GUI binaries for Mac (.app) and Windows (no console)
	@echo "Building standalone GUI binaries..."
	@mkdir -p dist
	@# ── Windows: -H windowsgui suppresses the console window ──
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS) -H windowsgui" -o dist/FERSCalc.exe ./cmd/fers-calc
	@# ── macOS Apple Silicon: wrap in .app bundle ──
	GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o dist/FERSCalc-mac-arm64 ./cmd/fers-calc
	@mkdir -p "dist/FERS Calculator.app/Contents/MacOS"
	@cp dist/FERSCalc-mac-arm64 "dist/FERS Calculator.app/Contents/MacOS/FERSCalc"
	@printf '<?xml version="1.0" encoding="UTF-8"?>\n\
	<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">\n\
	<plist version="1.0">\n\
	<dict>\n\
	  <key>CFBundleExecutable</key>\n\
	  <string>FERSCalc</string>\n\
	  <key>CFBundleIdentifier</key>\n\
	  <string>com.ferscalc.app</string>\n\
	  <key>CFBundleName</key>\n\
	  <string>FERS Calculator</string>\n\
	  <key>CFBundleVersion</key>\n\
	  <string>$(VERSION)</string>\n\
	  <key>CFBundlePackageType</key>\n\
	  <string>APPL</string>\n\
	  <key>LSUIElement</key>\n\
	  <true/>\n\
	</dict>\n\
	</plist>' > "dist/FERS Calculator.app/Contents/Info.plist"
	@# ── macOS Intel: wrap in .app bundle ──
	GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/FERSCalc-mac-amd64 ./cmd/fers-calc
	@mkdir -p "dist/FERS Calculator Intel.app/Contents/MacOS"
	@cp dist/FERSCalc-mac-amd64 "dist/FERS Calculator Intel.app/Contents/MacOS/FERSCalc"
	@printf '<?xml version="1.0" encoding="UTF-8"?>\n\
	<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">\n\
	<plist version="1.0">\n\
	<dict>\n\
	  <key>CFBundleExecutable</key>\n\
	  <string>FERSCalc</string>\n\
	  <key>CFBundleIdentifier</key>\n\
	  <string>com.ferscalc.app</string>\n\
	  <key>CFBundleName</key>\n\
	  <string>FERS Calculator</string>\n\
	  <key>CFBundleVersion</key>\n\
	  <string>$(VERSION)</string>\n\
	  <key>CFBundlePackageType</key>\n\
	  <string>APPL</string>\n\
	  <key>LSUIElement</key>\n\
	  <true/>\n\
	</dict>\n\
	</plist>' > "dist/FERS Calculator Intel.app/Contents/Info.plist"
	@echo ""
	@echo "Standalone GUI builds complete:"
	@echo "  macOS (Apple Silicon): dist/FERS Calculator.app"
	@echo "  macOS (Intel):         dist/FERS Calculator Intel.app"
	@echo "  Windows:               dist/FERSCalc.exe"
	@echo ""
	@echo "Double-click to launch — browser opens, no terminal window."

build-all: ## Build CLI binaries for all platforms
	@echo "Building for multiple platforms..."
	@mkdir -p dist
	GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY)-linux-amd64 ./cmd/fers-calc
	GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY)-darwin-amd64 ./cmd/fers-calc
	GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY)-darwin-arm64 ./cmd/fers-calc
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY)-windows-amd64.exe ./cmd/fers-calc
	@echo "Multi-platform build complete — see dist/"

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
	rm -rf dist/
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

serve: build ## Build and start the web UI server
	@echo "Starting $(BINARY) web server..."
	./$(BINARY) serve

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
