# Gin Framework Support Implementation Plan

> **Status:** ✅ **COMPLETED** - All tasks implemented, tested, and merged to main

**Goal:** Implement Gin framework support with Detector, Patcher, and Generator components using AST parsing.

**Architecture:** Follow Spring Boot pattern with three components: Detector (go.mod + Gin dependency check), Patcher (Gin needs no patching - no-op), Generator (AST parsing to extract routes, handlers, and schemas).

**Tech Stack:** Go, go/ast, go/parser, go/token, golang.org/x/mod/modfile, kin-openapi/openapi3, log/slog

**Implementation Date:** March 2026
**Status:** Production-ready with comprehensive logging and lint-clean code

---

## Task 1: Create gin package structure

**Files:**
- Create: `internal/extractor/gin/gin.go` (extractor interface implementation)
- Create: `internal/extractor/gin/detector.go` (stub)
- Create: `internal/extractor/gin/detector_test.go` (stub)
- Create: `internal/extractor/gin/patcher.go` (stub)
- Create: `internal/extractor/gin/patcher_test.go` (stub)
- Create: `internal/extractor/gin/generator.go` (stub)
- Create: `internal/extractor/gin/generator_test.go` (stub)
- Create: `internal/extractor/gin/ast_parser.go` (stub)
- Create: `internal/extractor/gin/ast_parser_test.go` (stub)
- Create: `internal/extractor/gin/handler_analyzer.go` (stub)
- Create: `internal/extractor/gin/handler_analyzer_test.go` (stub)
- Create: `internal/extractor/gin/schema_extractor.go` (stub)
- Create: `internal/extractor/gin/schema_extractor_test.go` (stub)

**Step 1: Create package directory and stub files**

```bash
mkdir -p internal/extractor/gin
touch internal/extractor/gin/{gin,detector,patcher,generator,ast_parser,handler_analyzer,schema_extractor}_test.go
```

**Step 2: Create gin.go - extractor entry point**

```go
// Package gin provides Gin framework specific extraction functionality.
package gin

import (
	"context"
	"fmt"

	"github.com/spencercjh/spec-forge/internal/extractor"
)

const FrameworkName = "gin"

// GinExtractor implements extractor.Extractor for Gin projects.
type GinExtractor struct {
	detector  *Detector
	patcher   *Patcher
	generator *Generator
}

// NewGinExtractor creates a new GinExtractor instance.
func NewGinExtractor() *GinExtractor {
	return &GinExtractor{
		detector:  NewDetector(),
		patcher:   NewPatcher(),
		generator: NewGenerator(),
	}
}

// Detect implements extractor.Extractor.Detect.
func (e *GinExtractor) Detect(projectPath string) (*extractor.ProjectInfo, error) {
	return e.detector.Detect(projectPath)
}

// Patch implements extractor.Extractor.Patch.
func (e *GinExtractor) Patch(ctx context.Context, projectPath string, info *extractor.ProjectInfo, opts *extractor.PatchOptions) (*extractor.PatchResult, error) {
	return e.patcher.Patch(ctx, projectPath, info, opts)
}

// Generate implements extractor.Extractor.Generate.
func (e *GinExtractor) Generate(ctx context.Context, projectPath string, info *extractor.ProjectInfo, opts *extractor.GenerateOptions) (*extractor.GenerateResult, error) {
	return e.generator.Generate(ctx, projectPath, info, opts)
}
```

**Step 3: Create detector.go stub**

```go
package gin

import "github.com/spencercjh/spec-forge/internal/extractor"

// Detector detects Gin projects.
type Detector struct{}

// NewDetector creates a new Detector instance.
func NewDetector() *Detector {
	return &Detector{}
}

// Detect analyzes a project and returns info if it's a Gin project.
func (d *Detector) Detect(projectPath string) (*extractor.ProjectInfo, error) {
	return nil, nil
}
```

**Step 4: Create patcher.go stub**

```go
package gin

import (
	"context"

	"github.com/spencercjh/spec-forge/internal/extractor"
)

// Patcher is a no-op for Gin projects (no patching needed).
type Patcher struct{}

// NewPatcher creates a new Patcher instance.
func NewPatcher() *Patcher {
	return &Patcher{}
}

// Patch performs no-op patching for Gin projects.
func (p *Patcher) Patch(ctx context.Context, projectPath string, info *extractor.ProjectInfo, opts *extractor.PatchOptions) (*extractor.PatchResult, error) {
	return &extractor.PatchResult{}, nil
}
```

**Step 5: Create generator.go stub**

```go
package gin

import (
	"context"

	"github.com/spencercjh/spec-forge/internal/extractor"
)

// Generator generates OpenAPI specs from Gin projects using AST parsing.
type Generator struct{}

// NewGenerator creates a new Generator instance.
func NewGenerator() *Generator {
	return &Generator{}
}

// Generate generates OpenAPI spec from Gin project.
func (g *Generator) Generate(ctx context.Context, projectPath string, info *extractor.ProjectInfo, opts *extractor.GenerateOptions) (*extractor.GenerateResult, error) {
	return nil, nil
}
```

**Step 6: Create ast_parser.go stub**

```go
package gin

// ASTParser parses Go AST to extract Gin routes.
type ASTParser struct {
	projectPath string
}

// NewASTParser creates a new ASTParser instance.
func NewASTParser(projectPath string) *ASTParser {
	return &ASTParser{projectPath: projectPath}
}
```

**Step 7: Create handler_analyzer.go stub**

```go
package gin

// HandlerAnalyzer analyzes Gin handler functions.
type HandlerAnalyzer struct{}

// NewHandlerAnalyzer creates a new HandlerAnalyzer instance.
func NewHandlerAnalyzer() *HandlerAnalyzer {
	return &HandlerAnalyzer{}
}
```

**Step 8: Create schema_extractor.go stub**

```go
package gin

// SchemaExtractor extracts OpenAPI schemas from Go structs.
type SchemaExtractor struct{}

// NewSchemaExtractor creates a new SchemaExtractor instance.
func NewSchemaExtractor() *SchemaExtractor {
	return &SchemaExtractor{}
}
```

**Step 9: Create test stubs**

```go
// gin_test.go
package gin

import "testing"

func TestNewGinExtractor(t *testing.T) {
	e := NewGinExtractor()
	if e == nil {
		t.Error("expected non-nil extractor")
	}
}
```

```go
// detector_test.go
package gin

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
package gin

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
package gin

import "testing"

func TestNewGenerator(t *testing.T) {
	g := NewGenerator()
	if g == nil {
		t.Error("expected non-nil generator")
	}
}
```

```go
// ast_parser_test.go
package gin

import "testing"

func TestNewASTParser(t *testing.T) {
	p := NewASTParser("/tmp")
	if p == nil {
		t.Error("expected non-nil parser")
	}
}
```

```go
// handler_analyzer_test.go
package gin

import "testing"

func TestNewHandlerAnalyzer(t *testing.T) {
	a := NewHandlerAnalyzer()
	if a == nil {
		t.Error("expected non-nil analyzer")
	}
}
```

```go
// schema_extractor_test.go
package gin

import "testing"

func TestNewSchemaExtractor(t *testing.T) {
	e := NewSchemaExtractor()
	if e == nil {
		t.Error("expected non-nil extractor")
	}
}
```

**Step 10: Verify stubs compile**

Run:
```bash
go build ./internal/extractor/gin/...
```

Expected: No errors

**Step 11: Run stub tests**

Run:
```bash
go test ./internal/extractor/gin/... -v
```

Expected: 7 tests pass

**Step 12: Commit**

```bash
git add internal/extractor/gin/
git commit -s -m "chore(gin): create package structure with stubs"
```

---

## Task 2: Define Gin types and data structures

**Files:**
- Create: `internal/extractor/gin/types.go`
- Create: `internal/extractor/gin/types_test.go`

**Step 1: Write the failing test**

```go
// internal/extractor/gin/types_test.go
package gin

import "testing"

func TestInfo(t *testing.T) {
	info := Info{
		GoVersion:  "1.21",
		ModuleName: "github.com/example/app",
		GinVersion: "v1.9.1",
		HasGin:     true,
	}

	if info.GoVersion != "1.21" {
		t.Errorf("expected GoVersion '1.21', got %s", info.GoVersion)
	}
	if info.ModuleName != "github.com/example/app" {
		t.Errorf("expected ModuleName 'github.com/example/app', got %s", info.ModuleName)
	}
}

func TestRouterGroup(t *testing.T) {
	rg := RouterGroup{
		BasePath: "/api/v1",
		Routes: []Route{
			{Method: "GET", Path: "/users"},
			{Method: "POST", Path: "/users"},
		},
	}

	if rg.BasePath != "/api/v1" {
		t.Errorf("expected BasePath '/api/v1', got %s", rg.BasePath)
	}
	if len(rg.Routes) != 2 {
		t.Errorf("expected 2 routes, got %d", len(rg.Routes))
	}
}

func TestRoute(t *testing.T) {
	route := Route{
		Method:      "GET",
		Path:        "/users/:id",
		FullPath:    "/api/v1/users/:id",
		HandlerName: "GetUser",
		HandlerFile: "user_handler.go",
		Middlewares: []string{"Auth"},
	}

	if route.Method != "GET" {
		t.Errorf("expected Method 'GET', got %s", route.Method)
	}
	if route.FullPath != "/api/v1/users/:id" {
		t.Errorf("expected FullPath '/api/v1/users/:id', got %s", route.FullPath)
	}
}

func TestHandlerInfo(t *testing.T) {
	hi := HandlerInfo{
		PathParams: []ParamInfo{
			{Name: "id", GoType: "string", Required: true},
		},
		QueryParams: []ParamInfo{
			{Name: "page", GoType: "string", Required: false},
		},
		BodyType: "CreateUserRequest",
		Responses: []ResponseInfo{
			{StatusCode: 200, GoType: "User"},
			{StatusCode: 404, GoType: "ErrorResponse"},
		},
	}

	if len(hi.PathParams) != 1 {
		t.Errorf("expected 1 path param, got %d", len(hi.PathParams))
	}
	if len(hi.Responses) != 2 {
		t.Errorf("expected 2 responses, got %d", len(hi.Responses))
	}
}

func TestParamInfo(t *testing.T) {
	param := ParamInfo{
		Name:     "id",
		GoType:   "string",
		Required: true,
	}

	if param.Name != "id" {
		t.Errorf("expected Name 'id', got %s", param.Name)
	}
	if !param.Required {
		t.Error("expected Required to be true")
	}
}

func TestResponseInfo(t *testing.T) {
	resp := ResponseInfo{
		StatusCode: 200,
		GoType:     "User",
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected StatusCode 200, got %d", resp.StatusCode)
	}
	if resp.GoType != "User" {
		t.Errorf("expected GoType 'User', got %s", resp.GoType)
	}
}
```

Run:
```bash
go test ./internal/extractor/gin/... -run TestInfo -v
```

Expected: FAIL - types not defined

**Step 2: Create types.go**

```go
// internal/extractor/gin/types.go
package gin

// Info contains information about a Gin project.
type Info struct {
	GoVersion    string        // Go version from go.mod
	ModuleName   string        // Module path
	GinVersion   string        // gin dependency version
	HasGin       bool          // Has gin dependency
	MainFiles    []string      // main.go or files with route registration
	HandlerFiles []string      // Handler file list
	RouterGroups []RouterGroup // Detected router groups
}

// RouterGroup represents a Gin router group.
type RouterGroup struct {
	BasePath string  // Group base path (e.g., "/api/v1")
	Routes   []Route // Routes in this group
}

// Route represents a single Gin route.
type Route struct {
	Method      string   // HTTP method: GET, POST, PUT, DELETE, PATCH
	Path        string   // Route path (e.g., "/users/:id")
	FullPath    string   // Full path including group prefix
	HandlerName string   // Handler function name
	HandlerFile string   // File containing handler definition
	Middlewares []string // Middleware names
}

// HandlerInfo contains information extracted from a handler function.
type HandlerInfo struct {
	PathParams   []ParamInfo    // Path parameters (c.Param)
	QueryParams  []ParamInfo    // Query parameters (c.Query)
	HeaderParams []ParamInfo    // Header parameters (c.GetHeader)
	BodyType     string         // Request body type (from ShouldBindJSON)
	Responses    []ResponseInfo // Response info (from c.JSON calls)
}

// ParamInfo represents a parameter extracted from handler.
type ParamInfo struct {
	Name     string // Parameter name
	GoType   string // Go type name
	Required bool   // Whether parameter is required
}

// ResponseInfo represents a response from handler analysis.
type ResponseInfo struct {
	StatusCode int    // HTTP status code
	GoType     string // Response Go type
}

// HandlerRef represents a reference to a handler function.
type HandlerRef struct {
	Name         string // Function name (empty for anonymous functions)
	File         string // File path
	IsAnonymous  bool   // Whether it's an anonymous function
	ReceiverType string // Receiver type for methods (empty for functions)
}
```

**Step 3: Run test to verify it passes**

Run:
```bash
go test ./internal/extractor/gin/... -run "TestInfo|TestRouter|TestRoute|TestHandler|TestParam|TestResponse" -v
```

Expected: PASS

**Step 4: Commit**

```bash
git add internal/extractor/gin/types.go internal/extractor/gin/types_test.go
git commit -s -m "feat(gin): add Gin project types and data structures"
```

---

## Task 3: Implement Detector with go.mod parsing

**Files:**
- Modify: `internal/extractor/gin/detector.go`
- Modify: `internal/extractor/gin/detector_test.go`

**Step 1: Add import and extend Detector**

```go
package gin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
	"github.com/spencercjh/spec-forge/internal/extractor"
)

const GinModule = "github.com/gin-gonic/gin"
const GoModFile = "go.mod"

// Detector detects Gin projects.
type Detector struct{}

// NewDetector creates a new Detector instance.
func NewDetector() *Detector {
	return &Detector{}
}
```

**Step 2: Write failing test for parseGinVersion**

```go
// Add to detector_test.go

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetector_parseGinVersion(t *testing.T) {
	tests := []struct {
		name     string
		goMod    string
		expected string
	}{
		{
			name: "has gin dependency",
			goMod: `module test

go 1.21

require github.com/gin-gonic/gin v1.9.1
`,
			expected: "v1.9.1",
		},
		{
			name: "no gin dependency",
			goMod: `module test

go 1.21

require github.com/gin-gonic/gin v1.9.0
`,
			expected: "v1.9.0",
		},
		{
			name: "no gin - other framework",
			goMod: `module test

go 1.21

require github.com/zeromicro/go-zero v1.6.0
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
			version, err := d.parseGinVersion(goModPath)
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
go test ./internal/extractor/gin/... -run TestDetector_parseGinVersion -v
```

Expected: FAIL - method not implemented

**Step 3: Implement parseGinVersion method**

```go
// Add to detector.go

func (d *Detector) parseGinVersion(goModPath string) (string, error) {
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return "", err
	}

	modFile, err := modfile.Parse("go.mod", content, nil)
	if err != nil {
		return "", err
	}

	for _, req := range modFile.Require {
		if req.Mod.Path == GinModule {
			return req.Mod.Version, nil
		}
	}

	return "", nil
}
```

Run:
```bash
go test ./internal/extractor/gin/... -run TestDetector_parseGinVersion -v
```

Expected: PASS

**Step 4: Write failing test for findMainFiles**

```go
// Add to detector_test.go

func TestDetector_findMainFiles(t *testing.T) {
	dir := t.TempDir()

	// Create main.go
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}"), 0644)

	// Create router.go with route registration
	os.WriteFile(filepath.Join(dir, "router.go"), []byte("package main\n\nfunc setupRouter() {}"), 0644)

	// Create non-main file
	os.WriteFile(filepath.Join(dir, "utils.go"), []byte("package main\n\nfunc helper() {}"), 0644)

	// Create vendor directory (should be excluded)
	os.MkdirAll(filepath.Join(dir, "vendor", "test"), 0755)
	os.WriteFile(filepath.Join(dir, "vendor", "test", "main.go"), []byte("package main"), 0644)

	d := NewDetector()
	files, err := d.findMainFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find main.go and router.go
	if len(files) < 1 {
		t.Errorf("expected at least 1 main file, got %d", len(files))
	}

	// Check main.go is found
	foundMain := false
	for _, f := range files {
		if filepath.Base(f) == "main.go" {
			foundMain = true
			break
		}
	}
	if !foundMain {
		t.Error("expected main.go to be found")
	}
}
```

Run:
```bash
go test ./internal/extractor/gin/... -run TestDetector_findMainFiles -v
```

Expected: FAIL - method not implemented

**Step 5: Implement findMainFiles method**

```go
// Add to detector.go

func (d *Detector) findMainFiles(projectPath string) ([]string, error) {
	var mainFiles []string

	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip vendor and hidden directories
		if info.IsDir() {
			if info.Name() == "vendor" || strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process .go files
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip vendor
		if strings.Contains(path, "/vendor/") {
			return nil
		}

		// For now, collect all .go files - AST parser will identify route registration files
		mainFiles = append(mainFiles, path)

		return nil
	})

	return mainFiles, err
}
```

Run:
```bash
go test ./internal/extractor/gin/... -run TestDetector_findMainFiles -v
```

Expected: PASS

**Step 6: Write failing test for full Detect method**

```go
// Add to detector_test.go

func TestDetector_Detect(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T) string
		wantErr     bool
		wantVersion string
		wantHasGin  bool
	}{
		{
			name: "valid gin project",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				goMod := `module test

go 1.21

require github.com/gin-gonic/gin v1.9.1
`
				os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0644)
				os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}"), 0644)
				return dir
			},
			wantErr:     false,
			wantVersion: "v1.9.1",
			wantHasGin:  true,
		},
		{
			name: "missing go.mod",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			wantErr: true,
		},
		{
			name: "no gin dependency",
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
		{
			name: "no go files",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				goMod := `module test

go 1.21

require github.com/gin-gonic/gin v1.9.1
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

			if info.Framework != FrameworkName {
				t.Errorf("expected framework %q, got %q", FrameworkName, info.Framework)
			}
			if info.GinVersion != tt.wantVersion {
				t.Errorf("expected version %q, got %q", tt.wantVersion, info.GinVersion)
			}
		})
	}
}
```

Run:
```bash
go test ./internal/extractor/gin/... -run TestDetector_Detect -v
```

Expected: FAIL - Detect method not fully implemented

**Step 7: Implement full Detect method**

```go
// Replace Detect method in detector.go

// Detect analyzes a project and returns info if it's a Gin project.
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

	// Parse go.mod for Gin dependency
	ginVersion, err := d.parseGinVersion(goModPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse go.mod: %w", err)
	}
	if ginVersion == "" {
		return nil, fmt.Errorf("no gin dependency found in go.mod")
	}

	// Extract module name
	content, _ := os.ReadFile(goModPath)
	modFile, _ := modfile.Parse("go.mod", content, nil)
	moduleName := ""
	if modFile != nil {
		moduleName = modFile.Module.Mod.Path
	}

	// Find Go files
	mainFiles, err := d.findMainFiles(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find main files: %w", err)
	}
	if len(mainFiles) == 0 {
		return nil, fmt.Errorf("no .go files found in %s", absPath)
	}

	return &extractor.ProjectInfo{
		Framework:     FrameworkName,
		BuildTool:     "go",
		BuildFilePath: goModPath,
		GinVersion:    ginVersion,
		ModuleName:    moduleName,
		MainFiles:     mainFiles,
		HasGin:        true,
	}, nil
}
```

**Step 8: Update ProjectInfo in types.go to add Gin fields**

```go
// Add to internal/extractor/types.go in ProjectInfo struct

// Gin specific fields (for Framework == "gin")
GinVersion string   // Gin framework version
ModuleName string   // Go module name
MainFiles  []string // Main Go files
HasGin     bool     // Whether Gin is detected
```

**Step 9: Run all Detector tests**

Run:
```bash
go test ./internal/extractor/gin/... -run TestDetector -v
```

Expected: All Detector tests PASS

**Step 10: Commit**

```bash
git add internal/extractor/gin/detector.go internal/extractor/gin/detector_test.go internal/extractor/types.go
git commit -s -m "feat(gin): implement Detector with go.mod parsing"
```

---

## Task 4: Implement Patcher (no-op for Gin)

**Files:**
- Modify: `internal/extractor/gin/patcher.go`
- Modify: `internal/extractor/gin/patcher_test.go`

**Step 1: Write failing test**

```go
// Add to patcher_test.go

import (
	"context"
	"testing"

	"github.com/spencercjh/spec-forge/internal/extractor"
)

func TestPatcher_Patch(t *testing.T) {
	p := NewPatcher()
	ctx := context.Background()
	info := &extractor.ProjectInfo{}
	opts := &extractor.PatchOptions{}

	result, err := p.Patch(ctx, "/tmp", info, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Patcher should return empty result for Gin
	if result == nil {
		t.Error("expected non-nil result")
	}
}
```

Run:
```bash
go test ./internal/extractor/gin/... -run TestPatcher_Patch -v
```

Expected: PASS (already implemented as stub)

**Step 2: Update ProjectInfo to track Patcher result**

```go
// Update patcher.go to set HasGin flag

func (p *Patcher) Patch(ctx context.Context, projectPath string, info *extractor.ProjectInfo, opts *extractor.PatchOptions) (*extractor.PatchResult, error) {
	// Gin projects don't need patching, just mark as ready
	info.HasGin = true
	return &extractor.PatchResult{}, nil
}
```

**Step 3: Commit**

```bash
git add internal/extractor/gin/patcher.go internal/extractor/gin/patcher_test.go
git commit -s -m "feat(gin): implement no-op Patcher for Gin projects"
```

---

## Task 5: Implement AST Parser for route extraction

**Files:**
- Modify: `internal/extractor/gin/ast_parser.go`
- Modify: `internal/extractor/gin/ast_parser_test.go`

**Step 1: Add imports and structure**

```go
package gin

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// ASTParser parses Go AST to extract Gin routes.
type ASTParser struct {
	fset        *token.FileSet
	projectPath string
	files       map[string]*ast.File
	routerVars  map[string]bool // Track router variable names
}

// NewASTParser creates a new ASTParser instance.
func NewASTParser(projectPath string) *ASTParser {
	return &ASTParser{
		projectPath: projectPath,
		fset:        token.NewFileSet(),
		files:       make(map[string]*ast.File),
		routerVars:  make(map[string]bool),
	}
}
```

**Step 2: Write failing test for ParseFiles**

```go
// Add to ast_parser_test.go

import (
	"os"
	"path/filepath"
	"testing"
)

func TestASTParser_ParseFiles(t *testing.T) {
	dir := t.TempDir()

	// Create a simple Go file
	code := `package main

import "github.com/gin-gonic/gin"

func main() {
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})
}
`
	os.WriteFile(filepath.Join(dir, "main.go"), []byte(code), 0644)

	parser := NewASTParser(dir)
	err := parser.ParseFiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(parser.files) != 1 {
		t.Errorf("expected 1 file, got %d", len(parser.files))
	}
}
```

Run:
```bash
go test ./internal/extractor/gin/... -run TestASTParser_ParseFiles -v
```

Expected: FAIL - method not implemented

**Step 3: Implement ParseFiles method**

```go
// Add to ast_parser.go

// ParseFiles parses all Go files in the project.
func (p *ASTParser) ParseFiles() error {
	return filepath.Walk(p.projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip vendor and hidden directories
		if info.IsDir() {
			if info.Name() == "vendor" || strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process .go files
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip test files
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// Parse the file
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		file, err := parser.ParseFile(p.fset, path, content, parser.ParseComments)
		if err != nil {
			// Log but continue with other files
			return nil
		}

		p.files[path] = file
		return nil
	})
}
```

Run:
```bash
go test ./internal/extractor/gin/... -run TestASTParser_ParseFiles -v
```

Expected: PASS

**Step 4: Write failing test for ExtractRoutes**

```go
// Add to ast_parser_test.go

func TestASTParser_ExtractRoutes(t *testing.T) {
	dir := t.TempDir()

	// Create a file with direct route registration
	code := `package main

import "github.com/gin-gonic/gin"

func main() {
	r := gin.Default()
	r.GET("/users", getUsers)
	r.POST("/users", createUser)
}

func getUsers(c *gin.Context) {}
func createUser(c *gin.Context) {}
`
	os.WriteFile(filepath.Join(dir, "main.go"), []byte(code), 0644)

	parser := NewASTParser(dir)
	parser.ParseFiles()

	routes, err := parser.ExtractRoutes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(routes) != 2 {
		t.Errorf("expected 2 routes, got %d", len(routes))
	}

	// Check first route
	foundGet := false
	foundPost := false
	for _, r := range routes {
		if r.Method == "GET" && r.Path == "/users" {
			foundGet = true
		}
		if r.Method == "POST" && r.Path == "/users" {
			foundPost = true
		}
	}
	if !foundGet {
		t.Error("expected GET /users route")
	}
	if !foundPost {
		t.Error("expected POST /users route")
	}
}
```

Run:
```bash
go test ./internal/extractor/gin/... -run TestASTParser_ExtractRoutes -v
```

Expected: FAIL - method not implemented

**Step 5: Implement ExtractRoutes and helper methods**

```go
// Add to ast_parser.go

// ExtractRoutes extracts all Gin routes from parsed files.
func (p *ASTParser) ExtractRoutes() ([]Route, error) {
	var routes []Route

	for path, file := range p.files {
		fileRoutes := p.extractRoutesFromFile(path, file)
		routes = append(routes, fileRoutes...)
	}

	return routes, nil
}

// extractRoutesFromFile extracts routes from a single file.
func (p *ASTParser) extractRoutesFromFile(path string, file *ast.File) []Route {
	var routes []Route

	// Inspect the AST
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			if route := p.parseRouteCall(path, node); route != nil {
				routes = append(routes, *route)
			}
		}
		return true
	})

	return routes
}

// parseRouteCall parses a route registration call.
func (p *ASTParser) parseRouteCall(file string, call *ast.CallExpr) *Route {
	// Pattern: r.GET("/path", handler)
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}

	method := sel.Sel.Name
	if !isHTTPMethod(method) {
		return nil
	}

	// Get the router variable
	_, ok = sel.X.(*ast.Ident)
	if !ok {
		return nil
	}

	// Need at least 2 args: path and handler
	if len(call.Args) < 2 {
		return nil
	}

	// Extract path
	path := extractStringLiteral(call.Args[0])
	if path == "" {
		return nil
	}

	// Extract handler name
	handlerName := extractHandlerName(call.Args[1])

	return &Route{
		Method:      method,
		Path:        path,
		FullPath:    path,
		HandlerName: handlerName,
		HandlerFile: file,
	}
}

// isHTTPMethod checks if a string is an HTTP method.
func isHTTPMethod(s string) bool {
	switch s {
	case "GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS":
		return true
	}
	return false
}

// extractStringLiteral extracts a string from an AST expression.
func extractStringLiteral(expr ast.Expr) string {
	if lit, ok := expr.(*ast.BasicLit); ok && lit.Kind == token.STRING {
		// Remove quotes
		return strings.Trim(lit.Value, `"`)
	}
	return ""
}

// extractHandlerName extracts handler name from an expression.
func extractHandlerName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.FuncLit:
		return "" // Anonymous function
	case *ast.SelectorExpr:
		if x, ok := e.X.(*ast.Ident); ok {
			return x.Name + "." + e.Sel.Name
		}
	}
	return ""
}
```

Run:
```bash
go test ./internal/extractor/gin/... -run TestASTParser_ExtractRoutes -v
```

Expected: PASS

**Step 6: Write failing test for Group extraction**

```go
// Add to ast_parser_test.go

func TestASTParser_ExtractGroupRoutes(t *testing.T) {
	dir := t.TempDir()

	// Create a file with route groups
	code := `package main

import "github.com/gin-gonic/gin"

func main() {
	r := gin.Default()
	api := r.Group("/api/v1")
	api.GET("/users", getUsers)
	api.POST("/users", createUser)
}

func getUsers(c *gin.Context) {}
func createUser(c *gin.Context) {}
`
	os.WriteFile(filepath.Join(dir, "main.go"), []byte(code), 0644)

	parser := NewASTParser(dir)
	parser.ParseFiles()

	routes, err := parser.ExtractRoutes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check full paths include group prefix
	found := false
	for _, r := range routes {
		if r.Method == "GET" && r.FullPath == "/api/v1/users" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected GET /api/v1/users route")
	}
}
```

Run:
```bash
go test ./internal/extractor/gin/... -run TestASTParser_ExtractGroupRoutes -v
```

Expected: FAIL - group extraction not implemented

**Step 7: Implement Group route extraction**

This requires tracking group variables and their base paths. Add to ast_parser.go:

```go
// GroupInfo stores information about a router group.
type GroupInfo struct {
	BasePath string
	VarName  string
}

// extractGroups extracts router group definitions.
func (p *ASTParser) extractGroups(file *ast.File) map[string]*GroupInfo {
	groups := make(map[string]*GroupInfo)

	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.AssignStmt:
			// Pattern: api := r.Group("/api")
			if len(node.Lhs) == 1 && len(node.Rhs) == 1 {
				if group := p.parseGroupAssignment(node.Rhs[0]); group != nil {
					if ident, ok := node.Lhs[0].(*ast.Ident); ok {
						group.VarName = ident.Name
						groups[ident.Name] = group
					}
				}
			}
		}
		return true
	})

	return groups
}

// parseGroupAssignment parses a Group assignment.
func (p *ASTParser) parseGroupAssignment(expr ast.Expr) *GroupInfo {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return nil
	}

	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Group" {
		return nil
	}

	if len(call.Args) < 1 {
		return nil
	}

	basePath := extractStringLiteral(call.Args[0])
	if basePath == "" {
		return nil
	}

	return &GroupInfo{BasePath: basePath}
}
```

Then update `extractRoutesFromFile` to use groups.

**Step 8: Commit**

```bash
git add internal/extractor/gin/ast_parser.go internal/extractor/gin/ast_parser_test.go
git commit -s -m "feat(gin): implement AST parser for route extraction"
```

---

## Task 6: Implement Handler Analyzer

**Files:**
- Modify: `internal/extractor/gin/handler_analyzer.go`
- Modify: `internal/extractor/gin/handler_analyzer_test.go`

**Step 1: Add imports and structure**

```go
package gin

import (
	"go/ast"
	"go/token"
)

// HandlerAnalyzer analyzes Gin handler functions.
type HandlerAnalyzer struct {
	fset      *token.FileSet
	files     map[string]*ast.File
	typeCache map[string]*ast.TypeSpec
}

// NewHandlerAnalyzer creates a new HandlerAnalyzer instance.
func NewHandlerAnalyzer(fset *token.FileSet, files map[string]*ast.File) *HandlerAnalyzer {
	return &HandlerAnalyzer{
		fset:      fset,
		files:     files,
		typeCache: make(map[string]*ast.TypeSpec),
	}
}
```

**Step 2: Write failing test for AnalyzeHandler**

```go
// Add to handler_analyzer_test.go

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestHandlerAnalyzer_AnalyzeHandler(t *testing.T) {
	// Create a simple handler function
	src := `package main

import "github.com/gin-gonic/gin"

type User struct {
	ID   int
	Name string
}

func getUser(c *gin.Context) {
	id := c.Param("id")
	c.JSON(200, User{ID: 1, Name: "test"})
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	files := map[string]*ast.File{"test.go": file}
	analyzer := NewHandlerAnalyzer(fset, files)

	// Find the getUser function
	var handlerDecl *ast.FuncDecl
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "getUser" {
			handlerDecl = fn
			break
		}
	}

	if handlerDecl == nil {
		t.Fatal("getUser function not found")
	}

	info, err := analyzer.AnalyzeHandler(handlerDecl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check path params
	if len(info.PathParams) != 1 {
		t.Errorf("expected 1 path param, got %d", len(info.PathParams))
	}
	if len(info.PathParams) > 0 && info.PathParams[0].Name != "id" {
		t.Errorf("expected param 'id', got %s", info.PathParams[0].Name)
	}

	// Check responses
	if len(info.Responses) != 1 {
		t.Errorf("expected 1 response, got %d", len(info.Responses))
	}
}
```

Run:
```bash
go test ./internal/extractor/gin/... -run TestHandlerAnalyzer_AnalyzeHandler -v
```

Expected: FAIL - method not implemented

**Step 3: Implement AnalyzeHandler method**

```go
// Add to handler_analyzer.go

// AnalyzeHandler analyzes a handler function and extracts information.
func (a *HandlerAnalyzer) AnalyzeHandler(fn *ast.FuncDecl) (*HandlerInfo, error) {
	info := &HandlerInfo{
		PathParams:   []ParamInfo{},
		QueryParams:  []ParamInfo{},
		HeaderParams: []ParamInfo{},
		Responses:    []ResponseInfo{},
	}

	// Inspect the function body
	if fn.Body == nil {
		return info, nil
	}

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			a.parseHandlerCall(node, info)
		}
		return true
	})

	return info, nil
}

// parseHandlerCall parses a call expression in a handler.
func (a *HandlerAnalyzer) parseHandlerCall(call *ast.CallExpr, info *HandlerInfo) {
	// Check for c.Param, c.Query, c.GetHeader, c.ShouldBindJSON, c.JSON
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	// Check if receiver is a context variable
	_, ok = sel.X.(*ast.Ident)
	if !ok {
		return
	}

	method := sel.Sel.Name

	switch method {
	case "Param":
		if len(call.Args) >= 1 {
			if name := extractStringLiteral(call.Args[0]); name != "" {
				info.PathParams = append(info.PathParams, ParamInfo{
					Name:     name,
					GoType:   "string",
					Required: true,
				})
			}
		}
	case "Query":
		if len(call.Args) >= 1 {
			if name := extractStringLiteral(call.Args[0]); name != "" {
				info.QueryParams = append(info.QueryParams, ParamInfo{
					Name:     name,
					GoType:   "string",
					Required: false,
				})
			}
		}
	case "GetHeader":
		if len(call.Args) >= 1 {
			if name := extractStringLiteral(call.Args[0]); name != "" {
				info.HeaderParams = append(info.HeaderParams, ParamInfo{
					Name:     name,
					GoType:   "string",
					Required: false,
				})
			}
		}
	case "ShouldBindJSON", "BindJSON", "ShouldBind":
		if len(call.Args) >= 1 {
			if typeName := extractTypeFromArg(call.Args[0]); typeName != "" {
				info.BodyType = typeName
			}
		}
	case "JSON":
		if len(call.Args) >= 2 {
			statusCode := extractStatusCode(call.Args[0])
			goType := extractTypeFromResponse(call.Args[1])
			info.Responses = append(info.Responses, ResponseInfo{
				StatusCode: statusCode,
				GoType:     goType,
			})
		}
	}
}
```

**Step 4: Add helper functions**

```go
// extractTypeFromArg extracts type name from a binding argument.
func extractTypeFromArg(expr ast.Expr) string {
	// Pattern: &variable or &Struct{}
	unary, ok := expr.(*ast.UnaryExpr)
	if !ok || unary.Op != token.AND {
		return ""
	}

	// Check for composite literal: &Type{}
	if comp, ok := unary.X.(*ast.CompositeLit); ok {
		if sel, ok := comp.Type.(*ast.Ident); ok {
			return sel.Name
		}
		if sel, ok := comp.Type.(*ast.SelectorExpr); ok {
			if x, ok := sel.X.(*ast.Ident); ok {
				return x.Name + "." + sel.Sel.Name
			}
		}
	}

	// Check for variable: &variable
	if ident, ok := unary.X.(*ast.Ident); ok {
		return ident.Name // Return variable name, type would need type checking
	}

	return ""
}

// extractStatusCode extracts HTTP status code from expression.
func extractStatusCode(expr ast.Expr) int {
	// Integer literal
	if lit, ok := expr.(*ast.BasicLit); ok {
		if lit.Kind == token.INT {
			var code int
			// Try to parse
			if _, err := fmt.Sscanf(lit.Value, "%d", &code); err == nil {
				return code
			}
		}
	}

	// http.StatusOK reference
	if sel, ok := expr.(*ast.SelectorExpr); ok {
		if x, ok := sel.X.(*ast.Ident); ok && x.Name == "http" {
			return statusCodeFromName(sel.Sel.Name)
		}
	}

	return 200 // Default
}

// statusCodeFromName converts http.StatusXxx to code.
func statusCodeFromName(name string) int {
	switch name {
	case "StatusOK":
		return 200
	case "StatusCreated":
		return 201
	case "StatusBadRequest":
		return 400
	case "StatusUnauthorized":
		return 401
	case "StatusForbidden":
		return 403
	case "StatusNotFound":
		return 404
	case "StatusInternalServerError":
		return 500
	}
	return 200
}

// extractTypeFromResponse extracts type from response argument.
func extractTypeFromResponse(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.CompositeLit:
		if ident, ok := e.Type.(*ast.Ident); ok {
			return ident.Name
		}
		if sel, ok := e.Type.(*ast.SelectorExpr); ok {
			if x, ok := sel.X.(*ast.Ident); ok {
				return x.Name + "." + sel.Sel.Name
			}
		}
	case *ast.CallExpr:
		// gin.H or similar
		if sel, ok := e.Fun.(*ast.SelectorExpr); ok {
			if sel.Sel.Name == "H" {
				return "map[string]any"
			}
		}
	}
	return ""
}
```

Add import for "fmt".

Run:
```bash
go test ./internal/extractor/gin/... -run TestHandlerAnalyzer_AnalyzeHandler -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/extractor/gin/handler_analyzer.go internal/extractor/gin/handler_analyzer_test.go
git commit -s -m "feat(gin): implement Handler analyzer for param and response extraction"
```

---

## Task 7: Implement Schema Extractor

**Files:**
- Modify: `internal/extractor/gin/schema_extractor.go`
- Modify: `internal/extractor/gin/schema_extractor_test.go`

**Step 1: Add imports and structure**

```go
package gin

import (
	"fmt"
	"go/ast"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// SchemaExtractor extracts OpenAPI schemas from Go structs.
type SchemaExtractor struct {
	files     map[string]*ast.File
	typeCache map[string]*openapi3.SchemaRef
}

// NewSchemaExtractor creates a new SchemaExtractor instance.
func NewSchemaExtractor(files map[string]*ast.File) *SchemaExtractor {
	return &SchemaExtractor{
		files:     files,
		typeCache: make(map[string]*openapi3.SchemaRef),
	}
}
```

**Step 2: Write failing test for ExtractSchema**

```go
// Add to schema_extractor_test.go

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestSchemaExtractor_ExtractSchema(t *testing.T) {
	src := `package main

type User struct {
	ID       int    ` + "`" + `json:"id" binding:"required"` + "`" + `
	Name     string ` + "`" + `json:"name"` + "`" + `
	Email    string ` + "`" + `json:"email" validate:"email"` + "`" + `
	Age      int    ` + "`" + `json:"age,omitempty" validate:"min=0,max=150"` + "`" + `
}
`
	fset := token.NewFileSet()
	file, _ := parser.ParseFile(fset, "test.go", src, 0)
	files := map[string]*ast.File{"test.go": file}

	extractor := NewSchemaExtractor(files)
	schema, err := extractor.ExtractSchema("User")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if schema == nil {
		t.Fatal("expected non-nil schema")
	}

	// Check required fields
	required := schema.Value.Required
	if len(required) != 1 || required[0] != "id" {
		t.Errorf("expected required=['id'], got %v", required)
	}

	// Check properties
	props := schema.Value.Properties
	if len(props) != 4 {
		t.Errorf("expected 4 properties, got %d", len(props))
	}

	// Check email format
	if emailProp := props["email"]; emailProp != nil {
		if emailProp.Value.Format != "email" {
			t.Errorf("expected email format, got %s", emailProp.Value.Format)
		}
	}
}
```

Run:
```bash
go test ./internal/extractor/gin/... -run TestSchemaExtractor_ExtractSchema -v
```

Expected: FAIL - method not implemented

**Step 3: Implement ExtractSchema method**

```go
// Add to schema_extractor.go

// ExtractSchema extracts an OpenAPI schema from a Go type.
func (e *SchemaExtractor) ExtractSchema(typeName string) (*openapi3.SchemaRef, error) {
	// Check cache
	if cached, ok := e.typeCache[typeName]; ok {
		return cached, nil
	}

	// Find the type definition
	typeSpec := e.findTypeSpec(typeName)
	if typeSpec == nil {
		return nil, fmt.Errorf("type %s not found", typeName)
	}

	// Extract schema from struct
	schema, err := e.extractStructSchema(typeSpec)
	if err != nil {
		return nil, err
	}

	ref := &openapi3.SchemaRef{Value: schema}
	e.typeCache[typeName] = ref
	return ref, nil
}

// findTypeSpec finds a type definition by name.
func (e *SchemaExtractor) findTypeSpec(name string) *ast.TypeSpec {
	for _, file := range e.files {
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}
			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if ok && typeSpec.Name.Name == name {
					return typeSpec
				}
			}
		}
	}
	return nil
}

// extractStructSchema extracts schema from a struct type.
func (e *SchemaExtractor) extractStructSchema(typeSpec *ast.TypeSpec) (*openapi3.Schema, error) {
	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		return nil, fmt.Errorf("type %s is not a struct", typeSpec.Name.Name)
	}

	schema := &openapi3.Schema{
		Type:       "object",
		Properties: make(openapi3.Schemas),
	}

	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 {
			continue // Embedded field - skip for now
		}

		fieldName := field.Names[0].Name
		fieldSchema := e.fieldToSchema(field.Type)

		// Parse struct tags
		if field.Tag != nil {
			tag := strings.Trim(field.Tag.Value, "`")
			e.applyTags(fieldSchema, tag, fieldName, schema)
		}

		schema.Properties[fieldName] = &openapi3.SchemaRef{Value: fieldSchema}
	}

	return schema, nil
}
```

**Step 4: Implement type mapping and tag processing**

```go
// fieldToSchema converts a Go type to OpenAPI schema.
func (e *SchemaExtractor) fieldToSchema(expr ast.Expr) *openapi3.Schema {
	switch t := expr.(type) {
	case *ast.Ident:
		return goTypeToSchema(t.Name)
	case *ast.StarExpr:
		// Pointer - unwrap and process underlying type
		return e.fieldToSchema(t.X)
	case *ast.ArrayType:
		itemSchema := e.fieldToSchema(t.Elt)
		return &openapi3.Schema{
			Type:  "array",
			Items: &openapi3.SchemaRef{Value: itemSchema},
		}
	case *ast.MapType:
		valueSchema := e.fieldToSchema(t.Value)
		return &openapi3.Schema{
			Type:                 "object",
			AdditionalProperties: openapi3.AdditionalProperties{Schema: &openapi3.SchemaRef{Value: valueSchema}},
		}
	case *ast.SelectorExpr:
		// Package qualified type (e.g., time.Time)
		if x, ok := t.X.(*ast.Ident); ok {
			fullName := x.Name + "." + t.Sel.Name
			return goTypeToSchema(fullName)
		}
	}

	return &openapi3.Schema{Type: "object"}
}

// goTypeToSchema converts a Go type name to OpenAPI schema.
func goTypeToSchema(goType string) *openapi3.Schema {
	switch goType {
	case "string":
		return &openapi3.Schema{Type: "string"}
	case "int", "int32":
		return &openapi3.Schema{Type: "integer", Format: "int32"}
	case "int64":
		return &openapi3.Schema{Type: "integer", Format: "int64"}
	case "uint", "uint32":
		return &openapi3.Schema{Type: "integer"}
	case "float32":
		return &openapi3.Schema{Type: "number", Format: "float"}
	case "float64":
		return &openapi3.Schema{Type: "number", Format: "double"}
	case "bool":
		return &openapi3.Schema{Type: "boolean"}
	case "time.Time":
		return &openapi3.Schema{Type: "string", Format: "date-time"}
	default:
		return &openapi3.Schema{Type: "object"}
	}
}

// applyTags processes struct tags and updates schema.
func (e *SchemaExtractor) applyTags(schema *openapi3.Schema, tag, fieldName string, parentSchema *openapi3.Schema) {
	// Parse json tag
	if jsonTag := extractTagValue(tag, "json"); jsonTag != "" {
		parts := strings.Split(jsonTag, ",")
		if parts[0] != "" {
			// Rename property
			oldKey := ""
			for k := range parentSchema.Properties {
				if _, ok := parentSchema.Properties[k]; ok && k == fieldName {
					oldKey = k
					break
				}
			}
			if oldKey != "" && parts[0] != fieldName {
				parentSchema.Properties[parts[0]] = parentSchema.Properties[oldKey]
				delete(parentSchema.Properties, oldKey)
			}
		}
		// Check for omitempty
		for _, part := range parts[1:] {
			if part == "omitempty" {
				// Not required
				return
			}
		}
	}

	// Parse binding tag
	if bindingTag := extractTagValue(tag, "binding"); bindingTag != "" {
		if bindingTag == "required" {
			parentSchema.Required = append(parentSchema.Required, fieldName)
		}
	}

	// Parse validate tag
	if validateTag := extractTagValue(tag, "validate"); validateTag != "" {
		e.applyValidation(schema, validateTag, fieldName, parentSchema)
	}
}

// extractTagValue extracts a specific tag value.
func extractTagValue(tag, key string) string {
	prefix := key + ":\""
	start := strings.Index(tag, prefix)
	if start == -1 {
		return ""
	}
	start += len(prefix)
	end := strings.Index(tag[start:], "\"")
	if end == -1 {
		return ""
	}
	return tag[start : start+end]
}

// applyValidation applies validation rules to schema.
func (e *SchemaExtractor) applyValidation(schema *openapi3.Schema, validateTag, fieldName string, parentSchema *openapi3.Schema) {
	rules := strings.Split(validateTag, ",")
	for _, rule := range rules {
		if rule == "required" {
			parentSchema.Required = append(parentSchema.Required, fieldName)
			continue
		}

		// Parse min/max
		if strings.HasPrefix(rule, "min=") {
			var val float64
			fmt.Sscanf(rule, "min=%f", &val)
			schema.Min = &val
		}
		if strings.HasPrefix(rule, "max=") {
			var val float64
			fmt.Sscanf(rule, "max=%f", &val)
			schema.Max = &val
		}

		// Parse minLength/maxLength
		if strings.HasPrefix(rule, "minLength=") {
			var val uint64
			fmt.Sscanf(rule, "minLength=%d", &val)
			schema.MinLength = val
		}
		if strings.HasPrefix(rule, "maxLength=") {
			var val uint64
			fmt.Sscanf(rule, "maxLength=%d", &val)
			schema.MaxLength = &val
		}

		// Parse format validators
		switch rule {
		case "email":
			schema.Format = "email"
		case "url":
			schema.Format = "uri"
		case "uuid":
			schema.Format = "uuid"
		}

		// Parse oneof enum
		if strings.HasPrefix(rule, "oneof=") {
			values := strings.Split(rule[6:], " ")
			for _, v := range values {
				schema.Enum = append(schema.Enum, v)
			}
		}
	}
}
```

Add import for "fmt" and "go/token".

Run:
```bash
go test ./internal/extractor/gin/... -run TestSchemaExtractor_ExtractSchema -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/extractor/gin/schema_extractor.go internal/extractor/gin/schema_extractor_test.go
git commit -s -m "feat(gin): implement Schema extractor for Go struct to OpenAPI conversion"
```

---

## Task 8: Implement Generator

**Files:**
- Modify: `internal/extractor/gin/generator.go`
- Modify: `internal/extractor/gin/generator_test.go`

**Step 1: Add imports and structure**

```go
package gin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/spencercjh/spec-forge/internal/extractor"
	"gopkg.in/yaml.v3"
)

// Generator generates OpenAPI specs from Gin projects using AST parsing.
type Generator struct{}

// NewGenerator creates a new Generator instance.
func NewGenerator() *Generator {
	return &Generator{}
}
```

**Step 2: Write failing test for Generate**

```go
// Add to generator_test.go

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spencercjh/spec-forge/internal/extractor"
)

func TestGenerator_Generate(t *testing.T) {
	// Create a simple Gin project
	dir := t.TempDir()

	goMod := `module test

go 1.21

require github.com/gin-gonic/gin v1.9.1
`
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0644)

	mainGo := `package main

import "github.com/gin-gonic/gin"

type User struct {
	ID   int    ` + "`" + `json:"id"` + "`" + `
	Name string ` + "`" + `json:"name"` + "`" + `
}

func main() {
	r := gin.Default()
	r.GET("/users/:id", getUser)
}

func getUser(c *gin.Context) {
	id := c.Param("id")
	c.JSON(200, User{ID: 1, Name: "test"})
}
`
	os.WriteFile(filepath.Join(dir, "main.go"), []byte(mainGo), 0644)

	g := NewGenerator()
	ctx := context.Background()
	info := &extractor.ProjectInfo{
		Framework:  "gin",
		GinVersion: "v1.9.1",
		MainFiles:  []string{filepath.Join(dir, "main.go")},
	}
	opts := &extractor.GenerateOptions{
		OutputDir:  dir,
		OutputFile: "openapi",
		Format:     "yaml",
	}

	result, err := g.Generate(ctx, dir, info, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Check output file exists
	if _, err := os.Stat(result.OutputPath); err != nil {
		t.Errorf("output file not created: %s", result.OutputPath)
	}
}
```

Run:
```bash
go test ./internal/extractor/gin/... -run TestGenerator_Generate -v
```

Expected: FAIL - method not implemented

**Step 3: Implement Generate method**

```go
// Generate generates OpenAPI spec from Gin project.
func (g *Generator) Generate(ctx context.Context, projectPath string, info *extractor.ProjectInfo, opts *extractor.GenerateOptions) (*extractor.GenerateResult, error) {
	// Step 1: Parse AST files
	parser := NewASTParser(projectPath)
	if err := parser.ParseFiles(); err != nil {
		return nil, fmt.Errorf("failed to parse files: %w", err)
	}

	// Step 2: Extract routes
	routes, err := parser.ExtractRoutes()
	if err != nil {
		return nil, fmt.Errorf("failed to extract routes: %w", err)
	}

	// Step 3: Analyze handlers
	analyzer := NewHandlerAnalyzer(parser.fset, parser.files)
	handlerInfos := make(map[string]*HandlerInfo)
	for _, route := range routes {
		// Find handler function
		handlerDecl := g.findHandlerDecl(route.HandlerName, parser.files)
		if handlerDecl != nil {
			handlerInfo, err := analyzer.AnalyzeHandler(handlerDecl)
			if err != nil {
				continue // Log but continue
			}
			handlerInfos[route.HandlerName] = handlerInfo
		}
	}

	// Step 4: Extract schemas
	extractor := NewSchemaExtractor(parser.files)
	schemas := make(openapi3.Schemas)
	for _, handlerInfo := range handlerInfos {
		if handlerInfo.BodyType != "" {
			if schema, err := extractor.ExtractSchema(handlerInfo.BodyType); err == nil {
				schemas[handlerInfo.BodyType] = schema
			}
		}
		for _, resp := range handlerInfo.Responses {
			if resp.GoType != "" && resp.GoType != "map[string]any" {
				if schema, err := extractor.ExtractSchema(resp.GoType); err == nil {
					schemas[resp.GoType] = schema
				}
			}
		}
	}

	// Step 5: Build OpenAPI document
	doc := g.buildOpenAPIDoc(info, routes, handlerInfos, schemas)

	// Step 6: Write output
	outputPath, err := g.writeOutput(doc, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to write output: %w", err)
	}

	return &extractor.GenerateResult{
		OutputPath: outputPath,
	}, nil
}

// findHandlerDecl finds a handler function declaration by name.
func (g *Generator) findHandlerDecl(name string, files map[string]*ast.File) *ast.FuncDecl {
	for _, file := range files {
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == name {
				return fn
			}
		}
	}
	return nil
}
```

**Step 4: Implement buildOpenAPIDoc and writeOutput**

```go
// buildOpenAPIDoc builds the OpenAPI document.
func (g *Generator) buildOpenAPIDoc(info *extractor.ProjectInfo, routes []Route, handlerInfos map[string]*HandlerInfo, schemas openapi3.Schemas) *openapi3.T {
	doc := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Gin API",
			Version: "1.0.0",
		},
		Paths:      make(openapi3.Paths),
		Components: &openapi3.Components{Schemas: schemas},
	}

	if info.ModuleName != "" {
		doc.Info.Title = info.ModuleName
	}

	// Build paths
	for _, route := range routes {
		pathItem := doc.Paths[route.FullPath]
		if pathItem == nil {
			pathItem = &openapi3.PathItem{}
			doc.Paths[route.FullPath] = pathItem
		}

		operation := g.buildOperation(route, handlerInfos[route.HandlerName])
		setOperationForMethod(pathItem, route.Method, operation)
	}

	return doc
}

// buildOperation builds an OpenAPI operation from a route.
func (g *Generator) buildOperation(route Route, handlerInfo *HandlerInfo) *openapi3.Operation {
	operation := &openapi3.Operation{
		OperationID: route.HandlerName,
		Summary:     route.HandlerName,
	}

	if handlerInfo == nil {
		return operation
	}

	// Add parameters
	operation.Parameters = make(openapi3.Parameters, 0)

	// Path parameters
	for _, param := range handlerInfo.PathParams {
		operation.Parameters = append(operation.Parameters, &openapi3.ParameterRef{
			Value: &openapi3.Parameter{
				Name:     param.Name,
				In:       "path",
				Required: param.Required,
				Schema:   &openapi3.SchemaRef{Value: &openapi3.Schema{Type: "string"}},
			},
		})
	}

	// Query parameters
	for _, param := range handlerInfo.QueryParams {
		operation.Parameters = append(operation.Parameters, &openapi3.ParameterRef{
			Value: &openapi3.Parameter{
				Name:     param.Name,
				In:       "query",
				Required: param.Required,
				Schema:   &openapi3.SchemaRef{Value: &openapi3.Schema{Type: "string"}},
			},
		})
	}

	// Request body
	if handlerInfo.BodyType != "" {
		operation.RequestBody = &openapi3.RequestBodyRef{
			Value: &openapi3.RequestBody{
				Content: openapi3.Content{
					"application/json": {
						Schema: &openapi3.SchemaRef{
							Ref: "#/components/schemas/" + handlerInfo.BodyType,
						},
					},
				},
			},
		}
	}

	// Responses
	operation.Responses = make(openapi3.Responses)
	for _, resp := range handlerInfo.Responses {
		response := &openapi3.Response{
			Description: &[]string{"Response"}[0],
			Content:     openapi3.Content{},
		}

		if resp.GoType != "" {
			if resp.GoType == "map[string]any" || resp.GoType == "" {
				response.Content["application/json"] = &openapi3.MediaType{
					Schema: &openapi3.SchemaRef{Value: &openapi3.Schema{Type: "object"}},
				}
			} else {
				response.Content["application/json"] = &openapi3.MediaType{
					Schema: &openapi3.SchemaRef{
						Ref: "#/components/schemas/" + resp.GoType,
					},
				}
			}
		}

		statusCode := fmt.Sprintf("%d", resp.StatusCode)
		operation.Responses[statusCode] = &openapi3.ResponseRef{Value: response}
	}

	// Default response if none specified
	if len(handlerInfo.Responses) == 0 {
		desc := "Success"
		operation.Responses["200"] = &openapi3.ResponseRef{
			Value: &openapi3.Response{
				Description: &desc,
			},
		}
	}

	return operation
}

// setOperationForMethod sets the operation for the given HTTP method.
func setOperationForMethod(pathItem *openapi3.PathItem, method string, operation *openapi3.Operation) {
	switch method {
	case "GET":
		pathItem.Get = operation
	case "POST":
		pathItem.Post = operation
	case "PUT":
		pathItem.Put = operation
	case "DELETE":
		pathItem.Delete = operation
	case "PATCH":
		pathItem.Patch = operation
	case "HEAD":
		pathItem.Head = operation
	case "OPTIONS":
		pathItem.Options = operation
	}
}

// writeOutput writes the OpenAPI document to file.
func (g *Generator) writeOutput(doc *openapi3.T, opts *extractor.GenerateOptions) (string, error) {
	outputDir := opts.OutputDir
	if outputDir == "" {
		outputDir = "."
	}

	outputFile := opts.OutputFile
	if outputFile == "" {
		outputFile = "openapi"
	}

	format := opts.Format
	if format == "" {
		format = "yaml"
	}

	var data []byte
	var err error
	var ext string

	switch format {
	case "json":
		data, err = doc.MarshalJSON()
		ext = ".json"
	default: // yaml
		data, err = yaml.Marshal(doc)
		ext = ".yaml"
	}

	if err != nil {
		return "", err
	}

	outputPath := filepath.Join(outputDir, outputFile+ext)
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return "", err
	}

	return outputPath, nil
}
```

Add import for "go/ast".

Run:
```bash
go test ./internal/extractor/gin/... -run TestGenerator_Generate -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/extractor/gin/generator.go internal/extractor/gin/generator_test.go
git commit -s -m "feat(gin): implement Generator with OpenAPI document building"
```

---

## Task 9: Register Gin Extractor in factory

**Files:**
- Modify: `internal/extractor/factory.go`
- Modify: `internal/extractor/factory_test.go`

**Step 1: Update factory to include Gin**

```go
// Add to internal/extractor/factory.go

import "github.com/spencercjh/spec-forge/internal/extractor/gin"

// In CreateExtractor function, add:
func CreateExtractor(framework string) (Extractor, error) {
	switch framework {
	case "spring":
		return spring.NewSpringExtractor(), nil
	case "gozero":
		return gozero.NewGoZeroExtractor(), nil
	case "gin":
		return gin.NewGinExtractor(), nil
	default:
		return nil, fmt.Errorf("unsupported framework: %s", framework)
	}
}
```

**Step 2: Add test for Gin extractor creation**

```go
// Add to factory_test.go

func TestCreateExtractor_Gin(t *testing.T) {
	extractor, err := CreateExtractor("gin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if extractor == nil {
		t.Error("expected non-nil extractor")
	}
}
```

Run:
```bash
go test ./internal/extractor/... -run TestCreateExtractor_Gin -v
```

Expected: PASS

**Step 3: Commit**

```bash
git add internal/extractor/factory.go internal/extractor/factory_test.go
git commit -s -m "feat(extractor): register Gin extractor in factory"
```

---

## Task 10: Update CLI for automatic Gin detection

**Files:**
- Modify: `cmd/generate.go`

**Step 1: Update detectFramework function**

```go
// Add to cmd/generate.go

func detectFramework(projectPath string) (string, error) {
	// Try Gin first
	ginDetector := gin.NewDetector()
	if _, err := ginDetector.Detect(projectPath); err == nil {
		return "gin", nil
	}

	// Then go-zero
	goZeroDetector := gozero.NewDetector()
	if _, err := goZeroDetector.Detect(projectPath); err == nil {
		return "gozero", nil
	}

	// Then Spring Boot
	springDetector := spring.NewDetector()
	if _, err := springDetector.Detect(projectPath); err == nil {
		return "spring", nil
	}

	return "", fmt.Errorf("could not detect framework for project: %s", projectPath)
}
```

Add import for gin detector.

**Step 2: Run tests**

```bash
go build ./cmd/...
go test ./cmd/... -v
```

Expected: Tests pass

**Step 3: Commit**

```bash
git add cmd/generate.go
git commit -s -m "feat(cmd): add automatic Gin framework detection"
```

---

## Task 11: Create integration test example

**Files:**
- Create: `integration-tests/gin-demo/` directory with sample project
- Create: `integration-tests/gin_demo_test.go`

**Step 1: Create example Gin project**

```bash
mkdir -p integration-tests/gin-demo
```

**Step 2: Create go.mod**

```go
// integration-tests/gin-demo/go.mod
module gin-demo

go 1.21

require github.com/gin-gonic/gin v1.9.1
```

**Step 3: Create main.go**

```go
// integration-tests/gin-demo/main.go
package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type User struct {
	ID    int    `json:"id" binding:"required"`
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
	Age   int    `json:"age,omitempty"`
}

type CreateUserRequest struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
	Age   int    `json:"age,omitempty"`
}

func main() {
	r := gin.Default()

	// Direct routes
	r.GET("/users", listUsers)
	r.POST("/users", createUser)

	// Route group
	api := r.Group("/api/v1")
	api.GET("/users/:id", getUser)
	api.PUT("/users/:id", updateUser)
	api.DELETE("/users/:id", deleteUser)

	r.Run()
}

func listUsers(c *gin.Context) {
	page := c.Query("page")
	size := c.DefaultQuery("size", "10")
	_ = page
	_ = size
	c.JSON(http.StatusOK, []User{})
}

func createUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, User{ID: 1, Name: req.Name})
}

func getUser(c *gin.Context) {
	id := c.Param("id")
	_ = id
	c.JSON(http.StatusOK, User{ID: 1, Name: "Test"})
}

func updateUser(c *gin.Context) {
	id := c.Param("id")
	_ = id
	var req CreateUserRequest
	c.ShouldBindJSON(&req)
	c.JSON(http.StatusOK, User{ID: 1})
}

func deleteUser(c *gin.Context) {
	id := c.Param("id")
	_ = id
	c.Status(http.StatusNoContent)
}
```

**Step 4: Create integration test**

```go
// integration-tests/gin_demo_test.go
//go:build e2e

package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/spencercjh/spec-forge/internal/extractor"
	"github.com/spencercjh/spec-forge/internal/extractor/gin"
)

func TestGinDemo(t *testing.T) {
	projectPath := "./gin-demo"

	// Detect project
	detector := gin.NewDetector()
	info, err := detector.Detect(projectPath)
	if err != nil {
		t.Fatalf("failed to detect project: %v", err)
	}

	if info.Framework != "gin" {
		t.Errorf("expected framework 'gin', got %s", info.Framework)
	}

	// Generate OpenAPI spec
	generator := gin.NewGenerator()
	ctx := t.Context()
	opts := &extractor.GenerateOptions{
		OutputDir:  t.TempDir(),
		OutputFile: "openapi",
		Format:     "yaml",
	}

	result, err := generator.Generate(ctx, projectPath, info, opts)
	if err != nil {
		t.Fatalf("failed to generate spec: %v", err)
	}

	// Verify output file exists
	if _, err := os.Stat(result.OutputPath); err != nil {
		t.Fatalf("output file not created: %s", result.OutputPath)
	}

	// Load and validate the spec
	loader := openapi3.NewLoader()
	spec, err := loader.LoadFromFile(result.OutputPath)
	if err != nil {
		t.Fatalf("failed to load spec: %v", err)
	}

	// Validate spec
	if err := spec.Validate(loader.Context); err != nil {
		t.Errorf("spec validation failed: %v", err)
	}

	// Check paths
	if len(spec.Paths) == 0 {
		t.Error("expected at least one path")
	}

	// Check schemas
	if spec.Components == nil || len(spec.Components.Schemas) == 0 {
		t.Error("expected at least one schema")
	}
}
```

**Step 5: Run integration test**

```bash
go test -v -tags=e2e ./integration-tests/... -run TestGinDemo
```

Expected: PASS (if dependencies are available)

**Step 6: Commit**

```bash
git add integration-tests/gin-demo/ integration-tests/gin_demo_test.go
git commit -s -m "test(integration): add Gin demo project and e2e test"
```

---

## Task 12: Update documentation

**Files:**
- Modify: `README.md`
- Modify: `AGENTS.md`

**Step 1: Update README.md with Gin support**

Add to README.md in the Framework Support section:

```markdown
### Gin Framework

For Gin projects, spec-forge uses AST parsing to extract routes from Go source code:

```bash
spec-forge generate ./my-gin-project
```

Features:
- Automatic detection of Gin projects (via go.mod)
- Route extraction from Go source files
- Support for route groups
- Parameter extraction (path, query, header, body)
- Schema generation from Go structs
- Support for json/binding/validate tags
```

**Step 2: Update AGENTS.md with Gin development notes**

Add to AGENTS.md:

```markdown
### Gin Framework Development

The Gin extractor is located in `internal/extractor/gin/`.

**Architecture:**
- Uses Go AST (go/ast, go/parser) for static analysis
- No runtime execution required (unlike Spring Boot)
- Patcher is a no-op (no dependencies to install)

**Key Components:**
- `ASTParser` - Parses Go files and extracts routes
- `HandlerAnalyzer` - Analyzes handler functions for params/responses
- `SchemaExtractor` - Converts Go structs to OpenAPI schemas

**Testing:**
- Example project: `integration-tests/gin-demo/`
- Run e2e test: `go test -v -tags=e2e ./integration-tests/... -run TestGinDemo`
```

**Step 3: Commit**

```bash
git add README.md AGENTS.md
git commit -s -m "docs: update README and AGENTS with Gin support documentation"
```

---

## Summary

This implementation plan adds complete Gin framework support to spec-forge:

1. **Package Structure** - Created all necessary files with stubs
2. **Types** - Defined data structures for Gin projects
3. **Detector** - Detects Gin projects via go.mod parsing
4. **Patcher** - No-op implementation (Gin doesn't need patching)
5. **AST Parser** - Extracts routes from Go source code
6. **Handler Analyzer** - Analyzes handler functions
7. **Schema Extractor** - Converts Go structs to OpenAPI schemas
8. **Generator** - Assembles complete OpenAPI documents
9. **Factory Integration** - Registered Gin extractor
10. **CLI Integration** - Added automatic detection
11. **Integration Tests** - Created demo project and e2e test
12. **Documentation** - Updated README and AGENTS

The implementation follows the same patterns as existing Spring Boot and go-zero extractors for consistency.
