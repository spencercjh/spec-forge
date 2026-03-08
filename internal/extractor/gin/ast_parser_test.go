package gin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewASTParser(t *testing.T) {
	p := NewASTParser("/tmp")
	if p == nil {
		t.Error("expected non-nil parser")
	}
}

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
	os.WriteFile(filepath.Join(dir, "main.go"), []byte(code), 0o644)

	parser := NewASTParser(dir)
	err := parser.ParseFiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(parser.files) != 1 {
		t.Errorf("expected 1 file, got %d", len(parser.files))
	}
}

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
	os.WriteFile(filepath.Join(dir, "main.go"), []byte(code), 0o644)

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
	os.WriteFile(filepath.Join(dir, "main.go"), []byte(code), 0o644)

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

func TestIsHTTPMethod(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		expected bool
	}{
		{"GET", "GET", true},
		{"POST", "POST", true},
		{"PUT", "PUT", true},
		{"DELETE", "DELETE", true},
		{"PATCH", "PATCH", true},
		{"HEAD", "HEAD", true},
		{"OPTIONS", "OPTIONS", true},
		{"INVALID", "INVALID", false},
		{"", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isHTTPMethod(tt.method)
			if result != tt.expected {
				t.Errorf("isHTTPMethod(%q) = %v, expected %v", tt.method, result, tt.expected)
			}
		})
	}
}

func TestExtractStringLiteral(t *testing.T) {
	// This function is tested indirectly through ExtractRoutes tests
	// as it requires AST nodes which are complex to create manually
}

func TestExtractHandlerName(t *testing.T) {
	// This function is tested indirectly through ExtractRoutes tests
}
