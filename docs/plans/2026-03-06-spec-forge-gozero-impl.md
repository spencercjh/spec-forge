# go-zero Framework Support Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement go-zero framework support with Detector, Patcher, and Generator components.

**Architecture:** Follow Spring Boot pattern with three components: Detector (find .api files + go.mod), Patcher (check/install goctl), Generator (run goctl api swagger + convert Swagger 2.0 to OpenAPI 3.0).

**Tech Stack:** Go, kin-openapi/openapi2conv, goctl CLI, executor.Interface

---

## Task 1: Create gozero package structure

**Files:**
- Create: `internal/extractor/gozero/detector.go` (stub)
- Create: `internal/extractor/gozero/detector_test.go` (stub)
- Create: `internal/extractor/gozero/patcher.go` (stub)
- Create: `internal/extractor/gozero/patcher_test.go` (stub)
- Create: `internal/extractor/gozero/generator.go` (stub)
- Create: `internal/extractor/gozero/generator_test.go` (stub)

**Step 1: Create package directory and stub files**

```bash
mkdir -p internal/extractor/gozero
touch internal/extractor/gozero/{detector,patcher,generator}_test.go
```

**Step 2: Create detector.go stub**

```go
// Package gozero provides go-zero framework specific extraction functionality.
package gozero

import "github.com/spencercjh/spec-forge/internal/extractor"

// Detector detects go-zero projects.
type Detector struct{}

// NewDetector creates a new Detector instance.
func NewDetector() *Detector {
	return &Detector{}
}

// Detect analyzes a project and returns info if it's a go-zero project.
func (d *Detector) Detect(projectPath string) (*extractor.ProjectInfo, error) {
	return nil, nil
}
```

**Step 3: Create patcher.go stub**

```go
package gozero

import (
	"context"

	"github.com/spencercjh/spec-forge/internal/extractor"
)

// Patcher checks and installs goctl.
type Patcher struct{}

// PatchResult contains the result of patching.
type PatchResult struct {
	GoctlInstalled bool
}

// NewPatcher creates a new Patcher instance.
func NewPatcher() *Patcher {
	return &Patcher{}
}

// Patch checks goctl installation.
func (p *Patcher) Patch(ctx context.Context, projectPath string, info *extractor.ProjectInfo, opts *extractor.PatchOptions) (*PatchResult, error) {
	return nil, nil
}
```

**Step 4: Create generator.go stub**

```go
package gozero

import (
	"context"

	"github.com/spencercjh/spec-forge/internal/extractor"
)

// Generator generates OpenAPI specs from go-zero projects.
type Generator struct{}

// NewGenerator creates a new Generator instance.
func NewGenerator() *Generator {
	return &Generator{}
}

// Generate generates OpenAPI spec from go-zero project.
func (g *Generator) Generate(ctx context.Context, projectPath string, info *extractor.ProjectInfo, opts *extractor.GenerateOptions) (*extractor.GenerateResult, error) {
	return nil, nil
}
```

**Step 5: Create test stubs**

```go
// detector_test.go
package gozero

import "testing"

func TestNewDetector(t *testing.T) {
	d := NewDetector()
	if d == nil {
		t.Error("expected non-nil detector")
	}
}
```

```go
// patcher_test.go
package gozero

import "testing"

func TestNewPatcher(t *testing.T) {
	p := NewPatcher()
	if p == nil {
		t.Error("expected non-nil patcher")
	}
}
```

```go
// generator_test.go
package gozero

import "testing"

func TestNewGenerator(t *testing.T) {
	g := NewGenerator()
	if g == nil {
		t.Error("expected non-nil generator")
	}
}
```

**Step 6: Verify stubs compile**

Run:
```bash
go build ./internal/extractor/gozero/...
```

Expected: No errors

**Step 7: Run stub tests**

Run:
```bash
go test ./internal/extractor/gozero/... -v
```

Expected: 3 tests pass

**Step 8: Commit**

```bash
git add internal/extractor/gozero/
git commit -s -m "chore(gozero): create package structure with stubs"
```

---

## Task 2: Implement Detector with go.mod parsing

**Files:**
- Modify: `internal/extractor/gozero/detector.go`
- Modify: `internal/extractor/gozero/detector_test.go`

**Step 1: Write failing test for go.mod detection**

Add to `detector_test.go`:

```go
func TestDetector_parseGoZeroVersion(t *testing.T) {
	tests := []struct {
		name     string
		goMod    string
		expected string
	}{
		{
			name: "has go-zero dependency",
			goMod: `module test

go 1.21

require github.com/zeromicro/go-zero v1.6.0
`,
			expected: "v1.6.0",
		},
		{
			name: "no go-zero dependency",
			goMod: `module test

go 1.21

require github.com/gin-gonic/gin v1.9.0
`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			goModPath := filepath.Join(dir, "go.mod")
			os.WriteFile(goModPath, []byte(tt.goMod), 0644)

			d := NewDetector()
			version, err := d.parseGoZeroVersion(goModPath)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if version != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, version)
			}
		})
	}
}
```

Run:
```bash
go test ./internal/extractor/gozero/... -run TestDetector_parseGoZeroVersion -v
```

Expected: FAIL - "d.parseGoZeroVersion undefined"

**Step 2: Implement parseGoZeroVersion method**

Add to `detector.go`:

```go
import (
	"os"
	"strings"
)

const GoZeroModule = "github.com/zeromicro/go-zero"

func (d *Detector) parseGoZeroVersion(goModPath string) (string, error) {
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
```

Run:
```bash
go test ./internal/extractor/gozero/... -run TestDetector_parseGoZeroVersion -v
```

Expected: PASS

**Step 3: Write failing test for .api file detection**

Add to `detector_test.go`:

```go
func TestDetector_findAPIFiles(t *testing.T) {
	dir := t.TempDir()

	// Create .api files
	os.WriteFile(filepath.Join(dir, "api.api"), []byte("service {}"), 0644)
	os.WriteFile(filepath.Join(dir, "types.api"), []byte("type {}"), 0644)

	// Create non-.api file
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644)

	// Create vendor directory with .api file (should be excluded)
	os.MkdirAll(filepath.Join(dir, "vendor", "test"), 0755)
	os.WriteFile(filepath.Join(dir, "vendor", "test", "api.api"), []byte("service {}"), 0644)

	d := NewDetector()
	files, err := d.findAPIFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("expected 2 .api files, got %d", len(files))
	}
}
```

Run:
```bash
go test ./internal/extractor/gozero/... -run TestDetector_findAPIFiles -v
```

Expected: FAIL - "d.findAPIFiles undefined"

**Step 4: Implement findAPIFiles method**

Add to `detector.go`:

```go
import (
	"os"
	"path/filepath"
)

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

Run:
```bash
go test ./internal/extractor/gozero/... -run TestDetector_findAPIFiles -v
```

Expected: PASS

**Step 5: Write failing test for full Detect method**

Add to `detector_test.go`:

```go
func TestDetector_Detect(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T) string
		wantErr     bool
		wantVersion string
		wantFiles   int
	}{
		{
			name: "valid go-zero project",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				goMod := `module test

go 1.21

require github.com/zeromicro/go-zero v1.6.0
`
				os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0644)
				os.WriteFile(filepath.Join(dir, "api.api"), []byte("service test-api {}"), 0644)
				return dir
			},
			wantErr:     false,
			wantVersion: "v1.6.0",
			wantFiles:   1,
		},
		{
			name: "missing go.mod",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			wantErr: true,
		},
		{
			name: "no go-zero dependency",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				goMod := `module test

go 1.21

require github.com/gin-gonic/gin v1.9.0
`
				os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0644)
				os.WriteFile(filepath.Join(dir, "api.api"), []byte("service {}"), 0644)
				return dir
			},
			wantErr: true,
		},
		{
			name: "no .api files",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				goMod := `module test

go 1.21

require github.com/zeromicro/go-zero v1.6.0
`
				os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0644)
				return dir
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setup(t)
			d := NewDetector()
			info, err := d.Detect(dir)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if info.Framework != "gozero" {
				t.Errorf("expected framework 'gozero', got %q", info.Framework)
			}
			if info.GoZeroVersion != tt.wantVersion {
				t.Errorf("expected version %q, got %q", tt.wantVersion, info.GoZeroVersion)
			}
			if len(info.APIFiles) != tt.wantFiles {
				t.Errorf("expected %d api files, got %d", tt.wantFiles, len(info.APIFiles))
			}
		})
	}
}
```

Run:
```bash
go test ./internal/extractor/gozero/... -run TestDetector_Detect -v
```

Expected: FAIL - framework not set correctly

**Step 6: Implement full Detect method**

Replace Detect method in `detector.go`:

```go
import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spencercjh/spec-forge/internal/extractor"
)

const FrameworkName = "gozero"
const GoModFile = "go.mod"

// Detect analyzes a project and returns info if it's a go-zero project.
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
```

Run:
```bash
go test ./internal/extractor/gozero/... -run TestDetector -v
```

Expected: All Detector tests PASS

**Step 7: Commit**

```bash
git add internal/extractor/gozero/detector.go internal/extractor/gozero/detector_test.go
git commit -s -m "feat(gozero): implement Detector with go.mod and .api file detection"
```

---

## Task 3: Extend ProjectInfo in types.go

**Files:**
- Modify: `internal/extractor/types.go`

**Step 1: Add new fields to ProjectInfo**

Add to `ProjectInfo` struct:

```go
// ProjectInfo contains detected information about a project.
type ProjectInfo struct {
	// ... existing fields ...

	// Framework type: "spring" or "gozero"
	Framework string

	// go-zero specific fields
	APIFiles      []string // List of .api files found
	GoZeroVersion string   // go-zero version from go.mod
	HasGoctl      bool     // Whether goctl is installed (set by Patcher)
}
```

**Step 2: Add constants for framework names**

Add at package level:

```go
// Framework constants
const (
	FrameworkSpring = "spring"
	FrameworkGoZero = "gozero"
)
```

**Step 3: Verify changes compile**

Run:
```bash
go build ./internal/extractor/...
```

Expected: No errors

**Step 4: Run existing tests**

Run:
```bash
go test ./internal/extractor/... -v
```

Expected: All existing tests still pass

**Step 5: Commit**

```bash
git add internal/extractor/types.go
git commit -s -m "feat(extractor): extend ProjectInfo with go-zero fields"
```

---

## Task 4: Implement Patcher with goctl check

**Files:**
- Modify: `internal/extractor/gozero/patcher.go`
- Modify: `internal/extractor/gozero/patcher_test.go`

**Step 1: Add executor import and constructor with injection**

```go
package gozero

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spencercjh/spec-forge/internal/executor"
	"github.com/spencercjh/spec-forge/internal/extractor"
)

const defaultTimeout = 30 * time.Second

// Patcher checks and installs goctl.
type Patcher struct {
	exec executor.Interface
}

// PatchResult contains the result of patching.
type PatchResult struct {
	GoctlInstalled     bool
	GoctlVersion       string
	InstallationOutput string
}

// NewPatcher creates a new Patcher with default executor.
func NewPatcher() *Patcher {
	return &Patcher{exec: executor.NewExecutor()}
}

// NewPatcherWithExecutor creates a Patcher with custom executor (for testing).
func NewPatcherWithExecutor(exec executor.Interface) *Patcher {
	return &Patcher{exec: exec}
}
```

**Step 2: Write failing test for goctl check**

Add to `patcher_test.go`:

```go
package gozero

import (
	"context"
	"testing"

	"github.com/spencercjh/spec-forge/internal/executor"
	"github.com/spencercjh/spec-forge/internal/extractor"
)

// mockExecutor is a test double for executor.Interface
type mockExecutor struct {
	result *executor.ExecuteResult
	err    error
}

func (m *mockExecutor) Execute(ctx context.Context, opts *executor.ExecuteOptions) (*executor.ExecuteResult, error) {
	return m.result, m.err
}

func TestPatcher_Patch_GoctlInstalled(t *testing.T) {
	mockExec := &mockExecutor{
		result: &executor.ExecuteResult{
			Stdout: "goctl version 1.6.0",
			Stderr: "",
		},
		err: nil,
	}

	p := NewPatcherWithExecutor(mockExec)
	info := &extractor.ProjectInfo{}
	opts := &extractor.PatchOptions{}

	result, err := p.Patch(context.Background(), "/tmp", info, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.GoctlInstalled {
		t.Error("expected goctl to be installed")
	}
	if result.GoctlVersion != "goctl version 1.6.0" {
		t.Errorf("unexpected version: %s", result.GoctlVersion)
	}
}

func TestPatcher_Patch_GoctlNotInstalled(t *testing.T) {
	mockExec := &mockExecutor{
		result: nil,
		err:    fmt.Errorf("command not found: goctl"),
	}

	p := NewPatcherWithExecutor(mockExec)
	info := &extractor.ProjectInfo{}
	opts := &extractor.PatchOptions{}

	_, err := p.Patch(context.Background(), "/tmp", info, opts)
	if err == nil {
		t.Error("expected error when goctl not installed")
	}

	// Error should contain installation hint
	if !strings.Contains(err.Error(), "go install") {
		t.Error("error should contain installation hint")
	}
}
```

Run:
```bash
go test ./internal/extractor/gozero/... -run TestPatcher -v
```

Expected: FAIL - imports and implementation missing

**Step 3: Implement Patch method with goctl check**

Add to `patcher.go`:

```go
// Patch checks goctl installation and returns installation info.
func (p *Patcher) Patch(ctx context.Context, projectPath string, info *extractor.ProjectInfo, opts *extractor.PatchOptions) (*PatchResult, error) {
	result := &PatchResult{}

	// Check if goctl is installed
	version, err := p.checkGoctl(ctx)
	if err == nil {
		result.GoctlInstalled = true
		result.GoctlVersion = version
		info.HasGoctl = true
		return result, nil
	}

	// goctl not installed - return error with installation hint
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
```

Run:
```bash
go test ./internal/extractor/gozero/... -run TestPatcher -v
```

Expected: PASS

**Step 4: Commit**

```bash
git add internal/extractor/gozero/patcher.go internal/extractor/gozero/patcher_test.go
git commit -s -m "feat(gozero): implement Patcher with goctl version check"
```

---

## Task 5: Implement Generator with Swagger conversion

**Files:**
- Modify: `internal/extractor/gozero/generator.go`
- Modify: `internal/extractor/gozero/generator_test.go`

**Step 1: Add imports and constructor with injection**

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
	"gopkg.in/yaml.v3"

	"github.com/spencercjh/spec-forge/internal/executor"
	"github.com/spencercjh/spec-forge/internal/extractor"
)

// Generator generates OpenAPI specs from go-zero projects.
type Generator struct {
	exec executor.Interface
}

// NewGenerator creates a new Generator with default executor.
func NewGenerator() *Generator {
	return &Generator{exec: executor.NewExecutor()}
}

// NewGeneratorWithExecutor creates a Generator with custom executor (for testing).
func NewGeneratorWithExecutor(exec executor.Interface) *Generator {
	return &Generator{exec: exec}
}
```

**Step 2: Write failing test for generateSwagger**

Add to `generator_test.go`:

```go
package gozero

import (
	"context"
	"testing"

	"github.com/spencercjh/spec-forge/internal/executor"
	"github.com/spencercjh/spec-forge/internal/extractor"
)

func TestGenerator_generateSwagger(t *testing.T) {
	mockExec := &mockExecutor{
		result: &executor.ExecuteResult{
			Stdout: "swagger generated successfully",
			Stderr: "",
		},
		err: nil,
	}

	g := NewGeneratorWithExecutor(mockExec)
	ctx := context.Background()
	opts := &extractor.GenerateOptions{
		OutputDir: t.TempDir(),
		Timeout:   defaultTimeout,
	}

	// Create a mock swagger.json file (generator expects it to exist after goctl)
	swaggerPath := filepath.Join(opts.OutputDir, "swagger.json")
	os.WriteFile(swaggerPath, []byte(`{"swagger":"2.0"}`), 0644)

	result, err := g.generateSwagger(ctx, "/path/to/api.api", opts.OutputDir, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != swaggerPath {
		t.Errorf("expected %s, got %s", swaggerPath, result)
	}
}
```

Run:
```bash
go test ./internal/extractor/gozero/... -run TestGenerator_generateSwagger -v
```

Expected: FAIL - method not implemented

**Step 3: Implement generateSwagger method**

Add to `generator.go`:

```go
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
```

Run:
```bash
go test ./internal/extractor/gozero/... -run TestGenerator_generateSwagger -v
```

Expected: PASS

**Step 4: Write failing test for convertToOpenAPI3**

Add to `generator_test.go`:

```go
func TestGenerator_convertToOpenAPI3(t *testing.T) {
	// Create a temporary swagger.json (Swagger 2.0)
	swagger2Content := `{
		"swagger": "2.0",
		"info": {
			"title": "Test API",
			"version": "1.0.0"
		},
		"paths": {}
	}`

	outputDir := t.TempDir()
	swaggerPath := filepath.Join(outputDir, "swagger.json")
	os.WriteFile(swaggerPath, []byte(swagger2Content), 0644)

	g := NewGenerator()
	opts := &extractor.GenerateOptions{
		OutputDir:  outputDir,
		OutputFile: "openapi",
		Format:     "json",
	}

	result, err := g.convertToOpenAPI3(swaggerPath, outputDir, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check file was created
	if _, err := os.Stat(result); err != nil {
		t.Errorf("output file not created: %s", result)
	}

	// Verify it's valid OpenAPI 3.0 by loading it
	loader := openapi3.NewLoader()
	spec, err := loader.LoadFromFile(result)
	if err != nil {
		t.Fatalf("failed to load generated spec: %v", err)
	}

	if spec.Info.Title != "Test API" {
		t.Errorf("expected title 'Test API', got %s", spec.Info.Title)
	}
}
```

Run:
```bash
go test ./internal/extractor/gozero/... -run TestGenerator_convertToOpenAPI3 -v
```

Expected: FAIL - method not implemented

**Step 5: Implement convertToOpenAPI3 method**

Add to `generator.go`:

```go
func (g *Generator) convertToOpenAPI3(swaggerPath, outputDir string, opts *extractor.GenerateOptions) (string, error) {
	// Load Swagger 2.0
	loader := openapi2.NewLoader()
	swagger2Doc, err := loader.LoadFromFile(swaggerPath)
	if err != nil {
		return "", fmt.Errorf("failed to load swagger.json: %w", err)
	}

	// Convert to OpenAPI 3.0
	openAPIDoc, err := openapi2conv.ToV3(swagger2Doc)
	if err != nil {
		return "", fmt.Errorf("failed to convert to OpenAPI 3.0: %w", err)
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

Run:
```bash
go test ./internal/extractor/gozero/... -run TestGenerator_convertToOpenAPI3 -v
```

Expected: PASS

**Step 6: Implement full Generate method**

Add to `generator.go`:

```go
// Generate generates OpenAPI spec from go-zero project.
func (g *Generator) Generate(ctx context.Context, projectPath string, info *extractor.ProjectInfo, opts *extractor.GenerateOptions) (*extractor.GenerateResult, error) {
	if len(info.APIFiles) == 0 {
		return nil, fmt.Errorf("no .api files found")
	}

	// Use the first .api file (typically api/api.api)
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
```

**Step 7: Run all generator tests**

Run:
```bash
go test ./internal/extractor/gozero/... -run TestGenerator -v
```

Expected: All PASS

**Step 8: Commit**

```bash
git add internal/extractor/gozero/generator.go internal/extractor/gozero/generator_test.go
git commit -s -m "feat(gozero): implement Generator with Swagger 2.0 to OpenAPI 3.0 conversion"
```

---

## Task 6: Implement goctl bug patches

**Files:**
- Create: `internal/extractor/gozero/swagger_patch.go`
- Create: `internal/extractor/gozero/swagger_patch_test.go`

**Step 1: Write failing test for array items fix (#5426)**

Add to `swagger_patch_test.go`:

```go
package gozero

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi2"
)

func TestPatchSwagger2Doc_ArrayItems(t *testing.T) {
	// Simulate goctl bug: array type missing items
	doc := &openapi2.T{
		Definitions: map[string]*openapi2.Schema{
			"TestResponse": {
				Type: "object",
				Properties: map[string]*openapi2.SchemaRef{
					"data": {
						Value: &openapi2.Schema{
							Type: "array",
							// Missing items - this is the bug
						},
					},
				},
			},
		},
	}

	g := &Generator{}
	g.patchSwagger2Doc(doc)

	// After patching, items should be present
	dataSchema := doc.Definitions["TestResponse"].Properties["data"].Value
	if dataSchema.Items == nil {
		t.Error("expected array items to be patched")
	}
	if dataSchema.Items.Value.Type != "object" {
		t.Errorf("expected default type 'object', got %s", dataSchema.Items.Value.Type)
	}
}
```

Run:
```bash
go test ./internal/extractor/gozero/... -run TestPatchSwagger2Doc_ArrayItems -v
```

Expected: FAIL - patch function not implemented

**Step 2: Implement patchSwagger2Doc and patchSchema**

Create `swagger_patch.go`:

```go
package gozero

import (
	"regexp"

	"github.com/getkin/kin-openapi/openapi2"
)

// patchSwagger2Doc applies fixes for known goctl bugs
// See: https://github.com/zeromicro/go-zero/issues/5426-5428
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
```

Run:
```bash
go test ./internal/extractor/gozero/... -run TestPatchSwagger2Doc_ArrayItems -v
```

Expected: PASS

**Step 3: Write failing test for form:"-" fix (#5427)**

Add to `swagger_patch_test.go`:

```go
func TestPatchPaths_FormDash(t *testing.T) {
	// Simulate goctl bug: form:"-" generates parameter with name "-"
	doc := &openapi2.T{
		Paths: map[string]*openapi2.PathItem{
			"/test": {
				Get: &openapi2.Operation{
					Parameters: openapi2.Parameters{
						{
							Value: &openapi2.Parameter{
								Name: "normal",
								In:   "query",
							},
						},
						{
							Value: &openapi2.Parameter{
								Name: "-", // Bug: should be ignored
								In:   "query",
							},
						},
					},
				},
			},
		},
	}

	g := &Generator{}
	g.patchSwagger2Doc(doc)

	// After patching, "-" parameters should be removed
	params := doc.Paths["/test"].Get.Parameters
	if len(params) != 1 {
		t.Errorf("expected 1 parameter, got %d", len(params))
	}
	if params[0].Value.Name != "normal" {
		t.Errorf("expected 'normal', got %s", params[0].Value.Name)
	}
}
```

Run:
```bash
go test ./internal/extractor/gozero/... -run TestPatchPaths_FormDash -v
```

Expected: FAIL - patchPaths not implemented

**Step 4: Implement patchPaths and patchOperation**

Add to `swagger_patch.go`:

```go
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
	// Match {param} syntax (Swagger 2.0 format)
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

Run:
```bash
go test ./internal/extractor/gozero/... -run TestPatchPaths_FormDash -v
```

Expected: PASS

**Step 5: Write failing test for orphan path params fix (#5428)**

Add to `swagger_patch_test.go`:

```go
func TestPatchPaths_OrphanPathParams(t *testing.T) {
	// Simulate goctl bug: path param declared but not in URL
	doc := &openapi2.T{
		Paths: map[string]*openapi2.PathItem{
			"/foo": { // No {id} in path
				Get: &openapi2.Operation{
					Parameters: openapi2.Parameters{
						{
							Value: &openapi2.Parameter{
								Name: "id",
								In:   "path",
							},
						},
					},
				},
			},
			"/bar/{id}": { // Has {id} in path
				Get: &openapi2.Operation{
					Parameters: openapi2.Parameters{
						{
							Value: &openapi2.Parameter{
								Name: "id",
								In:   "path",
							},
						},
					},
				},
			},
		},
	}

	g := &Generator{}
	g.patchSwagger2Doc(doc)

	// /foo should have no path params (orphan removed)
	fooParams := doc.Paths["/foo"].Get.Parameters
	if len(fooParams) != 0 {
		t.Errorf("expected 0 params for /foo, got %d", len(fooParams))
	}

	// /bar/{id} should still have id param
	barParams := doc.Paths["/bar/{id}"].Get.Parameters
	if len(barParams) != 1 {
		t.Errorf("expected 1 param for /bar/{id}, got %d", len(barParams))
	}
}
```

Run:
```bash
go test ./internal/extractor/gozero/... -run TestPatchPaths_OrphanPathParams -v
```

Expected: PASS (already implemented in step 4)

**Step 6: Run all patch tests**

Run:
```bash
go test ./internal/extractor/gozero/... -run TestPatch -v
```

Expected: All PASS

**Step 7: Commit**

```bash
git add internal/extractor/gozero/swagger_patch.go internal/extractor/gozero/swagger_patch_test.go
git commit -s -m "fix(gozero): patch goctl swagger generation bugs (#5426-5428)

- Fix nested arrays missing items field
- Remove parameters with name '-'
- Remove path parameters not present in URL"
```

---

## Task 7: Add gozero to generate command

**Files:**
- Modify: `cmd/generate.go`

**Files:**
- Modify: `cmd/generate.go`

**Step 1: Add gozero import**

Add import:

```go
import (
	// ... existing imports ...
	"github.com/spencercjh/spec-forge/internal/extractor/gozero"
)
```

**Step 2: Modify project detection logic**

Find the detection code and modify:

```go
// detectProject tries to detect the framework and return project info
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

	return nil, "", fmt.Errorf("no supported framework detected (Spring Boot or go-zero): %w", err)
}
```

**Step 3: Modify runGenerate to use framework-specific extraction**

In `runGenerate`, replace the Spring-specific detection with:

```go
// Step 1: Detect project
info, framework, err := detectProject(path)
if err != nil {
	return errWrap("detection failed", err)
}

slog.InfoContext(ctx, "Detected project",
	"framework", framework,
	"path", path,
)
```

**Step 4: Add framework-specific patch and generate logic**

After detection, add framework-specific handling:

```go
var patcher extractor.Patcher
var generator extractor.Generator

switch framework {
case "spring":
	patcher = spring.NewPatcher()
	generator = spring.NewGenerator()
case "gozero":
	patcher = gozero.NewPatcher()
	generator = gozero.NewGenerator()
default:
	return fmt.Errorf("unsupported framework: %s", framework)
}
```

**Step 5: Verify changes compile**

Run:
```bash
go build ./cmd/...
```

Expected: No errors

**Step 6: Commit**

```bash
git add cmd/generate.go
git commit -s -m "feat(cmd): integrate gozero extractor into generate command"
```

---

## Task 8: Create E2E test project

**Files:**
- Create: `integration-tests/gozero-demo/go.mod`
- Create: `integration-tests/gozero-demo/api/api.api`
- Create: `integration-tests/gozero-demo/main.go`

**Step 1: Create go.mod**

```go
module gozero-demo

go 1.21

require github.com/zeromicro/go-zero v1.6.0
```

**Step 2: Create api.api**

```go
syntax = "v1"

info (
	title:   "Demo API"
	desc:    "A demo go-zero API for testing"
	version: "1.0.0"
)

type Request {
	Name string `path:"name,options=you|me"`
}

type Response {
	Message string `json:"message"`
}

service demo-api {
	@handler DemoHandler
	get /demo/:name (Request) returns (Response)
}
```

**Step 3: Create main.go**

```go
package main

import "fmt"

func main() {
	fmt.Println("go-zero demo project")
}
```

**Step 4: Verify project structure**

Run:
```bash
ls -la integration-tests/gozero-demo/
```

Expected:
```
api/
  api.api
go.mod
main.go
```

**Step 5: Test detection works**

Run:
```bash
go run . generate ./integration-tests/gozero-demo --dry-run
```

Expected: Detects go-zero project (or appropriate error if --dry-run not supported)

**Step 6: Commit**

```bash
git add integration-tests/gozero-demo/
git commit -s -m "test: add go-zero demo project for E2E testing"
```

---

## Task 9: Final verification

**Step 1: Run full test suite**

Run:
```bash
make test
```

Expected: All tests pass

**Step 2: Run linter**

Run:
```bash
make lint
```

Expected: No issues

**Step 3: Build binary**

Run:
```bash
make build
```

Expected: Binary built successfully

**Step 4: Manual test with demo project**

Run:
```bash
# This will fail if goctl not installed, that's expected
./build/spec-forge generate ./integration-tests/gozero-demo -v 2>&1 | head -20
```

Expected: Shows detection of go-zero project, then fails at goctl step (if not installed)

**Step 5: Final commit**

If all good:
```bash
git commit --amend -s --no-edit  # amend if needed, or skip
```

---

## Summary

After completing all tasks:

1. **Detector**: Detects go-zero projects via `.api` files and `go.mod`
2. **Patcher**: Checks goctl installation with helpful error messages
3. **Generator**: Runs `goctl api swagger` and converts Swagger 2.0 → OpenAPI 3.0
4. **Bug Patches**: Fixes for goctl issues #5426 (array items), #5427 (form:"-"), #5428 (orphan path params)
5. **Integration**: gozero extractor integrated into `generate` command
6. **Tests**: Unit tests for all components + E2E demo project

Next PR will implement `specctx.GoZeroExtractor` to extract handler/struct documentation from go-zero source code.
