// Package spring provides Spring framework specific extraction functionality.
package spring

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spencercjh/spec-forge/internal/extractor"
)

// Detector detects Spring project information.
type Detector struct{}

// NewDetector creates a new Detector instance.
func NewDetector() *Detector {
	return &Detector{}
}

// Detect analyzes a Spring project and returns its information.
func (d *Detector) Detect(projectPath string) (*extractor.ProjectInfo, error) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	// Check for Maven project first
	pomPath := filepath.Join(absPath, "pom.xml")
	if _, err := os.Stat(pomPath); err == nil {
		return d.detectMavenProject(absPath, pomPath)
	}

	// Check for Gradle project
	gradlePath := filepath.Join(absPath, "build.gradle")
	if _, err := os.Stat(gradlePath); err == nil {
		return d.detectGradleProject(absPath, gradlePath)
	}

	return nil, fmt.Errorf("no build file found (pom.xml or build.gradle) in %s", absPath)
}

// detectMavenProject analyzes a Maven project.
func (d *Detector) detectMavenProject(_, pomPath string) (*extractor.ProjectInfo, error) {
	info := &extractor.ProjectInfo{
		BuildTool:     extractor.BuildToolMaven,
		BuildFilePath: pomPath,
	}

	// Parse pom.xml using maven parser
	mavenParser := NewMavenParser()
	pom, err := mavenParser.Parse(pomPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pom.xml: %w", err)
	}

	// Extract Spring Boot version
	info.SpringBootVersion = mavenParser.GetSpringBootVersion(pom)

	// Check for springdoc dependencies
	info.HasSpringdocDeps = mavenParser.HasSpringdocDependency(pom)
	info.SpringdocVersion = mavenParser.GetSpringdocVersion(pom)

	// Check for springdoc plugin
	info.HasSpringdocPlugin = mavenParser.HasSpringdocPlugin(pom)

	return info, nil
}

// detectGradleProject analyzes a Gradle project.
func (d *Detector) detectGradleProject(_, gradlePath string) (*extractor.ProjectInfo, error) {
	info := &extractor.ProjectInfo{
		BuildTool:     extractor.BuildToolGradle,
		BuildFilePath: gradlePath,
	}

	// Parse build.gradle using gradle parser
	gradleParser := NewGradleParser()
	build, err := gradleParser.Parse(gradlePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse build.gradle: %w", err)
	}

	// Extract Spring Boot version
	info.SpringBootVersion = gradleParser.GetSpringBootVersion(build)

	// Check for springdoc dependencies
	info.HasSpringdocDeps = gradleParser.HasSpringdocDependency(build)
	info.SpringdocVersion = gradleParser.GetSpringdocVersion(build)

	// Check for springdoc plugin
	info.HasSpringdocPlugin = gradleParser.HasSpringdocPlugin(build)

	return info, nil
}
