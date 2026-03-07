# Gin Framework Support Design

## Overview

This document describes the design for adding Gin framework support to spec-forge. Gin is a popular Go HTTP web framework. Unlike go-zero which uses `.api` files, Gin routes are defined in Go code. This design uses AST parsing to extract route information and generate OpenAPI specifications.

## Goals

- Support static analysis of Gin projects without starting the application
- Extract routes from 4 patterns: direct registration, route groups, middleware chains, and handler references
- Generate accurate OpenAPI 3.0 schemas from Go struct definitions
- Integrate with existing enricher for LLM-enhanced descriptions

## Non-Goals

- Multi-module/complex project structures (out of scope for initial implementation)
- Runtime reflection-based extraction
- Dynamic route patterns (e.g., paths constructed from variables)

## Architecture

### Package Structure

```
internal/extractor/gin/
├── gin.go              # Extractor entry point, implements extractor.Extractor interface
├── info.go             # Gin project info structures
├── detector.go         # Detect Gin projects (go.mod dependency check)
├── patcher.go          # Patch (Gin doesn't modify files, returns empty result)
├── generator.go        # Main OpenAPI generation entry point
├── ast_parser.go       # AST parser - route extraction
├── handler_analyzer.go # Handler function body analysis (param binding, responses)
├── schema_extractor.go # Generate OpenAPI Schema from Go structs
└── *_test.go           # Test files
```

### Data Structures

```go
// info.go
package gin

type Info struct {
    GoVersion      string        // Go version from go.mod
    ModuleName     string        // Module path
    GinVersion     string        // gin dependency version
    HasGin         bool          // Has gin dependency
    MainFiles      []string      // main.go or files with route registration
    HandlerFiles   []string      // Handler file list
    RouterGroups   []RouterGroup // Detected router groups
}

type RouterGroup struct {
    BasePath string
    Routes   []Route
}

type Route struct {
    Method      string   // GET, POST, PUT, DELETE, PATCH
    Path        string   // /users/:id
    FullPath    string   // /api/v1/users/:id (including group prefix)
    HandlerName string   // Function name
    HandlerFile string   // Definition file
    Middlewares []string // Middleware names
}

// handler_analyzer.go
type HandlerInfo struct {
    PathParams  []ParamInfo    // c.Param("id")
    QueryParams []ParamInfo    // c.Query("page")
    HeaderParams []ParamInfo   // c.GetHeader("Authorization")
    BodyType    string         // c.ShouldBindJSON type
    Responses   []ResponseInfo // c.JSON(200, resp)
}

type ParamInfo struct {
    Name     string
    GoType   string
    Required bool
}

type ResponseInfo struct {
    StatusCode int
    GoType    string
}
```

## AST Parsing Flow

```
go.mod Detection
    │
    ▼
Scan *.go Files
    │
    ▼
┌─────────────────────────────────────┐
│ Phase 1: Build Type Map             │
│ - Collect all struct definitions    │
│ - Collect all function definitions  │
│ - Build import alias map            │
└─────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────┐
│ Phase 2: Route Registration Detect  │
│ - r.GET("/path", handler)           │
│ - r.Group("/api") + sub-routes      │
│ - r.Use(middleware)                 │
│ - Track middleware chains           │
└─────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────┐
│ Phase 3: Handler Analysis           │
│ - Locate handler function definition│
│ - Analyze Gin Context calls in body │
│ - Extract parameter bindings        │
│ - Extract response calls            │
└─────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────┐
│ Phase 4: Schema Generation          │
│ - Generate OpenAPI Schema from defs │
│ - Parse json/binding/validate tags  │
│ - Handle nested structs             │
└─────────────────────────────────────┘
    │
    ▼
Generate OpenAPI 3.0 Document
```

## AST Pattern Matching

### Direct Route Registration

```go
// Code: r.GET("/users", handler)
// AST Pattern:
ExprStmt{
    CallExpr{
        SelectorExpr{X: Ident("r"), Sel: Ident("GET|POST|PUT|DELETE|PATCH")},
        Args: [BasicLit("/users"), Ident("handler") or FuncLit]
    }
}
```

### Route Groups

```go
// Code:
//   api := r.Group("/api")
//   api.GET("/users", handler)
// AST Pattern:
AssignStmt{
    Lhs: [Ident("api")],
    Rhs: [CallExpr{SelectorExpr{X: "r", Sel: "Group"}, Args: ["/api"]}]
}
// Track subsequent calls on the "api" variable
```

### Middleware Chains

```go
// Code: r.Use(authMiddleware).GET("/protected", handler)
// AST Pattern:
CallExpr{
    SelectorExpr{
        X: CallExpr{SelectorExpr{X: "r", Sel: "Use"}, Args: [...]},
        Sel: "GET"
    },
    Args: [...]
}
```

### Handler Parameter Binding

```go
// c.ShouldBindJSON(&req) - Request body
// c.Param("id") - Path parameter
// c.Query("page") - Query parameter
// c.GetHeader("Authorization") - Header parameter
// AST Pattern:
CallExpr{
    SelectorExpr{X: Ident("c"), Sel: Ident("ShouldBindJSON|Param|Query|GetHeader")},
    Args: [...]
}
```

### Handler Response

```go
// c.JSON(200, resp)
// c.JSON(http.StatusOK, response)
// AST Pattern:
CallExpr{
    SelectorExpr{X: Ident("c"), Sel: Ident("JSON")},
    Args: [BasicLit(200) or Ident("http.StatusOK"), ...]
}
```

## Schema Generation

### Type Mapping

| Go Type | OpenAPI Type | OpenAPI Format |
|---------|--------------|----------------|
| string | string | - |
| int, int32 | integer | int32 |
| int64 | integer | int64 |
| uint, uint32 | integer | - |
| float32 | number | float |
| float64 | number | double |
| bool | boolean | - |
| []T | array | items: T |
| map[string]T | object | additionalProperties: T |
| time.Time | string | date-time |
| Named struct | $ref | - |

### Tag Processing

| Tag | Processing |
|-----|------------|
| `json:"name"` | Property name |
| `json:"name,omitempty"` | Property name (not in required) |
| `binding:"required"` | Add to required list |
| `validate:"required"` | Add to required list |
| `validate:"min=X"` | Set minimum |
| `validate:"max=X"` | Set maximum |
| `validate:"minLength=X"` | Set minLength |
| `validate:"maxLength=X"` | Set maxLength |
| `validate:"email"` | Set format: email |
| `validate:"url"` | Set format: uri |
| `validate:"uuid"` | Set format: uuid |

### Example

```go
// Go struct
type CreateUserRequest struct {
    Name     string `json:"name" binding:"required" validate:"min=1,max=100"`
    Email    string `json:"email" binding:"required" validate:"email"`
    Age      int    `json:"age,omitempty" validate:"min=0,max=150"`
    Role     string `json:"role" validate:"oneof=admin user guest"`
}

// Generated OpenAPI Schema
{
    "type": "object",
    "required": ["name", "email"],
    "properties": {
        "name": {
            "type": "string",
            "minLength": 1,
            "maxLength": 100
        },
        "email": {
            "type": "string",
            "format": "email"
        },
        "age": {
            "type": "integer",
            "minimum": 0,
            "maximum": 150
        },
        "role": {
            "type": "string",
            "enum": ["admin", "user", "guest"]
        }
    }
}
```

## Responsibility Split: Generator vs Enricher

| Feature | Generator (Static) | Enricher (LLM) |
|---------|-------------------|----------------|
| Path, Method, OperationID | ✅ | - |
| Parameter types and locations | ✅ | - |
| Schema fields, types, required | ✅ | - |
| Operation summary | Function name as placeholder | ✅ Enhanced to natural language |
| Operation description | - | ✅ |
| Parameter description | - | ✅ |
| Schema field description | - | ✅ |

## Edge Cases

| Case | Handling |
|------|----------|
| Dynamic paths (variable concatenation) | Skip, log warning |
| Handler is a method (struct method) | Support, record receiver type |
| Anonymous function handler | Support, OperationID = "anonymous-N" |
| Cross-file type references | Use go/packages for full type loading |
| Unused type definitions | Still generate Schema (may be referenced) |
| c.JSON in conditional branches | Record all possible status codes |
| Third-party middleware | Record middleware name, no deep analysis |
| gin.Context wrapped in struct | Track context field access |
| Multiple router instances | Track each router variable separately |
| Route registration in init() | Support, scan all functions |

## Dependencies

- `go/ast` - AST parsing
- `go/parser` - Parse Go source files
- `go/token` - Token positions
- `golang.org/x/mod/modfile` - Parse go.mod
- `golang.org/x/tools/go/packages` - Load full type information (for schema extraction)
- `github.com/getkin/kin-openapi/openapi3` - OpenAPI document building

## Testing Strategy

1. **Unit Tests**: Each component (detector, parser, analyzer, schema extractor) has isolated tests
2. **Integration Tests**: Use example Gin projects in `integration-tests/gin-springboot-demo/`
3. **Golden Files**: Compare generated OpenAPI specs against expected outputs

## Implementation Phases

1. **Phase 1**: Basic detector and extractor skeleton
2. **Phase 2**: AST parser for route extraction (4 patterns)
3. **Phase 3**: Handler analyzer (param binding, responses)
4. **Phase 4**: Schema extractor from Go structs
5. **Phase 5**: Integration with enricher and CLI
6. **Phase 6**: Testing and documentation

## References

- Gin documentation: https://gin-gonic.com/docs/
- Go AST package: https://pkg.go.dev/go/ast
- OpenAPI 3.0 Specification: https://spec.openapis.org/oas/v3.0.0
