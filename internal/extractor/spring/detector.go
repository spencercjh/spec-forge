// Package spring provides Spring framework specific extraction functionality.
package spring

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

	// Check for multi-module project
	modules := mavenParser.GetModules(pom)
	if len(modules) > 0 {
		info.IsMultiModule = true
		info.Modules = modules
		// For multi-module Maven, the parent POM is the build file
		// but we may need to find the main module
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
func (d *Detector) detectGradleProject(projectPath, gradlePath string) (*extractor.ProjectInfo, error) {
	info := &extractor.ProjectInfo{
		BuildTool:     extractor.BuildToolGradle,
		BuildFilePath: gradlePath,
	}

	// Check for multi-module project via settings.gradle
	settingsPath := filepath.Join(projectPath, "settings.gradle")
	modules := d.parseGradleModules(settingsPath)
	if len(modules) > 0 {
		info.IsMultiModule = true
		info.Modules = modules

		// For multi-module projects, find the main module (one with Spring Boot)
		mainModule, mainModulePath := d.findMainGradleModule(projectPath, modules)
		if mainModule != "" {
			info.MainModule = mainModule
			info.MainModulePath = mainModulePath
		}
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

	// For multi-module projects, also check subproject build files
	if info.IsMultiModule && info.MainModulePath != "" {
		subBuild, err := gradleParser.Parse(info.MainModulePath)
		if err == nil {
			// If subproject has Spring Boot, use its info
			if subVersion := gradleParser.GetSpringBootVersion(subBuild); subVersion != "" {
				info.SpringBootVersion = subVersion
			}
			if gradleParser.HasSpringdocDependency(subBuild) {
				info.HasSpringdocDeps = true
			}
			if gradleParser.HasSpringdocPlugin(subBuild) {
				info.HasSpringdocPlugin = true
			}
		}
	}

	return info, nil
}

// parseGradleModules parses settings.gradle to find included modules.
func (d *Detector) parseGradleModules(settingsPath string) []string {
	file, err := os.Open(settingsPath)
	if err != nil {
		return nil
	}
	defer file.Close()

	var modules []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") {
			continue
		}
		// Parse include statements
		// Formats: include 'module1', 'module2' or include 'module1', "module2"
		if strings.HasPrefix(line, "include") {
			// Extract module names
			line = strings.TrimPrefix(line, "include")
			line = strings.TrimSpace(line)
			// Split by comma and clean up
			parts := strings.Split(line, ",")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				// Remove quotes
				part = strings.Trim(part, "'\"")
				if part != "" {
					modules = append(modules, part)
				}
			}
		}
	}

	return modules
}

// findMainGradleModule finds the main module that contains the Spring Boot application.
func (d *Detector) findMainGradleModule(projectPath string, modules []string) (string, string) {
	gradleParser := NewGradleParser()

	for _, module := range modules {
		modulePath := filepath.Join(projectPath, module, "build.gradle")
		if _, err := os.Stat(modulePath); err != nil {
			// Try build.gradle.kts
			modulePath = filepath.Join(projectPath, module, "build.gradle.kts")
			if _, err := os.Stat(modulePath); err != nil {
				continue
			}
		}

		build, err := gradleParser.Parse(modulePath)
		if err != nil {
			continue
		}

		// Check if this module has Spring Boot plugin
		if gradleParser.HasSpringBootPlugin(build) {
			return module, modulePath
		}
	}

	return "", ""
}
