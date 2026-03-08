# gRPC / Protobuf (Native protoc)

Spec Forge generates OpenAPI specs from `.proto` files using native `protoc` with the `protoc-gen-connect-openapi` plugin.

## Why Not Other Tools?

The gRPC OpenAPI tooling landscape is fragmented:

| Tool | Status | Limitation |
|------|--------|------------|
| `google/gnostic` (`protoc-gen-openapi`) | Inactive/unmaintained | No recent updates |
| `grpc-gateway`'s `protoc-gen-openapiv2` | Active | Only outputs Swagger 2.0, not OpenAPI 3.x |
| `buf` | Active | No official documentation for OpenAPI generation; developers rely on third-party blog posts |

**Spec Forge wraps [`protoc-gen-connect-openapi`](https://github.com/sudorandom/protoc-gen-connect-openapi)** — a maintained, OpenAPI 3.x-native solution.

## How It Works

1. **Detection**: Scans for `.proto` files (excludes `buf`-managed projects)
2. **Patching**: Verifies `protoc` and `protoc-gen-connect-openapi` are installed
3. **Generation**: Runs `protoc` with the connect-openapi plugin

## Prerequisites

- `protoc` installed ([install guide](https://github.com/protocolbuffers/protobuf/releases))
- `protoc-gen-connect-openapi` installed:
  ```bash
  go install github.com/sudorandom/protoc-gen-connect-openapi@latest
  ```

## Usage

```bash
# Generate OpenAPI spec from proto files
spec-forge generate ./my-grpc-project

# With additional import paths
spec-forge generate ./my-grpc-project --proto-import-path ./third_party --proto-import-path ./vendor

# Generate with AI enrichment
LLM_API_KEY="your-key" spec-forge generate ./my-grpc-project --enrich --language zh
```

## Limitations

- **buf-managed projects are not supported** in this mode. Use `buf generate` with the plugin, then use `spec-forge enrich` on the generated OpenAPI spec.

## References

- [protoc-gen-connect-openapi](https://github.com/sudorandom/protoc-gen-connect-openapi)
- [Protocol Buffers](https://protobuf.dev/)
