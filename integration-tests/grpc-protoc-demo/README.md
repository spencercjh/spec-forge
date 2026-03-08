# gRPC Protobuf Demo (Native protoc)

This is a demo gRPC project managed by native `protoc` (not buf), used for testing spec-forge's gRPC support.

## Project Structure

```
grpc-protoc-demo/
├── proto/
│   ├── common.proto      # Common messages (pagination, response, etc.)
│   └── user.proto        # User service with CRUD operations
├── third_party/
│   └── google/api/       # Google API annotations (HTTP mappings)
│       ├── annotations.proto
│       └── http.proto
├── gen/                  # Generated code (gitignored)
├── Makefile             # protoc commands
└── go.mod               # Go module
```

## Prerequisites

- `protoc` - Protocol Buffers compiler
- `protoc-gen-go` - Go code generator
- `protoc-gen-connect-openapi` - OpenAPI spec generator

## Installation

```bash
# Install protoc (macOS)
brew install protobuf

# Install Go tools
make install-tools
```

## Usage

### Manual Generation

```bash
# Generate Go code
make proto

# Generate OpenAPI spec
make openapi

# Generate both
make all
```

### With spec-forge

```bash
# spec-forge auto-detects proto files and generates OpenAPI
spec-forge generate ./grpc-protoc-demo

# With additional import paths
spec-forge generate ./grpc-protoc-demo --proto-import-path ./third_party

# Generate with AI enrichment
LLM_API_KEY="your-key" spec-forge generate ./grpc-protoc-demo --enrich --language zh
```

## API Overview

This demo provides user management operations with REST-style HTTP mappings:

| Operation | gRPC Method | HTTP Mapping |
|-----------|-------------|--------------|
| Get User | `GetUser` | GET /v1/users/{id} |
| List Users | `ListUsers` | GET /v1/users |
| Create User | `CreateUser` | POST /v1/users |
| Update Profile | `UpdateProfile` | PUT /v1/users/{id} |
| Upload File | `UploadFile` | POST /v1/users/{user_id}/files |

The HTTP mappings are defined using `google.api.http` annotations in the proto file.

## Differences from buf-managed projects

This project is intentionally **not** using buf to test spec-forge's ability to work with native protoc projects. Key differences:

1. No `buf.yaml` or `buf.gen.yaml`
2. Uses `Makefile` to manage protoc commands
3. Import paths managed manually via `-I` flags
4. Tool versions managed by Go modules

## spec-forge Integration Notes

When spec-forge detects this project:

1. **Detector** looks for `.proto` files without `buf.yaml`
2. **Patcher** checks for `protoc` and `protoc-gen-connect-openapi` availability
3. **Generator** runs protoc with:
   - Auto-detected import paths (`.`, `proto`, `third_party`)
   - `--connect-openapi_opt=features=google.api.http` if HTTP annotations are detected
   - Only service proto files (those with `service` definitions) to avoid duplicate errors
4. Import paths can be extended via `--proto-import-path` flag

### Features

- **HTTP Annotations**: Automatically detected and enabled when `google/api/annotations.proto` is imported
- **Service Detection**: Only processes proto files with `service` definitions
- **Comment Extraction**: Proto comments are automatically extracted to OpenAPI descriptions
- **REST Endpoints**: Generates REST-style paths from `google.api.http` annotations
