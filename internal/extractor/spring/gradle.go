package spring

import (
	"fmt"
	"strings"

	"github.com/scagogogo/gradle-parser/pkg/api"
	"github.com/scagogogo/gradle-parser/pkg/model"
)

// Gradle parser constants.
const (
	SpringdocGradlePluginID = "org.springdoc.openapi-gradle-plugin"
)

// GradleParser parses and modifies Gradle build.gradle files.
type GradleParser struct{}

// NewGradleParser creates a new GradleParser instance.
func NewGradleParser() *GradleParser {
	return &GradleParser{}
}

// Parse reads and parses a build.gradle file.
func (p *GradleParser) Parse(gradlePath string) (*model.Project, error) {
	result, err := api.ParseFile(gradlePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse build.gradle: %w", err)
	}

	if result.Project == nil {
		return nil, fmt.Errorf("parsed project is nil")
	}

	return result.Project, nil
}

// ParseString parses a build.gradle content string.
func (p *GradleParser) ParseString(content string) (*model.Project, error) {
	result, err := api.ParseString(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse build.gradle: %w", err)
	}

	if result.Project == nil {
		return nil, fmt.Errorf("parsed project is nil")
	}

	return result.Project, nil
}

// GetSpringBootVersion extracts the Spring Boot version from build.gradle.
func (p *GradleParser) GetSpringBootVersion(project *model.Project) string {
	// Check plugins block for spring-boot
	for _, plugin := range project.Plugins {
		if plugin.ID == "org.springframework.boot" || plugin.ID == "spring-boot" {
			if plugin.Version != "" {
				return plugin.Version
			}
		}
	}

	// Check dependencies for spring-boot-starter
	for _, dep := range project.Dependencies {
		if strings.Contains(dep.Name, "spring-boot-starter") {
			if dep.Version != "" {
				return dep.Version
			}
		}
	}

	return ""
}

// HasSpringdocDependency checks if build.gradle has springdoc dependencies.
func (p *GradleParser) HasSpringdocDependency(project *model.Project) bool {
	for _, dep := range project.Dependencies {
		if strings.Contains(dep.Name, "springdoc-openapi") {
			return true
		}
	}
	return false
}

// GetSpringdocVersion returns the springdoc version if present.
func (p *GradleParser) GetSpringdocVersion(project *model.Project) string {
	for _, dep := range project.Dependencies {
		if strings.Contains(dep.Name, "springdoc-openapi-starter-webmvc-ui") {
			return dep.Version
		}
	}
	return ""
}

// HasSpringdocPlugin checks if build.gradle has the springdoc gradle plugin.
func (p *GradleParser) HasSpringdocPlugin(project *model.Project) bool {
	for _, plugin := range project.Plugins {
		if plugin.ID == SpringdocGradlePluginID {
			return true
		}
	}
	return false
}

// FindDependency finds a dependency by name pattern.
func (p *GradleParser) FindDependency(project *model.Project, namePattern string) *model.Dependency {
	for i := range project.Dependencies {
		if strings.Contains(project.Dependencies[i].Name, namePattern) {
			return project.Dependencies[i]
		}
	}
	return nil
}

// FindPlugin finds a plugin by ID.
func (p *GradleParser) FindPlugin(project *model.Project, pluginID string) *model.Plugin {
	for i := range project.Plugins {
		if project.Plugins[i].ID == pluginID {
			return project.Plugins[i]
		}
	}
	return nil
}

// HasSpringBootPlugin checks if build.gradle has the Spring Boot plugin.
func (p *GradleParser) HasSpringBootPlugin(project *model.Project) bool {
	for _, plugin := range project.Plugins {
		if plugin.ID == "org.springframework.boot" || plugin.ID == "spring-boot" {
			return true
		}
	}
	return false
}
