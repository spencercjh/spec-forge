package spring

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"

	"github.com/spencercjh/spec-forge/internal/extractor"
	"github.com/vifraa/gopom"
)

// PatchResult contains the result of a patch operation.
type PatchResult struct {
	DependencyAdded bool
	PluginAdded     bool
	BuildFilePath   string
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

// patchMaven patches a Maven project.
func (p *Patcher) patchMaven(info *extractor.ProjectInfo, opts *extractor.PatchOptions) (*PatchResult, error) {
	result := &PatchResult{
		BuildFilePath: info.BuildFilePath,
	}

	parser := NewMavenParser()
	pom, err := parser.Parse(info.BuildFilePath)
	if err != nil {
		return nil, err
	}

	// Add dependency if needed
	if opts.Force || !info.HasSpringdocDeps {
		if !parser.HasSpringdocDependency(pom) {
			parser.AddDependency(pom, SpringdocGroupID, SpringdocWebMVCArtifactID, opts.SpringdocVersion)
			result.DependencyAdded = true
		}
	}

	// Add plugin if needed
	if opts.Force || !info.HasSpringdocPlugin {
		if !parser.HasSpringdocPlugin(pom) {
			parser.AddPlugin(pom, SpringdocGroupID, SpringdocMavenPluginArtifact, opts.MavenPluginVersion)
			result.PluginAdded = true
		}
	}

	// Write changes if not dry-run
	if !opts.DryRun && (result.DependencyAdded || result.PluginAdded) {
		if err := p.writePom(info.BuildFilePath, pom); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// patchGradle patches a Gradle project.
func (p *Patcher) patchGradle(info *extractor.ProjectInfo, opts *extractor.PatchOptions) (*PatchResult, error) {
	result := &PatchResult{
		BuildFilePath: info.BuildFilePath,
	}

	// Gradle modification is more complex due to the parser limitations
	// For now, we'll use text-based modification
	parser := NewGradleParser()
	build, err := parser.Parse(info.BuildFilePath)
	if err != nil {
		return nil, err
	}

	// Check what needs to be added
	if opts.Force || !info.HasSpringdocDeps {
		if !parser.HasSpringdocDependency(build) {
			result.DependencyAdded = true
		}
	}

	if opts.Force || !info.HasSpringdocPlugin {
		if !parser.HasSpringdocPlugin(build) {
			result.PluginAdded = true
		}
	}

	// Write changes if not dry-run
	if !opts.DryRun && (result.DependencyAdded || result.PluginAdded) {
		if err := p.patchGradleFile(info.BuildFilePath, opts, result); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// writePom serializes the pom back to XML and writes to file.
func (p *Patcher) writePom(path string, pom *gopom.Project) error {
	// Set XMLName if not set to ensure proper root element
	if pom.XMLName == nil {
		pom.XMLName = &xml.Name{Space: "http://maven.apache.org/POM/4.0.0", Local: "project"}
	}

	// Read original file to preserve header and formatting
	origContent, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read original pom: %w", err)
	}

	// Marshal with indentation
	output, err := xml.MarshalIndent(pom, "    ", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal pom: %w", err)
	}

	// Construct the final content with XML header
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	sb.WriteString("\n")
	sb.WriteString(`<project xmlns="http://maven.apache.org/POM/4.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"`)
	sb.WriteString("\n")
	sb.WriteString(`         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 https://maven.apache.org/xsd/maven-4.0.0.xsd">`)
	sb.WriteString("\n")

	// Skip the <project> tag from marshaled output since we added it manually
	lines := strings.Split(string(output), "\n")
	startIndex := 0
	for i, line := range lines {
		if strings.Contains(line, "<project") {
			startIndex = i + 1
			break
		}
	}

	// Find the end of project tag in marshaled output
	endIndex := len(lines)
	for i := startIndex; i < len(lines); i++ {
		if strings.Contains(lines[i], "</project>") {
			endIndex = i
			break
		}
	}

	// Write content between project tags
	for i := startIndex; i < endIndex; i++ {
		sb.WriteString(lines[i])
		sb.WriteString("\n")
	}
	sb.WriteString("</project>\n")

	// Write to file
	if err := os.WriteFile(path, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("failed to write pom: %w", err)
	}

	_ = origContent // Preserve for potential future use
	return nil
}

// patchGradleFile modifies the build.gradle file using text manipulation.
func (p *Patcher) patchGradleFile(path string, opts *extractor.PatchOptions, result *PatchResult) error {
	content, err := readFile(path)
	if err != nil {
		return err
	}

	// Add dependency if needed
	if result.DependencyAdded {
		// Find the dependencies block and add the springdoc dependency
		content = p.addGradleDependency(content, SpringdocGroupID+":"+SpringdocWebMVCArtifactID+":"+opts.SpringdocVersion)
	}

	// Add plugin if needed
	if result.PluginAdded {
		// Find the plugins block and add the springdoc plugin
		content = p.addGradlePlugin(content, SpringdocGradlePluginID, opts.GradlePluginVersion)
	}

	return writeFile(path, []byte(content))
}

// addGradleDependency adds a dependency to the dependencies block.
func (p *Patcher) addGradleDependency(content, dep string) string {
	// Find the dependencies block
	lines := strings.Split(content, "\n")
	var result []string
	inserted := false

	for i, line := range lines {
		result = append(result, line)
		if !inserted && strings.TrimSpace(line) == "dependencies {" {
			// Find proper indentation
			indent := ""
			for _, c := range line {
				if c == ' ' || c == '\t' {
					indent += string(c)
				} else {
					break
				}
			}
			// Add dependency with proper indentation
			result = append(result, indent+"    implementation '"+dep+"'")
			inserted = true

			// Skip if there are already dependencies
			if i+1 < len(lines) && strings.Contains(lines[i+1], "implementation") {
				// Already has dependencies, our insertion is complete
			}
		}
	}

	return strings.Join(result, "\n")
}

// addGradlePlugin adds a plugin to the plugins block.
func (p *Patcher) addGradlePlugin(content, pluginID, version string) string {
	// Find the plugins block
	lines := strings.Split(content, "\n")
	var result []string
	inserted := false

	for _, line := range lines {
		result = append(result, line)
		if !inserted && strings.TrimSpace(line) == "plugins {" {
			// Find proper indentation
			indent := ""
			for _, c := range line {
				if c == ' ' || c == '\t' {
					indent += string(c)
				} else {
					break
				}
			}
			// Add plugin with proper indentation
			result = append(result, indent+"    id '"+pluginID+"' version \""+version+"\"")
			inserted = true
		}
	}

	return strings.Join(result, "\n")
}
