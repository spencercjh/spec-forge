package gin

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestCrossFunctionCallTracking(t *testing.T) {
	// Test that custom helpers (not in knownResponseHelperNames) are traced
	// when they accept *gin.Context as first parameter
	src := `package main

import "github.com/gin-gonic/gin"

type Project struct {
	Name string
}

func myHelper(c *gin.Context, data any, err error) {
	if err != nil {
		c.JSON(400, gin.H{"error": "bad"})
		return
	}
	c.JSON(200, data)
}

func GetProject(c *gin.Context) {
	project := Project{Name: "test"}
	myHelper(c, project, nil)
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	files := map[string]*ast.File{"test.go": file}

	analyzer := NewHandlerAnalyzer(fset, files)

	// Find the GetProject handler
	var fn *ast.FuncDecl
	for _, decl := range file.Decls {
		if f, ok := decl.(*ast.FuncDecl); ok && f.Name.Name == "GetProject" {
			fn = f
			break
		}
	}
	if fn == nil {
		t.Fatal("GetProject handler not found")
	}

	info, err := analyzer.AnalyzeHandler(fn)
	if err != nil {
		t.Fatalf("failed to analyze: %v", err)
	}

	// Should have found responses by tracing into myHelper
	if len(info.Responses) == 0 {
		t.Fatal("expected at least one response from traced helper")
	}

	found200 := false
	for _, resp := range info.Responses {
		if resp.StatusCode == 200 {
			found200 = true
			if resp.GoType == "" {
				t.Error("expected non-empty GoType for 200 response")
			}
		}
	}
	if !found200 {
		t.Error("expected 200 response from traced helper")
	}
}

func TestParameterTypeInference_Strconv(t *testing.T) {
	src := `package main

import (
	"github.com/gin-gonic/gin"
	"strconv"
)

func listItems(c *gin.Context) {
	offset, _ := strconv.Atoi(c.Query("offset"))
	limit := c.Query("limit")
	_ = offset
	_ = limit
	c.JSON(200, gin.H{"ok": true})
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	files := map[string]*ast.File{"test.go": file}

	analyzer := NewHandlerAnalyzer(fset, files)

	var fn *ast.FuncDecl
	for _, decl := range file.Decls {
		if f, ok := decl.(*ast.FuncDecl); ok && f.Name.Name == "listItems" {
			fn = f
			break
		}
	}
	if fn == nil {
		t.Fatal("listItems handler not found")
	}

	info, err := analyzer.AnalyzeHandler(fn)
	if err != nil {
		t.Fatalf("failed to analyze: %v", err)
	}

	// offset should be inferred as integer from strconv.Atoi
	foundOffset := false
	for _, p := range info.QueryParams {
		if p.Name == "offset" {
			foundOffset = true
			if p.GoType != "integer" {
				t.Errorf("expected offset GoType 'integer', got %q", p.GoType)
			}
		}
	}
	if !foundOffset {
		t.Error("expected 'offset' query param")
	}

	// limit should remain string (no strconv wrapping)
	foundLimit := false
	for _, p := range info.QueryParams {
		if p.Name == "limit" {
			foundLimit = true
			if p.GoType != "string" {
				t.Errorf("expected limit GoType 'string', got %q", p.GoType)
			}
		}
	}
	if !foundLimit {
		t.Error("expected 'limit' query param")
	}
}

func TestParameterTypeInference_BoolComparison(t *testing.T) {
	src := `package main

import "github.com/gin-gonic/gin"

func getItems(c *gin.Context) {
	verbose := c.Query("verbose")
	if verbose == "true" {
		_ = verbose
	}
	c.JSON(200, nil)
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	files := map[string]*ast.File{"test.go": file}

	analyzer := NewHandlerAnalyzer(fset, files)

	var fn *ast.FuncDecl
	for _, decl := range file.Decls {
		if f, ok := decl.(*ast.FuncDecl); ok && f.Name.Name == "getItems" {
			fn = f
			break
		}
	}

	info, err := analyzer.AnalyzeHandler(fn)
	if err != nil {
		t.Fatalf("failed to analyze: %v", err)
	}

	foundVerbose := false
	for _, p := range info.QueryParams {
		if p.Name == "verbose" {
			foundVerbose = true
			if p.GoType != "boolean" {
				t.Errorf("expected verbose GoType 'boolean', got %q", p.GoType)
			}
		}
	}
	if !foundVerbose {
		t.Error("expected 'verbose' query param")
	}
}

func TestIsKnownResponseHelper(t *testing.T) {
	tests := []struct {
		name     string
		fnName   string
		expected bool
	}{
		{"done", "done", true},
		{"respond", "respond", true},
		{"response", "response", true},
		{"writeJSON", "writeJSON", true},
		{"sendJSON", "sendJSON", true},
		{"reply", "reply", true},
		{"unknown", "handleResponse", false},
		{"JSON", "JSON", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := isKnownResponseHelper(tt.fnName); result != tt.expected {
				t.Errorf("isKnownResponseHelper(%q) = %v, expected %v", tt.fnName, result, tt.expected)
			}
		})
	}
}

func TestParseHelperCall_DoneWithData(t *testing.T) {
	src := `package main

import "github.com/gin-gonic/gin"

func CreateProject(c *gin.Context) {
	var project models.Project
	done(c, project, nil)
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	files := map[string]*ast.File{"test.go": file}

	analyzer := NewHandlerAnalyzer(fset, files)

	var fn *ast.FuncDecl
	for _, decl := range file.Decls {
		if f, ok := decl.(*ast.FuncDecl); ok && f.Name.Name == "CreateProject" {
			fn = f
			break
		}
	}
	if fn == nil {
		t.Fatal("handler function not found")
	}

	info, err := analyzer.AnalyzeHandler(fn)
	if err != nil {
		t.Fatalf("failed to analyze handler: %v", err)
	}

	if len(info.Responses) == 0 {
		t.Fatal("expected at least one response from done() helper")
	}

	foundSuccess := false
	for _, resp := range info.Responses {
		if resp.StatusCode == 200 {
			foundSuccess = true
			if resp.GoType == "" {
				t.Error("expected non-empty GoType for success response")
			}
		}
	}
	if !foundSuccess {
		t.Error("expected 200 success response from done(c, project, nil)")
	}
}

func TestParseHelperCall_DoneWithCompositeLit(t *testing.T) {
	src := `package main

import "github.com/gin-gonic/gin"

func GetUsers(c *gin.Context) {
	users := []User{{Name: "a"}}
	done(c, users)
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	files := map[string]*ast.File{"test.go": file}

	analyzer := NewHandlerAnalyzer(fset, files)
	var fn *ast.FuncDecl
	for _, decl := range file.Decls {
		if f, ok := decl.(*ast.FuncDecl); ok && f.Name.Name == "GetUsers" {
			fn = f
			break
		}
	}

	info, err := analyzer.AnalyzeHandler(fn)
	if err != nil {
		t.Fatalf("failed to analyze handler: %v", err)
	}

	if len(info.Responses) == 0 {
		t.Fatal("expected at least one response from done() helper")
	}
}

func TestParseHelperCall_NotHelper(t *testing.T) {
	src := `package main

import "github.com/gin-gonic/gin"

func Handler(c *gin.Context) {
	process(c) // not a known helper
	c.JSON(200, gin.H{"ok": true})
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	files := map[string]*ast.File{"test.go": file}

	analyzer := NewHandlerAnalyzer(fset, files)
	var fn *ast.FuncDecl
	for _, decl := range file.Decls {
		if f, ok := decl.(*ast.FuncDecl); ok && f.Name.Name == "Handler" {
			fn = f
			break
		}
	}

	info, err := analyzer.AnalyzeHandler(fn)
	if err != nil {
		t.Fatalf("failed to analyze handler: %v", err)
	}

	// Should only have the c.JSON response, not from process()
	if len(info.Responses) != 1 {
		t.Errorf("expected 1 response (from c.JSON only), got %d", len(info.Responses))
	}
}

func TestParseHelperCall_MethodCallIgnored(t *testing.T) {
	src := `package main

import "github.com/gin-gonic/gin"

func Handler(c *gin.Context) {
	c.done() // method call, should not match helper
	c.JSON(200, gin.H{"ok": true})
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	files := map[string]*ast.File{"test.go": file}

	analyzer := NewHandlerAnalyzer(fset, files)
	var fn *ast.FuncDecl
	for _, decl := range file.Decls {
		if f, ok := decl.(*ast.FuncDecl); ok && f.Name.Name == "Handler" {
			fn = f
			break
		}
	}

	info, err := analyzer.AnalyzeHandler(fn)
	if err != nil {
		t.Fatalf("failed to analyze handler: %v", err)
	}

	// c.done() is a method call (*ast.SelectorExpr), not bare function call
	// Should only have c.JSON response
	if len(info.Responses) != 1 {
		t.Errorf("expected 1 response (c.JSON only), got %d: %+v", len(info.Responses), info.Responses)
	}
}

func TestIsErrorType(t *testing.T) {
	if isErrorType(&ast.Ident{Name: "nil"}, nil) {
		t.Error("expected nil ident to NOT be error type")
	}
	if isErrorType(&ast.Ident{Name: "e"}, nil) {
		t.Error("expected single-letter 'e' to NOT be error type")
	}
	if !isErrorType(&ast.Ident{Name: "err"}, nil) {
		t.Error("expected 'err' ident to be error type")
	}
	if isErrorType(&ast.Ident{Name: "data"}, nil) {
		t.Error("expected 'data' ident to NOT be error type")
	}
	varTypeMap := map[string]string{"myErr": "error"}
	if !isErrorType(&ast.Ident{Name: "myErr"}, varTypeMap) {
		t.Error("expected variable with error type to be error type")
	}
	varTypeMap2 := map[string]string{"data": "string"}
	if isErrorType(&ast.Ident{Name: "data"}, varTypeMap2) {
		t.Error("expected variable with string type to NOT be error type")
	}
}
