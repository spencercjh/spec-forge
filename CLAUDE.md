# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build, Test, and Lint Commands

```bash
# Build the binary
make build

# Run all tests
make test

# Run end-to-end tests (requires Maven/Gradle)
make test-e2e

# Run a single test
go test -v -run TestFunctionName ./internal/extractor/spring/...

# Run linter (golangci-lint v2)
make lint

# Format code (uses golangci-lint formatters: gofumpt, goimports, gci)
make fmt

# Run all verification (deps, fmt, lint, test)
make verify
```

> **Important:** `make verify` checks for uncommitted changes (git diff) and will fail if there are pending changes.
> Before committing code, use individual commands: `make fmt`, `make lint`, `make test`.
> Only use `make verify` after committing or in CI environments where working tree is clean.

## Architecture Overview

Spec Forge is a CLI tool that generates enriched OpenAPI specifications from various frameworks (Spring Boot, go-zero, gRPC-protoc).

**Core workflow:**
```
Source Code → Detect → Patch → Generate → Validate → Enrich → Publish
```

### Package Structure

```
cmd/                      # Cobra CLI commands
├── root.go               # Entry point, config initialization
├── generate.go           # `spec-forge generate` - full pipeline
├── enrich.go             # `spec-forge enrich` - standalone enrichment
├── publish.go            # `spec-forge publish` - publish to platforms

internal/
├── config/               # Viper configuration loading
├── executor/             # Shell command execution with timeout
├── extractor/            # OpenAPI spec extraction
│   ├── types.go          # GenerateOptions, GenerateResult, etc.
│   ├── spring/           # Spring Boot specific implementation
│   │   ├── detector.go   # Project type detection (Maven/Gradle)
│   │   ├── patcher.go    # springdoc dependency injection
│   │   ├── generator.go  # Maven/Gradle command execution
│   │   ├── maven.go      # POM parsing, spring-boot plugin config
│   │   └── gradle.go     # build.gradle parsing
│   ├── gozero/           # go-zero framework support
│   │   ├── detector.go   # go.mod parsing, dependency detection
│   │   ├── patcher.go    # go-swagger installation check
│   │   └── generator.go  # goctl command execution
│   ├── grpcprotoc/       # gRPC-protoc implementation
│   │   ├── detector.go   # .proto file detection, buf.yaml rejection
│   │   ├── patcher.go    # protoc tools check
│   │   ├── generator.go  # protoc command execution
│   │   └── grpcprotoc.go # Info struct with ProtoFiles, ServiceProtoFiles
│   └── gin/              # Gin framework support (AST-based)
│       ├── detector.go   # go.mod parsing for gin dependency
│       ├── patcher.go    # No-op (no patching needed)
│       ├── generator.go  # AST-based OpenAPI generation
│       ├── ast_parser.go # Go AST parsing for routes
│       ├── handler_analyzer.go # Handler function analysis
│       └── schema_extractor.go # Go struct to OpenAPI schema
├── validator/            # kin-openapi validation
├── enricher/             # LLM-based description enrichment
│   ├── enricher.go       # Main enricher interface
│   ├── config.go         # Enricher configuration
│   ├── prompt/           # Prompt templates
│   ├── processor/        # Batching and concurrent processing
│   ├── specctx/          # Spec context extraction (reserved for future)
│   └── provider/         # LLM providers (factory pattern)
│       └── factory.go    # Use NewProvider(cfg Config) to create providers
└── publisher/            # OpenAPI spec publishing
    ├── local.go          # Local file publishing
    └── readme.go         # ReadMe.com publishing via rdme CLI
```

### Data Flow

```
Spring Project → springdoc plugin → openapi.json → Enricher (LLM) → openapi.yaml
Gin Project → AST Parser → OpenAPI Generator → Enricher (LLM) → openapi.yaml
```

## Critical Constraints

### DCO (Developer Certificate of Origin) - MANDATORY

> **⚠️ IMPORTANT:** All commits MUST include a `Signed-off-by` line. PRs without this will be blocked by DCO check.

**When creating commits, always use the `-s` flag:**

```bash
git commit -s -m "your message"
```

This automatically adds:
```
Signed-off-by: Your Name <your.email@example.com>
```

**If DCO check fails on existing commits, fix with:**
```bash
# Rebase all commits since main and add signoff
git rebase -i main --exec "git commit --amend --no-edit -s"
git push --force-with-lease
```

### springdoc Commands (MUST follow official docs)

> **Reference:** https://springdoc.org/#plugins

The springdoc plugin requires the Spring Boot application to run. Correct commands:

| Build Tool | Command |
|------------|---------|
| **Maven** | `mvn verify` (NOT `springdoc:generate`) |
| **Gradle** | `gradle generateOpenApiDocs` |

**Why:** springdoc needs to access `/v3/api-docs` endpoint at runtime.

### Multi-module Maven Projects

Maven multi-module projects require `spring-boot-maven-plugin` with start/stop goals configured. The patcher automatically detects and adds this configuration in `internal/extractor/spring/maven.go` → `ConfigureSpringBootPlugin`.

### Provider Factory Pattern

When creating LLM providers, always use the factory method:

```go
import "github.com/spencercjh/spec-forge/internal/enricher/provider"

cfg := provider.Config{
    Provider: "openai",  // or "anthropic", "ollama", "custom"
    Model:    "gpt-4o",
    APIKey:   apiKey,
    BaseURL:  baseURL,   // for ollama/custom
}
p, err := provider.NewProvider(cfg)
```

Individual provider constructors (e.g., `newOpenAIProvider`) are internal (unexported). Use `provider.NewProvider()` as the entry point.

### Wrapper Priority

When executing Maven/Gradle commands, prioritize wrappers:

1. `./mvnw` or `./gradlew` in project root
2. Wrapper in parent directory (multi-module projects)
3. System `mvn` or `gradle`

### Configuration Priority

```
flag > env > config file > default
```

Config file: `.spec-forge.yaml` (see `.spec-forge.example.yaml`)

API keys should be provided via environment variables:
- `OPENAI_API_KEY` for OpenAI
- `ANTHROPIC_API_KEY` for Anthropic
- `LLM_API_KEY` (default) for custom providers

## Functional Testing with Example Projects

The `integration-tests/` directory contains example projects for testing:

```
integration-tests/
├── e2e_test.go                        # End-to-end tests
├── README.md                          # Test documentation
├── maven-springboot-openapi-demo/     # Maven-based Spring Boot project
├── gradle-springboot-openapi-demo/    # Gradle-based Spring Boot project
├── maven-multi-module-demo/           # Multi-module Maven project
├── gradle-multi-module-demo/          # Multi-module Gradle project
└── gin-demo/                          # Gin framework project
```

### Gin Framework Development

The Gin extractor is located in `internal/extractor/gin/`.

**Architecture:**
- Uses Go AST (go/ast, go/parser) for static analysis
- No runtime execution required (unlike Spring Boot)
- Patcher is a no-op (no dependencies to install)

**Key Components:**
- `ASTParser` - Parses Go files and extracts routes
- `HandlerAnalyzer` - Analyzes handler functions for params/responses
- `SchemaExtractor` - Converts Go structs to OpenAPI schemas

**Testing:**
```bash
# Run Gin-specific tests
go test -v ./internal/extractor/gin/...

# Run Gin e2e test (requires go.mod with gin dependency)
go test -v -tags=e2e ./integration-tests/... -run TestGinDemo
```

**Example Usage:**
```bash
# Generate OpenAPI spec from a Gin project
spec-forge generate ./integration-tests/gin-demo

# Generate with AI enrichment
LLM_API_KEY="your-key" spec-forge generate ./integration-tests/gin-demo \
    --enrich --language zh
```

### Running E2E Tests

```bash
# Run all e2e tests (requires Maven and/or Gradle)
make test-e2e

# Or directly with go test
go test -v -tags=e2e ./integration-tests/...
```

### Quick Test: Enrichment (M5)

Test schema field and API parameter enrichment with DeepSeek:

```bash
# Build the binary first
go build -o ./build/spec-forge .

# Run enrichment with Chinese descriptions (streaming enabled by default)
LLM_API_KEY="your-deepseek-api-key" ./build/spec-forge enrich \
    ./integration-tests/maven-springboot-openapi-demo/target/openapi.json \
    --provider custom \
    --model deepseek-chat \
    --custom-base-url https://api.deepseek.com/v1 \
    --language zh \
    -v

# Or use --no-stream for faster processing (enables concurrent LLM calls)
LLM_API_KEY="your-deepseek-api-key" ./build/spec-forge enrich \
    ./integration-tests/maven-springboot-openapi-demo/target/openapi.json \
    --provider custom \
    --model deepseek-chat \
    --custom-base-url https://api.deepseek.com/v1 \
    --language zh \
    -v \
    --no-stream
```

> **Note:** Streaming is enabled by default, showing real-time LLM output to stderr.
> With streaming on, batches are processed sequentially for readable output.
> Use `--no-stream` to enable concurrent processing across batches for faster enrichment.

Expected output:
- Schema fields get Chinese descriptions (e.g., `User.id` → "用户唯一标识符")
- API parameters get Chinese descriptions (e.g., `page` → "指定要获取的页码，用于分页查询")
- Batch count shows `batches=2` (API operations + Schema fields)

### Full Pipeline Test

```bash
# Generate OpenAPI spec from source, then enrich
./build/spec-forge generate \
    ./integration-tests/maven-springboot-openapi-demo \
    --enrich \
    --language zh \
    -v
```

### Configuration File

Use `.spec-forge.yaml` to simplify commands:

```yaml
enrich:
  enabled: true
  provider: custom
  model: deepseek-chat
  baseUrl: https://api.deepseek.com/v1
  apiKeyEnv: LLM_API_KEY
  language: zh
  timeout: 60s
```

Then run with config:
```bash
LLM_API_KEY="your-key" ./build/spec-forge enrich ./path/to/openapi.json -v
```

## Testing ReadMe.com Publisher

The ReadMe publisher uploads OpenAPI specs to ReadMe.com using the `rdme` CLI tool.

### Prerequisites

```bash
# Install rdme CLI globally
npm install -g rdme

# Verify installation
rdme --version
```

### ReadMe Configuration

Add to `.spec-forge.yaml`:

```yaml
readme:
  slug: your-api-slug           # Required: API identifier in ReadMe
  branch: stable                # Optional: version/branch (default: stable)
```

### Testing with Standalone Publish Command

**Recommended (Secure):** Use environment variable for API key:
```bash
README_API_KEY="rdme_xxx" ./build/spec-forge publish \
    ./integration-tests/maven-springboot-openapi-demo/target/openapi.json \
    --to readme \
    --readme-slug "your-api-slug"
```

**Security Note:** Do NOT pass API keys via `--readme-api-key` flag in production,
as it may be visible to other users via process listings (`ps`, `/proc/<pid>/cmdline`).
Always supply secrets via environment variables.

### Testing with Full Generate Pipeline

```bash
# Full pipeline: generate → validate → enrich → publish
LLM_API_KEY="your-llm-key" \
  README_API_KEY="rdme_xxx" \
  ./build/spec-forge generate \
  ./integration-tests/maven-springboot-openapi-demo \
  --publish-target readme \
  -v
```

### Overwrite Behavior

By default, `--publish-overwrite` is `false` for safety:
- **Safe mode (default)**: Preserves existing ReadMe spec, fails if already exists
- **CI mode (`--publish-overwrite`)**: Automatically overwrites existing spec

### ReadMe Publisher Security

- API key is passed via `README_API_KEY` environment variable to `rdme` CLI
- Never pass API key via command line arguments (would appear in `ps aux`)
- The publisher filters out any existing `README_API_KEY` entries before injecting the key

## Related Documentation

- Design doc: `docs/plans/2026-03-03-spec-forge-design.md`
- M3 implementation (Generator/Validator): `docs/plans/2026-03-04-spec-forge-m3-impl.md`
- M4 design (Enricher): `docs/plans/2026-03-04-spec-forge-m4-design.md`
- M5 design (Schema/Param Enrichment): `docs/plans/2026-03-05-spec-forge-m5-design.md`
