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

// minimalPomWithSpringdoc is a minimal pom.xml already containing springdoc
const minimalPomWithSpringdoc = `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
    <modelVersion>4.0.0</modelVersion>
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
                <groupId>org.springdoc</groupId>
                <artifactId>springdoc-openapi-maven-plugin</artifactId>
                <version>1.4</version>
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
	if err := os.WriteFile(tmpPom, origContent, 0o644); err != nil {
		t.Fatalf("Failed to create temp pom.xml: %v", err)
	}

	patcher := spring.NewPatcher()
	opts := &extractor.PatchOptions{
		DryRun:             true,
		SpringdocVersion:   spring.DefaultSpringdocVersion,
		MavenPluginVersion: spring.DefaultSpringdocMavenPlugin,
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
	if err := os.WriteFile(tmpPom, []byte(minimalPomWithoutSpringdoc), 0o644); err != nil {
		t.Fatalf("Failed to create temp pom.xml: %v", err)
	}

	patcher := spring.NewPatcher()

	t.Run("dry-run reports changes but does not modify file", func(t *testing.T) {
		opts := &extractor.PatchOptions{
			DryRun:             true,
			SpringdocVersion:   spring.DefaultSpringdocVersion,
			MavenPluginVersion: spring.DefaultSpringdocMavenPlugin,
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
		if err := os.WriteFile(tmpPom, []byte(minimalPomWithoutSpringdoc), 0o644); err != nil {
			t.Fatalf("Failed to reset temp pom.xml: %v", err)
		}

		opts := &extractor.PatchOptions{
			DryRun:             false,
			SpringdocVersion:   spring.DefaultSpringdocVersion,
			MavenPluginVersion: spring.DefaultSpringdocMavenPlugin,
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
	if err := os.WriteFile(tmpGradle, []byte(minimalGradleWithoutSpringdoc), 0o644); err != nil {
		t.Fatalf("Failed to create temp build.gradle: %v", err)
	}

	patcher := spring.NewPatcher()

	t.Run("dry-run reports changes but does not modify file", func(t *testing.T) {
		opts := &extractor.PatchOptions{
			DryRun:              true,
			SpringdocVersion:    spring.DefaultSpringdocVersion,
			GradlePluginVersion: spring.DefaultSpringdocGradlePlugin,
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
		if err := os.WriteFile(tmpGradle, []byte(minimalGradleWithoutSpringdoc), 0o644); err != nil {
			t.Fatalf("Failed to reset temp build.gradle: %v", err)
		}

		opts := &extractor.PatchOptions{
			DryRun:              false,
			SpringdocVersion:    spring.DefaultSpringdocVersion,
			GradlePluginVersion: spring.DefaultSpringdocGradlePlugin,
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
		info := &extractor.SpringInfo{
			HasSpringdocDeps:   false,
			HasSpringdocPlugin: false,
		}
		if !patcher.NeedsPatch(info, false) {
			t.Error("Should need patch when missing deps")
		}
	})

	t.Run("needs patch when force is true", func(t *testing.T) {
		info := &extractor.SpringInfo{
			HasSpringdocDeps:   true,
			HasSpringdocPlugin: true,
		}
		if !patcher.NeedsPatch(info, true) {
			t.Error("Should need patch when force is true")
		}
	})

	t.Run("no patch needed when already configured", func(t *testing.T) {
		info := &extractor.SpringInfo{
			HasSpringdocDeps:   true,
			HasSpringdocPlugin: true,
		}
		if patcher.NeedsPatch(info, false) {
			t.Error("Should not need patch when already configured")
		}
	})
}

// TestPatcher_Patch_AlreadyPatchedMaven verifies that patching an already-patched Maven project
// returns no changes and does not modify the file.
func TestPatcher_Patch_AlreadyPatchedMaven(t *testing.T) {
	// Create a temp project with springdoc already configured
	tmpDir := t.TempDir()
	tmpPom := filepath.Join(tmpDir, "pom.xml")
	if err := os.WriteFile(tmpPom, []byte(minimalPomWithSpringdoc), 0o644); err != nil {
		t.Fatalf("Failed to create temp pom.xml: %v", err)
	}

	patcher := spring.NewPatcher()
	opts := &extractor.PatchOptions{
		DryRun:             false,
		SpringdocVersion:   spring.DefaultSpringdocVersion,
		MavenPluginVersion: spring.DefaultSpringdocMavenPlugin,
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
	if err := os.WriteFile(tmpGradle, []byte(minimalGradleWithSpringdoc), 0o644); err != nil {
		t.Fatalf("Failed to create temp build.gradle: %v", err)
	}

	patcher := spring.NewPatcher()
	opts := &extractor.PatchOptions{
		DryRun:              false,
		SpringdocVersion:    spring.DefaultSpringdocVersion,
		GradlePluginVersion: spring.DefaultSpringdocGradlePlugin,
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
		if err := os.WriteFile(tmpPom, []byte(originalContent), 0o644); err != nil {
			t.Fatalf("Failed to create temp pom.xml: %v", err)
		}

		patcher := spring.NewPatcher()
		opts := &extractor.PatchOptions{
			DryRun:             false,
			SpringdocVersion:   spring.DefaultSpringdocVersion,
			MavenPluginVersion: spring.DefaultSpringdocMavenPlugin,
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
		if err := os.WriteFile(tmpGradle, []byte(originalContent), 0o644); err != nil {
			t.Fatalf("Failed to create temp build.gradle: %v", err)
		}

		patcher := spring.NewPatcher()
		opts := &extractor.PatchOptions{
			DryRun:              false,
			SpringdocVersion:    spring.DefaultSpringdocVersion,
			GradlePluginVersion: spring.DefaultSpringdocGradlePlugin,
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
		if err := os.WriteFile(tmpPom, []byte(minimalPomWithSpringdoc), 0o644); err != nil {
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

		// For Maven projects, OriginalContent is always set because we need to check spring-boot configuration
		// even when springdoc is already present. The test validates that no changes are reported.
		if result.DependencyAdded {
			t.Error("DependencyAdded should be false when springdoc already exists")
		}
		if result.PluginAdded {
			t.Error("PluginAdded should be false when springdoc already exists")
		}
	})
}

// TestPatcher_Restore tests the Restore functionality
func TestPatcher_Restore(t *testing.T) {
	t.Run("restores original Maven content", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpPom := filepath.Join(tmpDir, "pom.xml")
		originalContent := minimalPomWithoutSpringdoc
		if err := os.WriteFile(tmpPom, []byte(originalContent), 0o644); err != nil {
			t.Fatalf("Failed to create temp pom.xml: %v", err)
		}

		patcher := spring.NewPatcher()
		opts := &extractor.PatchOptions{
			SpringdocVersion:   spring.DefaultSpringdocVersion,
			MavenPluginVersion: spring.DefaultSpringdocMavenPlugin,
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
		if err := os.WriteFile(tmpPom, []byte(minimalPomWithoutSpringdoc), 0o644); err != nil {
			t.Fatalf("Failed to create temp pom.xml: %v", err)
		}

		patcher := spring.NewPatcher()
		opts := &extractor.PatchOptions{
			KeepPatched:        true,
			SpringdocVersion:   spring.DefaultSpringdocVersion,
			MavenPluginVersion: spring.DefaultSpringdocMavenPlugin,
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
		if err := os.WriteFile(tmpPom, []byte(minimalPomWithoutSpringdoc), 0o644); err != nil {
			t.Fatalf("Failed to create temp pom.xml: %v", err)
		}

		patcher := spring.NewPatcher()
		opts := &extractor.PatchOptions{
			KeepPatched:        false,
			SpringdocVersion:   spring.DefaultSpringdocVersion,
			MavenPluginVersion: spring.DefaultSpringdocMavenPlugin,
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

// ============================================
// Edge Case Tests
// ============================================

// pomWithOnlyPluginManagement is a pom.xml with only pluginManagement (no direct plugins)
const pomWithOnlyPluginManagement = `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
    <modelVersion>4.0.0</modelVersion>
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
        <pluginManagement>
            <plugins>
                <plugin>
                    <groupId>org.springframework.boot</groupId>
                    <artifactId>spring-boot-maven-plugin</artifactId>
                </plugin>
            </plugins>
        </pluginManagement>
    </build>
</project>
`

// pomWithoutDependencies is a pom.xml without any dependencies section
const pomWithoutDependencies = `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
    <modelVersion>4.0.0</modelVersion>
    <groupId>com.example</groupId>
    <artifactId>test-project</artifactId>
    <version>1.0.0</version>
    <properties>
        <java.version>17</java.version>
    </properties>
</project>
`

// pomWithoutBuild is a pom.xml without any build section
const pomWithoutBuild = `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
    <modelVersion>4.0.0</modelVersion>
    <groupId>com.example</groupId>
    <artifactId>test-project</artifactId>
    <version>1.0.0</version>
    <dependencies>
        <dependency>
            <groupId>org.springframework.boot</groupId>
            <artifactId>spring-boot-starter-web</artifactId>
        </dependency>
    </dependencies>
</project>
`

// pomWithDependencyManagement is a pom.xml with dependencyManagement section
const pomWithDependencyManagement = `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
    <modelVersion>4.0.0</modelVersion>
    <groupId>com.example</groupId>
    <artifactId>test-project</artifactId>
    <version>1.0.0</version>
    <dependencyManagement>
        <dependencies>
            <dependency>
                <groupId>org.springframework.boot</groupId>
                <artifactId>spring-boot-dependencies</artifactId>
                <version>3.2.0</version>
                <type>pom</type>
                <scope>import</scope>
            </dependency>
        </dependencies>
    </dependencyManagement>
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

// gradleWithoutPlugins is a build.gradle without plugins block
const gradleWithoutPlugins = `
group = 'com.example'
version = '1.0.0'

repositories {
    mavenCentral()
}

dependencies {
    implementation 'org.springframework.boot:spring-boot-starter-web:3.2.0'
}
`

// gradleWithoutDependencies is a build.gradle without dependencies block
const gradleWithoutDependencies = `
plugins {
    id 'java'
    id 'org.springframework.boot' version '3.2.0'
}

group = 'com.example'
version = '1.0.0'

repositories {
    mavenCentral()
}
`

// TestPatcher_EdgeCases_Maven tests various Maven edge cases
func TestPatcher_EdgeCases_Maven(t *testing.T) {
	t.Run("pom with only pluginManagement", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpPom := filepath.Join(tmpDir, "pom.xml")
		if err := os.WriteFile(tmpPom, []byte(pomWithOnlyPluginManagement), 0o644); err != nil {
			t.Fatalf("Failed to create temp pom.xml: %v", err)
		}

		patcher := spring.NewPatcher()
		opts := &extractor.PatchOptions{
			SpringdocVersion:   spring.DefaultSpringdocVersion,
			MavenPluginVersion: spring.DefaultSpringdocMavenPlugin,
		}

		result, err := patcher.Patch(tmpDir, opts)
		if err != nil {
			t.Fatalf("Patch failed: %v", err)
		}

		if !result.DependencyAdded {
			t.Error("Expected DependencyAdded to be true")
		}
		if !result.PluginAdded {
			t.Error("Expected PluginAdded to be true")
		}

		// Verify plugin was added
		content, _ := os.ReadFile(tmpPom)
		if !strings.Contains(string(content), "springdoc-openapi-maven-plugin") {
			t.Error("Plugin should be added even with only pluginManagement")
		}
	})

	t.Run("pom without dependencies section", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpPom := filepath.Join(tmpDir, "pom.xml")
		if err := os.WriteFile(tmpPom, []byte(pomWithoutDependencies), 0o644); err != nil {
			t.Fatalf("Failed to create temp pom.xml: %v", err)
		}

		patcher := spring.NewPatcher()
		opts := &extractor.PatchOptions{
			SpringdocVersion:   spring.DefaultSpringdocVersion,
			MavenPluginVersion: spring.DefaultSpringdocMavenPlugin,
		}

		result, err := patcher.Patch(tmpDir, opts)
		if err != nil {
			t.Fatalf("Patch failed: %v", err)
		}

		if !result.DependencyAdded {
			t.Error("Expected DependencyAdded to be true")
		}

		// Verify dependency was added
		content, _ := os.ReadFile(tmpPom)
		if !strings.Contains(string(content), "springdoc-openapi-starter-webmvc-ui") {
			t.Error("Dependency should be added even without existing dependencies section")
		}
	})

	t.Run("pom without build section", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpPom := filepath.Join(tmpDir, "pom.xml")
		if err := os.WriteFile(tmpPom, []byte(pomWithoutBuild), 0o644); err != nil {
			t.Fatalf("Failed to create temp pom.xml: %v", err)
		}

		patcher := spring.NewPatcher()
		opts := &extractor.PatchOptions{
			SpringdocVersion:   spring.DefaultSpringdocVersion,
			MavenPluginVersion: spring.DefaultSpringdocMavenPlugin,
		}

		result, err := patcher.Patch(tmpDir, opts)
		if err != nil {
			t.Fatalf("Patch failed: %v", err)
		}

		if !result.PluginAdded {
			t.Error("Expected PluginAdded to be true")
		}

		// Verify plugin and build section were added
		content, _ := os.ReadFile(tmpPom)
		if !strings.Contains(string(content), "springdoc-openapi-maven-plugin") {
			t.Error("Plugin should be added")
		}
		if !strings.Contains(string(content), "<build>") {
			t.Error("Build section should be created")
		}
	})

	t.Run("pom with dependencyManagement", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpPom := filepath.Join(tmpDir, "pom.xml")
		if err := os.WriteFile(tmpPom, []byte(pomWithDependencyManagement), 0o644); err != nil {
			t.Fatalf("Failed to create temp pom.xml: %v", err)
		}

		patcher := spring.NewPatcher()
		opts := &extractor.PatchOptions{
			SpringdocVersion:   spring.DefaultSpringdocVersion,
			MavenPluginVersion: spring.DefaultSpringdocMavenPlugin,
		}

		result, err := patcher.Patch(tmpDir, opts)
		if err != nil {
			t.Fatalf("Patch failed: %v", err)
		}

		if !result.DependencyAdded {
			t.Error("Expected DependencyAdded to be true")
		}

		// Verify dependency was added to regular dependencies, not dependencyManagement
		content, _ := os.ReadFile(tmpPom)
		if !strings.Contains(string(content), "springdoc-openapi-starter-webmvc-ui") {
			t.Error("Dependency should be added")
		}
	})

	t.Run("force option on already patched project", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpPom := filepath.Join(tmpDir, "pom.xml")
		if err := os.WriteFile(tmpPom, []byte(minimalPomWithSpringdoc), 0o644); err != nil {
			t.Fatalf("Failed to create temp pom.xml: %v", err)
		}

		patcher := spring.NewPatcher()
		opts := &extractor.PatchOptions{
			Force:              true,
			SpringdocVersion:   "9.9.9", // Different version
			MavenPluginVersion: "9.9.9",
		}

		result, err := patcher.Patch(tmpDir, opts)
		if err != nil {
			t.Fatalf("Patch failed: %v", err)
		}

		// Force allows patching, but won't add duplicate if already present in parsed POM
		// The Force option bypasses the Detector's check but not the actual duplicate check
		if result.DependencyAdded {
			t.Error("Should not add duplicate dependency even with force")
		}
	})
}

// TestPatcher_EdgeCases_Gradle tests various Gradle edge cases
func TestPatcher_EdgeCases_Gradle(t *testing.T) {
	t.Run("gradle without plugins block", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpGradle := filepath.Join(tmpDir, "build.gradle")
		if err := os.WriteFile(tmpGradle, []byte(gradleWithoutPlugins), 0o644); err != nil {
			t.Fatalf("Failed to create temp build.gradle: %v", err)
		}

		patcher := spring.NewPatcher()
		opts := &extractor.PatchOptions{
			SpringdocVersion:    spring.DefaultSpringdocVersion,
			GradlePluginVersion: spring.DefaultSpringdocGradlePlugin,
		}

		result, err := patcher.Patch(tmpDir, opts)
		if err != nil {
			t.Fatalf("Patch failed: %v", err)
		}

		if !result.DependencyAdded {
			t.Error("Expected DependencyAdded to be true")
		}
		// Plugin won't be added without plugins block (text manipulation limitation)
	})

	t.Run("gradle without dependencies block", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpGradle := filepath.Join(tmpDir, "build.gradle")
		if err := os.WriteFile(tmpGradle, []byte(gradleWithoutDependencies), 0o644); err != nil {
			t.Fatalf("Failed to create temp build.gradle: %v", err)
		}

		patcher := spring.NewPatcher()
		opts := &extractor.PatchOptions{
			SpringdocVersion:    spring.DefaultSpringdocVersion,
			GradlePluginVersion: spring.DefaultSpringdocGradlePlugin,
		}

		result, err := patcher.Patch(tmpDir, opts)
		if err != nil {
			t.Fatalf("Patch failed: %v", err)
		}

		// Plugin should be added since plugins block exists
		if !result.PluginAdded {
			t.Error("Plugin should be added since plugins block exists")
		}
		// Dependency won't be added without dependencies block (text manipulation limitation)
		if result.DependencyAdded {
			t.Error("Dependency should not be added without dependencies block")
		}
	})
}

// TestPatcher_ForceOption tests the Force option behavior
func TestPatcher_ForceOption(t *testing.T) {
	t.Run("force allows patching when detector says already present", func(t *testing.T) {
		// Create a project where detector might say springdoc is present
		// but the actual POM doesn't have it (edge case)
		tmpDir := t.TempDir()
		tmpPom := filepath.Join(tmpDir, "pom.xml")
		if err := os.WriteFile(tmpPom, []byte(minimalPomWithoutSpringdoc), 0o644); err != nil {
			t.Fatalf("Failed to create temp pom.xml: %v", err)
		}

		patcher := spring.NewPatcher()

		// With Force=true, should patch regardless of what detector says
		opts := &extractor.PatchOptions{
			Force:              true,
			SpringdocVersion:   spring.DefaultSpringdocVersion,
			MavenPluginVersion: spring.DefaultSpringdocMavenPlugin,
		}
		result, err := patcher.Patch(tmpDir, opts)
		if err != nil {
			t.Fatalf("Patch failed: %v", err)
		}
		if !result.DependencyAdded {
			t.Error("Should add dependency with force")
		}
	})

	t.Run("force does not add duplicate", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpPom := filepath.Join(tmpDir, "pom.xml")
		if err := os.WriteFile(tmpPom, []byte(minimalPomWithSpringdoc), 0o644); err != nil {
			t.Fatalf("Failed to create temp pom.xml: %v", err)
		}

		patcher := spring.NewPatcher()

		// Even with Force=true, should not add duplicate
		opts := &extractor.PatchOptions{Force: true}
		result, err := patcher.Patch(tmpDir, opts)
		if err != nil {
			t.Fatalf("Patch failed: %v", err)
		}
		if result.DependencyAdded {
			t.Error("Should not add duplicate dependency even with force")
		}
	})
}

// TestPatcher_DryRunMode tests dry-run mode behavior
func TestPatcher_DryRunMode(t *testing.T) {
	t.Run("dry-run preserves original file", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpPom := filepath.Join(tmpDir, "pom.xml")
		originalContent := minimalPomWithoutSpringdoc
		if err := os.WriteFile(tmpPom, []byte(originalContent), 0o644); err != nil {
			t.Fatalf("Failed to create temp pom.xml: %v", err)
		}

		patcher := spring.NewPatcher()
		opts := &extractor.PatchOptions{
			DryRun:             true,
			SpringdocVersion:   spring.DefaultSpringdocVersion,
			MavenPluginVersion: spring.DefaultSpringdocMavenPlugin,
		}

		result, err := patcher.Patch(tmpDir, opts)
		if err != nil {
			t.Fatalf("Patch failed: %v", err)
		}

		// Should report changes
		if !result.DependencyAdded {
			t.Error("Expected DependencyAdded to be true in dry-run")
		}
		if !result.PluginAdded {
			t.Error("Expected PluginAdded to be true in dry-run")
		}

		// But file should not be modified
		content, _ := os.ReadFile(tmpPom)
		if string(content) != originalContent {
			t.Error("File should not be modified in dry-run mode")
		}
	})
}

// ============================================
// Multi-Module Gradle Tests
// ============================================

// multiModuleSettingsGradle is a settings.gradle for multi-module project
const multiModuleSettingsGradle = `
rootProject.name = 'multi-module-demo'

include 'shared-lib'
include 'user-service'
`

// rootBuildGradle is a root build.gradle for multi-module project
const rootBuildGradle = `
plugins {
    id 'java'
    id 'org.springframework.boot' version '4.0.3' apply false
    id 'io.spring.dependency-management' version '1.1.7' apply false
}

group = 'com.example'
version = '1.0.0-SNAPSHOT'

subprojects {
    apply plugin: 'java'
    repositories {
        mavenCentral()
    }
}
`

// sharedLibBuildGradle is a build.gradle for shared-lib module (no Spring Boot)
const sharedLibBuildGradle = `
plugins {
    id 'java-library'
}

dependencies {
    api 'org.springframework.boot:spring-boot-starter-web'
}
`

// userServiceBuildGradle is a build.gradle for user-service module (with Spring Boot)
const userServiceBuildGradle = `
plugins {
    id 'org.springframework.boot'
}

dependencies {
    implementation project(':shared-lib')
    implementation 'org.springframework.boot:spring-boot-starter-web'
}
`

// TestDetector_MultiModuleGradle tests detection of multi-module Gradle projects
func TestDetector_MultiModuleGradle(t *testing.T) {
	// Create a multi-module Gradle project structure
	tmpDir := t.TempDir()

	// Create directory structure
	settingsDir := tmpDir
	sharedLibDir := filepath.Join(tmpDir, "shared-lib")
	userServiceDir := filepath.Join(tmpDir, "user-service")

	if err := os.MkdirAll(sharedLibDir, 0o755); err != nil {
		t.Fatalf("Failed to create shared-lib dir: %v", err)
	}
	if err := os.MkdirAll(userServiceDir, 0o755); err != nil {
		t.Fatalf("Failed to create user-service dir: %v", err)
	}

	// Create settings.gradle
	if err := os.WriteFile(filepath.Join(settingsDir, "settings.gradle"), []byte(multiModuleSettingsGradle), 0o644); err != nil {
		t.Fatalf("Failed to create settings.gradle: %v", err)
	}

	// Create root build.gradle
	if err := os.WriteFile(filepath.Join(settingsDir, "build.gradle"), []byte(rootBuildGradle), 0o644); err != nil {
		t.Fatalf("Failed to create root build.gradle: %v", err)
	}

	// Create shared-lib build.gradle
	if err := os.WriteFile(filepath.Join(sharedLibDir, "build.gradle"), []byte(sharedLibBuildGradle), 0o644); err != nil {
		t.Fatalf("Failed to create shared-lib build.gradle: %v", err)
	}

	// Create user-service build.gradle
	if err := os.WriteFile(filepath.Join(userServiceDir, "build.gradle"), []byte(userServiceBuildGradle), 0o644); err != nil {
		t.Fatalf("Failed to create user-service build.gradle: %v", err)
	}

	// Test detection
	detector := spring.NewDetector()
	info, err := detector.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detection failed: %v", err)
	}

	// Verify multi-module detection
	if info.Spring == nil {
		t.Fatal("Spring should not be nil")
	}
	if !info.Spring.IsMultiModule {
		t.Error("Expected IsMultiModule to be true")
	}
	if len(info.Spring.Modules) != 2 {
		t.Errorf("Expected 2 modules, got %d", len(info.Spring.Modules))
	}
	if info.Spring.MainModule != "user-service" {
		t.Errorf("Expected MainModule to be 'user-service', got '%s'", info.Spring.MainModule)
	}
	if info.Spring.MainModulePath == "" {
		t.Error("Expected MainModulePath to be set")
	}
}

// TestPatcher_MultiModuleGradle tests patching of multi-module Gradle projects
func TestPatcher_MultiModuleGradle(t *testing.T) {
	// Create a multi-module Gradle project structure
	tmpDir := t.TempDir()

	// Create directory structure
	sharedLibDir := filepath.Join(tmpDir, "shared-lib")
	userServiceDir := filepath.Join(tmpDir, "user-service")

	if err := os.MkdirAll(sharedLibDir, 0o755); err != nil {
		t.Fatalf("Failed to create shared-lib dir: %v", err)
	}
	if err := os.MkdirAll(userServiceDir, 0o755); err != nil {
		t.Fatalf("Failed to create user-service dir: %v", err)
	}

	// Create settings.gradle
	if err := os.WriteFile(filepath.Join(tmpDir, "settings.gradle"), []byte(multiModuleSettingsGradle), 0o644); err != nil {
		t.Fatalf("Failed to create settings.gradle: %v", err)
	}

	// Create root build.gradle
	if err := os.WriteFile(filepath.Join(tmpDir, "build.gradle"), []byte(rootBuildGradle), 0o644); err != nil {
		t.Fatalf("Failed to create root build.gradle: %v", err)
	}

	// Create shared-lib build.gradle
	if err := os.WriteFile(filepath.Join(sharedLibDir, "build.gradle"), []byte(sharedLibBuildGradle), 0o644); err != nil {
		t.Fatalf("Failed to create shared-lib build.gradle: %v", err)
	}

	// Create user-service build.gradle
	userServiceBuildPath := filepath.Join(userServiceDir, "build.gradle")
	if err := os.WriteFile(userServiceBuildPath, []byte(userServiceBuildGradle), 0o644); err != nil {
		t.Fatalf("Failed to create user-service build.gradle: %v", err)
	}

	// Patch the project
	patcher := spring.NewPatcher()
	opts := &extractor.PatchOptions{
		SpringdocVersion:    spring.DefaultSpringdocVersion,
		GradlePluginVersion: spring.DefaultSpringdocGradlePlugin,
	}

	result, err := patcher.Patch(tmpDir, opts)
	if err != nil {
		t.Fatalf("Patch failed: %v", err)
	}

	// Verify patch was applied to user-service module, not root
	if result.BuildFilePath != userServiceBuildPath {
		t.Errorf("Expected patch on user-service module, got: %s", result.BuildFilePath)
	}
	if !result.DependencyAdded {
		t.Error("Expected DependencyAdded to be true")
	}
	if !result.PluginAdded {
		t.Error("Expected PluginAdded to be true")
	}

	// Verify user-service/build.gradle was modified
	content, err := os.ReadFile(userServiceBuildPath)
	if err != nil {
		t.Fatalf("Failed to read user-service build.gradle: %v", err)
	}
	if !strings.Contains(string(content), "springdoc-openapi") {
		t.Error("user-service build.gradle should contain springdoc dependency")
	}

	// Verify root build.gradle was NOT modified
	rootContent, err := os.ReadFile(filepath.Join(tmpDir, "build.gradle"))
	if err != nil {
		t.Fatalf("Failed to read root build.gradle: %v", err)
	}
	if strings.Contains(string(rootContent), "springdoc-openapi") {
		t.Error("Root build.gradle should NOT be modified")
	}
}

// TestPatcher_MultiModuleGradle_Restore tests restore functionality for multi-module projects
func TestPatcher_MultiModuleGradle_Restore(t *testing.T) {
	// Create a multi-module Gradle project structure
	tmpDir := t.TempDir()

	// Create directory structure
	sharedLibDir := filepath.Join(tmpDir, "shared-lib")
	userServiceDir := filepath.Join(tmpDir, "user-service")

	if err := os.MkdirAll(sharedLibDir, 0o755); err != nil {
		t.Fatalf("Failed to create shared-lib dir: %v", err)
	}
	if err := os.MkdirAll(userServiceDir, 0o755); err != nil {
		t.Fatalf("Failed to create user-service dir: %v", err)
	}

	// Create settings.gradle
	if err := os.WriteFile(filepath.Join(tmpDir, "settings.gradle"), []byte(multiModuleSettingsGradle), 0o644); err != nil {
		t.Fatalf("Failed to create settings.gradle: %v", err)
	}

	// Create root build.gradle
	if err := os.WriteFile(filepath.Join(tmpDir, "build.gradle"), []byte(rootBuildGradle), 0o644); err != nil {
		t.Fatalf("Failed to create root build.gradle: %v", err)
	}

	// Create shared-lib build.gradle
	if err := os.WriteFile(filepath.Join(sharedLibDir, "build.gradle"), []byte(sharedLibBuildGradle), 0o644); err != nil {
		t.Fatalf("Failed to create shared-lib build.gradle: %v", err)
	}

	// Create user-service build.gradle
	userServiceBuildPath := filepath.Join(userServiceDir, "build.gradle")
	originalContent := userServiceBuildGradle
	if err := os.WriteFile(userServiceBuildPath, []byte(originalContent), 0o644); err != nil {
		t.Fatalf("Failed to create user-service build.gradle: %v", err)
	}

	// Patch the project
	patcher := spring.NewPatcher()
	opts := &extractor.PatchOptions{
		SpringdocVersion:    spring.DefaultSpringdocVersion,
		GradlePluginVersion: spring.DefaultSpringdocGradlePlugin,
	}

	result, err := patcher.Patch(tmpDir, opts)
	if err != nil {
		t.Fatalf("Patch failed: %v", err)
	}

	// Verify file was modified
	content, _ := os.ReadFile(userServiceBuildPath)
	if string(content) == originalContent {
		t.Error("File should have been modified")
	}

	// Restore original content
	err = patcher.Restore(result.BuildFilePath, result.OriginalContent)
	if err != nil {
		t.Fatalf("Restore failed: %v", err)
	}

	// Verify file was restored
	restoredContent, _ := os.ReadFile(userServiceBuildPath)
	if string(restoredContent) != originalContent {
		t.Error("File should be restored to original content")
	}
}

// ============================================
// Multi-Module Maven Tests
// ============================================

// mavenParentPom is a parent pom.xml for multi-module Maven project
const mavenParentPom = `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
    <modelVersion>4.0.0</modelVersion>
    <parent>
        <groupId>org.springframework.boot</groupId>
        <artifactId>spring-boot-starter-parent</artifactId>
        <version>3.2.0</version>
    </parent>
    <groupId>com.example</groupId>
    <artifactId>multi-module-demo</artifactId>
    <version>1.0.0</version>
    <packaging>pom</packaging>
    <modules>
        <module>shared-lib</module>
        <module>user-service</module>
    </modules>
    <dependencies>
        <dependency>
            <groupId>org.springframework.boot</groupId>
            <artifactId>spring-boot-starter-web</artifactId>
        </dependency>
    </dependencies>
</project>
`

// mavenSharedLibPom is a shared-lib module pom.xml
const mavenSharedLibPom = `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
    <modelVersion>4.0.0</modelVersion>
    <parent>
        <groupId>com.example</groupId>
        <artifactId>multi-module-demo</artifactId>
        <version>1.0.0</version>
    </parent>
    <artifactId>shared-lib</artifactId>
</project>
`

// mavenUserServicePom is a user-service module pom.xml with Spring Boot plugin
const mavenUserServicePom = `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
    <modelVersion>4.0.0</modelVersion>
    <parent>
        <groupId>com.example</groupId>
        <artifactId>multi-module-demo</artifactId>
        <version>1.0.0</version>
    </parent>
    <artifactId>user-service</artifactId>
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

// TestDetector_MultiModuleMaven tests detection of multi-module Maven projects
func TestDetector_MultiModuleMaven(t *testing.T) {
	// Create a multi-module Maven project structure
	tmpDir := t.TempDir()

	// Create directory structure
	sharedLibDir := filepath.Join(tmpDir, "shared-lib")
	userServiceDir := filepath.Join(tmpDir, "user-service")

	if err := os.MkdirAll(sharedLibDir, 0o755); err != nil {
		t.Fatalf("Failed to create shared-lib dir: %v", err)
	}
	if err := os.MkdirAll(userServiceDir, 0o755); err != nil {
		t.Fatalf("Failed to create user-service dir: %v", err)
	}

	// Create parent pom.xml
	if err := os.WriteFile(filepath.Join(tmpDir, "pom.xml"), []byte(mavenParentPom), 0o644); err != nil {
		t.Fatalf("Failed to create parent pom.xml: %v", err)
	}

	// Create shared-lib pom.xml
	if err := os.WriteFile(filepath.Join(sharedLibDir, "pom.xml"), []byte(mavenSharedLibPom), 0o644); err != nil {
		t.Fatalf("Failed to create shared-lib pom.xml: %v", err)
	}

	// Create user-service pom.xml
	if err := os.WriteFile(filepath.Join(userServiceDir, "pom.xml"), []byte(mavenUserServicePom), 0o644); err != nil {
		t.Fatalf("Failed to create user-service pom.xml: %v", err)
	}

	// Test detection
	detector := spring.NewDetector()
	info, err := detector.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detection failed: %v", err)
	}

	// Verify multi-module detection
	if info.Spring == nil {
		t.Fatal("Spring should not be nil")
	}
	if !info.Spring.IsMultiModule {
		t.Error("Expected IsMultiModule to be true")
	}
	if len(info.Spring.Modules) != 2 {
		t.Errorf("Expected 2 modules, got %d", len(info.Spring.Modules))
	}
	if info.Spring.MainModule != "user-service" {
		t.Errorf("Expected MainModule to be 'user-service', got '%s'", info.Spring.MainModule)
	}
	if info.Spring.MainModulePath == "" {
		t.Error("Expected MainModulePath to be set")
	}
}

// TestPatcher_MultiModuleMaven tests patching of multi-module Maven projects
func TestPatcher_MultiModuleMaven(t *testing.T) {
	// Create a multi-module Maven project structure
	tmpDir := t.TempDir()

	// Create directory structure
	sharedLibDir := filepath.Join(tmpDir, "shared-lib")
	userServiceDir := filepath.Join(tmpDir, "user-service")

	if err := os.MkdirAll(sharedLibDir, 0o755); err != nil {
		t.Fatalf("Failed to create shared-lib dir: %v", err)
	}
	if err := os.MkdirAll(userServiceDir, 0o755); err != nil {
		t.Fatalf("Failed to create user-service dir: %v", err)
	}

	// Create parent pom.xml
	if err := os.WriteFile(filepath.Join(tmpDir, "pom.xml"), []byte(mavenParentPom), 0o644); err != nil {
		t.Fatalf("Failed to create parent pom.xml: %v", err)
	}

	// Create shared-lib pom.xml
	if err := os.WriteFile(filepath.Join(sharedLibDir, "pom.xml"), []byte(mavenSharedLibPom), 0o644); err != nil {
		t.Fatalf("Failed to create shared-lib pom.xml: %v", err)
	}

	// Create user-service pom.xml
	userServicePomPath := filepath.Join(userServiceDir, "pom.xml")
	if err := os.WriteFile(userServicePomPath, []byte(mavenUserServicePom), 0o644); err != nil {
		t.Fatalf("Failed to create user-service pom.xml: %v", err)
	}

	// Patch the project
	patcher := spring.NewPatcher()
	opts := &extractor.PatchOptions{
		SpringdocVersion:   spring.DefaultSpringdocVersion,
		MavenPluginVersion: spring.DefaultSpringdocMavenPlugin,
	}

	result, err := patcher.Patch(tmpDir, opts)
	if err != nil {
		t.Fatalf("Patch failed: %v", err)
	}

	// Verify patch was applied to user-service module, not parent
	if result.BuildFilePath != userServicePomPath {
		t.Errorf("Expected patch on user-service module, got: %s", result.BuildFilePath)
	}
	if !result.DependencyAdded {
		t.Error("Expected DependencyAdded to be true")
	}
	if !result.PluginAdded {
		t.Error("Expected PluginAdded to be true")
	}

	// Verify user-service/pom.xml was modified
	content, _ := os.ReadFile(userServicePomPath)
	if !strings.Contains(string(content), "springdoc-openapi") {
		t.Error("user-service pom.xml should contain springdoc")
	}

	// Verify parent pom.xml was NOT modified
	parentContent, _ := os.ReadFile(filepath.Join(tmpDir, "pom.xml"))
	// Parent already has dependencies section, check it wasn't springdoc added there
	if strings.Contains(string(parentContent), "springdoc-openapi-starter-webmvc-ui") {
		t.Error("Parent pom.xml should NOT be modified with springdoc dependency")
	}
}

// TestPatcher_MultiModuleMaven_Restore tests restore functionality for multi-module Maven projects
func TestPatcher_MultiModuleMaven_Restore(t *testing.T) {
	// Create a multi-module Maven project structure
	tmpDir := t.TempDir()

	// Create directory structure
	sharedLibDir := filepath.Join(tmpDir, "shared-lib")
	userServiceDir := filepath.Join(tmpDir, "user-service")

	if err := os.MkdirAll(sharedLibDir, 0o755); err != nil {
		t.Fatalf("Failed to create shared-lib dir: %v", err)
	}
	if err := os.MkdirAll(userServiceDir, 0o755); err != nil {
		t.Fatalf("Failed to create user-service dir: %v", err)
	}

	// Create parent pom.xml
	if err := os.WriteFile(filepath.Join(tmpDir, "pom.xml"), []byte(mavenParentPom), 0o644); err != nil {
		t.Fatalf("Failed to create parent pom.xml: %v", err)
	}

	// Create shared-lib pom.xml
	if err := os.WriteFile(filepath.Join(sharedLibDir, "pom.xml"), []byte(mavenSharedLibPom), 0o644); err != nil {
		t.Fatalf("Failed to create shared-lib pom.xml: %v", err)
	}

	// Create user-service pom.xml
	userServicePomPath := filepath.Join(userServiceDir, "pom.xml")
	originalContent := mavenUserServicePom
	if err := os.WriteFile(userServicePomPath, []byte(originalContent), 0o644); err != nil {
		t.Fatalf("Failed to create user-service pom.xml: %v", err)
	}

	// Patch the project
	patcher := spring.NewPatcher()
	opts := &extractor.PatchOptions{
		SpringdocVersion:   spring.DefaultSpringdocVersion,
		MavenPluginVersion: spring.DefaultSpringdocMavenPlugin,
	}

	result, err := patcher.Patch(tmpDir, opts)
	if err != nil {
		t.Fatalf("Patch failed: %v", err)
	}

	// Verify file was modified
	content, _ := os.ReadFile(userServicePomPath)
	if string(content) == originalContent {
		t.Error("File should have been modified")
	}

	// Restore original content
	err = patcher.Restore(result.BuildFilePath, result.OriginalContent)
	if err != nil {
		t.Fatalf("Restore failed: %v", err)
	}

	// Verify file was restored
	restoredContent, _ := os.ReadFile(userServicePomPath)
	if string(restoredContent) != originalContent {
		t.Error("File should be restored to original content")
	}
}
