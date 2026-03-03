// Package spring_test tests the Spring extractor implementation.
package spring_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spencercjh/spec-forge/internal/extractor"
	"github.com/spencercjh/spec-forge/internal/extractor/spring"
)

// minimalPomWithoutSpringdoc is a minimal pom.xml without springdoc dependencies
const minimalPomWithoutSpringdoc = `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 https://maven.apache.org/xsd/maven-4.0.0.xsd">
    <modelVersion>4.0.0</modelVersion>
    <parent>
        <groupId>org.springframework.boot</groupId>
        <artifactId>spring-boot-starter-parent</artifactId>
        <version>3.2.0</version>
        <relativePath/>
    </parent>
    <groupId>com.example</groupId>
    <artifactId>test-project</artifactId>
    <version>1.0.0</version>
    <dependencies>
        <dependency>
            <groupId>org.springframework.boot</groupId>
            <artifactId>spring-boot-starter-web</artifactId>
        </dependency>
    </dependencies>
    <build>
        <plugins>
            <plugin>
                <groupId>org.springframework.boot</groupId>
                <artifactId>spring-boot-maven-plugin</artifactId>
            </plugin>
        </plugins>
    </build>
</project>
`

// minimalGradleWithoutSpringdoc is a minimal build.gradle without springdoc dependencies
const minimalGradleWithoutSpringdoc = `
plugins {
    id 'java'
    id 'org.springframework.boot' version '3.2.0'
    id 'io.spring.dependency-management' version '1.1.4'
}

group = 'com.example'
version = '1.0.0'

repositories {
    mavenCentral()
}

dependencies {
    implementation 'org.springframework.boot:spring-boot-starter-web'
}
`

func TestPatcher_Patch_MavenDryRun(t *testing.T) {
	// Read original pom.xml content
	origContent, err := os.ReadFile("../../../integration-tests/maven-springboot-openapi-demo/pom.xml")
	if err != nil {
		t.Skip("Integration test project not found")
	}

	// Create a temp copy for testing
	tmpDir := t.TempDir()
	tmpPom := filepath.Join(tmpDir, "pom.xml")
	if err := os.WriteFile(tmpPom, origContent, 0644); err != nil {
		t.Fatalf("Failed to create temp pom.xml: %v", err)
	}

	patcher := spring.NewPatcher()
	opts := &extractor.PatchOptions{
		DryRun:             true,
		SpringdocVersion:   extractor.DefaultSpringdocVersion,
		MavenPluginVersion: extractor.DefaultSpringdocMavenPlugin,
	}

	changes, err := patcher.Patch(tmpDir, opts)
	if err != nil {
		t.Fatalf("Patch failed: %v", err)
	}

	// In dry-run mode, changes should be reported
	t.Logf("Changes: %+v", changes)
}

func TestPatcher_Patch_MavenWithMissingDeps(t *testing.T) {
	// Create a temp project without springdoc
	tmpDir := t.TempDir()
	tmpPom := filepath.Join(tmpDir, "pom.xml")
	if err := os.WriteFile(tmpPom, []byte(minimalPomWithoutSpringdoc), 0644); err != nil {
		t.Fatalf("Failed to create temp pom.xml: %v", err)
	}

	patcher := spring.NewPatcher()

	t.Run("dry-run reports changes but does not modify file", func(t *testing.T) {
		opts := &extractor.PatchOptions{
			DryRun:             true,
			SpringdocVersion:   extractor.DefaultSpringdocVersion,
			MavenPluginVersion: extractor.DefaultSpringdocMavenPlugin,
		}

		changes, err := patcher.Patch(tmpDir, opts)
		if err != nil {
			t.Fatalf("Patch failed: %v", err)
		}

		if !changes.DependencyAdded {
			t.Error("Expected DependencyAdded to be true")
		}
		if !changes.PluginAdded {
			t.Error("Expected PluginAdded to be true")
		}

		// Verify file was not modified in dry-run mode
		content, err := os.ReadFile(tmpPom)
		if err != nil {
			t.Fatalf("Failed to read pom.xml: %v", err)
		}
		if strings.Contains(string(content), "springdoc-openapi") {
			t.Error("File should not be modified in dry-run mode")
		}
	})

	t.Run("actual patch modifies file", func(t *testing.T) {
		// Reset the pom file
		if err := os.WriteFile(tmpPom, []byte(minimalPomWithoutSpringdoc), 0644); err != nil {
			t.Fatalf("Failed to reset temp pom.xml: %v", err)
		}

		opts := &extractor.PatchOptions{
			DryRun:             false,
			SpringdocVersion:   extractor.DefaultSpringdocVersion,
			MavenPluginVersion: extractor.DefaultSpringdocMavenPlugin,
		}

		changes, err := patcher.Patch(tmpDir, opts)
		if err != nil {
			t.Fatalf("Patch failed: %v", err)
		}

		if !changes.DependencyAdded {
			t.Error("Expected DependencyAdded to be true")
		}
		if !changes.PluginAdded {
			t.Error("Expected PluginAdded to be true")
		}

		// Verify file was modified
		content, err := os.ReadFile(tmpPom)
		if err != nil {
			t.Fatalf("Failed to read pom.xml: %v", err)
		}
		if !strings.Contains(string(content), "springdoc-openapi-starter-webmvc-ui") {
			t.Error("File should contain springdoc dependency")
		}
		if !strings.Contains(string(content), "springdoc-openapi-maven-plugin") {
			t.Error("File should contain springdoc plugin")
		}
	})
}

func TestPatcher_Patch_GradleWithMissingDeps(t *testing.T) {
	// Create a temp project without springdoc
	tmpDir := t.TempDir()
	tmpGradle := filepath.Join(tmpDir, "build.gradle")
	if err := os.WriteFile(tmpGradle, []byte(minimalGradleWithoutSpringdoc), 0644); err != nil {
		t.Fatalf("Failed to create temp build.gradle: %v", err)
	}

	patcher := spring.NewPatcher()

	t.Run("dry-run reports changes but does not modify file", func(t *testing.T) {
		opts := &extractor.PatchOptions{
			DryRun:              true,
			SpringdocVersion:    extractor.DefaultSpringdocVersion,
			GradlePluginVersion: extractor.DefaultSpringdocGradlePlugin,
		}

		changes, err := patcher.Patch(tmpDir, opts)
		if err != nil {
			t.Fatalf("Patch failed: %v", err)
		}

		if !changes.DependencyAdded {
			t.Error("Expected DependencyAdded to be true")
		}
		if !changes.PluginAdded {
			t.Error("Expected PluginAdded to be true")
		}

		// Verify file was not modified in dry-run mode
		content, err := os.ReadFile(tmpGradle)
		if err != nil {
			t.Fatalf("Failed to read build.gradle: %v", err)
		}
		if strings.Contains(string(content), "springdoc-openapi") {
			t.Error("File should not be modified in dry-run mode")
		}
	})

	t.Run("actual patch modifies file", func(t *testing.T) {
		// Reset the gradle file
		if err := os.WriteFile(tmpGradle, []byte(minimalGradleWithoutSpringdoc), 0644); err != nil {
			t.Fatalf("Failed to reset temp build.gradle: %v", err)
		}

		opts := &extractor.PatchOptions{
			DryRun:              false,
			SpringdocVersion:    extractor.DefaultSpringdocVersion,
			GradlePluginVersion: extractor.DefaultSpringdocGradlePlugin,
		}

		changes, err := patcher.Patch(tmpDir, opts)
		if err != nil {
			t.Fatalf("Patch failed: %v", err)
		}

		if !changes.DependencyAdded {
			t.Error("Expected DependencyAdded to be true")
		}
		if !changes.PluginAdded {
			t.Error("Expected PluginAdded to be true")
		}

		// Verify file was modified
		content, err := os.ReadFile(tmpGradle)
		if err != nil {
			t.Fatalf("Failed to read build.gradle: %v", err)
		}
		if !strings.Contains(string(content), "springdoc-openapi-starter-webmvc-ui") {
			t.Error("File should contain springdoc dependency")
		}
		if !strings.Contains(string(content), "org.springdoc.openapi-gradle-plugin") {
			t.Error("File should contain springdoc plugin")
		}
	})
}

func TestPatcher_NeedsPatch(t *testing.T) {
	patcher := spring.NewPatcher()

	t.Run("needs patch when missing deps", func(t *testing.T) {
		info := &extractor.ProjectInfo{
			HasSpringdocDeps:   false,
			HasSpringdocPlugin: false,
		}
		if !patcher.NeedsPatch(info, false) {
			t.Error("Should need patch when missing deps")
		}
	})

	t.Run("needs patch when force is true", func(t *testing.T) {
		info := &extractor.ProjectInfo{
			HasSpringdocDeps:   true,
			HasSpringdocPlugin: true,
		}
		if !patcher.NeedsPatch(info, true) {
			t.Error("Should need patch when force is true")
		}
	})

	t.Run("no patch needed when already configured", func(t *testing.T) {
		info := &extractor.ProjectInfo{
			HasSpringdocDeps:   true,
			HasSpringdocPlugin: true,
		}
		if patcher.NeedsPatch(info, false) {
			t.Error("Should not need patch when already configured")
		}
	})
}

// minimalPomWithSpringdoc is a minimal pom.xml already containing springdoc dependencies
const minimalPomWithSpringdoc = `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 https://maven.apache.org/xsd/maven-4.0.0.xsd">
    <modelVersion>4.0.0</modelVersion>
    <parent>
        <groupId>org.springframework.boot</groupId>
        <artifactId>spring-boot-starter-parent</artifactId>
        <version>3.2.0</version>
        <relativePath/>
    </parent>
    <groupId>com.example</groupId>
    <artifactId>test-project</artifactId>
    <version>1.0.0</version>
    <dependencies>
        <dependency>
            <groupId>org.springframework.boot</groupId>
            <artifactId>spring-boot-starter-web</artifactId>
        </dependency>
        <dependency>
            <groupId>org.springdoc</groupId>
            <artifactId>springdoc-openapi-starter-webmvc-ui</artifactId>
            <version>2.3.0</version>
        </dependency>
    </dependencies>
    <build>
        <plugins>
            <plugin>
                <groupId>org.springframework.boot</groupId>
                <artifactId>spring-boot-maven-plugin</artifactId>
            </plugin>
            <plugin>
                <groupId>org.springdoc</groupId>
                <artifactId>springdoc-openapi-maven-plugin</artifactId>
                <version>1.4</version>
            </plugin>
        </plugins>
    </build>
</project>
`

// TestPatcher_Patch_AlreadyPatchedMaven verifies that patching an already-patched Maven project
// returns no changes and does not modify the file.
func TestPatcher_Patch_AlreadyPatchedMaven(t *testing.T) {
	// Create a temp project with springdoc already configured
	tmpDir := t.TempDir()
	tmpPom := filepath.Join(tmpDir, "pom.xml")
	if err := os.WriteFile(tmpPom, []byte(minimalPomWithSpringdoc), 0644); err != nil {
		t.Fatalf("Failed to create temp pom.xml: %v", err)
	}

	patcher := spring.NewPatcher()
	opts := &extractor.PatchOptions{
		DryRun:             false,
		SpringdocVersion:   extractor.DefaultSpringdocVersion,
		MavenPluginVersion: extractor.DefaultSpringdocMavenPlugin,
	}

	changes, err := patcher.Patch(tmpDir, opts)
	if err != nil {
		t.Fatalf("Patch failed: %v", err)
	}

	// Verify no changes were reported
	if changes.DependencyAdded {
		t.Error("Expected DependencyAdded to be false for already-patched project")
	}
	if changes.PluginAdded {
		t.Error("Expected PluginAdded to be false for already-patched project")
	}

	// Verify file was not modified
	content, err := os.ReadFile(tmpPom)
	if err != nil {
		t.Fatalf("Failed to read pom.xml: %v", err)
	}
	// The file should still contain exactly one springdoc dependency
	springdocCount := strings.Count(string(content), "springdoc-openapi-starter-webmvc-ui")
	if springdocCount != 1 {
		t.Errorf("Expected exactly 1 springdoc dependency, found %d", springdocCount)
	}
}

// minimalGradleWithSpringdoc is a minimal build.gradle already containing springdoc dependencies
const minimalGradleWithSpringdoc = `
plugins {
    id 'java'
    id 'org.springframework.boot' version '3.2.0'
    id 'io.spring.dependency-management' version '1.1.4'
    id 'org.springdoc.openapi-gradle-plugin' version "1.8.0"
}

group = 'com.example'
version = '1.0.0'

repositories {
    mavenCentral()
}

dependencies {
    implementation 'org.springframework.boot:spring-boot-starter-web'
    implementation 'org.springdoc:springdoc-openapi-starter-webmvc-ui:2.3.0'
}
`

// TestPatcher_Patch_AlreadyPatchedGradle verifies that patching an already-patched Gradle project
// returns no changes and does not modify the file.
func TestPatcher_Patch_AlreadyPatchedGradle(t *testing.T) {
	// Create a temp project with springdoc already configured
	tmpDir := t.TempDir()
	tmpGradle := filepath.Join(tmpDir, "build.gradle")
	if err := os.WriteFile(tmpGradle, []byte(minimalGradleWithSpringdoc), 0644); err != nil {
		t.Fatalf("Failed to create temp build.gradle: %v", err)
	}

	patcher := spring.NewPatcher()
	opts := &extractor.PatchOptions{
		DryRun:              false,
		SpringdocVersion:    extractor.DefaultSpringdocVersion,
		GradlePluginVersion: extractor.DefaultSpringdocGradlePlugin,
	}

	changes, err := patcher.Patch(tmpDir, opts)
	if err != nil {
		t.Fatalf("Patch failed: %v", err)
	}

	// Verify no changes were reported
	if changes.DependencyAdded {
		t.Error("Expected DependencyAdded to be false for already-patched project")
	}
	if changes.PluginAdded {
		t.Error("Expected PluginAdded to be false for already-patched project")
	}

	// Verify file was not modified
	content, err := os.ReadFile(tmpGradle)
	if err != nil {
		t.Fatalf("Failed to read build.gradle: %v", err)
	}
	// The file should still contain exactly one springdoc dependency
	springdocCount := strings.Count(string(content), "springdoc-openapi-starter-webmvc-ui")
	if springdocCount != 1 {
		t.Errorf("Expected exactly 1 springdoc dependency, found %d", springdocCount)
	}
}

// TestPatcher_Patch_NoBuildFile verifies that patching a project without a build file
// returns an appropriate error.
func TestPatcher_Patch_NoBuildFile(t *testing.T) {
	// Create a temp directory without any build files
	tmpDir := t.TempDir()

	patcher := spring.NewPatcher()
	opts := &extractor.PatchOptions{
		DryRun: false,
	}

	_, err := patcher.Patch(tmpDir, opts)
	if err == nil {
		t.Fatal("Expected error when no build file found")
	}
	if !strings.Contains(err.Error(), "no build file found") {
		t.Errorf("Expected 'no build file found' error, got: %v", err)
	}
}

// TestPatcher_OriginalContent tests that OriginalContent is correctly saved
func TestPatcher_OriginalContent(t *testing.T) {
	t.Run("Maven: saves original content", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpPom := filepath.Join(tmpDir, "pom.xml")
		originalContent := minimalPomWithoutSpringdoc
		if err := os.WriteFile(tmpPom, []byte(originalContent), 0644); err != nil {
			t.Fatalf("Failed to create temp pom.xml: %v", err)
		}

		patcher := spring.NewPatcher()
		opts := &extractor.PatchOptions{
			DryRun:             false,
			SpringdocVersion:   extractor.DefaultSpringdocVersion,
			MavenPluginVersion: extractor.DefaultSpringdocMavenPlugin,
		}

		result, err := patcher.Patch(tmpDir, opts)
		if err != nil {
			t.Fatalf("Patch failed: %v", err)
		}

		// Verify original content was saved
		if result.OriginalContent == "" {
			t.Error("OriginalContent should not be empty after patch")
		}
		if result.OriginalContent != originalContent {
			t.Errorf("OriginalContent mismatch:\nGot: %s\nWant: %s", result.OriginalContent, originalContent)
		}
	})

	t.Run("Gradle: saves original content", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpGradle := filepath.Join(tmpDir, "build.gradle")
		originalContent := minimalGradleWithoutSpringdoc
		if err := os.WriteFile(tmpGradle, []byte(originalContent), 0644); err != nil {
			t.Fatalf("Failed to create temp build.gradle: %v", err)
		}

		patcher := spring.NewPatcher()
		opts := &extractor.PatchOptions{
			DryRun:              false,
			SpringdocVersion:    extractor.DefaultSpringdocVersion,
			GradlePluginVersion: extractor.DefaultSpringdocGradlePlugin,
		}

		result, err := patcher.Patch(tmpDir, opts)
		if err != nil {
			t.Fatalf("Patch failed: %v", err)
		}

		// Verify original content was saved
		if result.OriginalContent == "" {
			t.Error("OriginalContent should not be empty after patch")
		}
		if result.OriginalContent != originalContent {
			t.Errorf("OriginalContent mismatch")
		}
	})

	t.Run("No changes: empty OriginalContent", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpPom := filepath.Join(tmpDir, "pom.xml")
		if err := os.WriteFile(tmpPom, []byte(minimalPomWithSpringdoc), 0644); err != nil {
			t.Fatalf("Failed to create temp pom.xml: %v", err)
		}

		patcher := spring.NewPatcher()
		opts := &extractor.PatchOptions{
			DryRun: false,
		}

		result, err := patcher.Patch(tmpDir, opts)
		if err != nil {
			t.Fatalf("Patch failed: %v", err)
		}

		// When no changes are made, OriginalContent should be empty
		if result.OriginalContent != "" {
			t.Error("OriginalContent should be empty when no changes are made")
		}
	})
}

// TestPatcher_Restore tests the Restore functionality
func TestPatcher_Restore(t *testing.T) {
	t.Run("restores original Maven content", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpPom := filepath.Join(tmpDir, "pom.xml")
		originalContent := minimalPomWithoutSpringdoc
		if err := os.WriteFile(tmpPom, []byte(originalContent), 0644); err != nil {
			t.Fatalf("Failed to create temp pom.xml: %v", err)
		}

		patcher := spring.NewPatcher()
		opts := &extractor.PatchOptions{
			SpringdocVersion:   extractor.DefaultSpringdocVersion,
			MavenPluginVersion: extractor.DefaultSpringdocMavenPlugin,
		}

		// Patch the file
		result, err := patcher.Patch(tmpDir, opts)
		if err != nil {
			t.Fatalf("Patch failed: %v", err)
		}

		// Verify file was modified
		content, _ := os.ReadFile(tmpPom)
		if strings.Contains(string(content), "springdoc-openapi") {
			// File was modified as expected
		} else {
			t.Fatal("File should have been modified with springdoc")
		}

		// Restore original content
		err = patcher.Restore(result.BuildFilePath, result.OriginalContent)
		if err != nil {
			t.Fatalf("Restore failed: %v", err)
		}

		// Verify file was restored
		restoredContent, err := os.ReadFile(tmpPom)
		if err != nil {
			t.Fatalf("Failed to read restored pom.xml: %v", err)
		}
		if string(restoredContent) != originalContent {
			t.Error("File content should be restored to original")
		}
	})

	t.Run("restore with empty content does nothing", func(t *testing.T) {
		patcher := spring.NewPatcher()
		err := patcher.Restore("/some/path", "")
		if err != nil {
			t.Errorf("Restore with empty content should not error, got: %v", err)
		}
	})
}

// TestPatcher_KeepPatchedOption tests the KeepPatched option behavior
func TestPatcher_KeepPatchedOption(t *testing.T) {
	// Note: KeepPatched is used by the caller (generate command) to decide
	// whether to call Restore(). The patcher itself always saves OriginalContent.
	// This test verifies that the option exists and doesn't break anything.

	t.Run("patcher works with KeepPatched=true", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpPom := filepath.Join(tmpDir, "pom.xml")
		if err := os.WriteFile(tmpPom, []byte(minimalPomWithoutSpringdoc), 0644); err != nil {
			t.Fatalf("Failed to create temp pom.xml: %v", err)
		}

		patcher := spring.NewPatcher()
		opts := &extractor.PatchOptions{
			KeepPatched:        true,
			SpringdocVersion:   extractor.DefaultSpringdocVersion,
			MavenPluginVersion: extractor.DefaultSpringdocMavenPlugin,
		}

		result, err := patcher.Patch(tmpDir, opts)
		if err != nil {
			t.Fatalf("Patch failed: %v", err)
		}

		// OriginalContent should still be saved (for potential restore by caller)
		if result.OriginalContent == "" {
			t.Error("OriginalContent should be saved regardless of KeepPatched")
		}
		if !result.DependencyAdded {
			t.Error("DependencyAdded should be true")
		}
	})

	t.Run("patcher works with KeepPatched=false", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpPom := filepath.Join(tmpDir, "pom.xml")
		if err := os.WriteFile(tmpPom, []byte(minimalPomWithoutSpringdoc), 0644); err != nil {
			t.Fatalf("Failed to create temp pom.xml: %v", err)
		}

		patcher := spring.NewPatcher()
		opts := &extractor.PatchOptions{
			KeepPatched:        false,
			SpringdocVersion:   extractor.DefaultSpringdocVersion,
			MavenPluginVersion: extractor.DefaultSpringdocMavenPlugin,
		}

		result, err := patcher.Patch(tmpDir, opts)
		if err != nil {
			t.Fatalf("Patch failed: %v", err)
		}

		// Same behavior - caller is responsible for restore
		if result.OriginalContent == "" {
			t.Error("OriginalContent should be saved regardless of KeepPatched")
		}
	})
}
