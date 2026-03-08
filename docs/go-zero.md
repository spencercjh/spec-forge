# go-zero

Spec Forge supports generating OpenAPI specs from [go-zero](https://go-zero.dev/) projects.

## How It Works

1. **Detection**: Parses `go.mod` to detect go-zero dependency and locate API definition files (`.api` files)
2. **Patching**: Checks for `goctl` installation
3. **Generation**: Uses `goctl api swagger` command to generate the OpenAPI spec

## Prerequisites

The `goctl` tool must be installed:

```bash
go install github.com/zeromicro/go-zero/tools/goctl@latest
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
