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

- **Java 25** - Required for Spring Boot projects. Install from [Adoptium](https://adoptium.net/) or use your package manager.
- **Maven** / **Gradle** - Demo projects include `mvnw`/`gradlew` wrappers, but you need a working Java environment. The wrappers will download dependencies automatically.
- **goctl** - Required for go-zero projects. Install with:
  ```bash
  go install github.com/zeromicro/go-zero/tools/goctl@v1.9.2
  ```

  **Note:** Version is pinned to v1.9.2 for supply chain security. Using `@latest` may cause golden test failures due to version field changes.
- **protoc** & **protoc-gen-connect-openapi** - Required for gRPC projects:
  ```bash
  # Install protoc (macOS)
  brew install protobuf

  # Install protoc-gen-connect-openapi
  go install github.com/sudorandom/protoc-gen-connect-openapi@latest
  ```

### Quick Setup (Ubuntu/Debian)

```bash
# Install Java 25
sudo apt install openjdk-25-jdk

# Install protoc
sudo apt install protobuf-compiler

# Install Go tools
go install github.com/zeromicro/go-zero/tools/goctl@v1.9.2
go install github.com/sudorandom/protoc-gen-connect-openapi@latest
```

## Running Tests

### Unit Tests

```bash
go test ./...
```

### End-to-End Tests

E2E tests use Cobra's `Execute` to test the complete CLI workflow:

```bash
# Run all E2E tests (recommended)
make test-e2e

# Or directly with go test
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

## Golden Fixture Tests

Golden fixtures are JSON snapshots of expected OpenAPI output, used to detect regressions in generated specs. They live under framework-specific `fixtures/golden/` directories.

### How It Works

1. Tests generate a spec from a demo project using `spec-forge generate`
2. The generated spec is compared (byte-equal after pretty-printing) against stored golden fixtures
3. Both full-spec and extracted sub-paths (individual schemas/operations) are compared

### Available Golden Suites

| Package | Golden Dir | Description |
|---------|-----------|-------------|
| `spring/` | `spring/fixtures/golden/` | Spring Boot (Maven) golden snapshots |
| `gin/` | `gin/fixtures/golden/` | Gin framework golden snapshots |
| `gozero/` | `gozero/fixtures/golden/` | go-zero framework golden snapshots |
| `grpcprotoc/` | `grpcprotoc/fixtures/golden/` | gRPC-protoc framework golden snapshots |

### Spring Golden Fixtures

The `spring/fixtures/golden/` directory contains:

```
spring/fixtures/golden/
├── openapi.json                    # Full OpenAPI spec snapshot
├── schemas/
│   ├── User.json                   # User schema structure
│   ├── PageResultUser.json         # Pagination wrapper schema
│   ├── ApiResponseUser.json        # API response wrapper schema
│   └── ...
└── paths/
    ├── api-v1-users-get.json       # GET /api/v1/users operation
    ├── api-v1-users-post.json      # POST /api/v1/users operation
    └── ...
```

**Purpose:** Golden fixtures detect regressions by comparing generated specs against known-good snapshots. Any change to the generated spec (field types, required properties, response codes) will cause a test failure, making regressions visible via `git diff`.

**Regenerating Spring golden files:**

```bash
REGENERATE_GOLDEN=true go test -v -tags=e2e ./integration-tests/spring/... -run TestRegenerateGolden
```

### go-zero Golden Fixtures

The `gozero/fixtures/golden/` directory contains:

```
gozero/fixtures/golden/
├── openapi.json                    # Full OpenAPI spec snapshot
└── paths/
    ├── api-v1-users-get.json       # GET /api/v1/users operation
    ├── api-v1-users-post.json      # POST /api/v1/users operation
    └── api-v1-users-id-get.json    # GET /api/v1/users/{id} operation
```

**Volatile Fields:** goctl generates runtime-dependent fields that change on every run:
- `x-date` - Timestamp of spec generation (changes every run)
- `x-goctl-version` - Tool version (may change when goctl is updated)

These fields are automatically stripped before golden comparison to ensure deterministic tests.

**Regenerating go-zero golden files:**

```bash
REGENERATE_GOLDEN=true go test -v -tags=e2e ./integration-tests/gozero/... -run TestRegenerateGolden
```

### gRPC-protoc Golden Fixtures

The `grpcprotoc/fixtures/golden/` directory contains:

```
grpcprotoc/fixtures/golden/
├── openapi.json                          # Full OpenAPI spec snapshot
├── schemas/
│   ├── demo-user-User.json               # User schema structure
│   ├── demo-user-CreateUserRequest.json   # CreateUserRequest schema
│   └── demo-user-ListUsersResponse.json   # ListUsersResponse schema
└── paths/
    ├── v1-users-get.json                  # GET /v1/users operation
    ├── v1-users-post.json                 # POST /v1/users operation
    ├── v1-users-id-get.json               # GET /v1/users/{id} operation
    └── v1-users-id-put.json               # PUT /v1/users/{id} operation
```

**Regenerating gRPC-protoc golden files:**

```bash
REGENERATE_GOLDEN=true go test -v -tags=e2e ./integration-tests/grpcprotoc/... -run TestRegenerateGolden
```

### Regenerating Golden Files

When the expected output legitimately changes (e.g., springdoc version bump, new endpoints):

```bash
# Regenerate Spring golden files
REGENERATE_GOLDEN=true go test -v -tags=e2e ./integration-tests/spring/... -run TestRegenerateGolden

# Regenerate go-zero golden files
REGENERATE_GOLDEN=true go test -v -tags=e2e ./integration-tests/gozero/... -run TestRegenerateGolden

# Regenerate gRPC-protoc golden files
REGENERATE_GOLDEN=true go test -v -tags=e2e ./integration-tests/grpcprotoc/... -run TestRegenerateGolden
```

After regeneration, review the diff carefully with `git diff` to confirm only expected changes.

## Writing New Tests

### Adding a Golden Snapshot Test

Golden snapshots compare generated spec fragments against stored fixtures. This detects regressions at a fine-grained level.

1. **Define the snapshot** in your test file:

```go
var goldenSnapshots = []helpers.GoldenSnapshot{
    {
        Name: "User Schema Structure",
        Path: "components.schemas.User",      // JSONPath-style path into the spec
        File: "schemas/User.json",            // Relative to fixtures/golden/
    },
    {
        Name: "GET /api/v1/users Operation",
        Path: "paths./api/v1/users.get",      // Supports OpenAPI path syntax
        File: "paths/api-v1-users-get.json",
    },
}
```

2. **Create the golden file** (first time only):

```bash
# Generate the spec, then extract snapshots
REGENERATE_GOLDEN=true go test -v -tags=e2e ./integration-tests/spring/... -run TestRegenerateGolden

# Review the generated files
git diff integration-tests/spring/fixtures/golden/
```

3. **Commit the golden files** to version control.

### Adding an Invariant Test

Invariant tests validate semantic properties that must always hold true, regardless of spec changes:

```go
t.Run("User Schema Must Have ID Field", func(t *testing.T) {
    validator.ValidateSchemaProperty("User", helpers.SchemaPropertyExpectation{
        Name: "id",
        Type: "integer",
    })
})
```

### Adding an Edge Case Test

Edge case tests verify graceful handling of unusual inputs. Always assert concrete outcomes:

```go
err := rootCmd.Execute()
if err != nil {
    t.Logf("Got expected error: %v", err)
    return
}
// If success, verify output was actually generated
files, _ := os.ReadDir(outputDir)
if len(files) == 0 {
    t.Fatal("expected output when Execute() returned nil")
}
```

### Important: Avoiding Port Collisions

Spring Boot tests start an application on port 8080 during spec generation. Since `go test` runs packages in parallel, tests that generate specs from the **same** demo project must be in the **same** Go package to prevent concurrent port binding conflicts. Do not duplicate Spring generation tests across packages.

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

### Spring Boot Golden & Invariant Tests

| Test File | Tests | Description |
|-----------|-------|-------------|
| `spring/golden_test.go` | `TestGoldenSnapshots` | Compares generated spec (full + extracted) against golden fixtures |
| `spring/golden_test.go` | `TestRegenerateGolden` | Regenerates golden files (gated by `REGENERATE_GOLDEN=true`) |
| `spring/invariant_test.go` | `TestCriticalInvariants` | Validates semantic invariants (field types, required params, refs) |
| `spring/edge_cases_test.go` | `TestMalformedPomGracefulDegradation` | Tests graceful error on malformed pom.xml |
| `spring/edge_cases_test.go` | `TestMissingSpringdocDependency` | Tests patcher behavior without springdoc dependency |

### go-zero Golden & Invariant Tests

| Test File | Tests | Description |
|-----------|-------|-------------|
| `gozero/golden_test.go` | `TestGoldenSnapshots` | Compares generated spec (full + extracted) against golden fixtures |
| `gozero/golden_test.go` | `TestRegenerateGolden` | Regenerates golden files (gated by `REGENERATE_GOLDEN=true`) |
| `gozero/invariant_test.go` | `TestAPITypeDefinitions` | Validates API type definitions are correctly parsed |
| `gozero/invariant_test.go` | `TestGoSwaggerFormatCompatibility` | Validates go-swagger format compatibility |
| `gozero/invariant_test.go` | `TestRouteGeneration` | Validates routes are correctly generated from .api files |
| `gozero/edge_cases_test.go` | `TestMissingGoctlGracefulSkip` | Tests graceful skip when goctl is not installed |
| `gozero/edge_cases_test.go` | `TestNonGoZeroProject` | Tests error handling for non-go-zero projects |
| `gozero/edge_cases_test.go` | `TestYAMLOutputFormat` | Tests YAML output format generation |
| `gozero/edge_cases_test.go` | `TestFormDataEndpoint` | Tests form-data endpoint handling |
| `gozero/edge_cases_test.go` | `TestUploadEndpoint` | Tests file upload endpoint handling |

### gRPC-protoc Golden & Invariant Tests

| Test File | Tests | Description |
|-----------|-------|-------------|
| `grpcprotoc/golden_test.go` | `TestGoldenSnapshots` | Compares generated spec (full + extracted) against golden fixtures |
| `grpcprotoc/golden_test.go` | `TestCriticalInvariants` | Validates semantic invariants (schema fields, operationIds, request bodies) |
| `grpcprotoc/golden_test.go` | `TestRegenerateGolden` | Regenerates golden files (gated by `REGENERATE_GOLDEN=true`) |
| `grpcprotoc/invariant_test.go` | `TestGRPCServiceMapping` | Validates gRPC service methods map to correct REST endpoints |
| `grpcprotoc/invariant_test.go` | `TestProtoFieldMapping` | Validates proto field name/type mapping (snake_case to camelCase) |
| `grpcprotoc/invariant_test.go` | `TestProtoMessageReferences` | Validates proto message $ref references (nested, cross-package) |
| `grpcprotoc/invariant_test.go` | `TestConnectProtocolSupport` | Validates Connect protocol features (error schema, responses) |
| `grpcprotoc/edge_cases_test.go` | `TestBufYAMLRejection` | Tests rejection of buf.yaml managed projects |
| `grpcprotoc/edge_cases_test.go` | `TestMissingProtocGracefulSkip` | Tests graceful skip when protoc is not installed |
| `grpcprotoc/edge_cases_test.go` | `TestYAMLOutputFormat` | Tests YAML output format generation |
| `grpcprotoc/edge_cases_test.go` | `TestMultipleProtoFiles` | Tests correct handling of multiple proto files |
| `grpcprotoc/edge_cases_test.go` | `TestNonProtocProject` | Tests error handling for non-protoc projects |

### CLI E2E Tests

| Test File | Tests | Description |
|-----------|-------|-------------|
| `e2e_generate_test.go` | `TestE2E_Generate_Help` | Tests generate command help |
| `e2e_generate_test.go` | `TestE2E_Generate_Version` | Tests CLI version output |
| `e2e_generate_test.go` | `TestE2E_Generate_InvalidProject` | Tests error handling for non-existent project |
| `e2e_enrich_test.go` | `TestE2E_Enrich_Help` | Tests enrich command help |
| `e2e_publish_test.go` | `TestE2E_Publish_Help` | Tests publish command help |
| `e2e_publish_test.go` | `TestE2E_Publish_MissingAPIKey` | Tests error handling for missing API key |
| `e2e_publish_test.go` | `TestE2E_Publish_MissingTarget` | Tests error handling for missing --to flag |
| `e2e_publish_test.go` | `TestE2E_Publish_InvalidTarget` | Tests error handling for invalid target |
| `e2e_publish_test.go` | `TestE2E_Publish_NonExistentSpec` | Tests error handling for non-existent spec file |
| `e2e_publish_test.go` | `TestE2E_Publish_InvalidSpec` | Tests error handling for invalid spec format |

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

The gRPC-protoc test suite (`grpcprotoc/`) verifies:
- Project detection (grpc-protoc framework)
- buf.yaml rejection with helpful error
- OpenAPI spec generation with REST endpoints from `google.api.http` annotations
- Golden snapshot stability (8 golden files)
- Proto field name mapping (snake_case → camelCase)
- Proto field type mapping (int32/int64 formats)
- Cross-package message references
- Multiple proto file handling
- YAML output format support
- Connect protocol features
