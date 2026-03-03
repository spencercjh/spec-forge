.PHONY: all build clean test lint fmt install build-linux dev verify help

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
	$(GOTEST) -v -coverprofile=coverage.out ./...
	@echo "Tests complete"

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
verify: deps fmt lint test
	@echo "All checks passed!"

help:
	@echo "Available targets:"
	@echo "  all           - Clean, download dependencies, format, lint, test, and build"
	@echo "  build         - Build the binary"
	@echo "  clean         - Remove build artifacts"
	@echo "  deps          - Download and tidy dependencies"
	@echo "  test          - Run tests with coverage"
	@echo "  test-coverage - Generate HTML coverage report"
	@echo "  lint          - Run golangci-lint (linters + formatters check)"
	@echo "  fmt           - Format code with golangci-lint formatters"
	@echo "  install       - Install binary to GOPATH/bin"
	@echo "  build-linux   - Build for Linux"
	@echo "  dev           - Build and show help"
	@echo "  verify        - Run all checks (deps, fmt, lint, test)"
	@echo "  help          - Show this help message"
