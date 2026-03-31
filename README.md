# Spec Forge

[![Go Report Card](https://goreportcard.com/badge/github.com/spencercjh/spec-forge)](https://goreportcard.com/report/github.com/spencercjh/spec-forge)
[![GoDoc](https://godoc.org/github.com/spencercjh/spec-forge?status.svg)](https://godoc.org/github.com/spencercjh/spec-forge)
[![CI](https://github.com/spencercjh/spec-forge/actions/workflows/ci.yml/badge.svg)](https://github.com/spencercjh/spec-forge/actions/workflows/ci.yml)
[![Copilot code review](https://github.com/spencercjh/spec-forge/actions/workflows/copilot-pull-request-reviewer/copilot-pull-request-reviewer/badge.svg)](https://github.com/spencercjh/spec-forge/actions/workflows/copilot-pull-request-reviewer/copilot-pull-request-reviewer)
[![Dependabot Updates](https://github.com/spencercjh/spec-forge/actions/workflows/dependabot/dependabot-updates/badge.svg)](https://github.com/spencercjh/spec-forge/actions/workflows/dependabot/dependabot-updates)

A CLI tool that solves the fragmented, painful world of OpenAPI spec generation — auto-detects your framework, generates
accurate specs from source code, and enriches them with AI.

## Quick Start

```bash
go install github.com/spencercjh/spec-forge@latest
spec-forge generate ./path/to/project
```

See [Quick Start Guide](./docs/quick-start.md) for detailed installation and usage.

---

## Why Spec Forge?

Generating OpenAPI specs from backend code is harder than it should be. Existing tools force you into painful
trade-offs: verbose annotations that break refactoring, unmaintained generators that produce broken output, or manual
specs that drift from your code.

Spec Forge solves this with **zero-annotation AST analysis** for Go web frameworks, **robust generation** where official
tools fall short, and **AI enrichment** that actually understands your code structure.

### The Team Lead Dilemma

> You're a Tech Lead. The PM just told you that your team's APIs need to be integrated by another team next week. They
> need proper API documentation — OpenAPI specs, not a Markdown file.
>
> You ask your backend developers. Blank stares.
>
> "We've never generated API docs before. We just write the code and maybe update the internal wiki."
>
> You check the codebase. Hundreds of endpoints across Spring Boot, Gin, and go-zero services. No annotations, no
> existing specs. Just hand-written Markdown tables that were last updated three months ago.

This is the reality in most engineering teams. **API documentation is an afterthought** because the tooling is too
complex, too fragmented, or requires habits developers never formed.

**Spec Forge changes the equation.** One command generates accurate specs from existing code — no annotations to add, no
complex setup, no "docs sprint." Your team delivers working APIs *and* proper documentation without changing how they
write code.

### Go Web Frameworks: The Annotation Trap

**Gin, Echo, and friends** — the dominant solution is [`swaggo/swag`](https://github.com/swaggo/swag), and it's an
annotation nightmare:

```go
// @Summary      Get user by ID
// @Description  Retrieve user details by their unique identifier
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id   path      int     true  "User ID"
// @Success      200  {object}  User    "User found"
// @Failure      404  {object}  Error   "User not found"
// @Failure      500  {object}  Error   "Internal server error"
// @Router       /users/{id} [get]
func GetUser(c *gin.Context) { ... }
```

**The annotation hell (black hole):**

1. **Not type-checked** — These comments are invisible to the Go compiler. Rename `User` to `UserResponse` and your spec
   breaks silently. Swag only catches this at generation time, if you remember to run it.

2. **Refactoring hell** — Change a field name, add a query param, or split a handler? You're manually updating dozens of
   annotations across multiple files. The annotations become a second, fragile codebase that mirrors your real code.

3. **Visual pollution** — 10 lines of noise for every handler. A medium-sized API accumulates hundreds of lines of
   comments that obscure the actual logic.

4. **Stale by default** — Since there's no compile-time verification, specs drift from reality. Developers forget to
   regenerate, or worse, stop trusting the spec because it's often wrong.

**Spec Forge requires zero annotations.** We parse Go AST to extract routes from `gin.Engine`, analyze
handler signatures, and map request/response structs directly. Rename a type, and the spec updates automatically. No
comments to maintain, no stale references, no visual noise.

### go-zero: The Stagnant Generator

go-zero's official `goctl api swagger` tool works for basic cases, but **development has stagnated**:

- **Stuck on Swagger 2.0** — Generates Swagger 2.0 specs instead of modern OpenAPI 3.x, requiring conversion for
  contemporary tooling
- **Slow issue resolution** — Community-reported bugs and feature requests see limited maintainer response
- **Minor quirks** — Various edge cases (certain field tags, complex nesting) produce imperfect output that needs manual
  cleanup
- **Fragmented ecosystem** — No single maintained alternative; teams patch the tool or maintain forks

**Spec Forge complements go-zero** with a more robust generator that produces clean OpenAPI 3.x specs directly, handling
edge cases the official tool misses.

### LLM Enrichment: Code-First, AI-Enhanced

Raw generated specs are accurate but sparse — they have the structure, not the story. Fields have types but no
descriptions, endpoints have paths but no context.

**The wrong way:** Ask an LLM to write the entire spec from scratch. It hallucinates types, invents fields, and produces
specs that diverge from reality the moment your code changes.

**The Spec Forge way:**

1. **Parse actual code** — AST analysis guarantees the spec structure matches your real types
2. **Generate base spec** — Accurate paths, schemas, parameters, zero hallucination
3. **AI enrichment** — LLM adds human-readable descriptions based on the real structure

```yaml
# Before enrichment
properties:
  user_id:
    type: string
    format: uuid

# After enrichment
properties:
  user_id:
    type: string
    format: uuid
    description: "Unique identifier for the user account, generated as UUID v4"
```

The LLM **never** invents types or changes structure — it only adds descriptions to what we already verified exists.
This keeps specs accurate while making them human-friendly and AI-agent-ready.

### Other Frameworks

**Spring Boot** — springdoc works well but requires manual dependency setup. Spec Forge auto-patches your `pom.xml` or
`build.gradle` and runs the generation pipeline.

**gRPC / Protobuf** — The tooling landscape is a mess: `protoc-gen-openapi` is unmaintained, `buf` lacks official
OpenAPI docs. Spec Forge wraps `protoc-gen-connect-openapi` — a maintained, OpenAPI 3.x-native solution.

**Hertz / Kitex (CloudWeGo)** — Official OpenAPI docs are outdated. Spec Forge will wrap the working tools from
`hertz-contrib/swagger-generate` into a single command (coming soon).

---

## How It Works

```
Source Code → Detect → Patch → Generate → Validate → Enrich → Publish
```

1. **Detect** — Identifies project type (Spring Boot, Gin, go-zero, gRPC)
2. **Patch** — Adds required dependencies/plugins if missing
3. **Generate** — Runs framework-specific generation
4. **Validate** — Validates OpenAPI spec compliance
5. **Enrich** — Uses LLM to add descriptions (optional)
6. **Publish** — Outputs to file or publishes to platforms

---

## Features

- 🔍 **Auto-detection** — Spring Boot, Gin, go-zero, gRPC
- 🔧 **Auto-patching** — Adds dependencies/plugins automatically
- 🤖 **AI Enrichment** — LLM-generated descriptions
- 🌐 **Multi-provider** — OpenAI, Anthropic, Ollama, custom
- ✍️ **Zero annotations for Gin** — Pure AST analysis

---

## Supported Frameworks

| Framework                              | Status     | Guide                |
|----------------------------------------|------------|----------------------|
| [Spring Boot](./docs/spring-boot.md)   | ✅ Ready    | Java/Maven/Gradle    |
| [Gin](./docs/gin.md)                   | ✅ Ready    | Go, zero annotations |
| [go-zero](./docs/go-zero.md)           | ✅ Ready    | Go                   |
| [gRPC (protoc)](./docs/grpc-protoc.md) | ✅ Ready    | Protobuf             |
| [Hertz](./docs/hertz.md)               | 🚧 Planned | Go                   |
| [Kitex](./docs/kitex.md)               | 🚧 Planned | Go                   |

---

## Configuration

Create `.spec-forge.yaml` in your project root:

```yaml
enrich:
  enabled: true
  provider: openai
  model: gpt-4o
  language: zh

output:
  dir: ./openapi
  format: yaml
```

See [.spec-forge.example.yaml](.spec-forge.example.yaml) for all options.

---

## Documentation

- [Quick Start](./docs/quick-start.md) — Installation and first steps
- [Configuration](./docs/configuration.md) — All config options
- [AI Enrichment](./docs/ai-enrichment.md) — LLM providers and prompts
- [Publishing](./docs/publishing.md) — ReadMe.com and more

---

## License

MIT
