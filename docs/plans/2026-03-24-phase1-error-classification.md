# Phase 1: Error Classification System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create a unified error classification system that standardizes error handling across all spec-forge packages.

**Architecture:** Create a centralized `internal/errors` package with typed error categories. Each error type includes: category code, message, cause, and recovery hints. Replace existing ad-hoc error handling with structured errors using a phased migration approach.

**Tech Stack:** Go 1.22+, errors.Is/As/Unwrap patterns, structured logging

---

## File Structure

```
internal/
├── errors/
│   ├── errors.go         # Core error types and constructors
│   ├── errors_test.go    # Unit tests for error package
│   ├── codes.go          # Error category codes
│   ├── codes_test.go     # Code validation tests
│   └── doc.go            # Package documentation
├── extractor/
│   ├── spring/           # Migrate to internal/errors
│   ├── gozero/           # Migrate to internal/errors
│   ├── grpcprotoc/       # Migrate to internal/errors
│   └── gin/              # Migrate to internal/errors
├── enricher/
│   └── errors.go         # Refactor to use internal/errors
├── executor/
│   └── executor.go       # Refactor to use internal/errors
└── publisher/
    └── publisher.go      # Refactor to use internal/errors
```

---

## Error Categories

| Code | Category | Description | Recovery |
|------|----------|-------------|----------|
| `CONFIG` | Configuration | Invalid config, missing env vars | Check .spec-forge.yaml, env vars |
| `DETECT` | Detection | Framework not detected, invalid project | Verify project structure |
| `PATCH` | Patching | Dependency injection failed | Check build file permissions |
| `GENERATE` | Generation | Spec generation failed | Check build logs, dependencies |
| `VALIDATE` | Validation | OpenAPI spec invalid | Fix spec compliance issues |
| `LLM` | LLM/Enrichment | AI provider errors | Check API key, rate limits |
| `PUBLISH` | Publishing | Upload to platform failed | Check credentials, network |
| `SYSTEM` | System | File I/O, command execution, timeout | Check system resources |

---

## Task 1: Create Core Error Package

**Files:**
- Create: `internal/errors/doc.go`
- Create: `internal/errors/codes.go`
- Create: `internal/errors/errors.go`
- Create: `internal/errors/errors_test.go`
- Create: `internal/errors/codes_test.go`

- [ ] **Step 1: Write failing test for error codes**

```go
// internal/errors/codes_test.go
package errors

import "testing"

func TestErrorCodesAreValid(t *testing.T) {
    codes := []string{
        CodeConfig,
        CodeDetect,
        CodePatch,
        CodeGenerate,
        CodeValidate,
        CodeLLM,
        CodePublish,
        CodeSystem,
    }
    for _, code := range codes {
        if len(code) == 0 {
            t.Errorf("error code should not be empty")
        }
        if code != string([]byte(code)) {
            t.Errorf("error code %q should be ASCII only", code)
        }
    }
}

func TestCodeRecoveryHint(t *testing.T) {
    tests := []struct {
        code     string
        wantHint string
    }{
        {CodeConfig, "configuration"},
        {CodeDetect, "project structure"},
        {CodeLLM, "API key"},
    }
    for _, tt := range tests {
        hint := RecoveryHint(tt.code)
        if hint == "" {
            t.Errorf("Code %q should have a recovery hint", tt.code)
        }
        if !containsSubstring(hint, tt.wantHint) {
            t.Errorf("Code %q hint %q should contain %q", tt.code, hint, tt.wantHint)
        }
    }
}

func containsSubstring(s, substr string) bool {
    return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s[1:], substr))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/errors/... -v -run TestErrorCodes`
Expected: FAIL (package not found)

- [ ] **Step 3: Write error codes implementation**

```go
// internal/errors/doc.go
// Package errors provides standardized error types for spec-forge.
//
// All errors in spec-forge should use this package to ensure consistent
// error handling, logging, and user feedback.
package errors
```

```go
// internal/errors/codes.go
package errors

// Error category codes. Each code represents a distinct error category
// with specific semantics for recovery and logging.
const (
    // CodeConfig indicates configuration errors (invalid config, missing env vars).
    CodeConfig = "CONFIG"

    // CodeDetect indicates framework detection errors (framework not found, invalid project).
    CodeDetect = "DETECT"

    // CodePatch indicates patching errors (dependency injection failed, file modification).
    CodePatch = "PATCH"

    // CodeGenerate indicates spec generation errors (build failed, no output produced).
    CodeGenerate = "GENERATE"

    // CodeValidate indicates OpenAPI validation errors (invalid spec, compliance issues).
    CodeValidate = "VALIDATE"

    // CodeLLM indicates LLM/enrichment errors (API errors, rate limits, parsing failures).
    CodeLLM = "LLM"

    // CodePublish indicates publishing errors (upload failed, authentication issues).
    CodePublish = "PUBLISH"

    // CodeSystem indicates system-level errors (file I/O, command execution, timeout).
    CodeSystem = "SYSTEM"
)

// RecoveryHint returns a human-readable hint for error recovery.
func RecoveryHint(code string) string {
    switch code {
    case CodeConfig:
        return "Check your .spec-forge.yaml configuration and environment variables"
    case CodeDetect:
        return "Verify the project structure matches the expected framework layout"
    case CodePatch:
        return "Check file permissions and ensure the build file is writable"
    case CodeGenerate:
        return "Review build logs for errors and ensure all dependencies are installed"
    case CodeValidate:
        return "Fix the OpenAPI spec compliance issues reported by the validator"
    case CodeLLM:
        return "Verify your API key is valid and check for rate limiting"
    case CodePublish:
        return "Check your credentials and network connectivity"
    case CodeSystem:
        return "Check system resources, file permissions, and command availability"
    default:
        return "Review the error details and consult documentation"
    }
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/errors/... -v -run TestErrorCodes`
Expected: PASS

- [ ] **Step 5: Write failing test for Error type**

```go
// internal/errors/errors_test.go
package errors

import (
    "errors"
    "testing"
)

func TestNew(t *testing.T) {
    cause := errors.New("underlying error")
    err := New(CodeConfig, "invalid configuration", cause)

    if err.Code != CodeConfig {
        t.Errorf("expected code %q, got %q", CodeConfig, err.Code)
    }
    if err.Message != "invalid configuration" {
        t.Errorf("expected message %q, got %q", "invalid configuration", err.Message)
    }
    if !errors.Is(err, cause) {
        t.Error("error should wrap the cause")
    }
}

func TestError_Error(t *testing.T) {
    tests := []struct {
        name     string
        err      *Error
        wantCont string
    }{
        {
            name: "with cause",
            err:  New(CodeConfig, "invalid config", errors.New("missing key")),
            wantCont: "[CONFIG] invalid config: missing key",
        },
        {
            name: "without cause",
            err:  New(CodeDetect, "framework not found", nil),
            wantCont: "[DETECT] framework not found",
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := tt.err.Error()
            if !containsSubstring(got, tt.wantCont) {
                t.Errorf("Error() = %q, want to contain %q", got, tt.wantCont)
            }
        })
    }
}

func TestIsCode(t *testing.T) {
    configErr := New(CodeConfig, "test", nil)
    detectErr := New(CodeDetect, "test", nil)

    if !IsCode(configErr, CodeConfig) {
        t.Error("IsCode should return true for matching code")
    }
    if IsCode(configErr, CodeDetect) {
        t.Error("IsCode should return false for non-matching code")
    }
    if IsCode(detectErr, CodeConfig) {
        t.Error("IsCode should return false for different error")
    }
}

func TestAsCode(t *testing.T) {
    err := New(CodeLLM, "API error", errors.New("timeout"))

    var target *Error
    if !errors.As(err, &target) {
        t.Fatal("errors.As should succeed")
    }
    if target.Code != CodeLLM {
        t.Errorf("expected code %q, got %q", CodeLLM, target.Code)
    }
}
```

- [ ] **Step 6: Run test to verify it fails**

Run: `go test ./internal/errors/... -v -run TestNew`
Expected: FAIL (type not defined)

- [ ] **Step 7: Write Error type implementation**

```go
// internal/errors/errors.go
package errors

import (
    "errors"
    "fmt"
)

// Error represents a structured error with classification and recovery hints.
type Error struct {
    // Code is the error category code (CONFIG, DETECT, etc.)
    Code string
    // Message is a human-readable error description
    Message string
    // Cause is the underlying error (may be nil)
    Cause error
    // Context contains additional key-value pairs for debugging
    Context map[string]any
}

// Error implements the error interface.
func (e *Error) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
    }
    return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause for errors.Is/As.
func (e *Error) Unwrap() error {
    return e.Cause
}

// WithContext adds context key-value pairs to the error.
func (e *Error) WithContext(key string, value any) *Error {
    if e.Context == nil {
        e.Context = make(map[string]any)
    }
    e.Context[key] = value
    return e
}

// Hint returns the recovery hint for this error's code.
func (e *Error) Hint() string {
    return RecoveryHint(e.Code)
}

// New creates a new Error with the given code, message, and optional cause.
func New(code, message string, cause error) *Error {
    return &Error{
        Code:    code,
        Message: message,
        Cause:   cause,
    }
}

// Newf creates a new Error with formatted message.
func Newf(code string, cause error, format string, args ...any) *Error {
    return &Error{
        Code:    code,
        Message: fmt.Sprintf(format, args...),
        Cause:   cause,
    }
}

// IsCode checks if the error has the specified code.
func IsCode(err error, code string) bool {
    var e *Error
    if errors.As(err, &e) {
        return e.Code == code
    }
    return false
}

// GetCode extracts the error code, or returns empty string if not a classified error.
func GetCode(err error) string {
    var e *Error
    if errors.As(err, &e) {
        return e.Code
    }
    return ""
}

// IsRetryable returns true if the error category is typically retryable.
func IsRetryable(err error) bool {
    var e *Error
    if errors.As(err, &e) {
        switch e.Code {
        case CodeLLM, CodePublish, CodeSystem:
            return true
        default:
            return false
        }
    }
    return false
}
```

- [ ] **Step 8: Run test to verify it passes**

Run: `go test ./internal/errors/... -v`
Expected: PASS

- [ ] **Step 9: Commit core error package**

```bash
git add internal/errors/
git commit -s -m "feat(errors): add unified error classification system

- Add 8 error category codes (CONFIG, DETECT, PATCH, GENERATE, VALIDATE, LLM, PUBLISH, SYSTEM)
- Add Error type with code, message, cause, and context
- Add helper functions: IsCode, GetCode, IsRetryable, RecoveryHint
- Add comprehensive unit tests"
```

---

## Task 2: Create Convenience Constructors

**Files:**
- Modify: `internal/errors/errors.go`
- Modify: `internal/errors/errors_test.go`

- [ ] **Step 1: Write failing test for convenience constructors**

```go
// Add to internal/errors/errors_test.go

func TestConvenienceConstructors(t *testing.T) {
    tests := []struct {
        name     string
        fn       func(string, error) *Error
        wantCode string
    }{
        {"Config", ConfigError, CodeConfig},
        {"Detect", DetectError, CodeDetect},
        {"Patch", PatchError, CodePatch},
        {"Generate", GenerateError, CodeGenerate},
        {"Validate", ValidateError, CodeValidate},
        {"LLM", LLMError, CodeLLM},
        {"Publish", PublishError, CodePublish},
        {"System", SystemError, CodeSystem},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.fn("test message", nil)
            if err.Code != tt.wantCode {
                t.Errorf("expected code %q, got %q", tt.wantCode, err.Code)
            }
        })
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/errors/... -v -run TestConvenience`
Expected: FAIL (functions not defined)

- [ ] **Step 3: Add convenience constructors**

```go
// Add to internal/errors/errors.go

// ConfigError creates a configuration error.
func ConfigError(message string, cause error) *Error {
    return New(CodeConfig, message, cause)
}

// DetectError creates a framework detection error.
func DetectError(message string, cause error) *Error {
    return New(CodeDetect, message, cause)
}

// PatchError creates a patching error.
func PatchError(message string, cause error) *Error {
    return New(CodePatch, message, cause)
}

// GenerateError creates a spec generation error.
func GenerateError(message string, cause error) *Error {
    return New(CodeGenerate, message, cause)
}

// ValidateError creates a validation error.
func ValidateError(message string, cause error) *Error {
    return New(CodeValidate, message, cause)
}

// LLMError creates an LLM/enrichment error.
func LLMError(message string, cause error) *Error {
    return New(CodeLLM, message, cause)
}

// PublishError creates a publishing error.
func PublishError(message string, cause error) *Error {
    return New(CodePublish, message, cause)
}

// SystemError creates a system-level error.
func SystemError(message string, cause error) *Error {
    return New(CodeSystem, message, cause)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/errors/... -v -run TestConvenience`
Expected: PASS

- [ ] **Step 5: Commit convenience constructors**

```bash
git add internal/errors/
git commit -s -m "feat(errors): add convenience constructors for each error category"
```

---

## Task 3: Migrate Executor Package

**Files:**
- Modify: `internal/executor/executor.go`
- Modify: `internal/executor/executor_test.go`

- [ ] **Step 1: Write test for migrated error types**

```go
// Add to internal/executor/executor_test.go

import (
    "testing"
    forgeerrors "github.com/spencercjh/spec-forge/internal/errors"
)

func TestCommandNotFoundError_IsSystemError(t *testing.T) {
    err := &CommandNotFoundError{Command: "mvn"}
    if !forgeerrors.IsCode(err, forgeerrors.CodeSystem) {
        t.Error("CommandNotFoundError should be a SYSTEM error")
    }
}

func TestCommandFailedError_IsSystemError(t *testing.T) {
    err := &CommandFailedError{
        Command:  "gradle",
        ExitCode: 1,
        Stderr:   "build failed",
    }
    if !forgeerrors.IsCode(err, forgeerrors.CodeSystem) {
        t.Error("CommandFailedError should be a SYSTEM error")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/executor/... -v -run TestCommand`
Expected: FAIL (errors not migrated)

- [ ] **Step 3: Migrate executor errors to implement Error interface**

```go
// Modify internal/executor/executor.go

import (
    forgeerrors "github.com/spencercjh/spec-forge/internal/errors"
    // ... other imports
)

// CommandNotFoundError indicates the command was not found in PATH.
type CommandNotFoundError struct {
    Command string
    *forgeerrors.Error
}

func NewCommandNotFoundError(command string) *CommandNotFoundError {
    return &CommandNotFoundError{
        Command: command,
        Error: forgeerrors.SystemError(
            fmt.Sprintf("command '%s' not found in PATH", command),
            exec.ErrNotFound,
        ),
    }
}

// Update existing constructor pattern if different

// CommandFailedError indicates the command executed but returned non-zero exit code.
type CommandFailedError struct {
    Command  string
    Args     []string
    ExitCode int
    Stdout   string
    Stderr   string
    Err      error
    *forgeerrors.Error
}

// Update to embed forgeerrors.Error and implement proper classification
```

**Migration Strategy:**
We use **Option B: Wrapper approach** - Keep existing error types but add `IsCode()` compatibility:

1. Keep existing `CommandNotFoundError` and `CommandFailedError` structs unchanged
2. Add a `forgeerrors.Error` as embedded field OR implement a `Classify()` method
3. Ensure `errors.As()` works for both the original type AND `*forgeerrors.Error`

Example approach:
```go
type CommandNotFoundError struct {
    Command string
    classified *forgeerrors.Error  // Embedded classified error
}

func (e *CommandNotFoundError) Error() string {
    return e.classified.Error()  // Delegate to classified
}

func (e *CommandNotFoundError) Unwrap() error {
    return e.classified.Unwrap()
}

// For errors.As to find *forgeerrors.Error
func (e *CommandNotFoundError) Is(target error) bool {
    return errors.Is(e.classified, target)
}
```

This ensures backward compatibility: existing code using `errors.As(err, &CommandNotFoundError{})` continues to work.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/executor/... -v`
Expected: PASS

- [ ] **Step 5: Commit executor migration**

```bash
git add internal/executor/
git commit -s -m "refactor(executor): migrate to unified error classification

- CommandNotFoundError now returns SYSTEM code
- CommandFailedError now returns SYSTEM code
- Maintain backward compatibility with existing error interface"
```

---

## Task 4: Migrate Enricher Package

**Files:**
- Modify: `internal/enricher/errors.go`
- Modify: `internal/enricher/errors_test.go`

- [ ] **Step 1: Write test for migrated enricher errors**

```go
// Add to internal/enricher/errors_test.go

import forgeerrors "github.com/spencercjh/spec-forge/internal/errors"

func TestEnrichmentError_CodeMapping(t *testing.T) {
    tests := []struct {
        errorType string
        wantCode  string
    }{
        {ErrorTypeConfig, forgeerrors.CodeConfig},
        {ErrorTypeLLMCall, forgeerrors.CodeLLM},
        {ErrorTypeParse, forgeerrors.CodeLLM},
        {ErrorTypeTemplate, forgeerrors.CodeConfig},
    }

    for _, tt := range tests {
        err := NewEnrichmentError(tt.errorType, "test", nil)
        if !forgeerrors.IsCode(err, tt.wantCode) {
            t.Errorf("error type %q should map to code %q", tt.errorType, tt.wantCode)
        }
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/enricher/... -v -run TestEnrichmentError_CodeMapping`
Expected: FAIL (mapping not implemented)

- [ ] **Step 3: Implement error type mapping**

```go
// Modify internal/enricher/errors.go

import forgeerrors "github.com/spencercjh/spec-forge/internal/errors"

// mapErrorType maps enricher-specific error types to unified codes.
func mapErrorType(errorType string) string {
    switch errorType {
    case ErrorTypeConfig, ErrorTypeTemplate:
        return forgeerrors.CodeConfig
    case ErrorTypeLLMCall, ErrorTypeParse:
        return forgeerrors.CodeLLM
    default:
        return forgeerrors.CodeSystem
    }
}

// Update EnrichmentError to implement forgeerrors interface or wrap forgeerrors.Error
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/enricher/... -v`
Expected: PASS

- [ ] **Step 5: Commit enricher migration**

```bash
git add internal/enricher/
git commit -s -m "refactor(enricher): migrate to unified error classification

- Map enricher error types to unified codes (CONFIG, LLM, SYSTEM)
- Maintain backward compatibility with IsConfigError"
```

---

## Task 5: Migrate Extractor Packages (Spring, Gin, GoZero, gRPC)

**Files:**
- Modify: `internal/extractor/spring/detector.go`
- Modify: `internal/extractor/spring/patcher.go`
- Modify: `internal/extractor/spring/generator.go`
- Modify: `internal/extractor/gin/detector.go`
- Modify: `internal/extractor/gin/generator.go`
- Modify: `internal/extractor/gozero/detector.go`
- Modify: `internal/extractor/gozero/generator.go`
- Modify: `internal/extractor/grpcprotoc/detector.go`
- Modify: `internal/extractor/grpcprotoc/generator.go`
- Create: Test files for each package with error classification tests

- [ ] **Step 1: Create detector error test for Spring**

```go
// Add to internal/extractor/spring/detector_test.go or create new test file

import forgeerrors "github.com/spencercjh/spec-forge/internal/errors"

func TestDetect_ReturnsDetectError(t *testing.T) {
    detector := NewDetector()
    _, err := detector.Detect("/nonexistent/path")

    if err == nil {
        t.Fatal("expected error for nonexistent path")
    }
    if !forgeerrors.IsCode(err, forgeerrors.CodeDetect) {
        t.Errorf("expected DETECT error, got code: %s", forgeerrors.GetCode(err))
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/extractor/spring/... -v -run TestDetect_ReturnsDetectError`
Expected: FAIL (returns fmt.Errorf)

- [ ] **Step 3: Migrate Spring detector errors**

```go
// Modify internal/extractor/spring/detector.go

import forgeerrors "github.com/spencercjh/spec-forge/internal/errors"

func (d *Detector) Detect(projectPath string) (*extractor.ProjectInfo, error) {
    absPath, err := filepath.Abs(projectPath)
    if err != nil {
        return nil, forgeerrors.DetectError(
            fmt.Sprintf("failed to resolve path: %s", projectPath),
            err,
        )
    }
    // ... continue migration
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/extractor/spring/... -v`
Expected: PASS

- [ ] **Step 5: Repeat for all extractor packages**

Follow the same pattern for:
- `internal/extractor/spring/patcher.go` → CodePatch
- `internal/extractor/spring/generator.go` → CodeGenerate
- `internal/extractor/gin/*` → CodeDetect, CodeGenerate
- `internal/extractor/gozero/*` → CodeDetect, CodePatch, CodeGenerate
- `internal/extractor/grpcprotoc/*` → CodeDetect, CodePatch, CodeGenerate

- [ ] **Step 6: Commit extractor migrations**

```bash
git add internal/extractor/
git commit -s -m "refactor(extractor): migrate all extractors to unified error classification

- Spring: DETECT errors for detection, PATCH for patching, GENERATE for generation
- Gin: DETECT for detection, GENERATE for AST parsing failures
- GoZero: DETECT for detection, PATCH for goctl check, GENERATE for generation
- gRPC-protoc: DETECT for detection, PATCH for protoc check, GENERATE for generation"
```

---

## Task 6: Migrate Publisher Package

**Files:**
- Modify: `internal/publisher/publisher.go`
- Modify: `internal/publisher/readme.go`
- Modify: `internal/publisher/publisher_test.go`

- [ ] **Step 1: Write test for publisher error classification**

- [ ] **Step 2: Migrate publisher errors to use CodePublish**

- [ ] **Step 3: Run tests and verify**

- [ ] **Step 4: Commit publisher migration**

```bash
git add internal/publisher/
git commit -s -m "refactor(publisher): migrate to unified error classification

- Publishing errors now return PUBLISH code
- Local file errors return SYSTEM code"
```

---

## Task 7: Update CLI Commands

**Files:**
- Modify: `cmd/generate.go`
- Modify: `cmd/enrich.go`
- Modify: `cmd/publish.go`

- [ ] **Step 1: Update CLI error handling to use error codes**

```go
// Example in cmd/generate.go

import forgeerrors "github.com/spencercjh/spec-forge/internal/errors"

// In error handling section:
if err != nil {
    code := forgeerrors.GetCode(err)
    hint := ""
    if classified, ok := err.(*forgeerrors.Error); ok {
        hint = classified.Hint()
    }

    if hint != "" {
        fmt.Fprintf(os.Stderr, "Error: %v\nHint: %s\n", err, hint)
    } else {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
    }

    // Exit with appropriate code based on error category
    os.Exit(exitCodeForError(code))
}
```

- [ ] **Step 2: Run full test suite**

Run: `make test`
Expected: All tests pass

- [ ] **Step 3: Commit CLI updates**

```bash
git add cmd/
git commit -s -m "feat(cmd): use error codes for better error messages

- Display recovery hints based on error code
- Exit with appropriate codes per error category"
```

---

## Task 8: Add Integration Test for Error Classification

**Files:**
- Create: `integration-tests/error_classification_test.go`

- [ ] **Step 1: Write integration test**

```go
// integration-tests/error_classification_test.go

//go:build e2e

package e2e

import (
    "testing"

    "github.com/spencercjh/spec-forge/internal/executor"
    forgeerrors "github.com/spencercjh/spec-forge/internal/errors"
)

func TestExecutorErrorClassification(t *testing.T) {
    exec := executor.NewExecutor()

    // Test command not found
    _, err := exec.Execute(context.Background(), &executor.ExecuteOptions{
        Command: "nonexistent_command_12345",
    })

    if !forgeerrors.IsCode(err, forgeerrors.CodeSystem) {
        t.Errorf("command not found should be SYSTEM error, got %s", forgeerrors.GetCode(err))
    }
}
```

- [ ] **Step 2: Run integration test**

Run: `go test -v -tags=e2e ./integration-tests/... -run TestExecutorErrorClassification`
Expected: PASS

- [ ] **Step 3: Commit integration test**

```bash
git add integration-tests/
git commit -s -m "test(e2e): add error classification integration test"
```

---

## Task 9: Documentation

**Files:**
- Create: `docs/error-handling.md`
- Update: `CLAUDE.md`

- [ ] **Step 1: Write error handling documentation**

```markdown
# docs/error-handling.md

# Error Handling Guide

## Overview

Spec Forge uses a unified error classification system in `internal/errors`. All errors are classified into one of 8 categories.

## Error Categories

| Code | Description | Example |
|------|-------------|---------|
| CONFIG | Configuration errors | Invalid .spec-forge.yaml |
| DETECT | Framework detection errors | Framework not found |
| PATCH | Patching errors | Dependency injection failed |
| GENERATE | Generation errors | Build failed |
| VALIDATE | Validation errors | Invalid OpenAPI spec |
| LLM | LLM errors | API rate limit |
| PUBLISH | Publishing errors | Upload failed |
| SYSTEM | System errors | Command not found |

## Usage

### Creating Errors

```go
import forgeerrors "github.com/spencercjh/spec-forge/internal/errors"

// Using convenience constructors
err := forgeerrors.DetectError("framework not found", nil)

// With cause
err := forgeerrors.GenerateError("build failed", underlyingErr)

// With context
err := forgeerrors.New(forgeerrors.CodeConfig, "invalid config", nil).
    WithContext("file", ".spec-forge.yaml").
    WithContext("field", "enrich.provider")
```

### Checking Errors

```go
// Check specific code
if forgeerrors.IsCode(err, forgeerrors.CodeLLM) {
    // Handle LLM error (maybe retry)
}

// Check if retryable
if forgeerrors.IsRetryable(err) {
    // Retry logic
}

// Get recovery hint
if classified, ok := err.(*forgeerrors.Error); ok {
    fmt.Println("Hint:", classified.Hint())
}
```

## Migration Guide

When adding new code, use `internal/errors` package:

1. Import: `import forgeerrors "github.com/spencercjh/spec-forge/internal/errors"`
2. Choose appropriate code
3. Use convenience constructor or `New()`
4. Add context if helpful for debugging
```

- [ ] **Step 2: Update CLAUDE.md**

Add reference to error handling documentation in the Architecture section.

- [ ] **Step 3: Commit documentation**

```bash
git add docs/ CLAUDE.md
git commit -s -m "docs: add error handling guide and update CLAUDE.md"
```

---

## Task 10: Backward Compatibility Verification

**Files:**
- Create: `internal/errors/backward_compat_test.go`

- [ ] **Step 1: Write backward compatibility tests**

```go
// internal/errors/backward_compat_test.go
package errors_test

import (
    "errors"
    "testing"

    "github.com/spencercjh/spec-forge/internal/executor"
    forgeerrors "github.com/spencercjh/spec-forge/internal/errors"
)

func TestBackwardCompat_ExecutorErrors(t *testing.T) {
    // Test that existing error type assertions still work
    t.Run("CommandNotFoundError", func(t *testing.T) {
        err := &executor.CommandNotFoundError{Command: "test"}

        // Old-style assertion should still work
        var cmdErr *executor.CommandNotFoundError
        if !errors.As(err, &cmdErr) {
            t.Error("errors.As for CommandNotFoundError should still work")
        }

        // New-style classification should also work
        if !forgeerrors.IsCode(err, forgeerrors.CodeSystem) {
            t.Error("error should also be classifiable as SYSTEM")
        }
    })

    t.Run("CommandFailedError", func(t *testing.T) {
        err := &executor.CommandFailedError{
            Command:  "test",
            ExitCode: 1,
        }

        var cmdErr *executor.CommandFailedError
        if !errors.As(err, &cmdErr) {
            t.Error("errors.As for CommandFailedError should still work")
        }

        if !forgeerrors.IsCode(err, forgeerrors.CodeSystem) {
            t.Error("error should also be classifiable as SYSTEM")
        }
    })
}
```

- [ ] **Step 2: Run backward compatibility tests**

Run: `go test ./internal/errors/... -v -run TestBackwardCompat`
Expected: PASS (after executor migration complete)

- [ ] **Step 3: Commit backward compatibility tests**

```bash
git add internal/errors/backward_compat_test.go
git commit -s -m "test(errors): add backward compatibility verification tests"
```

---

## Verification Checklist

- [ ] All tests pass: `make test`
- [ ] Linter passes: `make lint`
- [ ] All packages migrated to use `internal/errors`
- [ ] Documentation complete
- [ ] Backward compatibility verified:
  - [ ] `errors.As(err, &CommandNotFoundError{})` still works
  - [ ] `errors.As(err, &CommandFailedError{})` still works
  - [ ] `IsConfigError()` in enricher still works
  - [ ] Existing test assertions using error types pass
- [ ] No breaking changes to public API

---

## Summary

This plan creates a unified error classification system that:
1. Defines 8 clear error categories
2. Provides recovery hints for each category
3. Enables programmatic error handling (IsRetryable, IsCode)
4. Maintains backward compatibility
5. Includes comprehensive tests and documentation
