.PHONY: all build clean deps test test-e2e test-all test-coverage lint fmt install build-linux dev verify help

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
	$(GOTEST) -race -v -coverprofile=coverage.out ./...
	@echo "Tests complete"

# Run end-to-end tests (tests CLI via Cobra ExecuteContext)
# Automatically detects HTTP_PROXY/HTTPS_PROXY (and lowercase variants) and configures Java tools
test-e2e:
	@echo "Running e2e tests..."
	@PROXY_URL=""; \
	if [ -n "$$HTTP_PROXY" ] || [ -n "$$http_proxy" ]; then \
		PROXY_URL=$${HTTP_PROXY:-$$http_proxy}; \
	elif [ -n "$$HTTPS_PROXY" ] || [ -n "$$https_proxy" ]; then \
		PROXY_URL=$${HTTPS_PROXY:-$$https_proxy}; \
	fi; \
	if [ -n "$$PROXY_URL" ]; then \
		PROXY_NO_SCHEME=$${PROXY_URL#*://}; \
		PROXY_AUTH_AND_HOST=$${PROXY_NO_SCHEME%%/*}; \
		PROXY_HOSTPORT=$${PROXY_AUTH_AND_HOST#*@}; \
		if printf '%s' "$$PROXY_HOSTPORT" | grep -q '^\['; then \
			PROXY_HOST=$${PROXY_HOSTPORT%%]*}; \
			PROXY_HOST=$${PROXY_HOST#\[}; \
			PROXY_PORT=$${PROXY_HOSTPORT#*]:}; \
		else \
			PROXY_HOST=$${PROXY_HOSTPORT%%:*}; \
			if [ "$$PROXY_HOST" = "$$PROXY_HOSTPORT" ]; then \
				PROXY_PORT=""; \
			else \
				PROXY_PORT=$${PROXY_HOSTPORT##*:}; \
			fi; \
		fi; \
		if [ -n "$$PROXY_HOST" ]; then \
			PROXY_JAVA_OPTS="-Dhttp.proxyHost=$$PROXY_HOST -Dhttps.proxyHost=$$PROXY_HOST"; \
			if [ -n "$$PROXY_PORT" ]; then \
				PROXY_JAVA_OPTS="$$PROXY_JAVA_OPTS -Dhttp.proxyPort=$$PROXY_PORT -Dhttps.proxyPort=$$PROXY_PORT"; \
			fi; \
			if [ -n "$$JAVA_TOOL_OPTIONS" ]; then \
				JAVA_TOOL_OPTIONS="$$PROXY_JAVA_OPTS $$JAVA_TOOL_OPTIONS"; \
			else \
				JAVA_TOOL_OPTIONS="$$PROXY_JAVA_OPTS"; \
			fi; \
			echo "Detected proxy: $$PROXY_HOST:$${PROXY_PORT:-default}"; \
			export JAVA_TOOL_OPTIONS; \
		fi; \
	fi; \
	$(GOTEST) -v -tags=e2e ./integration-tests/...
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
	@echo "  deps          - Download and tidy dependencies"
	@echo "  test          - Run unit tests with coverage"
	@echo "  test-e2e      - Run E2E tests (requires Maven/Gradle/goctl/protoc)"
	@echo "  test-all      - Run all tests (unit + e2e)"
	@echo "  test-coverage - Generate HTML coverage report"
	@echo "  lint          - Run golangci-lint (linters + formatters check)"
	@echo "  fmt           - Format code with golangci-lint formatters"
	@echo "  install       - Install binary to GOPATH/bin"
	@echo "  build-linux   - Build for Linux"
	@echo "  dev           - Build and show help"
	@echo "  verify        - Run all checks (deps tidy, fmt, lint, test) with change detection"
	@echo "  help          - Show this help message"
