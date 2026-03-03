package spring

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"

	"github.com/vifraa/gopom"

	"github.com/spencercjh/spec-forge/internal/extractor"
)

// PatchResult contains the result of a patch operation.
type PatchResult struct {
	DependencyAdded bool
	PluginAdded     bool
	BuildFilePath   string
	OriginalContent string // Original file content for potential restoration
}

// Patcher modifies Spring projects to add springdoc dependencies.
type Patcher struct {
	detector *Detector
}

// NewPatcher creates a new Patcher instance.
func NewPatcher() *Patcher {
	return &Patcher{
		detector: NewDetector(),
	}
}

// NeedsPatch checks if the project needs to be patched.
func (p *Patcher) NeedsPatch(info *extractor.ProjectInfo, force bool) bool {
	if force {
		return true
	}
	return !info.HasSpringdocDeps || !info.HasSpringdocPlugin
}

// Patch adds springdoc dependencies to the project.
func (p *Patcher) Patch(projectPath string, opts *extractor.PatchOptions) (*PatchResult, error) {
	// Detect project info
	info, err := p.detector.Detect(projectPath)
	if err != nil {
		return nil, fmt.Errorf("detection failed: %w", err)
	}

	// Check if patch is needed
	if !p.NeedsPatch(info, opts.Force) {
		return &PatchResult{
			DependencyAdded: false,
			PluginAdded:     false,
			BuildFilePath:   info.BuildFilePath,
			OriginalContent: "",
		}, nil
	}

	// Apply defaults
	if opts.SpringdocVersion == "" {
		opts.SpringdocVersion = extractor.DefaultSpringdocVersion
	}
	if opts.MavenPluginVersion == "" {
		opts.MavenPluginVersion = extractor.DefaultSpringdocMavenPlugin
	}
	if opts.GradlePluginVersion == "" {
		opts.GradlePluginVersion = extractor.DefaultSpringdocGradlePlugin
	}

	// Patch based on build tool
	switch info.BuildTool {
	case extractor.BuildToolMaven:
		return p.patchMaven(info, opts)
	case extractor.BuildToolGradle:
		return p.patchGradle(info, opts)
	default:
		return nil, fmt.Errorf("unsupported build tool: %s", info.BuildTool)
	}
}

// Restore restores the original file content.
func (p *Patcher) Restore(buildFilePath, originalContent string) error {
	if originalContent == "" {
		return nil
	}
	return os.WriteFile(buildFilePath, []byte(originalContent), 0644)
}

// patchMaven patches a Maven project using gopom.
func (p *Patcher) patchMaven(info *extractor.ProjectInfo, opts *extractor.PatchOptions) (*PatchResult, error) {
	// For multi-module projects, patch the main module instead of parent POM
	buildFilePath := info.BuildFilePath
	if info.IsMultiModule && info.MainModulePath != "" {
		buildFilePath = info.MainModulePath
	}

	result := &PatchResult{
		BuildFilePath: buildFilePath,
	}

	// Read and save original content
	content, err := os.ReadFile(buildFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read pom.xml: %w", err)
	}
	result.OriginalContent = string(content)

	if opts.DryRun {
		// In dry-run mode, don't modify anything
		parser := NewMavenParser()
		pom, err := parser.Parse(buildFilePath)
		if err != nil {
			return nil, err
		}
		result.DependencyAdded = !parser.HasSpringdocDependency(pom)
		result.PluginAdded = !parser.HasSpringdocPlugin(pom)
		return result, nil
	}

	// Parse with gopom
	pom, err := gopom.Parse(buildFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pom.xml: %w", err)
	}

	// Add dependency if needed
	if opts.Force || !info.HasSpringdocDeps {
		if !hasSpringdocDependency(pom) {
			addMavenDependency(pom, opts.SpringdocVersion)
			result.DependencyAdded = true
		}
	}

	// Add plugin if needed
	if opts.Force || !info.HasSpringdocPlugin {
		if !hasSpringdocPlugin(pom) {
			addMavenPlugin(pom, opts.MavenPluginVersion)
			result.PluginAdded = true
		}
	}

	// Write changes if any modifications were made
	if result.DependencyAdded || result.PluginAdded {
		output, err := xml.MarshalIndent(pom, "  ", "    ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal pom.xml: %w", err)
		}

		// Add XML header and newline
		xmlContent := xml.Header + string(output) + "\n"
		if err := os.WriteFile(buildFilePath, []byte(xmlContent), 0644); err != nil {
			return nil, fmt.Errorf("failed to write pom.xml: %w", err)
		}
	}

	return result, nil
}

// hasSpringdocDependency checks if springdoc dependency exists.
func hasSpringdocDependency(pom *gopom.Project) bool {
	if pom.Dependencies == nil {
		return false
	}
	for _, dep := range *pom.Dependencies {
		if dep.GroupID != nil && *dep.GroupID == SpringdocGroupID {
			return true
		}
	}
	return false
}

// addMavenDependency adds springdoc dependency to pom.
func addMavenDependency(pom *gopom.Project, version string) {
	dep := gopom.Dependency{
		GroupID:    new(SpringdocGroupID),
		ArtifactID: new(SpringdocWebMVCArtifactID),
		Version:    new(version),
	}

	if pom.Dependencies == nil {
		pom.Dependencies = &[]gopom.Dependency{dep}
	} else {
		*pom.Dependencies = append(*pom.Dependencies, dep)
	}
}

// hasSpringdocPlugin checks if springdoc plugin exists.
func hasSpringdocPlugin(pom *gopom.Project) bool {
	if pom.Build == nil || pom.Build.Plugins == nil {
		return false
	}
	for _, plugin := range *pom.Build.Plugins {
		if plugin.GroupID != nil && *plugin.GroupID == SpringdocGroupID {
			return true
		}
	}
	return false
}

// addMavenPlugin adds springdoc maven plugin to pom.
func addMavenPlugin(pom *gopom.Project, version string) {
	plugin := gopom.Plugin{
		GroupID:    new(SpringdocGroupID),
		ArtifactID: new(SpringdocMavenPluginArtifact),
		Version:    new(version),
		Executions: &[]gopom.PluginExecution{
			{
				ID:    new("generate-openapi"),
				Goals: &[]string{"generate"},
			},
		},
	}

	// Ensure Build section exists
	if pom.Build == nil {
		pom.Build = &gopom.Build{}
	}

	// Ensure Plugins section exists
	if pom.Build.Plugins == nil {
		pom.Build.Plugins = &[]gopom.Plugin{plugin}
	} else {
		*pom.Build.Plugins = append(*pom.Build.Plugins, plugin)
	}
}

// patchGradle patches a Gradle project.
func (p *Patcher) patchGradle(info *extractor.ProjectInfo, opts *extractor.PatchOptions) (*PatchResult, error) {
	// For multi-module projects, patch the main module instead of root
	buildFilePath := info.BuildFilePath
	if info.IsMultiModule && info.MainModulePath != "" {
		buildFilePath = info.MainModulePath
	}

	result := &PatchResult{
		BuildFilePath: buildFilePath,
	}

	// Read and save original content
	content, err := os.ReadFile(buildFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read build.gradle: %w", err)
	}
	result.OriginalContent = string(content)

	if opts.DryRun {
		// In dry-run mode, don't modify anything
		parser := NewGradleParser()
		build, err := parser.Parse(buildFilePath)
		if err != nil {
			return nil, err
		}
		result.DependencyAdded = !parser.HasSpringdocDependency(build)
		result.PluginAdded = !parser.HasSpringdocPlugin(build)
		return result, nil
	}

	// For Gradle, we use text manipulation since there's no reliable Gradle parser
	modified := result.OriginalContent

	// Add dependency if needed
	if opts.Force || !info.HasSpringdocDeps {
		parser := NewGradleParser()
		build, err := parser.Parse(buildFilePath)
		if err != nil {
			return nil, err
		}
		if !parser.HasSpringdocDependency(build) {
			newContent := addGradleDependencyText(modified, opts.SpringdocVersion)
			if newContent != modified {
				modified = newContent
				result.DependencyAdded = true
			}
		}
	}

	// Add plugin if needed
	if opts.Force || !info.HasSpringdocPlugin {
		parser := NewGradleParser()
		build, err := parser.Parse(buildFilePath)
		if err != nil {
			return nil, err
		}
		if !parser.HasSpringdocPlugin(build) {
			newContent := addGradlePluginText(modified, opts.GradlePluginVersion)
			if newContent != modified {
				modified = newContent
				result.PluginAdded = true
			}
		}
	}

	// Write changes if modified
	if result.DependencyAdded || result.PluginAdded {
		if err := os.WriteFile(buildFilePath, []byte(modified), 0644); err != nil {
			return nil, fmt.Errorf("failed to write build.gradle: %w", err)
		}
	}

	return result, nil
}

// addGradleDependencyText adds springdoc dependency using text manipulation.
func addGradleDependencyText(content, version string) string {
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
	lineStart := lastIndexByte(content[:depsIdx], '\n')
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

// addGradlePluginText adds springdoc plugin using text manipulation.
func addGradlePluginText(content, version string) string {
	plugin := fmt.Sprintf("id '%s' version \"%s\"", SpringdocGradlePluginID, version)

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
	lineStart := lastIndexByte(content[:pluginsIdx], '\n')
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

// lastIndexByte finds the last occurrence of c in s.
func lastIndexByte(s string, c byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			return i
		}
	}
	return -1
}
