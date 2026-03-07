# go-zero Framework Support Design

> **Goal:** Add go-zero framework support to generate OpenAPI specs from go-zero API definition files.

**Architecture:** Implement go-zero extractor following the Spring Boot pattern: Detector → Patcher → Generator. The generator uses `goctl api swagger` to generate Swagger 2.0 specs, then converts to OpenAPI 3.0 using `openapi2conv.ToV3()`.

**Tech Stack:** Go, kin-openapi (openapi2conv), goctl CLI

---

## Overview

go-zero is a popular microservices framework in Go. This feature enables spec-forge to extract OpenAPI specifications from go-zero projects using the official `goctl` tool.

### Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| **Detect via .api files + go.mod** | go-zero projects use `.api` files for API definitions and require go-zero dependency |
| **Use goctl CLI** | Official tool, maintained by go-zero team, generates accurate specs |
| **Swagger 2.0 → OAS 3.0 conversion** | goctl generates Swagger 2.0; kin-openapi's openapi2conv handles conversion |
| **Patcher installs goctl** | Complete support: auto-detect missing goctl and guide installation |
| **NoGoZeroExtractor in this PR** | Context extraction deferred to next PR to keep scope manageable |

---

## Architecture

### Component Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                    go-zero Project                               │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │ *.api files │  │  go.mod     │  │ go-zero dependency      │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│ [Detector]                                                      │
│  - Find .api files in project                                   │
│  - Parse go.mod for github.com/zeromicro/go-zero                │
│  - Extract go-zero version                                      │
└─────────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│ [Patcher]                                                       │
│  - Check if goctl is installed                                  │
│  - Return installation hint or auto-install (configurable)      │
└─────────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│ [Generator]                                                     │
│  1. Execute: goctl api swagger --api api/api.api --dir ./doc    │
│  2. Read swagger.json (Swagger 2.0)                             │
│  3. Convert: openapi2conv.ToV3() → OpenAPI 3.0                  │
│  4. Save as openapi.json                                        │
└─────────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│ [Enricher] (existing, no changes needed)                        │
│  - Process OpenAPI 3.0 spec                                     │
│  - Generate AI descriptions                                     │
└─────────────────────────────────────────────────────────────────┘
```

---

## API Reference

### goctl Commands

```bash
# Install goctl
go install github.com/zeromicro/go-zero/tools/goctl@latest

# Generate Swagger 2.0 spec
goctl api swagger --api api/api.api --dir ./doc

# Output: doc/swagger.json (Swagger 2.0 format)
```

**Reference:** https://go-zero.dev/reference/cli-guide/swagger/

### Swagger 2.0 → OpenAPI 3.0 Conversion

```go
import (
    "github.com/getkin/kin-openapi/openapi2"
    "github.com/getkin/kin-openapi/openapi2conv"
    "github.com/getkin/kin-openapi/openapi3"
)

// Load Swagger 2.0
swagger2Doc, err := openapi2.NewLoader().LoadFromFile("swagger.json")
if err != nil {
    return nil, fmt.Errorf("failed to load swagger: %w", err)
}

// Convert to OpenAPI 3.0
openAPIDoc, err := openapi2conv.ToV3(swagger2Doc)
if err != nil {
    return nil, fmt.Errorf("failed to convert to OpenAPI 3.0: %w", err)
}

// Use openAPIDoc (*openapi3.T)
```

---

## Implementation

### File Structure

```
internal/extractor/
├── spring/              # Existing Spring Boot implementation
├── gozero/              # NEW go-zero implementation
│   ├── detector.go      # Project detection
│   ├── detector_test.go
│   ├── patcher.go       # goctl check/install
│   ├── patcher_test.go
│   ├── generator.go     # goctl execution + conversion
│   ├── generator_test.go
│   └── goctl.go         # goctl tool utilities
└── types.go             # Extended ProjectInfo
```

### Core Types

```go
// ProjectInfo extensions (in types.go)
type ProjectInfo struct {
    // ... existing fields ...

    // Framework type: "spring" or "gozero"
    Framework string

    // go-zero specific
    APIFiles       []string // List of .api files found
    GoZeroVersion  string   // go-zero version from go.mod
    HasGoctl       bool     // Whether goctl is installed
}

// gozero package types
package gozero

// Detector detects go-zero projects
type Detector struct{}

// Patcher checks and optionally installs goctl
type Patcher struct {
    exec executor.Interface
}

type PatchResult struct {
    GoctlInstalled     bool
    GoctlVersion       string
    InstallationOutput string
}

// Generator generates OpenAPI specs from go-zero projects
type Generator struct {
    exec executor.Interface
}
```

### Detector Implementation

```go
package gozero

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "github.com/spencercjh/spec-forge/internal/extractor"
)

const (
    FrameworkName = "gozero"
    GoModFile     = "go.mod"
    GoZeroModule  = "github.com/zeromicro/go-zero"
)

// Detector detects go-zero projects
type Detector struct{}

// NewDetector creates a new Detector
func NewDetector() *Detector {
    return &Detector{}
}

// Detect analyzes a project and returns info if it's a go-zero project
func (d *Detector) Detect(projectPath string) (*extractor.ProjectInfo, error) {
    absPath, err := filepath.Abs(projectPath)
    if err != nil {
        return nil, fmt.Errorf("failed to resolve path: %w", err)
    }

    // Check for go.mod
    goModPath := filepath.Join(absPath, GoModFile)
    if _, err := os.Stat(goModPath); err != nil {
        return nil, fmt.Errorf("no go.mod found in %s", absPath)
    }

    // Parse go.mod for go-zero dependency
    version, err := d.parseGoZeroVersion(goModPath)
    if err != nil {
        return nil, fmt.Errorf("failed to parse go.mod: %w", err)
    }
    if version == "" {
        return nil, fmt.Errorf("no go-zero dependency found in go.mod")
    }

    // Find .api files
    apiFiles, err := d.findAPIFiles(absPath)
    if err != nil {
        return nil, fmt.Errorf("failed to find .api files: %w", err)
    }
    if len(apiFiles) == 0 {
        return nil, fmt.Errorf("no .api files found in %s", absPath)
    }

    return &extractor.ProjectInfo{
        Framework:     FrameworkName,
        BuildTool:     "go",
        BuildFilePath: goModPath,
        GoZeroVersion: version,
        APIFiles:      apiFiles,
    }, nil
}

func (d *Detector) parseGoZeroVersion(goModPath string) (string, error) {
    // Read and parse go.mod
    // Look for: github.com/zeromicro/go-zero v1.x.x
    content, err := os.ReadFile(goModPath)
    if err != nil {
        return "", err
    }

    lines := strings.Split(string(content), "\n")
    for _, line := range lines {
        line = strings.TrimSpace(line)
        if strings.HasPrefix(line, GoZeroModule+" ") {
            parts := strings.Fields(line)
            if len(parts) >= 2 {
                return parts[1], nil
            }
        }
    }
    return "", nil
}

func (d *Detector) findAPIFiles(projectPath string) ([]string, error) {
    var apiFiles []string

    err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        if !info.IsDir() && strings.HasSuffix(path, ".api") {
            // Exclude vendor directory
            if strings.Contains(path, "/vendor/") {
                return nil
            }
            apiFiles = append(apiFiles, path)
        }
        return nil
    })

    return apiFiles, err
}
```

### Patcher Implementation

```go
package gozero

import (
    "context"
    "fmt"

    "github.com/spencercjh/spec-forge/internal/executor"
    "github.com/spencercjh/spec-forge/internal/extractor"
)

// Patcher checks and installs goctl
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

// Patch checks goctl installation and returns installation info
func (p *Patcher) Patch(ctx context.Context, projectPath string, info *extractor.ProjectInfo, opts *extractor.PatchOptions) (*PatchResult, error) {
    result := &PatchResult{}

    // Check if goctl is installed
    version, err := p.checkGoctl(ctx)
    if err == nil {
        result.GoctlInstalled = true
        result.GoctlVersion = version
        return result, nil
    }

    // goctl not installed
    if opts != nil && opts.AutoInstall {
        // Auto-install goctl
        output, installErr := p.installGoctl(ctx)
        if installErr != nil {
            return nil, fmt.Errorf("goctl not installed and auto-install failed: %w\nInstall manually: go install github.com/zeromicro/go-zero/tools/goctl@latest", installErr)
        }
        result.GoctlInstalled = true
        result.InstallationOutput = output
        return result, nil
    }

    // Return error with installation hint
    return nil, fmt.Errorf("goctl not found: %w\n\nTo install goctl, run:\n  go install github.com/zeromicro/go-zero/tools/goctl@latest", err)
}

func (p *Patcher) checkGoctl(ctx context.Context) (string, error) {
    opts := &executor.ExecuteOptions{
        Command: "goctl",
        Args:    []string{"--version"},
        Timeout: defaultTimeout,
    }

    result, err := p.exec.Execute(ctx, opts)
    if err != nil {
        return "", err
    }

    return strings.TrimSpace(result.Stdout), nil
}

func (p *Patcher) installGoctl(ctx context.Context) (string, error) {
    opts := &executor.ExecuteOptions{
        Command: "go",
        Args:    []string{"install", "github.com/zeromicro/go-zero/tools/goctl@latest"},
        Timeout: 5 * time.Minute,
    }

    result, err := p.exec.Execute(ctx, opts)
    if err != nil {
        return "", err
    }

    return result.Stdout, nil
}
```

### Generator Implementation

```go
package gozero

import (
    "context"
    "fmt"
    "os"
    "path/filepath"

    "github.com/getkin/kin-openapi/openapi2"
    "github.com/getkin/kin-openapi/openapi2conv"
    "github.com/getkin/kin-openapi/openapi3"

    "github.com/spencercjh/spec-forge/internal/executor"
    "github.com/spencercjh/spec-forge/internal/extractor"
)

// Generator generates OpenAPI specs from go-zero projects
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

// Generate generates OpenAPI spec from go-zero project
func (g *Generator) Generate(ctx context.Context, projectPath string, info *extractor.ProjectInfo, opts *extractor.GenerateOptions) (*extractor.GenerateResult, error) {
    if len(info.APIFiles) == 0 {
        return nil, fmt.Errorf("no .api files found")
    }

    // Use the first .api file (typically api/api.api)
    // TODO: Handle multiple .api files if needed
    apiFile := info.APIFiles[0]

    // Create output directory
    outputDir := opts.OutputDir
    if outputDir == "" {
        outputDir = filepath.Join(projectPath, "doc")
    }
    if err := os.MkdirAll(outputDir, 0755); err != nil {
        return nil, fmt.Errorf("failed to create output directory: %w", err)
    }

    // Generate Swagger 2.0 using goctl
    swaggerPath, err := g.generateSwagger(ctx, apiFile, outputDir, opts)
    if err != nil {
        return nil, fmt.Errorf("failed to generate swagger: %w", err)
    }

    // Convert Swagger 2.0 to OpenAPI 3.0
    openAPIPath, err := g.convertToOpenAPI3(swaggerPath, outputDir, opts)
    if err != nil {
        return nil, fmt.Errorf("failed to convert to OpenAPI 3.0: %w", err)
    }

    return &extractor.GenerateResult{
        SpecFilePath: openAPIPath,
        Format:       opts.Format,
    }, nil
}

func (g *Generator) generateSwagger(ctx context.Context, apiFile, outputDir string, opts *extractor.GenerateOptions) (string, error) {
    execOpts := &executor.ExecuteOptions{
        Command: "goctl",
        Args: []string{
            "api", "swagger",
            "--api", apiFile,
            "--dir", outputDir,
        },
        Timeout: opts.Timeout,
    }

    _, err := g.exec.Execute(ctx, execOpts)
    if err != nil {
        return "", fmt.Errorf("goctl api swagger failed: %w", err)
    }

    // goctl generates swagger.json in outputDir
    swaggerPath := filepath.Join(outputDir, "swagger.json")
    if _, err := os.Stat(swaggerPath); err != nil {
        return "", fmt.Errorf("swagger.json not generated at %s", swaggerPath)
    }

    return swaggerPath, nil
}

func (g *Generator) convertToOpenAPI3(swaggerPath, outputDir string, opts *extractor.GenerateOptions) (string, error) {
    // Load Swagger 2.0
    loader := openapi2.NewLoader()
    swagger2Doc, err := loader.LoadFromFile(swaggerPath)
    if err != nil {
        return "", fmt.Errorf("failed to load swagger.json: %w", err)
    }

    // Apply patches for known goctl bugs
    // See: https://github.com/zeromicro/go-zero/issues/5425-5428
    g.patchSwagger2Doc(swagger2Doc)

    // Convert to OpenAPI 3.0
    openAPIDoc, err := openapi2conv.ToV3(swagger2Doc)
    if err != nil {
        return "", fmt.Errorf("failed to convert: %w", err)
    }

    // Determine output filename
    outputFile := opts.OutputFile
    if outputFile == "" {
        outputFile = "openapi"
    }

    // Marshal and save
    var data []byte
    ext := ".json"

    if opts.Format == "yaml" || opts.Format == "yml" {
        yamlData, err := openAPIDoc.MarshalYAML()
        if err != nil {
            return "", fmt.Errorf("failed to marshal YAML: %w", err)
        }
        data, err = yaml.Marshal(yamlData)
        if err != nil {
            return "", fmt.Errorf("failed to encode YAML: %w", err)
        }
        ext = ".yaml"
    } else {
        data, err = openAPIDoc.MarshalJSON()
        if err != nil {
            return "", fmt.Errorf("failed to marshal JSON: %w", err)
        }
    }

    outputPath := filepath.Join(outputDir, outputFile+ext)
    if err := os.WriteFile(outputPath, data, 0644); err != nil {
        return "", fmt.Errorf("failed to write output: %w", err)
    }

    return outputPath, nil
}
```

### Integration in cmd/generate.go

```go
// Modified detection logic in runGenerate
func detectProject(projectPath string) (*extractor.ProjectInfo, string, error) {
    // Try Spring Boot first
    springDetector := spring.NewDetector()
    info, err := springDetector.Detect(projectPath)
    if err == nil {
        return info, "spring", nil
    }

    // Try go-zero
    gozeroDetector := gozero.NewDetector()
    info, err = gozeroDetector.Detect(projectPath)
    if err == nil {
        return info, "gozero", nil
    }

    return nil, "", fmt.Errorf("no supported framework detected (Spring Boot or go-zero)")
}
```

---

## Testing

### Unit Tests

```go
// detector_test.go
func TestDetector_Detect(t *testing.T) {
    tests := []struct {
        name        string
        setup       func(t *testing.T) string
        wantErr     bool
        wantVersion string
    }{
        {
            name: "valid go-zero project",
            setup: func(t *testing.T) string {
                dir := t.TempDir()
                // Create go.mod with go-zero
                goMod := `module test

go 1.21

require github.com/zeromicro/go-zero v1.6.0
`
                os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0644)
                // Create .api file
                os.WriteFile(filepath.Join(dir, "api.api"), []byte("service test-api {}"), 0644)
                return dir
            },
            wantErr:     false,
            wantVersion: "v1.6.0",
        },
        // ... more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            dir := tt.setup(t)
            d := NewDetector()
            info, err := d.Detect(dir)
            // ... assertions
        })
    }
}
```

### E2E Tests

Create test project in `integration-tests/gozero-demo/`:

```
integration-tests/
├── gozero-demo/
│   ├── go.mod
│   ├── api/
│   │   └── api.api
│   └── gozero_demo.go
```

---

## Dependencies

Add to `go.mod`:

```go
require (
    // ... existing dependencies ...
    github.com/getkin/kin-openapi/openapi2 v0.128.0
)
```

---

## Bug Fixes for goctl api swagger

goctl's swagger generation has known bugs that produce invalid/semantically incorrect specs. We patch these after generation but before conversion to OpenAPI 3.0.

References:
- [#5426](https://github.com/zeromicro/go-zero/issues/5426): Nested Array Types Missing `items`
- [#5427](https://github.com/zeromicro/go-zero/issues/5427): `form:"-"` Not Treated as Ignore
- [#5428](https://github.com/zeromicro/go-zero/issues/5428): Orphan Path Parameters Generated

### Patch Implementation

```go
// patchSwagger2Doc applies fixes for known goctl bugs
func (g *Generator) patchSwagger2Doc(doc *openapi2.T) {
    if doc == nil {
        return
    }

    // Fix #5426: Nested arrays missing items
    // Fix #5427: Remove parameters with name "-"
    g.patchDefinitions(doc.Definitions)

    // Fix #5428: Remove orphan path parameters
    g.patchPaths(doc.Paths)
}

// patchDefinitions fixes schema definitions
func (g *Generator) patchDefinitions(defs map[string]*openapi2.Schema) {
    for _, schema := range defs {
        g.patchSchema(schema)
    }
}

// patchSchema recursively patches schema properties
func (g *Generator) patchSchema(schema *openapi2.Schema) {
    if schema == nil {
        return
    }

    // Fix #5426: Array types missing items
    if schema.Type == "array" && schema.Items == nil {
        // Default to object if items is missing
        schema.Items = &openapi2.SchemaRef{
            Value: &openapi2.Schema{Type: "object"},
        }
    }

    // Recursively patch nested schemas
    for _, prop := range schema.Properties {
        g.patchSchema(prop.Value)
    }

    // Patch items if present
    if schema.Items != nil {
        g.patchSchema(schema.Items.Value)
    }

    // Patch additional properties
    if schema.AdditionalProperties != nil {
        g.patchSchema(schema.AdditionalProperties.Value)
    }
}

// patchPaths fixes path items and parameters
func (g *Generator) patchPaths(paths map[string]*openapi2.PathItem) {
    for pathStr, pathItem := range paths {
        if pathItem == nil {
            continue
        }

        // Extract actual path parameters from URL
        pathParams := g.extractPathParams(pathStr)

        // Patch operations
        g.patchOperation(pathItem.Get, pathParams)
        g.patchOperation(pathItem.Post, pathParams)
        g.patchOperation(pathItem.Put, pathParams)
        g.patchOperation(pathItem.Delete, pathParams)
        g.patchOperation(pathItem.Patch, pathParams)
        g.patchOperation(pathItem.Head, pathParams)
        g.patchOperation(pathItem.Options, pathParams)
    }
}

// patchOperation fixes operation parameters
func (g *Generator) patchOperation(op *openapi2.Operation, validPathParams []string) {
    if op == nil {
        return
    }

    var filteredParams openapi2.Parameters
    for _, param := range op.Parameters {
        if param == nil || param.Value == nil {
            continue
        }

        p := param.Value

        // Fix #5427: Skip parameters with name "-"
        if p.Name == "-" {
            continue
        }

        // Fix #5428: Remove path params not in URL
        if p.In == "path" && !g.containsString(validPathParams, p.Name) {
            continue
        }

        filteredParams = append(filteredParams, param)
    }

    op.Parameters = filteredParams
}

// extractPathParams extracts {param} or :param from URL path
func (g *Generator) extractPathParams(path string) []string {
    var params []string
    // Match {param} syntax
    re := regexp.MustCompile(`\{(\w+)\}`)
    matches := re.FindAllStringSubmatch(path, -1)
    for _, m := range matches {
        if len(m) > 1 {
            params = append(params, m[1])
        }
    }
    return params
}

func (g *Generator) containsString(slice []string, s string) bool {
    for _, item := range slice {
        if item == s {
            return true
        }
    }
    return false
}
```

---

## Future Improvements

### 1. Multiple .api Files Support

Some projects may have multiple API definition files. Handle by:
- Merging multiple swagger.json files
- Or prompting user to select main API file

### 2. specctx.GoZeroExtractor

Next PR: Extract context from go-zero source code:

```go
type GoZeroExtractor struct{}

func (e *GoZeroExtractor) Extract(ctx context.Context, projectPath string, spec *openapi3.T) (*EnrichmentContext, error) {
    // 1. Parse .api files for handler documentation
    // 2. Extract Go struct comments
    // 3. Fill EnrichmentContext with descriptions
}
```

### 3. goctl Version Check

Validate goctl version compatibility with go-zero version in go.mod.

---

## Verification

```bash
# Run tests
go test ./internal/extractor/gozero/... -v

# Run linter
golangci-lint run ./internal/extractor/gozero/...

# Manual integration test
go build -o ./build/spec-forge .

# Test with go-zero demo project
./build/spec-forge generate ./integration-tests/gozero-demo -v
```

---

## References

- [go-zero Documentation](https://go-zero.dev/)
- [goctl Swagger Guide](https://go-zero.dev/reference/cli-guide/swagger/)
- [kin-openapi openapi2conv](https://pkg.go.dev/github.com/getkin/kin-openapi/openapi2conv)
