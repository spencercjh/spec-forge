# E2E Tests

This directory contains end-to-end tests for the spec-forge CLI tool.

## Test Categories

| Category | Build Tag | Description |
|----------|-----------|-------------|
| **E2E Tests** | `e2e` | Test complete CLI workflow via Cobra ExecuteContext |
| **Unit Tests** | (none) | Fast unit tests without external dependencies |

## Test Projects

| Project | Build Tool | Description |
|---------|------------|-------------|
| `maven-springboot-openapi-demo` | Maven | Single module Spring Boot project |
| `gradle-springboot-openapi-demo` | Gradle | Single module Spring Boot project |
| `maven-multi-module-demo` | Maven | Multi-module Spring Boot project |
| `gradle-multi-module-demo` | Gradle | Multi-module Gradle project |
| `gozero-demo` | Go Modules | go-zero framework project |
| `gin-demo` | Go Modules | Gin framework project |
| `grpc-protoc-demo` | protoc | gRPC project with native protoc (not buf) |

## Prerequisites

E2E tests require external tools to be installed:

- **Java 25** - Required for Spring Boot projects (Maven/Gradle wrappers are included in demo projects)
- **goctl** - Required for go-zero projects. Install with:
  ```bash
  go install github.com/zeromicro/go-zero/tools/goctl@latest
  ```
- **protoc** & **protoc-gen-connect-openapi** - Required for gRPC projects:
  ```bash
  # Install protoc (macOS)
  brew install protobuf

  # Install protoc-gen-connect-openapi
  go install github.com/sudorandom/protoc-gen-connect-openapi@latest
  ```

## Running Tests

### Unit Tests

```bash
go test ./...
```

### End-to-End Tests

E2E tests use Cobra's `ExecuteContext` to test the complete CLI workflow:

```bash
# Run all E2E tests
go test -tags=e2e ./integration-tests/...

# Run specific E2E test
go test -tags=e2e ./integration-tests/... -run TestE2E_MavenSpringBoot_Generate

# Run with verbose output
go test -v -tags=e2e ./integration-tests/...
```

### Skipping E2E Tests in CI

E2E tests are gated by the `e2e` build tag. They won't run by default:

```bash
# This only runs unit tests
go test ./...

# Run E2E tests
go test -tags=e2e ./...
```

## Test Coverage

### Framework E2E Tests

| Test File | Tests | Description |
|-----------|-------|-------------|
| `e2e_spring_maven_test.go` | `TestE2E_MavenSpringBoot_Generate` | Tests generate flow for Maven project with full spec validation |
| `e2e_spring_maven_test.go` | `TestE2E_MavenSpringBoot_GenerateEnrich` | Tests complete generate flow |
| `e2e_spring_gradle_test.go` | `TestE2E_GradleSpringBoot_Generate` | Tests generate flow for Gradle project with full spec validation |
| `e2e_gozero_test.go` | `TestE2E_GoZero_Generate` | Tests generate flow for go-zero project with full spec validation |
| `e2e_gin_test.go` | `TestE2E_GinDemo_Generate` | Tests generate flow for Gin project with full spec validation |
| `e2e_grpc_protoc_test.go` | `TestE2E_GrpcProtoc_Generate` | Tests generate flow for gRPC-protoc project with full spec validation |

### Multi-Module E2E Tests

| Test File | Tests | Description |
|-----------|-------|-------------|
| `e2e_multi_module_test.go` | `TestE2E_MavenMultiModule_Generate` | Tests generate flow for Maven multi-module project |
| `e2e_multi_module_test.go` | `TestE2E_GradleMultiModule_Generate` | Tests generate flow for Gradle multi-module project |

### CLI E2E Tests

| Test File | Tests | Description |
|-----------|-------|-------------|
| `e2e_generate_test.go` | `TestE2E_Generate_Help` | Tests generate command help |
| `e2e_generate_test.go` | `TestE2E_Generate_MavenSpringBoot` | Tests full CLI generate flow |
| `e2e_generate_test.go` | `TestE2E_Generate_Gin` | Tests full CLI generate flow for Gin |
| `e2e_enrich_test.go` | `TestE2E_Enrich_Help` | Tests enrich command help |
| `e2e_publish_test.go` | `TestE2E_Publish_Help` | Tests publish command help |
| `e2e_publish_test.go` | `TestE2E_Publish_MissingAPIKey` | Tests error handling for missing API key |
| `e2e_publish_test.go` | `TestE2E_Publish_MissingTarget` | Tests error handling for missing --to flag |
| `e2e_publish_test.go` | `TestE2E_Publish_InvalidTarget` | Tests error handling for invalid target |
| `e2e_publish_test.go` | `TestE2E_Publish_NonExistentSpec` | Tests error handling for non-existent spec file |
| `e2e_publish_test.go` | `TestE2E_Publish_InvalidSpec` | Tests error handling for invalid spec format |

### Other Test Files

| Test File | Type | Description |
|-----------|------|-------------|
| `error_test.go` | E2E | Tests error handling for missing commands |

### Spec Assertion Helpers

The `spec_assertions_test.go` file provides comprehensive OpenAPI spec validation utilities:

```go
// Example: Full validation with all checks
validator := NewSpecValidator(t, specFile)
validator.FullValidation(ValidationConfig{
    ExpectedPaths: []string{"/api/v1/users", "/api/v1/users/{id}"},
    Operations: []OperationConfig{
        {
            Path:                    "/api/v1/users",
            Method:                  "get",
            WantOperationID:         true,
            WantSummary:             true,
            ExpectedResponseCodes:   []string{"200", "401"},
            ValidateResponseContent: "application/json",
            ExpectedParams:          []string{"page", "size"},
        },
    },
    ExpectedSchemas: []string{"User", "CreateUserRequest"},
})
```

**Available Validations:**
- `ValidateOpenAPIVersion()` - Validates spec version (3.x)
- `ValidateInfo()` - Validates title and version fields
- `ValidatePaths()` - Validates expected paths exist
- `ValidateOperationFields()` - Validates operationId, summary
- `ValidateResponseCodes()` - Validates HTTP response codes (200, 404, etc.)
- `ValidateResponseContent()` - Validates content-type in responses
- `ValidateRequestBody()` - Validates request body content
- `ValidateParameters()` - Validates path/query parameters
- `ValidateSchemas()` - Validates component schemas

### gRPC-protoc Test Details

The `TestE2E_GrpcProtoc_Generate` test verifies:
- Project detection (grpc-protoc framework)
- OpenAPI spec generation with REST endpoints
- Expected REST paths from `google.api.http` annotations
