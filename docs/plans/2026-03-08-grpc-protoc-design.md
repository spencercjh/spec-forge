# gRPC-Protoc Framework Support Design

> **Status:** ✅ Implemented (2026-03-08)
>
> **Goal:** Add gRPC/protobuf support for native protoc projects (not buf-managed) to generate OpenAPI specs using `protoc-gen-connect-openapi`.

**Architecture:** Implement grpc-protoc extractor following the Spring Boot/go-zero pattern: Detector → Patcher → Generator.

**Tech Stack:** Go, protoc, protoc-gen-connect-openapi

---

## Overview

This feature enables spec-forge to extract OpenAPI specifications from gRPC projects that use native `protoc` (not buf). This is **Phase 1** of gRPC support - buf-managed projects will be handled separately in Phase 2.

### Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| **Only native protoc projects** | Buf projects have their own workflow; we'll handle them in Phase 2 |
| **Use protoc-gen-connect-openapi** | Most actively maintained, supports `google.api.http` annotations |
| **Explicit rejection of buf projects** | Clear error message guides users to appropriate workflow |
| **Extensible import paths** | Projects may have proto files in multiple locations (proto/, third_party/, vendor/) |
| **No auto-install** | protoc installation is platform-specific; provide clear instructions instead |

---

## Architecture

### Component Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                    gRPC Project (Native protoc)                  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │ *.proto     │  │ NO buf.yaml │  │ go.mod (optional)       │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│ [Detector]                                                      │
│  - Check buf.yaml does NOT exist                                │
│  - Find all .proto files                                        │
│  - Detect import paths (proto/, third_party/, etc.)             │
│  - Check for google.api.http annotations                        │
└─────────────────────────────────────────────────────────────────┘
                            │
                            ▼
        ┌───────────────────────────────────┐
        │ Buf project detected?             │
        │ Return ErrBufProjectDetected      │
        └───────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│ [Patcher]                                                       │
│  - Check protoc is installed                                    │
│  - Check protoc-gen-connect-openapi is installed                │
│  - Return installation hints if missing                         │
└─────────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│ [Generator]                                                     │
│  1. Build protoc command with -I flags                          │
│  2. Execute: protoc --connect-openapi_out=... *.proto           │
│  3. Return path to generated openapi.json                       │
└─────────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│ [Enricher] (existing, no changes needed)                        │
│  - Process OpenAPI 3.x spec                                     │
│  - Generate AI descriptions                                     │
└─────────────────────────────────────────────────────────────────┘
```

---

## API Reference

### protoc-gen-connect-openapi Commands

```bash
# Install the plugin
go install github.com/sudorandom/protoc-gen-connect-openapi@latest

# Generate OpenAPI from proto files
protoc -Iproto -Ithird_party \
  --connect-openapi_out=./gen/openapi \
  --connect-openapi_opt=features=google.api.http \
  proto/*.proto

# Output: gen/openapi/*.openapi.json (or .yaml)
```

**Reference:** https://github.com/sudorandom/protoc-gen-connect-openapi

**Features:**
- Supports `google.api.http` annotations (gRPC-Gateway style)
- Generates OpenAPI 3.1
- Handles Connect RPC, gRPC, and gRPC-Web

---

## Implementation

### File Structure

```
internal/extractor/
├── spring/                  # Existing Spring Boot implementation
├── gozero/                  # Existing go-zero implementation
├── grpcprotoc/              # NEW gRPC-protoc implementation
│   ├── detector.go          # Project detection
│   ├── detector_test.go
│   ├── patcher.go           # protoc + plugin check
│   ├── patcher_test.go
│   ├── generator.go         # protoc execution
│   ├── generator_test.go
│   └── grpcprotoc.go        # Package types and constants
└── types.go                 # Existing types (no changes)
```

### Core Types

```go
// Package grpcprotoc provides gRPC-protoc framework extraction
package grpcprotoc

const (
    FrameworkName = "grpc-protoc"
)

// Info holds gRPC-protoc specific project information
type Info struct {
    ProtoFiles        []string // All .proto files found
    ServiceProtoFiles []string // Proto files with service definitions (main entry points)
    ProtoRoot         string   // Root directory containing proto files
    HasGoogleAPI      bool     // Whether google/api/annotations.proto is imported
    HasBuf            bool     // Whether buf.yaml exists (should be false)
    ImportPaths       []string // Detected import paths
}

// Detector detects native protoc gRPC projects
type Detector struct{}

// Patcher checks protoc and protoc-gen-connect-openapi installation
type Patcher struct {
    exec executor.Interface
}

type PatchResult struct {
    ProtocInstalled                  bool
    ProtocVersion                    string
    ProtocGenConnectOpenAPIInstalled bool
    ProtocGenConnectOpenAPIVersion   string
}

// Generator generates OpenAPI specs from proto files
type Generator struct {
    exec executor.Interface
}
```

### Detector Implementation

```go
package grpcprotoc

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "github.com/spencercjh/spec-forge/internal/extractor"
)

var ErrBufProjectDetected = fmt.Errorf(
    "buf.yaml detected: this is a buf-managed project. " +
        "spec-forge currently only supports native protoc projects for gRPC. " +
        "Please use 'buf generate' with protoc-gen-connect-openapi, " +
        "then use 'spec-forge enrich' on the generated OpenAPI spec")

// Detector detects native protoc gRPC projects
type Detector struct{}

// NewDetector creates a new Detector
func NewDetector() *Detector {
    return &Detector{}
}

// Detect analyzes a project and returns info if it's a native protoc gRPC project
func (d *Detector) Detect(projectPath string) (*extractor.ProjectInfo, error) {
    absPath, err := filepath.Abs(projectPath)
    if err != nil {
        return nil, fmt.Errorf("failed to resolve path: %w", err)
    }

    // Check for buf.yaml - if exists, this is not our target
    bufYamlPath := filepath.Join(absPath, "buf.yaml")
    if _, err := os.Stat(bufYamlPath); err == nil {
        return nil, ErrBufProjectDetected
    }

    // Find all .proto files
    protoFiles, importPaths, err := d.findProtoFiles(absPath)
    if err != nil {
        return nil, fmt.Errorf("failed to find proto files: %w", err)
    }
    if len(protoFiles) == 0 {
        return nil, fmt.Errorf("no .proto files found in %s", absPath)
    }

    // Check for google.api.http usage
    hasGoogleAPI := d.hasGoogleAPIAnnotations(protoFiles)

    // Find proto files with service definitions (main entry points)
    serviceProtoFiles := d.findServiceProtoFiles(protoFiles)

    return &extractor.ProjectInfo{
        Framework:     FrameworkName,
        BuildTool:     "protoc",
        BuildFilePath: protoFiles[0], // Use first proto file as reference
        FrameworkData: &Info{
            ProtoFiles:        protoFiles,
            ServiceProtoFiles: serviceProtoFiles,
            ProtoRoot:         absPath,
            HasGoogleAPI:      hasGoogleAPI,
            HasBuf:            false,
            ImportPaths:       importPaths,
        },
    }, nil
}

func (d *Detector) findServiceProtoFiles(protoFiles []string) []string {
    var serviceFiles []string
    for _, protoFile := range protoFiles {
        if d.hasServiceDefinition(protoFile) {
            serviceFiles = append(serviceFiles, protoFile)
        }
    }
    return serviceFiles
}

func (d *Detector) hasServiceDefinition(protoFile string) bool {
    content, err := os.ReadFile(protoFile)
    if err != nil {
        return false
    }
    // Check for service keyword at the beginning of a line
    for _, line := range strings.Split(string(content), "\n") {
        line = strings.TrimSpace(line)
        if strings.HasPrefix(line, "service ") || strings.HasPrefix(line, "service\t") {
            return true
        }
    }
    return false
}

func (d *Detector) findProtoFiles(projectPath string) ([]string, []string, error) {
    var protoFiles []string
    importPaths := []string{projectPath} // Always include project root

    err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        if info.IsDir() {
            // Skip vendor and common non-proto directories
            if info.Name() == "vendor" || info.Name() == ".git" || info.Name() == "node_modules" {
                return filepath.SkipDir
            }
            // Track potential import paths
            if info.Name() == "proto" || info.Name() == "third_party" || info.Name() == "protos" {
                importPaths = append(importPaths, path)
            }
            return nil
        }
        if strings.HasSuffix(path, ".proto") {
            protoFiles = append(protoFiles, path)
        }
        return nil
    })

    return protoFiles, importPaths, err
}

func (d *Detector) hasGoogleAPIAnnotations(protoFiles []string) bool {
    for _, f := range protoFiles {
        content, err := os.ReadFile(f)
        if err != nil {
            continue
        }
        if strings.Contains(string(content), "google/api/annotations.proto") {
            return true
        }
    }
    return false
}
```

### Patcher Implementation

```go
package grpcprotoc

import (
    "context"
    "fmt"
    "strings"
    "time"

    "github.com/spencercjh/spec-forge/internal/executor"
    "github.com/spencercjh/spec-forge/internal/extractor"
)

var (
    ErrProtocNotInstalled = fmt.Errorf(
        "protoc not found. Install from: https://github.com/protocolbuffers/protobuf/releases")

    ErrProtocGenConnectOpenAPINotInstalled = fmt.Errorf(
        "protoc-gen-connect-openapi not found. Install with: " +
            "go install github.com/sudorandom/protoc-gen-connect-openapi@latest")
)

// Patcher checks protoc and protoc-gen-connect-openapi installation
type Patcher struct {
    exec executor.Interface
}

// NewPatcher creates a new Patcher
func NewPatcher() *Patcher {
    return &Patcher{exec: executor.NewExecutor()}
}

// NewPatcherWithExecutor creates a Patcher with custom executor (for testing)
func NewPatcherWithExecutor(exec executor.Interface) *Patcher {
    return &Patcher{exec: exec}
}

// Patch checks protoc and protoc-gen-connect-openapi installation
func (p *Patcher) Patch(ctx context.Context, projectPath string, info *extractor.ProjectInfo, opts *extractor.PatchOptions) (*PatchResult, error) {
    result := &PatchResult{}

    // Check protoc
    version, err := p.checkProtoc(ctx)
    if err != nil {
        return nil, fmt.Errorf("%w: %v", ErrProtocNotInstalled, err)
    }
    result.ProtocInstalled = true
    result.ProtocVersion = version

    // Check protoc-gen-connect-openapi
    pluginVersion, err := p.checkProtocGenConnectOpenAPI(ctx)
    if err != nil {
        return nil, fmt.Errorf("%w: %v", ErrProtocGenConnectOpenAPINotInstalled, err)
    }
    result.ProtocGenConnectOpenAPIInstalled = true
    result.ProtocGenConnectOpenAPIVersion = pluginVersion

    return result, nil
}

func (p *Patcher) checkProtoc(ctx context.Context) (string, error) {
    opts := &executor.ExecuteOptions{
        Command: "protoc",
        Args:    []string{"--version"},
        Timeout: 30 * time.Second,
    }

    result, err := p.exec.Execute(ctx, opts)
    if err != nil {
        return "", err
    }

    return strings.TrimSpace(result.Stdout), nil
}

func (p *Patcher) checkProtocGenConnectOpenAPI(ctx context.Context) (string, error) {
    opts := &executor.ExecuteOptions{
        Command: "protoc-gen-connect-openapi",
        Args:    []string{"--version"},
        Timeout: 30 * time.Second,
    }

    result, err := p.exec.Execute(ctx, opts)
    if err != nil {
        return "", err
    }

    return strings.TrimSpace(result.Stdout), nil
}
```

### Generator Implementation

```go
package grpcprotoc

import (
    "context"
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "time"

    "github.com/spencercjh/spec-forge/internal/executor"
    "github.com/spencercjh/spec-forge/internal/extractor"
)

// Generator generates OpenAPI specs from proto files
type Generator struct {
    exec executor.Interface
}

// NewGenerator creates a new Generator
func NewGenerator() *Generator {
    return &Generator{exec: executor.NewExecutor()}
}

// NewGeneratorWithExecutor creates a Generator with custom executor (for testing)
func NewGeneratorWithExecutor(exec executor.Interface) *Generator {
    return &Generator{exec: exec}
}

// Generate generates OpenAPI spec from proto files
func (g *Generator) Generate(ctx context.Context, projectPath string, info *extractor.ProjectInfo, opts *extractor.GenerateOptions) (*extractor.GenerateResult, error) {
    grpcInfo, ok := info.FrameworkData.(*Info)
    if !ok {
        return nil, fmt.Errorf("invalid FrameworkData type: expected *grpcprotoc.Info")
    }

    // Only process service proto files (those with service definitions)
    if len(grpcInfo.ServiceProtoFiles) == 0 {
        return nil, fmt.Errorf("no proto files with service definitions found")
    }

    // Create output directory
    outputDir := opts.OutputDir
    if outputDir == "" {
        outputDir = filepath.Join(projectPath, "gen", "openapi")
    }
    if err := os.MkdirAll(outputDir, 0755); err != nil {
        return nil, fmt.Errorf("failed to create output directory: %w", err)
    }

    // Build protoc command
    args := g.buildProtocArgs(grpcInfo, outputDir, opts)

    // Execute protoc
    execOpts := &executor.ExecuteOptions{
        Command: "protoc",
        Args:    args,
        Timeout: opts.Timeout,
        Dir:     projectPath,
    }

    _, err := g.exec.Execute(ctx, execOpts)
    if err != nil {
        return nil, fmt.Errorf("protoc execution failed: %w", err)
    }

    // Find generated OpenAPI file
    outputFile := g.findOutputFile(outputDir)
    if outputFile == "" {
        return nil, fmt.Errorf("no OpenAPI file generated in %s", outputDir)
    }

    return &extractor.GenerateResult{
        SpecFilePath: outputFile,
        Format:       opts.Format,
    }, nil
}

func (g *Generator) buildProtocArgs(info *Info, outputDir string, opts *extractor.GenerateOptions) []string {
    var args []string

    // Add import paths
    importPaths := info.ImportPaths

    // Merge extra import paths from CLI flags
    importPaths = append(importPaths, opts.ProtoImportPaths...)

    // Add -I flags (deduplicated)
    seen := make(map[string]bool)
    for _, path := range importPaths {
        if !seen[path] {
            seen[path] = true
            args = append(args, "-I"+path)
        }
    }

    // Add connect-openapi output
    outputArg := fmt.Sprintf("--connect-openapi_out=%s", outputDir)
    args = append(args, outputArg)

    // Add format option (if YAML requested)
    if opts.Format == "yaml" || opts.Format == "yml" {
        args = append(args, "--connect-openapi_opt=format=yaml")
    }

    // CONDITIONAL: Only enable google.api.http if detected in project
    if info.HasGoogleAPI {
        args = append(args, "--connect-openapi_opt=features=google.api.http")
    }

    // Add only service proto files (those with service definitions)
    // to avoid duplicate definition errors from importing common proto files
    for _, protoFile := range info.ServiceProtoFiles {
        args = append(args, protoFile)
    }

    return args
}

func (g *Generator) findOutputFile(outputDir string) string {
    // protoc-gen-connect-openapi generates files like: <proto_filename>.openapi.json
    entries, err := os.ReadDir(outputDir)
    if err != nil {
        return ""
    }

    for _, entry := range entries {
        name := entry.Name()
        if strings.HasSuffix(name, ".openapi.json") || strings.HasSuffix(name, ".openapi.yaml") {
            return filepath.Join(outputDir, name)
        }
    }

    return ""
}
```

### CLI Integration

```go
// cmd/generate.go - Add new flag
func init() {
    generateCmd.Flags().StringSlice("proto-import-path", nil,
        "Additional import paths for protoc (-I flags), can be specified multiple times")
}

// In runGenerate(), pass proto import paths to detector
func runGenerate(cmd *cobra.Command, args []string) error {
    // ... existing code ...

    protoImportPaths, _ := cmd.Flags().GetStringSlice("proto-import-path")

    // Pass to generate options
    opts := &extractor.GenerateOptions{
        // ... existing options ...
        FrameworkData: map[string]any{
            "proto_import_paths": protoImportPaths,
        },
    }

    // ... rest of generation logic ...
}
```

---

## Testing

### Unit Tests

```go
// detector_test.go
func TestDetector_Detect(t *testing.T) {
    tests := []struct {
        name    string
        setup   func(t *testing.T) string
        wantErr bool
        errMsg  string
    }{
        {
            name: "valid protoc project",
            setup: func(t *testing.T) string {
                dir := t.TempDir()
                // Create proto file
                os.WriteFile(filepath.Join(dir, "user.proto"), []byte(`
syntax = "proto3";
package user;
message User { string name = 1; }
`), 0644)
                return dir
            },
            wantErr: false,
        },
        {
            name: "buf project should be rejected",
            setup: func(t *testing.T) string {
                dir := t.TempDir()
                os.WriteFile(filepath.Join(dir, "buf.yaml"), []byte("version: v1"), 0644)
                os.WriteFile(filepath.Join(dir, "user.proto"), []byte("syntax = \"proto3\";"), 0644)
                return dir
            },
            wantErr: true,
            errMsg:  "buf.yaml detected",
        },
        {
            name: "no proto files",
            setup: func(t *testing.T) string {
                return t.TempDir()
            },
            wantErr: true,
            errMsg:  "no .proto files found",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            dir := tt.setup(t)
            d := NewDetector()
            info, err := d.Detect(dir)
            // ... assertions ...
        })
    }
}
```

### E2E Test

Test with the demo project in `integration-tests/grpc-protoc-demo/`.

---

## Demo Project

Location: `integration-tests/grpc-protoc-demo/`

Structure:
```
grpc-protoc-demo/
├── proto/
│   ├── common.proto      # Common messages (pagination, response)
│   └── user.proto        # User service (CRUD operations)
├── third_party/
│   └── google/api/       # Google API HTTP annotations
│       ├── annotations.proto
│       └── http.proto
├── Makefile             # protoc commands
├── go.mod               # Go module
└── README.md            # Documentation
```

---

## Verification

```bash
# Run tests
go test ./internal/extractor/grpcprotoc/... -v

# Run linter
golangci-lint run ./internal/extractor/grpcprotoc/...

# Manual integration test with demo project
cd integration-tests/grpc-protoc-demo
make install-tools
make openapi

# Test with spec-forge (after implementation)
go build -o ./build/spec-forge .
./build/spec-forge generate ./integration-tests/grpc-protoc-demo -v
```

---

## Future Improvements

### Phase 2: Buf Support

```go
// grpcbuf package for buf-managed projects
package grpcbuf

// Detector checks for buf.yaml
// Generator runs: buf generate --template buf.gen.openapi.yaml
```

### Additional Features

1. **Multiple proto roots** - Support complex project structures
2. **Custom protoc plugins** - Allow other OpenAPI generators
3. **Proto validation** - Run protoc with --dry-run first

---

## References

- [protoc-gen-connect-openapi](https://github.com/sudorandom/protoc-gen-connect-openapi)
- [Protocol Buffers](https://protobuf.dev/)
- [gRPC-Gateway](https://github.com/grpc-ecosystem/grpc-gateway)
- [google.api.http annotation](https://github.com/googleapis/googleapis/blob/master/google/api/annotations.proto)
