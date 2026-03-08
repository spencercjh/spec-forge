# Gin Framework

Spec Forge uses **AST-based static analysis** to extract OpenAPI specs from Gin projects — **zero annotations required**.

## Why Not `swaggo/swag`?

The dominant solution ([`swaggo/swag`](https://github.com/swaggo/swag)) requires hundreds of verbose annotations:

```go
// @Summary  Get user by ID
// @Param    id   path  int  true  "User ID"
// @Success  200  {object}  User
// @Router   /users/{id} [get]
func GetUser(c *gin.Context) { ... }
```

These annotations are **not validated by the Go compiler** — typos and stale references only surface at generation time. Renaming a type means manually updating annotations everywhere.

**Spec Forge requires zero annotations.** It uses Go AST analysis to read your routes, handler signatures, and struct definitions directly from source.

## How It Works

1. **Detection**: Parses `go.mod` to detect Gin dependency
2. **Patching**: No-op (no dependencies to install)
3. **Generation**: Uses Go AST parser to extract routes, handlers, and types

## Supported Patterns

- Direct route registration: `r.GET("/users", handler)`
- Route groups: `api := r.Group("/api")`
- Middleware chains: `r.Use(auth).GET("/protected", handler)`
- Parameter binding: `c.Param()`, `c.Query()`, `c.ShouldBindJSON()`
- Response types: extracted from `c.JSON()` calls with type inference

## Usage

```bash
# Basic generation
cd my-gin-project
spec-forge generate . -o ./openapi

# Generate with AI enrichment
LLM_API_KEY="sk-xxx" spec-forge generate . \
    --enrich \
    --provider custom \
    --model deepseek-chat \
    --language zh

# Verbose mode to see extraction details
spec-forge generate . -v
```

## References

- [Gin Web Framework](https://gin-gonic.com/)
