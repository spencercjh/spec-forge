//go:build e2e

package spring

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spencercjh/spec-forge/cmd"
)

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

	// If it somehow succeeds, verify that something was actually generated
	files, readErr := os.ReadDir(outputDir)
	if readErr != nil {
		t.Fatalf("failed to read output directory %q: %v", outputDir, readErr)
	}
	if len(files) == 0 {
		t.Fatalf("expected generated output in %q when Execute() returned nil, but directory is empty", outputDir)
	}

	t.Logf("Malformed pom.xml handled gracefully without error; generated %d item(s) in %q", len(files), outputDir)
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
	// Use --keep-patched to preserve the patched pom.xml for verification
	// Without this flag, generate restores the original file after execution
	rootCmd.SetArgs([]string{
		"generate",
		tempDir,
		"--output-dir", outputDir,
		"--output", "json",
		"--skip-enrich",
		"--skip-publish",
		"--keep-patched",
	})

	// This may fail (no Maven wrapper, no source code) but shouldn't panic
	err := rootCmd.Execute()
	if err != nil {
		t.Logf("Expected error when springdoc is not set up: %v", err)

		// Verify the patcher modified the pom.xml to add springdoc
		updatedPom, readErr := os.ReadFile(filepath.Join(tempDir, "pom.xml"))
		if readErr != nil {
			t.Fatalf("failed to read pom.xml after generate attempt: %v", readErr)
		}

		// Assert that springdoc dependency was added by the patcher
		if !bytes.Contains(updatedPom, []byte("springdoc-openapi")) {
			t.Errorf("expected patcher to add springdoc-openapi dependency to pom.xml, but it was not found")
		} else {
			t.Log("Patcher successfully added springdoc dependency before build failure")
		}
		return
	}

	// If it succeeds, verify output was generated
	files, readErr := os.ReadDir(outputDir)
	if readErr != nil {
		t.Fatalf("failed to read output directory %q: %v", outputDir, readErr)
	}
	if len(files) == 0 {
		t.Fatalf("expected generated output in %q when Execute() returned nil, but directory is empty", outputDir)
	}

	t.Logf("Missing springdoc dependency handled - patcher added it; generated %d item(s)", len(files))
}
