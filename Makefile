.PHONY: all build clean deps test test-e2e e2e-deps test-all test-coverage lint fmt install build-linux dev verify help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOLINT=golangci-lint

# Binary names
BINARY_NAME=spec-forge
BINARY_UNIX=$(BINARY_NAME)_unix

# Main package
MAIN_PACKAGE=.

# Build directory
BUILD_DIR=./build

# E2E test tool versions (pinned for supply chain security)
GOCTL_VERSION=v1.9.2
PROTOC_GEN_CONNECT_OPENAPI_VERSION=v0.25.5

# All-in-one: clean, deps, format, lint, test, build
all: clean deps fmt lint test build

build:
	@echo "Building..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	@echo "Clean complete"

deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "Dependencies downloaded"

test:
	@echo "Running tests..."
	$(GOTEST) -race -v -coverprofile=coverage.out ./...
	@echo "Tests complete"

# Install E2E test dependencies
e2e-deps:
	@echo "Installing E2E test dependencies..."
	@# Check for protoc (required for gRPC-protoc tests)
	@if ! command -v protoc >/dev/null 2>&1; then \
		echo "⚠️  WARNING: protoc not found in PATH. gRPC-protoc tests will be skipped."; \
		echo "   Install protoc from: https://grpc.io/docs/protoc-installation/"; \
	fi
	@# Check for java (required for Spring Boot tests)
	@if ! command -v java >/dev/null 2>&1; then \
		echo "⚠️  WARNING: java not found in PATH. Spring Boot tests will be skipped."; \
		echo "   Install Java 25 from: https://adoptium.net/ or your package manager"; \
	fi
	@echo "Installing goctl $(GOCTL_VERSION)..."
	$(GOCMD) install github.com/zeromicro/go-zero/tools/goctl@$(GOCTL_VERSION)
	@echo "Installing protoc-gen-connect-openapi $(PROTOC_GEN_CONNECT_OPENAPI_VERSION)..."
	$(GOCMD) install github.com/sudorandom/protoc-gen-connect-openapi@$(PROTOC_GEN_CONNECT_OPENAPI_VERSION)
	@echo "E2E dependencies installed"

# Run end-to-end tests (tests CLI via Cobra ExecuteContext)
# Use -p 1 to run packages sequentially to avoid port conflicts between Spring Boot apps.
# Without this flag, Go runs test packages in parallel by default, causing:
# - Port 8080 conflicts when multiple Spring Boot apps try to start simultaneously
# - JMX port 9001 conflicts for the spring-boot-maven-plugin stop goal
# - Random "Spring application lifecycle JMX bean not found" failures
test-e2e: e2e-deps
	@echo "Running e2e tests..."
	$(GOTEST) -v -p 1 -tags=e2e ./integration-tests/...
	@echo "E2E tests complete"

# Run all tests (unit + e2e)
test-all: test test-e2e
	@echo "All tests complete"

test-coverage: test
	@echo "Generating coverage report..."
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run golangci-lint (includes linters + formatters check)
lint:
	@echo "Running linter..."
	$(GOLINT) run ./...
	@echo "Lint complete"

# Format code using golangci-lint formatters (gofumpt, goimports, gci)
fmt:
	@echo "Formatting code..."
	$(GOLINT) fmt ./...
	$(GOCMD) fix ./...
	@echo "Format complete"

install:
	@echo "Installing..."
	$(GOCMD) install $(MAIN_PACKAGE)
	@echo "Install complete"

# Cross compilation
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_UNIX) $(MAIN_PACKAGE)

# Development
dev: build
	@echo "Running in development mode..."
	./$(BUILD_DIR)/$(BINARY_NAME) --help

# Verify all checks pass (deps, fmt, lint, test)
# Mirrors CI workflow: checks for uncommitted changes after deps/fmt
verify: deps
	@echo "Checking go.mod/go.sum are tidy..."
	@git diff --exit-code -- go.mod go.sum
	$(GOLINT) fmt ./...
	$(GOCMD) fix ./...
	@echo "Checking formatting produced no changes..."
	@git diff --exit-code
	$(GOLINT) run ./...
	$(GOTEST) -race -v -coverprofile=coverage.out ./...
	@echo "All checks passed!"

help:
	@echo "Available targets:"
	@echo "  all           - Clean, download dependencies, format, lint, test, and build"
	@echo "  build         - Build the binary"
	@echo "  clean         - Remove build artifacts"
	@echo "  deps          - Download and tidy Go module dependencies"
	@echo "  e2e-deps      - Install E2E test dependencies (goctl)"
	@echo "  test          - Run unit tests with coverage"
	@echo "  test-e2e      - Run E2E tests (auto-installs goctl)"
	@echo "  test-all      - Run all tests (unit + e2e)"
	@echo "  test-coverage - Generate HTML coverage report"
	@echo "  lint          - Run golangci-lint (linters + formatters check)"
	@echo "  fmt           - Format code with golangci-lint formatters"
	@echo "  install       - Install binary to GOPATH/bin"
	@echo "  build-linux   - Build for Linux"
	@echo "  dev           - Build and show help"
	@echo "  verify        - Run all checks (deps tidy, fmt, lint, test) with change detection"
	@echo "  help          - Show this help message"
