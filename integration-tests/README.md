# Integration Tests

This directory contains end-to-end tests for the spec-forge CLI tool.

## Test Projects

| Project | Build Tool | Description |
|---------|------------|-------------|
| `maven-springboot-openapi-demo` | Maven | Single module Spring Boot project |
| `gradle-springboot-openapi-demo` | Gradle | Single module Spring Boot project |
| `maven-multi-module-demo` | Maven | Multi-module Spring Boot project |
| `gradle-multi-module-demo` | Gradle | Multi-module Spring Boot project |
| `gozero-demo` | Go Modules | go-zero framework project |

## Running Tests

### Unit Tests

```bash
go test ./...
```

### End-to-End Tests

E2E tests require the build tools (Maven/Gradle) to be installed.

```bash
# Run all e2e tests
go test -tags=e2e ./integration-tests/...

# Run specific test
go test -tags=e2e ./integration-tests/... -run TestE2E_MavenSpringBoot_Generate

# Run with verbose output
go test -v -tags=e2e ./integration-tests/...
```

### Skipping E2E Tests in CI

E2E tests are gated by the `e2e` build tag. They won't run by default:

```bash
# This won't run e2e tests
go test ./...

# This will run e2e tests
go test -tags=e2e ./...
```

## Test Coverage

| Test File | Tests | Description |
|-----------|-------|-------------|
| `spring_maven_test.go` | `TestE2E_MavenSpringBoot_Generate` | Tests generate flow for Maven project |
| `spring_maven_test.go` | `TestE2E_MavenSpringBoot_GenerateEnrich` | Tests complete generate → enrich flow |
| `spring_gradle_test.go` | `TestE2E_GradleSpringBoot_Generate` | Tests generate flow for Gradle project |
| `gozero_test.go` | `TestE2E_GoZero_Generate` | Tests generate flow for go-zero project |
| `gozero_test.go` | `TestE2E_GoZero_Detect` | Tests detection of go-zero project info |
| `gozero_test.go` | `TestE2E_GoZero_NoGoctl` | Tests graceful handling when goctl missing |
| `error_test.go` | `TestE2E_ErrorHandling_CommandNotFound` | Tests error handling for missing commands |
| `mock_provider_test.go` | `countingMockProvider` | Shared mock provider for enrichment tests |
