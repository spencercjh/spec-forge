# ReadMe Publisher Implementation Plan

> **Goal:** Add ReadMe.com publisher to upload OpenAPI specs using the `rdme` CLI tool.

**Architecture:** Implement `ReadMePublisher` that wraps the `rdme openapi upload` CLI command. The publisher writes the spec to a temp file, uses the shared `executor.Interface` to invoke the `rdme` CLI with appropriate flags, and returns the result.

**Tech Stack:** Go, shared `executor.Interface` for CLI invocation (backed by `os/exec` in production), existing `Publisher` interface

---

## Overview

The ReadMe Publisher enables uploading OpenAPI specifications directly to ReadMe.com documentation platform using the official `rdme` CLI tool.

### Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| **API Key via Env Var** | Security: Prevents API key from appearing in process listings (`ps aux`) |
| **Overwrite Control** | Safety: `--confirm-overwrite` only added when `PublishOptions.Overwrite=true` |
| **Factory Function Returns Error** | Correctness: Unknown publisher types are explicit errors, not silent defaults |
| **Constants for Strings** | Maintainability: Eliminates magic strings, satisfies linter |

---

## API Reference

### rdme CLI Command

```bash
rdme openapi upload [SPEC] [--branch=<version>] [--slug=<slug>] [--confirm-overwrite] [--useSpecVersion]
```

**Authentication:**
API key is passed via `README_API_KEY` environment variable (not command line) to prevent leaking in process listings.

**Flags:**
- `--branch` - ReadMe project version (default: `stable`)
- `--slug` - Unique identifier for the API definition
- `--confirm-overwrite` - Skip confirmation prompts (CI-friendly)
- `--useSpecVersion` - Use OpenAPI `info.version` as ReadMe version

---

## Implementation

### File Structure

```
internal/publisher/
├── publisher.go       # Interface, options, factory function
├── local.go           # Local file publisher
├── readme.go          # ReadMe.com publisher (NEW)
└── readme_test.go     # Unit tests (NEW)
```

### Core Types

```go
// Publisher interface (existing)
type Publisher interface {
    Publish(ctx context.Context, spec *openapi3.T, opts *PublishOptions) (*PublishResult, error)
    Name() string
}

// PublishOptions with ReadMe support
type PublishOptions struct {
    OutputPath string
    Format     string
    Overwrite  bool
    ReadMe     *ReadMeOptions  // ReadMe-specific options
}

// ReadMeOptions contains ReadMe-specific publishing options
type ReadMeOptions struct {
    APIKey         string
    Branch         string
    Slug           string
    UseSpecVersion bool
}

// PublishResult with Message support
type PublishResult struct {
    Path         string
    Format       string
    BytesWritten int
    Message      string  // CLI output for remote publishers
}
```

### ReadMePublisher Implementation

```go
package publisher

import (
    "context"
    "errors"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "strings"

    "github.com/getkin/kin-openapi/openapi3"
    "gopkg.in/yaml.v3"
)

type ReadMePublisher struct{}

func NewReadMePublisher() *ReadMePublisher {
    return &ReadMePublisher{}
}

func (p *ReadMePublisher) Name() string {
    return "readme"
}

func (p *ReadMePublisher) Publish(ctx context.Context, spec *openapi3.T, opts *PublishOptions) (*PublishResult, error) {
    if spec == nil {
        return nil, errors.New("spec is nil")
    }

    if opts == nil || opts.ReadMe == nil {
        return nil, errors.New("readme options are required")
    }

    if opts.ReadMe.APIKey == "" {
        return nil, errors.New("readme API key is required")
    }

    // Create temp file for the spec
    tmpDir, err := os.MkdirTemp("", "spec-forge-readme-*")
    if err != nil {
        return nil, fmt.Errorf("failed to create temp directory: %w", err)
    }
    defer os.RemoveAll(tmpDir)

    // Determine format
    format := opts.Format
    if format == "" {
        format = formatYAML
    }

    // Write spec to temp file
    tmpFile := filepath.Join(tmpDir, "openapi."+format)
    data, err := p.marshalSpec(spec, format)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal spec: %w", err)
    }

    if writeErr := os.WriteFile(tmpFile, data, 0o600); writeErr != nil {
        return nil, fmt.Errorf("failed to write temp file: %w", writeErr)
    }

    // Build rdme command args (without API key)
    args := p.buildArgs(tmpFile, opts)

    // Execute rdme CLI with API key via environment variable
    // SECURITY: API key is passed via env var to avoid leaking in process listings
    cmd := exec.CommandContext(ctx, "rdme", args...)
    cmd.Env = append(os.Environ(), "README_API_KEY="+opts.ReadMe.APIKey)

    output, err := cmd.CombinedOutput()
    if err != nil {
        return nil, fmt.Errorf("rdme command failed: %w\noutput: %s", err, string(output))
    }

    // Build location identifier for the uploaded spec
    location := p.buildLocation(opts.ReadMe)

    return &PublishResult{
        Path:         location,
        Format:       format,
        BytesWritten: len(data),
        Message:      strings.TrimSpace(string(output)),
    }, nil
}

func (p *ReadMePublisher) marshalSpec(spec *openapi3.T, format string) ([]byte, error) {
    switch format {
    case formatJSON:
        return spec.MarshalJSON()
    default:
        yamlData, err := spec.MarshalYAML()
        if err != nil {
            return nil, err
        }
        return yaml.Marshal(yamlData)
    }
}

func (p *ReadMePublisher) buildArgs(specPath string, opts *PublishOptions) []string {
    args := []string{
        "openapi", "upload",
        specPath,
    }

    readmeOpts := opts.ReadMe

    if readmeOpts.Slug != "" {
        args = append(args, "--slug", readmeOpts.Slug)
    }

    if readmeOpts.Branch != "" {
        args = append(args, "--branch", readmeOpts.Branch)
    }

    if readmeOpts.UseSpecVersion {
        args = append(args, "--useSpecVersion")
    }

    // Only add confirm-overwrite if Overwrite is explicitly set to true
    if opts.Overwrite {
        args = append(args, "--confirm-overwrite")
    }

    return args
}

func (p *ReadMePublisher) buildLocation(opts *ReadMeOptions) string {
    var parts []string

    if opts.Slug != "" {
        parts = append(parts, "slug:"+opts.Slug)
    }

    if opts.Branch != "" {
        parts = append(parts, "branch:"+opts.Branch)
    }

    if len(parts) == 0 {
        return "readme.com (default)"
    }

    return "readme.com/" + strings.Join(parts, "/")
}
```

### Factory Function

```go
// ErrUnknownPublisher is returned when an unknown publisher type is requested.
var ErrUnknownPublisher = errors.New("unknown publisher type")

// NewPublisher creates a Publisher based on the destination type.
// Supported types: "local" (default), "readme"
// For unknown types, returns error.
func NewPublisher(destType string) (Publisher, error) {
    normalizedType := strings.ToLower(strings.TrimSpace(destType))
    switch normalizedType {
    case publisherLocal, "":
        return NewLocalPublisher(), nil
    case publisherReadme:
        return NewReadMePublisher(), nil
    default:
        return nil, fmt.Errorf("%w: %q", ErrUnknownPublisher, destType)
    }
}
```

---

## Testing

### Unit Tests

```go
func TestReadMePublisher_BuildArgs(t *testing.T) {
    p := NewReadMePublisher()

    tests := []struct {
        name     string
        path     string
        opts     *PublishOptions
        expected []string
    }{
        {
            name: "minimal options without overwrite",
            path: "/tmp/spec.yaml",
            opts: &PublishOptions{
                ReadMe: &ReadMeOptions{APIKey: "test-key"},
            },
            expected: []string{
                "openapi", "upload", "/tmp/spec.yaml",
            },
        },
        {
            name: "minimal options with overwrite",
            path: "/tmp/spec.yaml",
            opts: &PublishOptions{
                Overwrite: true,
                ReadMe:    &ReadMeOptions{APIKey: "test-key"},
            },
            expected: []string{
                "openapi", "upload", "/tmp/spec.yaml",
                "--confirm-overwrite",
            },
        },
        {
            name: "full options",
            path: "/tmp/spec.yaml",
            opts: &PublishOptions{
                Overwrite: true,
                ReadMe:    &ReadMeOptions{APIKey: "test-key", Branch: "v1.0.0", Slug: "my-api"},
            },
            expected: []string{
                "openapi", "upload", "/tmp/spec.yaml",
                "--slug", "my-api",
                "--branch", "v1.0.0",
                "--confirm-overwrite",
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            args := p.buildArgs(tt.path, tt.opts)
            if !equalArgs(args, tt.expected) {
                t.Errorf("args mismatch\ngot:      %v\nexpected: %v", args, tt.expected)
            }
        })
    }
}
```

### Test Coverage

- Nil spec handling
- Nil options handling
- Missing API key validation
- Build args with various option combinations
- Overwrite flag behavior

---

## Future Improvements

### 1. ✓ Executor Interface Integration (COMPLETED)

**Status:** Implemented. ReadMePublisher now uses `executor.Interface` for consistent timeout handling, error wrapping, and testability.

```go
type ReadMePublisher struct {
    exec executor.Interface
}

func NewReadMePublisherWithExecutor(exec executor.Interface) *ReadMePublisher {
    return &ReadMePublisher{exec: exec}
}
```

Benefits delivered:
- Consistent timeout handling
- Easier mocking for tests
- Centralized command execution logic
- Command-not-found hints for rdme

### 2. ✓ Environment Variable Handling (COMPLETED)

**Status:** Implemented. `buildEnv()` method filters all existing `README_API_KEY` entries before injecting the resolved key.

```go
func sanitizeEnv(env []string) []string {
    var result []string
    for _, e := range env {
        if !strings.HasPrefix(e, "README_API_KEY=") {
            result = append(result, e)
        }
    }
    return result
}
```

### 3. Configuration Integration

Add to `internal/config/config.go`:

```go
type ReadMeConfig struct {
    APIKey         string `mapstructure:"apiKey"`
    APIKeyEnv      string `mapstructure:"apiKeyEnv"`  // Env var name for API key
    Branch         string `mapstructure:"branch"`
    Slug           string `mapstructure:"slug"`
    UseSpecVersion bool   `mapstructure:"useSpecVersion"`
}

type Config struct {
    Enrich  EnrichConfig  `mapstructure:"enrich"`
    Output  OutputConfig  `mapstructure:"output"`
    Extract ExtractConfig `mapstructure:"extract"`
    ReadMe  ReadMeConfig  `mapstructure:"readme"`
}
```

### 4. CLI Integration

Add flags to `publish` command:

```go
// Security: API key should be supplied via README_API_KEY env var, not CLI flag
// to prevent leaking in process listings (ps, /proc/<pid>/cmdline)
publishCmd.Flags().String("readme-branch", "", "ReadMe version/branch")
publishCmd.Flags().String("readme-slug", "", "ReadMe API slug")
publishCmd.Flags().Bool("readme-use-spec-version", false, "Use OpenAPI version as ReadMe version")
```

**Security Note:** The `--readme-api-key` flag is intentionally NOT provided. API keys must be supplied via the `README_API_KEY` environment variable to prevent credential leakage via process listings. The publisher automatically reads from this environment variable.

### 5. Dry Run Mode

Support `--dry-run` to preview what would be uploaded without making changes:

```go
if opts.DryRun {
    return &PublishResult{
        Path:         p.buildLocation(opts.ReadMe),
        Format:       format,
        BytesWritten: len(data),
        Message:      fmt.Sprintf("[DRY RUN] Would upload %d bytes to %s", len(data), location),
    }, nil
}
```

---

## Verification

```bash
# Run unit tests
go test ./internal/publisher/... -v

# Run linter
golangci-lint run ./internal/publisher/...

# Manual integration test (requires rdme CLI)
npm install -g rdme
export README_API_KEY="your-api-key"
go run . publish ./openapi.yaml --to readme --readme-slug my-api
```

---

## References

- [rdme CLI Documentation](https://github.com/readmeio/rdme)
- [ReadMe API Documentation](https://docs.readme.com/docs/rdme)
- Original PR: #5
