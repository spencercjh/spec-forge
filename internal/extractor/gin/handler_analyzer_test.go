package gin

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
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
	if handlerDecl == nil {
		t.Fatal("listUsers handler not found")
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
	if handlerDecl == nil {
		t.Fatal("handler not found")
	}

	info, _ := analyzer.AnalyzeHandler(handlerDecl)

	if info.BodyType != "CreateUserRequest" {
		t.Errorf("expected BodyType 'CreateUserRequest', got %s", info.BodyType)
	}

	// Check that we have both error and success responses
	if len(info.Responses) != 2 {
		t.Errorf("expected 2 responses (400 and 201), got %d", len(info.Responses))
	}
}

func TestHandlerAnalyzer_XMLBinding(t *testing.T) {
	src := `package main

import "github.com/gin-gonic/gin"

type User struct {
	Name string
}

func createUser(c *gin.Context) {
	var user User
	if err := c.ShouldBindXML(&user); err != nil {
		c.XML(400, gin.H{"error": err.Error()})
		return
	}
	c.XML(201, user)
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
	if handlerDecl == nil {
		t.Fatal("handler not found")
	}

	info, _ := analyzer.AnalyzeHandler(handlerDecl)

	if info.BodyType != "User" {
		t.Errorf("expected BodyType 'User', got '%s'", info.BodyType)
	}

	// Check responses - should have XML responses
	found201 := false
	found400 := false
	for _, resp := range info.Responses {
		if resp.StatusCode == 201 {
			found201 = true
		}
		if resp.StatusCode == 400 {
			found400 = true
		}
	}
	if !found201 {
		t.Error("expected 201 response")
	}
	if !found400 {
		t.Error("expected 400 response")
	}
}

func TestHandlerAnalyzer_QueryBinding(t *testing.T) {
	src := `package main

import "github.com/gin-gonic/gin"

type ListRequest struct {
	Page int
	Size int
}

func listUsers(c *gin.Context) {
	var req ListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, req)
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
	if handlerDecl == nil {
		t.Fatal("handler not found")
	}

	info, _ := analyzer.AnalyzeHandler(handlerDecl)

	if info.BodyType != "ListRequest" {
		t.Errorf("expected BodyType 'ListRequest', got '%s'", info.BodyType)
	}
}

func TestHandlerAnalyzer_FormData(t *testing.T) {
	src := `package main

import "github.com/gin-gonic/gin"

func uploadFile(c *gin.Context) {
	file, _ := c.FormFile("file")
	userID := c.PostForm("userId")
	_ = file
	_ = userID
	c.JSON(200, gin.H{"message": "uploaded"})
}
`
	fset := token.NewFileSet()
	file, _ := parser.ParseFile(fset, "test.go", src, 0)
	files := map[string]*ast.File{"test.go": file}
	analyzer := NewHandlerAnalyzer(fset, files)

	var handlerDecl *ast.FuncDecl
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "uploadFile" {
			handlerDecl = fn
			break
		}
	}
	if handlerDecl == nil {
		t.Fatal("handler not found")
	}

	info, _ := analyzer.AnalyzeHandler(handlerDecl)

	// Check for file parameter in FileParams
	foundFile := false
	for _, param := range info.FileParams {
		if param.Name == "file" {
			foundFile = true
			if param.GoType != "file" {
				t.Errorf("expected file type for 'file', got '%s'", param.GoType)
			}
		}
	}
	if !foundFile {
		t.Error("expected 'file' parameter in FileParams")
	}

	// Check for form parameter in FormParams
	foundUserID := false
	for _, param := range info.FormParams {
		if param.Name == "userId" {
			foundUserID = true
		}
	}
	if !foundUserID {
		t.Error("expected 'userId' parameter in FormParams")
	}
}
