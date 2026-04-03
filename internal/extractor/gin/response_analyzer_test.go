package gin

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestExtractStatusCode(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected int
	}{
		{"200", "200", 200},
		{"201", "201", 201},
		{"400", "400", 400},
		{"404", "404", 404},
		{"500", "500", 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a simple expression with the status code
			src := `package main
	func test() { _ = ` + tt.code + ` }`
			fset := token.NewFileSet()
			file, _ := parser.ParseFile(fset, "test.go", src, 0)

			// Extract the basic lit from the assignment
			var lit *ast.BasicLit
			ast.Inspect(file, func(n ast.Node) bool {
				if l, ok := n.(*ast.BasicLit); ok {
					lit = l
					return false
				}
				return true
			})

			if lit == nil {
				t.Fatal("failed to find basic lit")
			}

			result := extractStatusCode(lit)
			if result != tt.expected {
				t.Errorf("extractStatusCode() = %d, expected %d", result, tt.expected)
			}
		})
	}
}

func TestStatusCodeFromName(t *testing.T) {
	tests := []struct {
		name     string
		expected int
	}{
		{"StatusOK", 200},
		{"StatusCreated", 201},
		{"StatusNoContent", 204},
		{"StatusBadRequest", 400},
		{"StatusNotFound", 404},
		{"StatusInternalServerError", 500},
		{"UnknownStatus", 200}, // default
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := statusCodeFromName(tt.name)
			if result != tt.expected {
				t.Errorf("statusCodeFromName(%q) = %d, expected %d", tt.name, result, tt.expected)
			}
		})
	}
}

func TestHandlerAnalyzer_StringResponse(t *testing.T) {
	src := `package main

import "github.com/gin-gonic/gin"

func hello(c *gin.Context) {
	c.String(200, "Hello World")
}
`
	fset := token.NewFileSet()
	file, _ := parser.ParseFile(fset, "test.go", src, 0)
	files := map[string]*ast.File{"test.go": file}
	analyzer := NewHandlerAnalyzer(fset, files)

	var handlerDecl *ast.FuncDecl
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "hello" {
			handlerDecl = fn
			break
		}
	}

	info, _ := analyzer.AnalyzeHandler(handlerDecl)

	// Check for string response
	found200 := false
	for _, resp := range info.Responses {
		if resp.StatusCode == 200 && resp.GoType == "string" {
			found200 = true
		}
	}
	if !found200 {
		t.Error("expected 200 response with string type")
	}
}

func TestHandlerAnalyzer_Redirect(t *testing.T) {
	src := `package main

import "github.com/gin-gonic/gin"

func redirect(c *gin.Context) {
	c.Redirect(302, "/new-location")
}
`
	fset := token.NewFileSet()
	file, _ := parser.ParseFile(fset, "test.go", src, 0)
	files := map[string]*ast.File{"test.go": file}
	analyzer := NewHandlerAnalyzer(fset, files)

	var handlerDecl *ast.FuncDecl
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "redirect" {
			handlerDecl = fn
			break
		}
	}

	info, _ := analyzer.AnalyzeHandler(handlerDecl)

	// Check for redirect response
	found302 := false
	for _, resp := range info.Responses {
		if resp.StatusCode == 302 {
			found302 = true
		}
	}
	if !found302 {
		t.Error("expected 302 redirect response")
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
