# Quick Start

Complete installation and usage guide for Spec Forge.

## Installation

### Prerequisites

- **Go 1.26** — Required for all frameworks
- **Java + Maven/Gradle** — For Spring Boot projects
- **protoc + plugins** — For gRPC projects

### Install Spec Forge

```bash
go install github.com/spencercjh/spec-forge@latest
```

Verify installation:

```bash
spec-forge --help
```

---

## First Command

### Basic Generation

```bash
# Generate from any supported project (auto-detects framework)
spec-forge generate ./path/to/project
```

Output is written to the project's output directory (default: project root). For gRPC/protoc, the filename follows `<service>.openapi.yaml`.

### With AI Enrichment

AI enrichment runs automatically when `enrich.provider` and `enrich.model` are configured:

```bash
export OPENAI_API_KEY="sk-xxx"
spec-forge generate ./path/to/project --language zh
```

To disable enrichment during generation:

```bash
spec-forge generate ./path/to/project --language zh --skip-enrich
```

---

## Framework-Specific Quick Start

### Spring Boot

```bash
# From project root
spec-forge generate .

# Output: target/openapi.json (Maven) or build/openapi.json (Gradle)
```

Requirements:
- `pom.xml` or `build.gradle` present
- Spring Boot 2.x or 3.x

### Gin

```bash
# From project root
spec-forge generate .

# No annotations needed — AST-based analysis
```

Requirements:
- `go.mod` with `gin` dependency
- Routes registered in `main()` or init functions

### go-zero

```bash
# From project root
spec-forge generate .
```

Requirements:
- `goctl` installed: `go install github.com/zeromicro/go-zero/tools/goctl@latest`
- `.api` definition files

### gRPC (protoc)

```bash
# Install protoc plugin first
go install github.com/sudorandom/protoc-gen-connect-openapi@latest

# Then generate
spec-forge generate .
```

Requirements:
- `protoc` installed
- `.proto` files with service definitions

---

## Configuration

Create `.spec-forge.yaml` in your **current working directory**:

```yaml
# AI Enrichment
enrich:
  enabled: true
  provider: openai          # openai, anthropic, ollama, custom
  model: gpt-4o
  apiKeyEnv: OPENAI_API_KEY
  language: zh              # zh, en, ja, etc.
  timeout: 60s

# Output
output:
  dir: ./openapi
  format: yaml              # yaml or json
```

**Important:** Spec Forge reads `.spec-forge.yaml` from the current working directory, not the project directory. If you run `spec-forge generate ./path/to/project`, ensure the config file is in your current directory, not `./path/to/project`.

**Priority:** CLI flags > Environment variables > Config file > Defaults

---

## Common Commands

### Generate

```bash
# Basic
spec-forge generate ./project

# With enrichment (requires enrich.provider and enrich.model configured)
spec-forge generate ./project --language zh

# Disable enrichment
spec-forge generate ./project --language zh --skip-enrich

# Custom output path (format auto-detected from extension)
spec-forge generate ./project --output ./specs/api.yaml
spec-forge generate ./project --output ./specs/api.json

# Custom output directory (uses default filename)
spec-forge generate ./project --output-dir ./specs

# Verbose logging
spec-forge generate ./project -v
```

### Enrich (Standalone)

```bash
# Enrich existing OpenAPI spec
spec-forge enrich ./openapi.json --provider openai --model gpt-4o

# With custom provider (DeepSeek example)
LLM_API_KEY="sk-xxx" spec-forge enrich ./openapi.json \
    --provider custom \
    --custom-base-url https://api.deepseek.com/v1 \
    --model deepseek-chat

# Disable streaming for faster concurrent processing
spec-forge enrich ./openapi.json --no-stream
```

### Publish

```bash
# Publish to ReadMe.com
README_API_KEY="rdme_xxx" spec-forge publish ./openapi.json --to readme --readme-slug my-api

# Or with full pipeline
README_API_KEY="rdme_xxx" spec-forge generate ./project --publish-target readme
```

---

## Environment Variables

| Variable            | Purpose                      |
|---------------------|------------------------------|
| `OPENAI_API_KEY`    | OpenAI API key               |
| `ANTHROPIC_API_KEY` | Anthropic API key            |
| `LLM_API_KEY`       | Default for custom providers |
| `README_API_KEY`    | ReadMe.com API key           |

---

## Troubleshooting

### "command not found"

Ensure `$GOPATH/bin` is in your `PATH`:

```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

### Spring Boot: "No OpenAPI output found"

1. Check `springdoc-openapi` is in dependencies
2. For multi-module projects, ensure `spring-boot-maven-plugin` has start/stop goals
3. Try running with `-v` for verbose output

### Gin: "No routes found"

- Ensure routes are registered in `main()` or package init
- Check that `gin` import is present in `go.mod`

### gRPC: "protoc-gen-connect-openapi not found"

```bash
go install github.com/sudorandom/protoc-gen-connect-openapi@latest
```

Ensure `$GOPATH/bin` is in your `PATH`.

---

## Next Steps

- [Configuration Guide](./configuration.md) — All options explained
- [AI Enrichment](./ai-enrichment.md) — LLM setup and custom prompts
- [Publishing](./publishing.md) — ReadMe.com and other platforms
- Framework guides: [Spring Boot](./spring-boot.md), [Gin](./gin.md), [go-zero](./go-zero.md), [gRPC](./grpc-protoc.md)
