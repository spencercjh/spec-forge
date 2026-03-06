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
	springInfo := &Info{}

	// Parse pom.xml
	pom, err := d.mavenParser.Parse(pomPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pom.xml: %w", err)
	}

	// Check for multi-module project
	modules := d.mavenParser.GetModules(pom)
	if len(modules) > 0 {
		springInfo.IsMultiModule = true
		springInfo.Modules = modules

		// Find the main module (one with Spring Boot plugin)
		mainModule, mainModulePath := d.mavenParser.FindMainModule(projectPath, modules)
		if mainModule != "" {
			springInfo.MainModule = mainModule
			springInfo.MainModulePath = mainModulePath
		}
	}

	// Extract project information from parent POM
	springInfo.SpringBootVersion = d.mavenParser.GetSpringBootVersion(pom)
	springInfo.HasSpringdocDeps = d.mavenParser.HasSpringdocDependency(pom)
	springInfo.SpringdocVersion = d.mavenParser.GetSpringdocVersion(pom)
	springInfo.HasSpringdocPlugin = d.mavenParser.HasSpringdocPlugin(pom)

	// For multi-module projects, also check subproject pom files
	if springInfo.IsMultiModule && springInfo.MainModulePath != "" {
		subPom, err := d.mavenParser.Parse(springInfo.MainModulePath)
		if err == nil {
			// Merge subproject info
			if subVersion := d.mavenParser.GetSpringBootVersion(subPom); subVersion != "" {
				springInfo.SpringBootVersion = subVersion
			}
			if d.mavenParser.HasSpringdocDependency(subPom) {
				springInfo.HasSpringdocDeps = true
				if v := d.mavenParser.GetSpringdocVersion(subPom); v != "" {
					springInfo.SpringdocVersion = v
				}
			}
			if d.mavenParser.HasSpringdocPlugin(subPom) {
				springInfo.HasSpringdocPlugin = true
			}
		}
	}

	info := &extractor.ProjectInfo{
		Framework:     extractor.FrameworkSpringBoot,
		BuildTool:     BuildToolMaven,
		BuildFilePath: pomPath,
		FrameworkData: springInfo,
	}

	return info, nil
}

// detectGradleProject analyzes a Gradle project.
func (d *Detector) detectGradleProject(projectPath, gradlePath string) (*extractor.ProjectInfo, error) {
	springInfo := &Info{}

	// Check for multi-module project via settings.gradle
	settingsPath := filepath.Join(projectPath, "settings.gradle")
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		settingsPath = filepath.Join(projectPath, "settings.gradle.kts")
	}

	modules := d.gradleParser.ParseModules(settingsPath)
	if len(modules) > 0 {
		springInfo.IsMultiModule = true
		springInfo.Modules = modules

		// Find the main module (one with Spring Boot plugin)
		mainModule, mainModulePath := d.gradleParser.FindMainModule(projectPath, modules)
		if mainModule != "" {
			springInfo.MainModule = mainModule
			springInfo.MainModulePath = mainModulePath
		}
	}

	// Parse build.gradle
	build, err := d.gradleParser.Parse(gradlePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse build.gradle: %w", err)
	}

	// Extract project information
	springInfo.SpringBootVersion = d.gradleParser.GetSpringBootVersion(build)
	springInfo.HasSpringdocDeps = d.gradleParser.HasSpringdocDependency(build)
	springInfo.SpringdocVersion = d.gradleParser.GetSpringdocVersion(build)
	springInfo.HasSpringdocPlugin = d.gradleParser.HasSpringdocPlugin(build)

	// For multi-module projects, also check subproject build files
	if springInfo.IsMultiModule && springInfo.MainModulePath != "" {
		subBuild, err := d.gradleParser.Parse(springInfo.MainModulePath)
		if err == nil {
			// Merge subproject info
			if subVersion := d.gradleParser.GetSpringBootVersion(subBuild); subVersion != "" {
				springInfo.SpringBootVersion = subVersion
			}
			if d.gradleParser.HasSpringdocDependency(subBuild) {
				springInfo.HasSpringdocDeps = true
			}
			if d.gradleParser.HasSpringdocPlugin(subBuild) {
				springInfo.HasSpringdocPlugin = true
			}
		}
	}

	info := &extractor.ProjectInfo{
		Framework:     extractor.FrameworkSpringBoot,
		BuildTool:     BuildToolGradle,
		BuildFilePath: gradlePath,
		FrameworkData: springInfo,
	}

	return info, nil
}
