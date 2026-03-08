package gin

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

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

	info, _ := analyzer.AnalyzeHandler(handlerDecl)

	// Check for form parameters
	foundFile := false
	foundUserID := false
	for _, param := range info.QueryParams {
		if param.Name == "file" {
			foundFile = true
			if param.GoType != "file" {
				t.Errorf("expected file type for 'file', got '%s'", param.GoType)
			}
		}
		if param.Name == "userId" {
			foundUserID = true
		}
	}
	if !foundFile {
		t.Error("expected 'file' parameter")
	}
	if !foundUserID {
		t.Error("expected 'userId' parameter")
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
