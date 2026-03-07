# gRPC Protobuf Demo (Native protoc)

This is a demo gRPC project managed by native `protoc` (not buf), used for testing spec-forge's gRPC support.

## Project Structure

```
grpc-protoc-demo/
├── proto/
│   ├── common.proto      # Common messages (pagination, response, etc.)
│   └── user.proto        # User service with CRUD operations
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

### With spec-forge (planned)

```bash
# spec-forge will auto-detect proto files and generate OpenAPI
spec-forge generate ./grpc-protoc-demo
```

## API Overview

This demo provides the same functionality as the Spring Boot demo:

| Operation | gRPC Method | HTTP Mapping |
|-----------|-------------|--------------|
| Get User | `GetUser` | GET /api/v1/users/{id} |
| List Users | `ListUsers` | GET /api/v1/users |
| Create User | `CreateUser` | POST /api/v1/users |
| Update Profile | `UpdateProfile` | POST /api/v1/users/{id}/profile |
| Upload File | `UploadFile` | POST /api/v1/users/upload |

## Differences from buf-managed projects

This project is intentionally **not** using buf to test spec-forge's ability to work with native protoc projects. Key differences:

1. No `buf.yaml` or `buf.gen.yaml`
2. Uses `Makefile` to manage protoc commands
3. Import paths managed manually via `-I` flags
4. Tool versions managed by Go modules

## spec-forge Integration Notes

When spec-forge detects this project:

1. **Detector** looks for `.proto` files without `buf.yaml`
2. **Patcher** checks for `protoc` and `protoc-gen-connect-openapi`
3. **Generator** runs protoc with appropriate import paths
4. Import paths can be extended via `--proto-import-path` flag
