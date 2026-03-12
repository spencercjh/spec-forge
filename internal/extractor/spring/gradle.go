package spring

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/scagogogo/gradle-parser/pkg/api"
	"github.com/scagogogo/gradle-parser/pkg/model"
)

// GradleParser parses and modifies Gradle build.gradle files.
type GradleParser struct{}

// newGradleParser creates a new GradleParser instance.
func newGradleParser() *GradleParser {
	return &GradleParser{}
}

// Parse reads and parses a build.gradle file.
func (p *GradleParser) Parse(gradlePath string) (*model.Project, error) {
	result, err := api.ParseFile(gradlePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse build.gradle: %w", err)
	}

	if result.Project == nil {
		return nil, errors.New("parsed project is nil")
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
		return nil, errors.New("parsed project is nil")
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

// HasSpringBootPlugin checks if build.gradle has the Spring Boot plugin.
func (p *GradleParser) HasSpringBootPlugin(project *model.Project) bool {
	for _, plugin := range project.Plugins {
		if plugin.ID == "org.springframework.boot" || plugin.ID == "spring-boot" {
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

// ParseModules parses settings.gradle to find included modules.
func (p *GradleParser) ParseModules(settingsPath string) []string {
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
		if after, ok := strings.CutPrefix(line, "include"); ok {
			line = strings.TrimSpace(after)
			// Split by comma and clean up
			parts := strings.SplitSeq(line, ",")
			for part := range parts {
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

// FindMainModule finds the main module that contains the Spring Boot application.
// Returns (moduleName, moduleBuildPath). Returns empty strings if not found.
func (p *GradleParser) FindMainModule(projectPath string, modules []string) (moduleName, moduleBuildPath string) {
	for _, module := range modules {
		// Try build.gradle first
		modulePath := filepath.Join(projectPath, module, "build.gradle")
		if _, err := os.Stat(modulePath); err != nil {
			// Try build.gradle.kts
			modulePath = filepath.Join(projectPath, module, "build.gradle.kts")
			if _, err := os.Stat(modulePath); err != nil {
				continue
			}
		}

		build, err := p.Parse(modulePath)
		if err != nil {
			continue
		}

		// Check if this module has Spring Boot plugin
		if p.HasSpringBootPlugin(build) {
			moduleName = module
			moduleBuildPath = modulePath
			return moduleName, moduleBuildPath
		}
	}

	return moduleName, moduleBuildPath
}

// AddDependencyText adds springdoc dependency using text manipulation.
func (p *GradleParser) AddDependencyText(content, version string) string {
	dep := fmt.Sprintf("implementation '%s:%s:%s'", SpringdocGroupID, SpringdocWebMVCArtifactID, version)

	// Find the dependencies block
	depsIdx := strings.Index(content, "dependencies {")
	if depsIdx == -1 {
		depsIdx = strings.Index(content, "dependencies{")
	}
	if depsIdx == -1 {
		return content
	}

	// Find the end of the line
	lineEnd := strings.Index(content[depsIdx:], "\n")
	if lineEnd == -1 {
		return content
	}

	// Get the indentation of the "dependencies" line
	lineStart := lastIndexOf(content[:depsIdx], '\n')
	if lineStart == -1 {
		lineStart = 0
	} else {
		lineStart++ // Move past the newline
	}
	indent := content[lineStart:depsIdx]

	// Insert the dependency
	insertPos := depsIdx + lineEnd + 1
	return content[:insertPos] + indent + "    " + dep + "\n" + content[insertPos:]
}

// AddPluginText adds springdoc plugin using text manipulation.
func (p *GradleParser) AddPluginText(content, version string) string {
	plugin := fmt.Sprintf("id %q version %q", SpringdocGradlePluginID, version)

	// Find the plugins block
	pluginsIdx := strings.Index(content, "plugins {")
	if pluginsIdx == -1 {
		pluginsIdx = strings.Index(content, "plugins{")
	}
	if pluginsIdx == -1 {
		return content
	}

	// Find the end of the line
	lineEnd := strings.Index(content[pluginsIdx:], "\n")
	if lineEnd == -1 {
		return content
	}

	// Get the indentation of the "plugins" line
	lineStart := lastIndexOf(content[:pluginsIdx], '\n')
	if lineStart == -1 {
		lineStart = 0
	} else {
		lineStart++
	}
	indent := content[lineStart:pluginsIdx]

	// Insert the plugin
	insertPos := pluginsIdx + lineEnd + 1
	return content[:insertPos] + indent + "    " + plugin + "\n" + content[insertPos:]
}

// lastIndexOf finds the last occurrence of a byte in a string.
func lastIndexOf(s string, c byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			return i
		}
	}
	return -1
}
