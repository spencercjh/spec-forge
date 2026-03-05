package spring

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/vifraa/gopom"

	"github.com/spencercjh/spec-forge/internal/extractor"
)

// PatchResult contains the result of a patch operation.
type PatchResult struct {
	DependencyAdded      bool
	PluginAdded          bool
	BuildFilePath        string
	OriginalContent      string // Original file content for potential restoration
	SpringBootConfigured bool   // Whether spring-boot-maven-plugin was configured with start/stop goals
}

// Patcher modifies Spring projects to add springdoc dependencies.
type Patcher struct {
	detector     *Detector
	mavenParser  *MavenParser
	gradleParser *GradleParser
}

// NewPatcher creates a new Patcher instance.
func NewPatcher() *Patcher {
	return &Patcher{
		detector:     NewDetector(),
		mavenParser:  NewMavenParser(),
		gradleParser: NewGradleParser(),
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

	// Apply defaults
	if opts.SpringdocVersion == "" {
		opts.SpringdocVersion = DefaultSpringdocVersion
	}
	if opts.MavenPluginVersion == "" {
		opts.MavenPluginVersion = DefaultSpringdocMavenPlugin
	}
	if opts.GradlePluginVersion == "" {
		opts.GradlePluginVersion = DefaultSpringdocGradlePlugin
	}

	// For Maven projects, we always need to check spring-boot-maven-plugin configuration
	// even if springdoc is already configured, because start/stop goals might be missing
	if info.BuildTool == BuildToolMaven {
		return p.patchMaven(info, opts)
	}

	// Check if patch is needed for other build tools
	if !p.NeedsPatch(info, opts.Force) {
		return &PatchResult{
			DependencyAdded: false,
			PluginAdded:     false,
			BuildFilePath:   info.BuildFilePath,
			OriginalContent: "",
		}, nil
	}

	// Patch based on build tool
	switch info.BuildTool {
	case BuildToolGradle:
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
	//nolint:gosec // G306: 0644 is appropriate for build files (pom.xml, build.gradle)
	return os.WriteFile(buildFilePath, []byte(originalContent), 0o644)
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
		pom, dryRunErr := p.mavenParser.Parse(buildFilePath)
		if dryRunErr != nil {
			return nil, dryRunErr
		}
		result.DependencyAdded = !p.mavenParser.HasSpringdocDependency(pom)
		result.PluginAdded = !p.mavenParser.HasSpringdocPlugin(pom)
		result.SpringBootConfigured = !p.mavenParser.HasSpringBootStartStopGoals(pom)
		return result, nil
	}

	// Parse with gopom
	pom, err := gopom.Parse(buildFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pom.xml: %w", err)
	}

	// Add dependency if needed
	if opts.Force || !info.HasSpringdocDeps {
		if !p.mavenParser.HasDependency(pom) {
			p.mavenParser.AddDependency(pom, SpringdocGroupID, SpringdocWebMVCArtifactID, opts.SpringdocVersion)
			result.DependencyAdded = true
		}
	}

	// Add plugin if needed
	if opts.Force || !info.HasSpringdocPlugin {
		if !p.mavenParser.HasPlugin(pom) {
			p.mavenParser.AddPlugin(pom, SpringdocGroupID, SpringdocMavenPluginArtifact, opts.MavenPluginVersion)
			result.PluginAdded = true
		}
	}

	// Always check spring-boot-maven-plugin configuration
	// This is required for springdoc plugin to work during integration-test phase
	// We do this even if springdoc plugin already exists, because start/stop might be missing
	if p.mavenParser.ConfigureSpringBootPlugin(pom) {
		result.SpringBootConfigured = true
		slog.Debug("Configured spring-boot-maven-plugin with start/stop goals", "status", "configured")
	}

	// Write changes if any modifications were made
	if result.DependencyAdded || result.PluginAdded || result.SpringBootConfigured {
		output, err := p.mavenParser.MarshalPom(pom)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal pom.xml: %w", err)
		}
		//nolint:gosec // 0644 is appropriate for build files (pom.xml)
		if err := os.WriteFile(buildFilePath, output, 0o644); err != nil {
			return nil, fmt.Errorf("failed to write pom.xml: %w", err)
		}
	}

	return result, nil
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
		build, err := p.gradleParser.Parse(buildFilePath)
		if err != nil {
			return nil, err
		}
		result.DependencyAdded = !p.gradleParser.HasSpringdocDependency(build)
		result.PluginAdded = !p.gradleParser.HasSpringdocPlugin(build)
		return result, nil
	}

	// For Gradle, we use text manipulation since there's no reliable Gradle parser
	modified := result.OriginalContent

	// Add dependency if needed
	if opts.Force || !info.HasSpringdocDeps {
		build, err := p.gradleParser.Parse(buildFilePath)
		if err != nil {
			return nil, err
		}
		if !p.gradleParser.HasSpringdocDependency(build) {
			newContent := p.gradleParser.AddDependencyText(modified, opts.SpringdocVersion)
			if newContent != modified {
				modified = newContent
				result.DependencyAdded = true
			}
		}
	}

	// Add plugin if needed
	if opts.Force || !info.HasSpringdocPlugin {
		build, err := p.gradleParser.Parse(buildFilePath)
		if err != nil {
			return nil, err
		}
		if !p.gradleParser.HasSpringdocPlugin(build) {
			newContent := p.gradleParser.AddPluginText(modified, opts.GradlePluginVersion)
			if newContent != modified {
				modified = newContent
				result.PluginAdded = true
			}
		}
	}

	// Write changes if modified
	if result.DependencyAdded || result.PluginAdded {
		//nolint:gosec // 0644 is appropriate for build files (build.gradle)
		if err := os.WriteFile(buildFilePath, []byte(modified), 0o644); err != nil {
			return nil, fmt.Errorf("failed to write build.gradle: %w", err)
		}
	}

	return result, nil
}
