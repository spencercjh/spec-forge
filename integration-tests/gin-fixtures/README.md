# Gin Test Fixtures

This directory contains focused Gin framework test fixtures for testing the spec-forge extractor.

## Structure

| Fixture | Purpose | Coverage |
|---------|---------|----------|
| `gin-demo/` | Main showcase | Full-featured demo with all Gin patterns |
| `gin-basic/` | Minimal example | Single file, basic CRUD operations |
| `gin-multifile/` | Multi-file structure | Handlers and models in separate packages |
| `gin-unsupported/` | Edge cases | Wildcard routes, middleware, anonymous handlers |
| `gin-anonymous-handler/` | Anonymous handler focus | Various inline handler patterns |
| `gin-nested-models/` | Complex types | Nested structs, pointers, slices, custom types |

## Usage

Run tests for specific fixtures:

```bash
# All Gin fixtures
go test -v -tags=e2e ./integration-tests/... -run TestGin

# Specific fixture
go test -v -tags=e2e ./integration-tests/... -run TestGinBasic
```

## Fixture Details

### gin-basic
- Single file with basic CRUD
- Simple structs
- Named handlers only

### gin-multifile
- Separated handlers and models
- Multiple route groups
- Cross-package type references

### gin-unsupported
- Wildcard routes (`/*filepath`)
- Middleware functions
- Complex inline handlers
- Routes that may not be fully extractable

### gin-anonymous-handler
- Various anonymous handler patterns
- Inline handlers with different bindings
- Variable-assigned handlers
- Grouped anonymous handlers

### gin-nested-models
- Deeply nested structs (Address in Company)
- Self-referencing types (Employee.Manager)
- Pointer fields
- Custom type aliases
- Generic/paginated responses
