// Package spring_test tests the Spring extractor implementation.
package spring_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spencercjh/spec-forge/internal/extractor"
	"github.com/spencercjh/spec-forge/internal/extractor/spring"
)

func TestDetector_Detect_NoBuildFile(t *testing.T) {
	// Create temp dir without build files
	tmpDir := t.TempDir()

	detector := spring.NewDetector()
	_, err := detector.Detect(tmpDir)

	if err == nil {
		t.Error("Expected error when no build file found")
	}
}

func TestDetector_Detect_MavenProject(t *testing.T) {
	// Use the integration test project
	projectPath := "../../../integration-tests/maven-springboot-openapi-demo"

	// Skip if project doesn't exist (CI environment)
	if _, err := os.Stat(filepath.Join(projectPath, "pom.xml")); os.IsNotExist(err) {
		t.Skip("Integration test project not found")
	}

	detector := spring.NewDetector()
	info, err := detector.Detect(projectPath)

	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if info.BuildTool != extractor.BuildToolMaven {
		t.Errorf("BuildTool = %s, want %s", info.BuildTool, extractor.BuildToolMaven)
	}

	if info.BuildFilePath == "" {
		t.Error("BuildFilePath should not be empty")
	}

	if !info.HasSpringdocDeps {
		t.Error("HasSpringdocDeps should be true for this project")
	}
}

func TestDetector_Detect_GradleProject(t *testing.T) {
	// Use the integration test project
	projectPath := "../../../integration-tests/gradle-springboot-openapi-demo"

	// Skip if project doesn't exist (CI environment)
	if _, err := os.Stat(filepath.Join(projectPath, "build.gradle")); os.IsNotExist(err) {
		t.Skip("Integration test project not found")
	}

	detector := spring.NewDetector()
	info, err := detector.Detect(projectPath)

	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if info.BuildTool != extractor.BuildToolGradle {
		t.Errorf("BuildTool = %s, want %s", info.BuildTool, extractor.BuildToolGradle)
	}

	if info.BuildFilePath == "" {
		t.Error("BuildFilePath should not be empty")
	}

	if !info.HasSpringdocDeps {
		t.Error("HasSpringdocDeps should be true for this project")
	}
}
