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

## Architecture Overview

Spec Forge is a CLI tool that generates enriched OpenAPI specifications from Spring Boot projects.

**Core workflow:**
```
Source Code → Detect → Patch → Generate → Validate → Enrich → Restore
```

### Package Structure

```
cmd/                      # Cobra CLI commands
├── root.go               # Entry point, config initialization
├── generate.go           # `spec-forge generate` - full pipeline
├── enrich.go             # `spec-forge enrich` - standalone enrichment
├── spring.go             # `spec-forge spring` - patch/detect subcommands

internal/
├── config/               # Viper configuration loading
├── executor/             # Shell command execution with timeout
├── extractor/            # OpenAPI spec extraction
│   ├── types.go          # GenerateOptions, GenerateResult, etc.
│   └── spring/           # Spring Boot specific implementation
│       ├── detector.go   # Project type detection (Maven/Gradle)
│       ├── patcher.go    # springdoc dependency injection
│       ├── generator.go  # Maven/Gradle command execution
│       ├── maven.go      # POM parsing, spring-boot plugin config
│       └── gradle.go     # build.gradle parsing
├── validator/            # kin-openapi validation
└── enricher/             # LLM-based description enrichment
    ├── enricher.go       # Main enricher interface
    ├── config.go         # Enricher configuration
    ├── prompt/           # Prompt templates
    ├── processor/        # Batching and concurrent processing
    └── provider/         # LLM providers (factory pattern)
        └── factory.go    # Use NewProvider(cfg Config) to create providers
```

### Data Flow

```
Spring Project → springdoc plugin → openapi.json → Enricher (LLM) → openapi.yaml
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

The `integration-tests/` directory contains example Spring Boot projects for testing:

```
integration-tests/
├── e2e_test.go                        # End-to-end tests
├── README.md                          # Test documentation
├── maven-springboot-openapi-demo/     # Maven-based Spring Boot project
├── gradle-springboot-openapi-demo/    # Gradle-based Spring Boot project
├── maven-multi-module-demo/           # Multi-module Maven project
└── gradle-multi-module-demo/          # Multi-module Gradle project
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

# Run enrichment with Chinese descriptions
LLM_API_KEY="your-deepseek-api-key" ./build/spec-forge enrich \
    ./integration-tests/maven-springboot-openapi-demo/target/openapi.json \
    --provider custom \
    --model deepseek-chat \
    --custom-base-url https://api.deepseek.com/v1 \
    --language zh \
    -v
```

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

## Related Documentation

- Design doc: `docs/plans/2026-03-03-spec-forge-design.md`
- M3 implementation (Generator/Validator): `docs/plans/2026-03-04-spec-forge-m3-impl.md`
- M4 design (Enricher): `docs/plans/2026-03-04-spec-forge-m4-design.md`
- M5 design (Schema/Param Enrichment): `docs/plans/2026-03-05-spec-forge-m5-design.md`
