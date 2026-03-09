# go-zero

Spec Forge supports generating OpenAPI specs from [go-zero](https://go-zero.dev/) projects.

## How It Works

1. **Detection**: Parses `go.mod` to detect go-zero dependency and locate API definition files (`.api` files)
2. **Patching**: Checks for `goctl` installation
   - For goctl < 1.9.2: Patches multi-hyphen prefix values to avoid parsing errors (issue #5425)
   - For goctl >= 1.9.2: Skips patching (issue fixed upstream)
3. **Generation**: Uses `goctl api swagger` command to generate the OpenAPI spec
4. **Post-processing**: Fixes generated Swagger issues
   - Adds missing `items` field for array types (issue #5426)
   - Removes invalid parameters with name "-" (issue #5427)
   - Fixes path parameter mismatches (issue #5428)

## Known Limitations

1. **Nested array types**: `[][]interface{}` generates Swagger without nested `items` field (issue #5426)
   - Workaround: Post-processing adds missing `items: { type: "object" }` for each array layer
2. **Multi-hyphen prefixes**: For goctl < 1.9.2, prefix values like `/api/alert-center` need quotes
   - Workaround: Pre-processing wraps unquoted values in quotes for goctl < 1.9.2
   - For goctl >= 1.9.2: No patching needed (issue fixed upstream)

## Prerequisites

The `goctl` tool must be installed (v1.8.0+, recommended v1.9.2+)

```bash
# Install goctl
go install github.com/zeromicro/go-zero/tools/goctl@latest

# Verify installation
goctl --version
```

## Usage

```bash
# Basic generation
spec-forge generate ./my-go-zero-project

# With AI enrichment
spec-forge generate ./my-go-zero-project --language zh
```

## References

- [go-zero Documentation](https://go-zero.dev/)
- [go-zero Swagger CLI Reference](https://go-zero.dev/reference/cli-guide/swagger)
