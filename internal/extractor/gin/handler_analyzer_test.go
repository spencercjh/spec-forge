package gin

import (
	"go/parser"
	"go/token"
	"testing"

	"go/ast"
)

func TestNewHandlerAnalyzer(t *testing.T) {
	a := NewHandlerAnalyzer(nil, nil)
	if a == nil {
		t.Error("expected non-nil analyzer")
	}
}

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
	_ = id
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

func TestHandlerAnalyzer_AnalyzeHandler_WithQueryParams(t *testing.T) {
	src := `package main

import "github.com/gin-gonic/gin"

func listUsers(c *gin.Context) {
	page := c.Query("page")
	size := c.DefaultQuery("size", "10")
	_ = page
	_ = size
	c.JSON(200, nil)
}
`
	fset := token.NewFileSet()
	file, _ := parser.ParseFile(fset, "test.go", src, 0)
	files := map[string]*ast.File{"test.go": file}
	analyzer := NewHandlerAnalyzer(fset, files)

	var handlerDecl *ast.FuncDecl
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "listUsers" {
			handlerDecl = fn
			break
		}
	}

	info, _ := analyzer.AnalyzeHandler(handlerDecl)

	if len(info.QueryParams) != 2 {
		t.Errorf("expected 2 query params, got %d", len(info.QueryParams))
	}
}

func TestHandlerAnalyzer_AnalyzeHandler_WithBodyBinding(t *testing.T) {
	src := `package main

import "github.com/gin-gonic/gin"

type CreateUserRequest struct {
	Name string ` + "`" + `json:"name"` + "`" + `
}

func createUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, req)
}
`
	fset := token.NewFileSet()
	file, _ := parser.ParseFile(fset, "test.go", src, 0)
	files := map[string]*ast.File{"test.go": file}
	analyzer := NewHandlerAnalyzer(fset, files)

	var handlerDecl *ast.FuncDecl
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "createUser" {
			handlerDecl = fn
			break
		}
	}

	info, _ := analyzer.AnalyzeHandler(handlerDecl)

	// Note: For variable binding (&req), we return the variable name.
	// Full type resolution would require type checking.
	if info.BodyType != "req" && info.BodyType != "CreateUserRequest" {
		t.Errorf("expected BodyType 'req' or 'CreateUserRequest', got %s", info.BodyType)
	}

	// Check that we have both error and success responses
	if len(info.Responses) != 2 {
		t.Errorf("expected 2 responses (400 and 201), got %d", len(info.Responses))
	}
}

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
