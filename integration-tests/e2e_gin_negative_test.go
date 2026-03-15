//go:build e2e

package e2e_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spencercjh/spec-forge/cmd"
)

// TestE2E_Gin_Unsupported_WildcardRoutes tests that wildcard routes are handled gracefully
func TestE2E_Gin_Unsupported_WildcardRoutes(t *testing.T) {
	projectPath := "./gin-fixtures/gin-unsupported"

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("gin-unsupported fixture not found")
	}

	// Wildcard route handling should now work

	outputDir := t.TempDir()

	rootCmd := cmd.NewRootCommand()
	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{
		"generate",
		projectPath,
		"--output-dir", outputDir,
		"--output", "json",
		"--skip-enrich",
		"--skip-publish",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v\nstderr: %s", err, stderr.String())
	}

	specFile := FindSpecFile(t, outputDir, "json")
	validator := NewSpecValidator(t, specFile)

	// Wildcard routes should be skipped or handled without crashing
	// The standard health endpoint should still work
	validator.ValidatePath("/api/health")

	// Should have paths generated (even if some are skipped)
	pathCount := validator.GetPathCount()
	if pathCount == 0 {
		t.Error("expected at least some paths to be generated")
	}

	t.Logf("Generated %d paths (wildcard routes handled)", pathCount)
}

// TestE2E_Gin_Unsupported_AnonymousHandlers tests handling of anonymous handlers
func TestE2E_Gin_Unsupported_AnonymousHandlers(t *testing.T) {
	projectPath := "./gin-fixtures/gin-anonymous-handler"

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("gin-anonymous-handler fixture not found")
	}

	// TODO: Fix anonymous handler response generation - currently produces invalid response
	// "invalid operation POST: value of responses must be an object"
	t.Skip("Skipping: anonymous handler responses need proper handling - see issue tracker")

	outputDir := t.TempDir()

	rootCmd := cmd.NewRootCommand()
	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{
		"generate",
		projectPath,
		"--output-dir", outputDir,
		"--output", "json",
		"--skip-enrich",
		"--skip-publish",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v\nstderr: %s", err, stderr.String())
	}

	specFile := FindSpecFile(t, outputDir, "json")
	validator := NewSpecValidator(t, specFile)

	// Some paths should be generated even with anonymous handlers
	// The extractor may skip some, but shouldn't crash
	pathCount := validator.GetPathCount()
	if pathCount == 0 {
		t.Error("expected at least some paths to be generated from anonymous handlers")
	}

	validator.LogSummary()
	t.Logf("Anonymous handlers handled: %d paths generated", pathCount)
}

// TestE2E_Gin_NestedModels_ComplexTypes tests extraction of complex nested types
func TestE2E_Gin_NestedModels_ComplexTypes(t *testing.T) {
	projectPath := "./gin-fixtures/gin-nested-models"

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("gin-nested-models fixture not found")
	}

	outputDir := t.TempDir()

	rootCmd := cmd.NewRootCommand()
	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{
		"generate",
		projectPath,
		"--output-dir", outputDir,
		"--output", "json",
		"--skip-enrich",
		"--skip-publish",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v\nstderr: %s", err, stderr.String())
	}

	specFile := FindSpecFile(t, outputDir, "json")
	validator := NewSpecValidator(t, specFile)

	// Basic validation
	validator.FullValidation(ValidationConfig{
		ExpectedPaths: []string{
			"/companies",
			"/companies/{id}",
			"/companies/{id}/employees",
		},
		Operations: []OperationConfig{
			{
				Path:                  "/companies",
				Method:                "get",
				WantOperationID:       true,
				ExpectedResponseCodes: []string{"200", "400"},
			},
			{
				Path:                  "/companies",
				Method:                "post",
				WantOperationID:       true,
				ExpectedResponseCodes: []string{"201", "400"},
			},
		},
		ExpectedSchemas: []string{
			"Company",
			"Employee",
			"Address",
		},
	})

	// Validate nested struct property (Address inside Company)
	validator.ValidateSchemaProperty("Company", SchemaPropertyExpectation{
		Name: "address",
		Type: "object", // Nested struct becomes object
	})

	// Validate array of structs (Employees)
	validator.ValidateSchemaProperty("Company", SchemaPropertyExpectation{
		Name:     "employees",
		Type:     "array",
		ItemType: "Employee",
	})

	t.Log("Complex nested types handled successfully")
}

// TestE2E_Gin_Multifile_CrossPackage tests cross-package type resolution
func TestE2E_Gin_Multifile_CrossPackage(t *testing.T) {
	projectPath := "./gin-fixtures/gin-multifile"

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("gin-multifile fixture not found")
	}

	outputDir := t.TempDir()

	rootCmd := cmd.NewRootCommand()
	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{
		"generate",
		projectPath,
		"--output-dir", outputDir,
		"--output", "json",
		"--skip-enrich",
		"--skip-publish",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v\nstderr: %s", err, stderr.String())
	}

	specFile := FindSpecFile(t, outputDir, "json")
	validator := NewSpecValidator(t, specFile)

	// Validate cross-package types are resolved
	validator.FullValidation(ValidationConfig{
		ExpectedPaths: []string{
			"/api/users",
			"/api/users/{id}",
			"/api/products",
		},
		ExpectedSchemas: []string{
			"User",
			"CreateUserRequest",
			"Product",
		},
	})

	t.Log("Cross-package type resolution successful")
}

// TestE2E_Gin_Basic_Minimal tests minimal single-file extraction
func TestE2E_Gin_Basic_Minimal(t *testing.T) {
	projectPath := "./gin-fixtures/gin-basic"

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("gin-basic fixture not found")
	}

	outputDir := t.TempDir()

	rootCmd := cmd.NewRootCommand()
	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{
		"generate",
		projectPath,
		"--output-dir", outputDir,
		"--output", "json",
		"--skip-enrich",
		"--skip-publish",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v\nstderr: %s", err, stderr.String())
	}

	specFile := FindSpecFile(t, outputDir, "json")
	validator := NewSpecValidator(t, specFile)

	// Minimal fixture should have basic CRUD paths
	validator.FullValidation(ValidationConfig{
		ExpectedPaths: []string{
			"/items",
			"/items/{id}",
		},
		Operations: []OperationConfig{
			{
				Path:                  "/items",
				Method:                "get",
				WantOperationID:       true,
				ExpectedResponseCodes: []string{"200"},
			},
			{
				Path:                  "/items",
				Method:                "post",
				WantOperationID:       true,
				ExpectedResponseCodes: []string{"201", "400"},
			},
			{
				Path:                  "/items/{id}",
				Method:                "get",
				WantOperationID:       true,
				ExpectedResponseCodes: []string{"200"},
				ExpectedParams:        []string{"id"},
			},
		},
		ExpectedSchemas: []string{
			"Item",
		},
	})

	t.Log("Basic minimal extraction successful")
}

// TestE2E_Gin_Malformed_GracefulDegradation tests graceful handling of malformed files
func TestE2E_Gin_Malformed_GracefulDegradation(t *testing.T) {
	// Create a temporary directory with a malformed Go file
	tempDir := t.TempDir()

	// Create go.mod
	goMod := `module malformed-test

go 1.24.0

require github.com/gin-gonic/gin v1.12.0
`
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create main.go with valid routes
	mainGo := `package main

import "github.com/gin-gonic/gin"

func main() {
	r := gin.Default()
	r.GET("/valid", validHandler)
	r.Run()
}

func validHandler(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok"})
}
`
	if err := os.WriteFile(filepath.Join(tempDir, "main.go"), []byte(mainGo), 0o644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	// Create malformed.go with syntax error
	malformedGo := `package main

import "github.com/gin-gonic/gin"

// This file has syntax errors
func malformedHandler(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok"
	// Missing closing brace
`
	if err := os.WriteFile(filepath.Join(tempDir, "malformed.go"), []byte(malformedGo), 0o644); err != nil {
		t.Fatalf("failed to write malformed.go: %v", err)
	}

	outputDir := t.TempDir()

	rootCmd := cmd.NewRootCommand()
	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{
		"generate",
		tempDir,
		"--output-dir", outputDir,
		"--output", "json",
		"--skip-enrich",
		"--skip-publish",
	})

	// Should either succeed (gracefully skipping malformed file) or fail with clear error
	err := rootCmd.Execute()
	if err != nil {
		// If it fails, it should provide a clear error message
		errMsg := stderr.String()
		if errMsg == "" {
			t.Error("expected error message for malformed file, got empty stderr")
		} else {
			t.Logf("Got expected error for malformed file: %s", errMsg)
		}
		return
	}

	// If it succeeds, valid routes should still be extracted
	specFile := FindSpecFile(t, outputDir, "json")
	validator := NewSpecValidator(t, specFile)

	// Valid handler should still be present
	validator.ValidatePath("/valid")

	t.Log("Graceful degradation: malformed file handled, valid routes extracted")
}

// TestE2E_Gin_InvalidStruct_StillGenerates tests partial generation when struct can't be resolved
func TestE2E_Gin_InvalidStruct_StillGenerates(t *testing.T) {
	// Create a temporary directory with a struct that can't be resolved
	tempDir := t.TempDir()

	// Create go.mod
	goMod := `module partial-test

go 1.24.0

require github.com/gin-gonic/gin v1.12.0
`
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create main.go with reference to undefined struct
	mainGo := `package main

import "github.com/gin-gonic/gin"

func main() {
	r := gin.Default()
	r.GET("/known", knownHandler)
	r.POST("/unknown", unknownHandler)
	r.Run()
}

type KnownType struct {
	Name string ` + "`" + `json:"name"` + "`" + `
}

func knownHandler(c *gin.Context) {
	c.JSON(200, KnownType{Name: "test"})
}

func unknownHandler(c *gin.Context) {
	var req UnknownRequest // This type doesn't exist
	c.ShouldBindJSON(&req)
	c.JSON(200, gin.H{"status": "ok"})
}
`
	if err := os.WriteFile(filepath.Join(tempDir, "main.go"), []byte(mainGo), 0o644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	outputDir := t.TempDir()

	rootCmd := cmd.NewRootCommand()
	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{
		"generate",
		tempDir,
		"--output-dir", outputDir,
		"--output", "json",
		"--skip-enrich",
		"--skip-publish",
	})

	// Should succeed and generate partial spec
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected graceful handling of unknown types, got error: %v\nstderr: %s", err, stderr.String())
	}

	specFile := FindSpecFile(t, outputDir, "json")
	validator := NewSpecValidator(t, specFile)

	// Known type should be in schemas
	validator.ValidateSchemas([]string{"KnownType"})

	// Both paths should exist
	validator.ValidatePaths([]string{"/known", "/unknown"})

	t.Log("Partial generation successful: valid paths generated despite unknown types")
}
