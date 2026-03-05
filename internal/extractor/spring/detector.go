// Package spring provides Spring framework specific extraction functionality.
package spring

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spencercjh/spec-forge/internal/extractor"
)

// Detector detects Spring project information.
type Detector struct {
	mavenParser  *MavenParser
	gradleParser *GradleParser
}

// NewDetector creates a new Detector instance.
func NewDetector() *Detector {
	return &Detector{
		mavenParser:  NewMavenParser(),
		gradleParser: NewGradleParser(),
	}
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

	// Also check for build.gradle.kts
	gradleKtsPath := filepath.Join(absPath, "build.gradle.kts")
	if _, err := os.Stat(gradleKtsPath); err == nil {
		return d.detectGradleProject(absPath, gradleKtsPath)
	}

	return nil, fmt.Errorf("no build file found (pom.xml or build.gradle) in %s", absPath)
}

// detectMavenProject analyzes a Maven project.
func (d *Detector) detectMavenProject(projectPath, pomPath string) (*extractor.ProjectInfo, error) {
	info := &extractor.ProjectInfo{
		BuildTool:     BuildToolMaven,
		BuildFilePath: pomPath,
	}

	// Parse pom.xml
	pom, err := d.mavenParser.Parse(pomPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pom.xml: %w", err)
	}

	// Check for multi-module project
	modules := d.mavenParser.GetModules(pom)
	if len(modules) > 0 {
		info.IsMultiModule = true
		info.Modules = modules

		// Find the main module (one with Spring Boot plugin)
		mainModule, mainModulePath := d.mavenParser.FindMainModule(projectPath, modules)
		if mainModule != "" {
			info.MainModule = mainModule
			info.MainModulePath = mainModulePath
		}
	}

	// Extract project information from parent POM
	info.SpringBootVersion = d.mavenParser.GetSpringBootVersion(pom)
	info.HasSpringdocDeps = d.mavenParser.HasSpringdocDependency(pom)
	info.SpringdocVersion = d.mavenParser.GetSpringdocVersion(pom)
	info.HasSpringdocPlugin = d.mavenParser.HasSpringdocPlugin(pom)

	// For multi-module projects, also check subproject pom files
	if info.IsMultiModule && info.MainModulePath != "" {
		subPom, err := d.mavenParser.Parse(info.MainModulePath)
		if err == nil {
			// Merge subproject info
			if subVersion := d.mavenParser.GetSpringBootVersion(subPom); subVersion != "" {
				info.SpringBootVersion = subVersion
			}
			if d.mavenParser.HasSpringdocDependency(subPom) {
				info.HasSpringdocDeps = true
				if v := d.mavenParser.GetSpringdocVersion(subPom); v != "" {
					info.SpringdocVersion = v
				}
			}
			if d.mavenParser.HasSpringdocPlugin(subPom) {
				info.HasSpringdocPlugin = true
			}
		}
	}

	return info, nil
}

// detectGradleProject analyzes a Gradle project.
func (d *Detector) detectGradleProject(projectPath, gradlePath string) (*extractor.ProjectInfo, error) {
	info := &extractor.ProjectInfo{
		BuildTool:     BuildToolGradle,
		BuildFilePath: gradlePath,
	}

	// Check for multi-module project via settings.gradle
	settingsPath := filepath.Join(projectPath, "settings.gradle")
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		settingsPath = filepath.Join(projectPath, "settings.gradle.kts")
	}

	modules := d.gradleParser.ParseModules(settingsPath)
	if len(modules) > 0 {
		info.IsMultiModule = true
		info.Modules = modules

		// Find the main module (one with Spring Boot plugin)
		mainModule, mainModulePath := d.gradleParser.FindMainModule(projectPath, modules)
		if mainModule != "" {
			info.MainModule = mainModule
			info.MainModulePath = mainModulePath
		}
	}

	// Parse build.gradle
	build, err := d.gradleParser.Parse(gradlePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse build.gradle: %w", err)
	}

	// Extract project information
	info.SpringBootVersion = d.gradleParser.GetSpringBootVersion(build)
	info.HasSpringdocDeps = d.gradleParser.HasSpringdocDependency(build)
	info.SpringdocVersion = d.gradleParser.GetSpringdocVersion(build)
	info.HasSpringdocPlugin = d.gradleParser.HasSpringdocPlugin(build)

	// For multi-module projects, also check subproject build files
	if info.IsMultiModule && info.MainModulePath != "" {
		subBuild, err := d.gradleParser.Parse(info.MainModulePath)
		if err == nil {
			// Merge subproject info
			if subVersion := d.gradleParser.GetSpringBootVersion(subBuild); subVersion != "" {
				info.SpringBootVersion = subVersion
			}
			if d.gradleParser.HasSpringdocDependency(subBuild) {
				info.HasSpringdocDeps = true
			}
			if d.gradleParser.HasSpringdocPlugin(subBuild) {
				info.HasSpringdocPlugin = true
			}
		}
	}

	return info, nil
}
