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
			DryRun:               true,
			SpringdocVersion:     extractor.DefaultSpringdocVersion,
			GradlePluginVersion:  extractor.DefaultSpringdocGradlePlugin,
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
			DryRun:               false,
			SpringdocVersion:     extractor.DefaultSpringdocVersion,
			GradlePluginVersion:  extractor.DefaultSpringdocGradlePlugin,
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
