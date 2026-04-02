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

func TestASTParser_ParseFiles_RelativeDotPath(t *testing.T) {
	dir := t.TempDir()

	code := `package main

import "github.com/gin-gonic/gin"

func main() {
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {})
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(code), 0o644); err != nil {
		t.Fatalf("failed to create main.go: %v", err)
	}

	// Use "." relative path — this is how users invoke: spec-forge generate .
	// Regression test: info.Name() returns "." for the root, which must not be
	// treated as a hidden directory and skip the entire tree.
	parser := NewASTParser(".")

	// Change working directory to temp dir so "." resolves correctly
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	}()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	if err := parser.ParseFiles(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(parser.files) == 0 {
		t.Fatal("expected at least 1 parsed file with relative \".\" path, got 0")
	}

	routes, err := parser.ExtractRoutes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(routes) == 0 {
		t.Fatal("expected at least 1 route with relative \".\" path, got 0")
	}
}

func TestASTParser_ParseFiles_HiddenDirsSkipped(t *testing.T) {
	dir := t.TempDir()

	// Normal Go file should be parsed
	os.WriteFile(filepath.Join(dir, "main.go"), []byte(`package main; func main(){}`), 0o644)

	// Go file inside hidden directory should be skipped
	os.MkdirAll(filepath.Join(dir, ".hidden"), 0o755)
	os.WriteFile(filepath.Join(dir, ".hidden", "secret.go"), []byte(`package main; func secret(){}`), 0o644)

	parser := NewASTParser(dir)
	if err := parser.ParseFiles(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(parser.files) != 1 {
		t.Errorf("expected 1 file (hidden dir skipped), got %d", len(parser.files))
	}
	for path := range parser.files {
		if filepath.Base(filepath.Dir(path)) == ".hidden" {
			t.Errorf("file from hidden directory should not be parsed: %s", path)
		}
	}
}

func TestExtractStringLiteral(t *testing.T) {
	// This function is tested indirectly through ExtractRoutes tests
	// as it requires AST nodes which are complex to create manually
}

func TestExtractHandlerName(t *testing.T) {
	// This function is tested indirectly through ExtractRoutes tests
}

func TestShouldExcludeRoute(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"/swagger/{any}", true},
		{"/swagger/index.html", true},
		{"/docs", true},
		{"/docs/openapi.yaml", true},
		{"/debug/pprof", true},
		{"/static/css/main.css", true},
		{"/public/image.png", true},
		{"/favicon.ico", true},
		{"/api/v1/users", false},
		{"/api/v1/projects/{name}", false},
		{"/ping", false},
		{"/health", false},
		{"/srv/ping", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := shouldExcludeRoute(tt.path)
			if result != tt.expected {
				t.Errorf("shouldExcludeRoute(%q) = %v, expected %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestFilterRoutes(t *testing.T) {
	routes := []Route{
		{Method: "GET", FullPath: "/api/v1/users", HandlerName: "ListUsers"},
		{Method: "GET", FullPath: "/swagger/{any}", HandlerName: "SwaggerUI"},
		{Method: "POST", FullPath: "/api/v1/projects", HandlerName: "CreateProject"},
		{Method: "GET", FullPath: "/docs/openapi.yaml", HandlerName: "Docs"},
		{Method: "GET", FullPath: "/ping", HandlerName: "Ping"},
		{Method: "GET", FullPath: "/debug/pprof", HandlerName: "Pprof"},
	}

	filtered := filterRoutes(routes)

	if len(filtered) != 3 {
		t.Errorf("expected 3 routes after filtering, got %d: %v", len(filtered), filtered)
	}

	expectedPaths := map[string]bool{
		"/api/v1/users":    false,
		"/api/v1/projects": false,
		"/ping":            false,
	}
	for _, route := range filtered {
		if _, ok := expectedPaths[route.FullPath]; !ok {
			t.Errorf("unexpected route in filtered results: %s", route.FullPath)
		}
		expectedPaths[route.FullPath] = true
	}
	for path, found := range expectedPaths {
		if !found {
			t.Errorf("expected route %s not found in filtered results", path)
		}
	}
}
