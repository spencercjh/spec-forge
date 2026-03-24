# Error Handling Guide

This document describes the unified error classification system used across spec-forge.

## Overview

All spec-forge errors are classified into one of eight categories using the `internal/errors` package.
Each error carries a **code**, a **message**, an optional **cause**, and optional **context** key/value pairs.

The classification enables:
- Consistent, machine-readable error codes
- Human-readable recovery hints
- Proper exit codes from the CLI
- Retry logic based on error category

## Error Categories

| Code | Category | Description | Retryable | Exit Code |
|------|----------|-------------|-----------|-----------|
| `CONFIG` | Configuration | Invalid config, missing env vars | No | 2 |
| `DETECT` | Detection | Framework not detected, invalid project | No | 2 |
| `PATCH` | Patching | Dependency injection failed | No | 2 |
| `GENERATE` | Generation | Spec generation failed | No | 1 |
| `VALIDATE` | Validation | OpenAPI spec invalid | No | 1 |
| `LLM` | LLM/Enrichment | AI provider errors | **Yes** | 1 |
| `PUBLISH` | Publishing | Upload to platform failed | **Yes** | 1 |
| `SYSTEM` | System | File I/O, command execution, timeout | **Yes** | 1 |

## Recovery Hints

Each error category has a built-in recovery hint:

| Code | Recovery Hint |
|------|---------------|
| `CONFIG` | Check your `.spec-forge.yaml` configuration file and ensure all required environment variables are set |
| `DETECT` | Verify the project structure and ensure it contains the expected build files (pom.xml, build.gradle, go.mod, .proto files) |
| `PATCH` | Check that build files are writable and the project has correct permissions |
| `GENERATE` | Check build logs for compilation errors and ensure all dependencies are available |
| `VALIDATE` | Review the generated OpenAPI spec for compliance issues and fix any schema errors |
| `LLM` | Check your API key, model name, and network connectivity; consider retrying as this may be a transient error |
| `PUBLISH` | Verify your publishing credentials and network connectivity; consider retrying |
| `SYSTEM` | Check system resources, file permissions, and ensure required tools are installed; consider retrying |

## API Usage

Import the package using the `forgeerrors` alias to avoid collision with the standard `errors` package:

```go
import forgeerrors "github.com/spencercjh/spec-forge/internal/errors"
```

### Creating Errors

Use the convenience constructors:

```go
// By category
err := forgeerrors.ConfigError("API key not configured", nil)
err := forgeerrors.DetectError("no pom.xml found", nil)
err := forgeerrors.PatchError("write permission denied", ioErr)
err := forgeerrors.GenerateError("maven build failed", mavenErr)
err := forgeerrors.ValidateError("missing required field", nil)
err := forgeerrors.LLMError("rate limit exceeded", apiErr)
err := forgeerrors.PublishError("rdme CLI not found", execErr)
err := forgeerrors.SystemError("command timed out", ctxErr)

// Generic constructor
err := forgeerrors.New(forgeerrors.CodeGenerate, "custom message", cause)

// Formatted message
err := forgeerrors.Newf(forgeerrors.CodeSystem, cause, "command '%s' failed with code %d", cmd, code)

// Adding debug context
err = err.WithContext("path", projectPath).WithContext("tool", "mvn")
```

### Checking Error Categories

```go
// Check for a specific category
if forgeerrors.IsCode(err, forgeerrors.CodeLLM) {
    // handle LLM error
}

// Get the category code
code := forgeerrors.GetCode(err) // e.g. "LLM", "" if not classified

// Check if the error is retryable
if forgeerrors.IsRetryable(err) {
    // retry the operation
}
```

### Recovery Hints

```go
// From an error
if fe, ok := err.(*forgeerrors.Error); ok {
    fmt.Println("Hint:", fe.Hint())
}

// By code
hint := forgeerrors.RecoveryHint(forgeerrors.CodeConfig)
```

## Backward Compatibility

Existing error types (`CommandNotFoundError`, `CommandFailedError`, `EnrichmentError`,
`ErrNotGoZeroProject`, `ErrNotProtocProject`, `ErrBufProjectDetected`, etc.) are preserved.

The classified error is embedded as an internal field (or as the `Cause`) so that:

- `errors.As(err, &CommandNotFoundError{})` still works
- `errors.As(err, &EnrichmentError{})` still works
- `errors.Is(err, ErrBufProjectDetected)` still works
- `forgeerrors.IsCode(err, forgeerrors.CodeSystem)` also works via the error chain

## CLI Exit Codes

When the `spec-forge` CLI encounters a classified error, it:

1. Prints the error message (by Cobra)
2. Prints the recovery hint to stderr if available
3. Exits with an appropriate exit code:
   - **2**: User/configuration errors (`CONFIG`, `DETECT`, `PATCH`)
   - **1**: Execution/external-service errors (all other codes)

## Package Mapping

| Package | Operation | Error Code |
|---------|-----------|------------|
| `internal/executor` | Command not found | `SYSTEM` |
| `internal/executor` | Command execution failed | `SYSTEM` |
| `internal/enricher` | Configuration errors | `CONFIG` |
| `internal/enricher` | LLM call / parse / template | `LLM` |
| `internal/extractor/spring` | Project detection | `DETECT` |
| `internal/extractor/spring` | Dependency patching | `PATCH` |
| `internal/extractor/spring` | Spec generation | `GENERATE` |
| `internal/extractor/gin` | Project detection | `DETECT` |
| `internal/extractor/gin` | Spec generation | `GENERATE` |
| `internal/extractor/gozero` | Project detection | `DETECT` |
| `internal/extractor/gozero` | Tool availability check | `PATCH` |
| `internal/extractor/gozero` | Spec generation | `GENERATE` |
| `internal/extractor/grpcprotoc` | Project detection | `DETECT` |
| `internal/extractor/grpcprotoc` | Tool availability check | `PATCH` |
| `internal/extractor/grpcprotoc` | Spec generation | `GENERATE` |
| `internal/publisher` | Configuration / unknown target | `CONFIG` |
| `internal/publisher` | Publishing | `PUBLISH` |
