//go:build e2e

package spring

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spencercjh/spec-forge/cmd"
	"github.com/spencercjh/spec-forge/integration-tests/helpers"
)

// TestMultiModuleMaven tests multi-module Maven project generation
func TestMultiModuleMaven(t *testing.T) {
	projectPath := "../maven-multi-module-demo"

	skipIfProjectMissing(t, projectPath)

	outputDir := t.TempDir()
	specFile := generateSpec(t, projectPath, outputDir)
	validator := helpers.NewSpecValidator(t, specFile)

	validator.ValidateOpenAPIVersion()
	validator.ValidateInfo()

	pathCount := validator.GetPathCount()
	if pathCount == 0 {
		t.Error("expected at least some paths to be generated from multi-module project")
	}

	t.Logf("Multi-module Maven project: generated %d paths", pathCount)
	validator.LogSummary()
}

// TestMultiModuleGradle tests multi-module Gradle project generation
func TestMultiModuleGradle(t *testing.T) {
	projectPath := "../gradle-multi-module-demo"

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("Gradle multi-module demo project not found")
	}

	gradlewPath := filepath.Join(projectPath, "gradlew")
	if _, err := os.Stat(gradlewPath); os.IsNotExist(err) {
		t.Skip("Gradle wrapper not found, skipping test")
	}

	outputDir := t.TempDir()
	specFile := generateSpec(t, projectPath, outputDir)
	validator := helpers.NewSpecValidator(t, specFile)

	validator.ValidateOpenAPIVersion()
	validator.ValidateInfo()

	pathCount := validator.GetPathCount()
	if pathCount == 0 {
		t.Error("expected at least some paths to be generated from multi-module project")
	}

	t.Logf("Multi-module Gradle project: generated %d paths", pathCount)
	validator.LogSummary()
}

// TestMalformedPomGracefulDegradation tests graceful handling of malformed pom.xml
func TestMalformedPomGracefulDegradation(t *testing.T) {
	tempDir := t.TempDir()

	// Create a malformed pom.xml that is not valid XML
	malformedPom := `<?xml version="1.0" encoding="UTF-8"?>
<project>
    <modelVersion>4.0.0</modelVersion>
    <groupId>com.example</groupId>
    <artifactId>malformed-demo</artifactId>
    <!-- Missing closing tags and required elements -->
    <dependencies>
        <dependency>
            <groupId>org.springframework.boot</groupId>
`
	if err := os.WriteFile(filepath.Join(tempDir, "pom.xml"), []byte(malformedPom), 0o644); err != nil {
		t.Fatalf("failed to write malformed pom.xml: %v", err)
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
		// Expected: malformed pom.xml should cause a graceful error
		t.Logf("Got expected error for malformed pom.xml: %v", err)
		return
	}

	// If it somehow succeeds, that's also acceptable (graceful degradation)
	t.Log("Malformed pom.xml handled gracefully without error")
}

// TestMissingSpringdocDependency tests behavior when springdoc dependency is not present
func TestMissingSpringdocDependency(t *testing.T) {
	tempDir := t.TempDir()

	// Create a minimal pom.xml without springdoc dependency
	// The patcher should add springdoc dependency automatically
	pomXML := `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 https://maven.apache.org/xsd/maven-4.0.0.xsd">
    <modelVersion>4.0.0</modelVersion>
    <parent>
        <groupId>org.springframework.boot</groupId>
        <artifactId>spring-boot-starter-parent</artifactId>
        <version>3.4.0</version>
        <relativePath/>
    </parent>
    <groupId>com.example</groupId>
    <artifactId>no-springdoc-demo</artifactId>
    <version>0.0.1-SNAPSHOT</version>
    <properties>
        <java.version>17</java.version>
    </properties>
    <dependencies>
        <dependency>
            <groupId>org.springframework.boot</groupId>
            <artifactId>spring-boot-starter-web</artifactId>
        </dependency>
    </dependencies>
</project>
`
	if err := os.WriteFile(filepath.Join(tempDir, "pom.xml"), []byte(pomXML), 0o644); err != nil {
		t.Fatalf("failed to write pom.xml: %v", err)
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

	// This may fail (no Maven wrapper, no source code) but shouldn't panic
	err := rootCmd.Execute()
	if err != nil {
		t.Logf("Expected error when springdoc is not set up: %v", err)
		return
	}

	t.Log("Missing springdoc dependency handled - patcher may have added it")
}

// TestGradleSpringBoot tests Gradle-based Spring Boot project generation
func TestGradleSpringBoot(t *testing.T) {
	projectPath := "../gradle-springboot-openapi-demo"

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("Gradle Spring Boot demo project not found")
	}

	gradlewPath := filepath.Join(projectPath, "gradlew")
	if _, err := os.Stat(gradlewPath); os.IsNotExist(err) {
		t.Skip("Gradle wrapper not found, skipping test")
	}

	outputDir := t.TempDir()
	specFile := generateSpec(t, projectPath, outputDir)
	validator := helpers.NewSpecValidator(t, specFile)

	validator.FullValidation(helpers.ValidationConfig{
		ExpectedPaths: []string{
			"/api/v1/users",
			"/api/v1/users/{id}",
			"/api/v1/users/{id}/profile",
			"/api/v1/users/upload",
		},
		Operations: []helpers.OperationConfig{
			{
				Path:            "/api/v1/users",
				Method:          "get",
				WantOperationID: true,
			},
			{
				Path:            "/api/v1/users",
				Method:          "post",
				WantOperationID: true,
			},
			{
				Path:            "/api/v1/users/{id}",
				Method:          "get",
				WantOperationID: true,
				ExpectedParams:  []string{"id"},
			},
		},
		ExpectedSchemas: []string{
			"User",
			"FileUploadResult",
		},
	})

	t.Log("Gradle Spring Boot project generation successful")
}
