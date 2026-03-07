package gin

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/spencercjh/spec-forge/internal/extractor"
)

func TestNewGenerator(t *testing.T) {
	g := NewGenerator()
	if g == nil {
		t.Error("expected non-nil generator")
	}
}

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
	_ = id
	c.JSON(200, User{ID: 1, Name: "test"})
}
`
	os.WriteFile(filepath.Join(dir, "main.go"), []byte(mainGo), 0644)

	g := NewGenerator()
	ctx := context.Background()
	info := &extractor.ProjectInfo{
		Framework:  "gin",
		FrameworkData: &Info{
			ModuleName: "test",
			GinVersion: "v1.9.1",
		},
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
	if _, err := os.Stat(result.SpecFilePath); err != nil {
		t.Errorf("output file not created: %s", result.SpecFilePath)
	}

	// Load and validate the spec
	loader := openapi3.NewLoader()
	spec, err := loader.LoadFromFile(result.SpecFilePath)
	if err != nil {
		t.Fatalf("failed to load spec: %v", err)
	}

	// Check paths
	if spec.Paths.Len() == 0 {
		t.Error("expected at least one path")
	}

	// Check if User schema exists
	if spec.Components == nil || spec.Components.Schemas == nil {
		t.Error("expected Components.Schemas to be defined")
	} else if spec.Components.Schemas["User"] == nil {
		t.Error("expected User schema to be defined")
	}
}

func TestGenerator_Generate_JSON(t *testing.T) {
	dir := t.TempDir()

	goMod := `module test

go 1.21

require github.com/gin-gonic/gin v1.9.1
`
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0644)

	mainGo := `package main

import "github.com/gin-gonic/gin"

func main() {
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})
}
`
	os.WriteFile(filepath.Join(dir, "main.go"), []byte(mainGo), 0644)

	g := NewGenerator()
	ctx := context.Background()
	info := &extractor.ProjectInfo{
		Framework:  "gin",
		FrameworkData: &Info{
			ModuleName: "test",
		},
	}
	opts := &extractor.GenerateOptions{
		OutputDir:  dir,
		OutputFile: "openapi",
		Format:     "json",
	}

	result, err := g.Generate(ctx, dir, info, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !contains(result.SpecFilePath, ".json") {
		t.Errorf("expected json file extension, got %s", result.SpecFilePath)
	}

	// Load and validate
	loader := openapi3.NewLoader()
	_, err = loader.LoadFromFile(result.SpecFilePath)
	if err != nil {
		t.Fatalf("failed to load spec: %v", err)
	}
}

func TestSetOperationForMethod(t *testing.T) {
	tests := []struct {
		method string
	}{
		{"GET"},
		{"POST"},
		{"PUT"},
		{"DELETE"},
		{"PATCH"},
		{"HEAD"},
		{"OPTIONS"},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			pathItem := &openapi3.PathItem{}
			operation := &openapi3.Operation{Summary: "test"}

			setOperationForMethod(pathItem, tt.method, operation)

			var result *openapi3.Operation
			switch tt.method {
			case "GET":
				result = pathItem.Get
			case "POST":
				result = pathItem.Post
			case "PUT":
				result = pathItem.Put
			case "DELETE":
				result = pathItem.Delete
			case "PATCH":
				result = pathItem.Patch
			case "HEAD":
				result = pathItem.Head
			case "OPTIONS":
				result = pathItem.Options
			}

			if result == nil {
				t.Errorf("expected %s operation to be set", tt.method)
			}
			if result != nil && result.Summary != "test" {
				t.Errorf("expected operation summary 'test', got %s", result.Summary)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr || len(s) > len(substr) && contains(s[:len(s)-1], substr)
}
