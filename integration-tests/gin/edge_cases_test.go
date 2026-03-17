//go:build e2e

package gin

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spencercjh/spec-forge/cmd"
	"github.com/spencercjh/spec-forge/integration-tests/helpers"
)

// TestWildcardRoutes tests wildcard route handling
func TestWildcardRoutes(t *testing.T) {
	projectPath := "./fixtures/gin-unsupported"

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("gin-unsupported fixture not found")
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

	specFile := helpers.FindSpecFile(t, outputDir, "json")
	validator := helpers.NewSpecValidator(t, specFile)

	validator.ValidatePath("/api/health")

	pathCount := validator.GetPathCount()
	if pathCount == 0 {
		t.Error("expected at least some paths to be generated")
	}

	t.Logf("Generated %d paths (wildcard routes handled)", pathCount)
}

// TestAnonymousHandlers tests anonymous handler handling
func TestAnonymousHandlers(t *testing.T) {
	projectPath := "./fixtures/gin-anonymous-handler"

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("gin-anonymous-handler fixture not found")
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

	specFile := helpers.FindSpecFile(t, outputDir, "json")
	validator := helpers.NewSpecValidator(t, specFile)

	pathCount := validator.GetPathCount()
	if pathCount == 0 {
		t.Error("expected at least some paths to be generated from anonymous handlers")
	}

	validator.LogSummary()
	t.Logf("Anonymous handlers handled: %d paths generated", pathCount)
}

// TestNestedModels tests extraction of complex nested types
func TestNestedModels(t *testing.T) {
	projectPath := "./fixtures/gin-nested-models"

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

	specFile := helpers.FindSpecFile(t, outputDir, "json")
	validator := helpers.NewSpecValidator(t, specFile)

	validator.FullValidation(helpers.ValidationConfig{
		ExpectedPaths: []string{
			"/companies",
			"/companies/{id}",
			"/companies/{id}/employees",
		},
		Operations: []helpers.OperationConfig{
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

	validator.ValidateSchemaProperty("Company", helpers.SchemaPropertyExpectation{
		Name: "address",
		Ref:  "Address",
	})

	validator.ValidateSchemaProperty("Company", helpers.SchemaPropertyExpectation{
		Name:     "employees",
		Type:     "array",
		ItemType: "Employee",
	})

	t.Log("Complex nested types handled successfully")
}

// TestMultifileCrossPackage tests cross-package type resolution
func TestMultifileCrossPackage(t *testing.T) {
	projectPath := "./fixtures/gin-multifile"

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

	specFile := helpers.FindSpecFile(t, outputDir, "json")
	validator := helpers.NewSpecValidator(t, specFile)

	validator.FullValidation(helpers.ValidationConfig{
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

// TestBasicMinimal tests minimal single-file extraction
func TestBasicMinimal(t *testing.T) {
	projectPath := "./fixtures/gin-basic"

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

	specFile := helpers.FindSpecFile(t, outputDir, "json")
	validator := helpers.NewSpecValidator(t, specFile)

	validator.FullValidation(helpers.ValidationConfig{
		ExpectedPaths: []string{
			"/items",
			"/items/{id}",
		},
		Operations: []helpers.OperationConfig{
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

// TestMalformedGracefulDegradation tests graceful handling of malformed files
func TestMalformedGracefulDegradation(t *testing.T) {
	tempDir := t.TempDir()

	goMod := `module malformed-test

go 1.24.0

require github.com/gin-gonic/gin v1.12.0
`
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

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

	malformedGo := `package main

import "github.com/gin-gonic/gin"

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

	err := rootCmd.Execute()
	if err != nil {
		errMsg := stderr.String()
		if errMsg == "" {
			t.Error("expected error message for malformed file, got empty stderr")
		} else {
			t.Logf("Got expected error for malformed file: %s", errMsg)
		}
		return
	}

	specFile := helpers.FindSpecFile(t, outputDir, "json")
	validator := helpers.NewSpecValidator(t, specFile)

	validator.ValidatePath("/valid")

	t.Log("Graceful degradation: malformed file handled, valid routes extracted")
}

// TestInvalidStructStillGenerates tests partial generation when struct can't be resolved
func TestInvalidStructStillGenerates(t *testing.T) {
	tempDir := t.TempDir()

	goMod := `module partial-test

go 1.24.0

require github.com/gin-gonic/gin v1.12.0
`
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

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
	var req UnknownRequest
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

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected graceful handling of unknown types, got error: %v\nstderr: %s", err, stderr.String())
	}

	specFile := helpers.FindSpecFile(t, outputDir, "json")
	validator := helpers.NewSpecValidator(t, specFile)

	validator.ValidateSchemas([]string{"KnownType"})
	validator.ValidatePaths([]string{"/known", "/unknown"})

	t.Log("Partial generation successful: valid paths generated despite unknown types")
}
