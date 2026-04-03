package gin

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

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
