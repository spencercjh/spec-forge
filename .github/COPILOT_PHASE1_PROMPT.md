# Phase 1: Unified Error Classification System - Implementation Guide

## Context

You are implementing Phase 1 of the spec-forge stability roadmap. This phase creates a unified error classification system to standardize error handling across all packages.

**Main Issue:** https://github.com/spencercjh/spec-forge/issues/27
**Sub-Tasks:** #28-#37

## Current State Analysis

Before implementing, explore the existing error patterns:

1. **Executor errors** (`internal/executor/executor.go`):
   - `CommandNotFoundError` - command not found in PATH
   - `CommandFailedError` - non-zero exit code

2. **Enricher errors** (`internal/enricher/errors.go`):
   - `EnrichmentError` with types: CONFIG, LLM_CALL, PARSE, TEMPLATE
   - `IsConfigError()` helper function

3. **Extractor errors** (scattered across packages):
   - Mostly using `fmt.Errorf()` without classification
   - Located in: `internal/extractor/spring/`, `gin/`, `gozero/`, `grpcprotoc/`

## Implementation Requirements

### 1. Create Core Error Package

Create `internal/errors/` with:

```go
// codes.go
const (
    CodeConfig   = "CONFIG"   // Configuration errors
    CodeDetect   = "DETECT"   // Framework detection errors
    CodePatch    = "PATCH"    // Patching errors
    CodeGenerate = "GENERATE" // Spec generation errors
    CodeValidate = "VALIDATE" // Validation errors
    CodeLLM      = "LLM"      // LLM/enrichment errors
    CodePublish  = "PUBLISH"  // Publishing errors
    CodeSystem   = "SYSTEM"   // System-level errors
)

func RecoveryHint(code string) string { /* ... */ }
```

```go
// errors.go
type Error struct {
    Code    string
    Message string
    Cause   error
    Context map[string]any
}

func (e *Error) Error() string
func (e *Error) Unwrap() error
func (e *Error) WithContext(key string, value any) *Error
func (e *Error) Hint() string

func New(code, message string, cause error) *Error
func IsCode(err error, code string) bool
func GetCode(err error) string
func IsRetryable(err error) bool

// Convenience constructors
func ConfigError(message string, cause error) *Error
func DetectError(message string, cause error) *Error
// ... etc for each code
```

### 2. Migration Strategy

**CRITICAL: Maintain backward compatibility**

When migrating existing error types:

```go
// BEFORE (executor.go)
type CommandNotFoundError struct {
    Command string
}

func (e *CommandNotFoundError) Error() string {
    return fmt.Sprintf("command '%s' not found in PATH", e.Command)
}

// AFTER - Option A: Embed classified error
type CommandNotFoundError struct {
    Command string
    classified *forgeerrors.Error
}

func (e *CommandNotFoundError) Error() string {
    return e.classified.Error()
}

func (e *CommandNotFoundError) Unwrap() error {
    return e.classified.Unwrap()
}
```

This ensures:
- `errors.As(err, &CommandNotFoundError{})` still works
- `forgeerrors.IsCode(err, forgeerrors.CodeSystem)` also works

### 3. Error Code Mapping

| Package | File | Operations | Error Code |
|---------|------|------------|------------|
| executor | executor.go | Command not found | SYSTEM |
| executor | executor.go | Command failed | SYSTEM |
| enricher | errors.go | Config errors | CONFIG |
| enricher | errors.go | LLM errors | LLM |
| extractor/spring | detector.go | Detection | DETECT |
| extractor/spring | patcher.go | Patching | PATCH |
| extractor/spring | generator.go | Generation | GENERATE |
| extractor/gin | detector.go | Detection | DETECT |
| extractor/gin | generator.go | AST parsing | GENERATE |
| extractor/gozero | detector.go | Detection | DETECT |
| extractor/gozero | patcher.go | goctl check | PATCH |
| extractor/gozero | generator.go | Generation | GENERATE |
| extractor/grpcprotoc | detector.go | Detection | DETECT |
| extractor/grpcprotoc | patcher.go | protoc check | PATCH |
| extractor/grpcprotoc | generator.go | Generation | GENERATE |
| publisher | readme.go | Publishing | PUBLISH |

### 4. CLI Integration

Update `cmd/generate.go`, `cmd/enrich.go`, `cmd/publish.go`:

```go
if err != nil {
    if classified, ok := err.(*forgeerrors.Error); ok {
        fmt.Fprintf(os.Stderr, "Error: %v\nHint: %s\n", err, classified.Hint())
        os.Exit(exitCodeForCategory(classified.Code))
    }
    fmt.Fprintf(os.Stderr, "Error: %v\n", err)
    os.Exit(1)
}
```

### 5. Testing Requirements

- Write unit tests for `internal/errors/` package
- Add backward compatibility tests ensuring existing error assertions work
- Add integration tests in `integration-tests/error_classification_test.go`

## Acceptance Criteria

- [ ] All tests pass: `make test`
- [ ] Linter passes: `make lint`
- [ ] 100% of packages use `internal/errors`
- [ ] `docs/error-handling.md` created
- [ ] Backward compatibility verified:
  - `errors.As(err, &CommandNotFoundError{})` works
  - `errors.As(err, &CommandFailedError{})` works
  - `IsConfigError()` in enricher works
- [ ] DCO signed on all commits (`git commit -s`)

## Commit Convention

Follow conventional commits with DCO sign-off:

```bash
git commit -s -m "feat(errors): add unified error classification system

- Add 8 error category codes
- Add Error type with code, message, cause, and context
- Add helper functions: IsCode, GetCode, IsRetryable, RecoveryHint"
```

## Key Files to Reference

- `internal/executor/executor.go` - existing error patterns
- `internal/enricher/errors.go` - existing error patterns
- `internal/extractor/spring/detector.go` - typical detection errors
- `CLAUDE.md` - project conventions and build commands

## Build Commands

```bash
# Run tests
make test

# Run linter
make lint

# Format code
make fmt

# Full verification
make verify
```

## Notes

- Use `forgeerrors` as import alias to avoid collision with standard `errors`
- All errors should have recovery hints
- `LLM`, `PUBLISH`, `SYSTEM` codes are retryable
- Keep existing error types for backward compatibility
