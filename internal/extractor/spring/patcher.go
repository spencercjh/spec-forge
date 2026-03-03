package spring

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
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

// patchMaven patches a Maven project using pure text manipulation to preserve formatting.
func (p *Patcher) patchMaven(info *extractor.ProjectInfo, opts *extractor.PatchOptions) (*PatchResult, error) {
	result := &PatchResult{
		BuildFilePath: info.BuildFilePath,
	}

	// Read original content as bytes to preserve exact formatting
	content, err := os.ReadFile(info.BuildFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read pom.xml: %w", err)
	}
	originalContent := string(content)

	// Parse with gopom just for detection (not for modification)
	parser := NewMavenParser()
	pom, err := parser.Parse(info.BuildFilePath)
	if err != nil {
		return nil, err
	}

	modified := originalContent

	// Add dependency if needed
	if opts.Force || !info.HasSpringdocDeps {
		if !parser.HasSpringdocDependency(pom) {
			modified, err = p.addMavenDependencyText(modified, opts.SpringdocVersion, pom)
			if err != nil {
				return nil, fmt.Errorf("failed to add dependency: %w", err)
			}
			result.DependencyAdded = true
		}
	}

	// Add plugin if needed
	if opts.Force || !info.HasSpringdocPlugin {
		if !parser.HasSpringdocPlugin(pom) {
			modified, err = p.addMavenPluginText(modified, opts.MavenPluginVersion, pom)
			if err != nil {
				return nil, fmt.Errorf("failed to add plugin: %w", err)
			}
			result.PluginAdded = true
		}
	}

	// Write changes if not dry-run and content actually changed
	if !opts.DryRun && modified != originalContent {
		if err := os.WriteFile(info.BuildFilePath, []byte(modified), 0644); err != nil {
			return nil, fmt.Errorf("failed to write pom.xml: %w", err)
		}
	}

	return result, nil
}

// addMavenDependencyText adds springdoc dependency using pure text manipulation.
func (p *Patcher) addMavenDependencyText(content, version string, pom *gopom.Project) (string, error) {
	// Create the dependency XML with proper indentation (4 spaces per level)
	depLines := []string{
		`        <dependency>`,
		fmt.Sprintf(`            <groupId>%s</groupId>`, SpringdocGroupID),
		fmt.Sprintf(`            <artifactId>%s</artifactId>`, SpringdocWebMVCArtifactID),
		fmt.Sprintf(`            <version>%s</version>`, version),
		`        </dependency>`,
	}
	depXML := "\n" + strings.Join(depLines, "\n")

	// Check if it's a parent POM with modules
	isParentPOM := strings.Contains(content, "<modules>")

	// For parent POMs, prefer adding to dependencyManagement
	if isParentPOM && strings.Contains(content, "<dependencyManagement>") {
		// Find <dependencyManagement><dependencies>
		pattern := regexp.MustCompile(`(?s)<dependencyManagement>\s*<dependencies>`)
		if loc := pattern.FindStringIndex(content); loc != nil {
			return content[:loc[1]] + depXML + content[loc[1]:], nil
		}
	}

	// Find regular <dependencies> section (not inside dependencyManagement)
	// We need to find the first <dependencies> that's NOT inside <dependencyManagement>
	if idx := findDependenciesSection(content); idx != -1 {
		insertPos := idx + len("<dependencies>")
		// Find end of line
		if eol := strings.Index(content[insertPos:], "\n"); eol != -1 {
			insertPos += eol + 1
		}
		return content[:insertPos] + depXML + content[insertPos:], nil
	}

	// No dependencies section - create one after </properties>, </packaging>, or </description>
	depBlock := fmt.Sprintf(`

    <dependencies>
%s
    </dependencies>`, depXML)

	// Try to find a good insertion point
	for _, tag := range []string{"</properties>", "</packaging>", "</description>"} {
		if idx := strings.LastIndex(content, tag); idx != -1 {
			// Find end of line
			eol := strings.Index(content[idx:], "\n")
			if eol != -1 {
				return content[:idx+eol+1] + depBlock + content[idx+eol+1:], nil
			}
		}
	}

	return "", fmt.Errorf("could not find suitable location to add dependency")
}

// findDependenciesSection finds the <dependencies> section that is NOT inside <dependencyManagement>.
func findDependenciesSection(content string) int {
	// Strategy: find all <dependencies> tags and return the first one that's not inside dependencyManagement
	depsPattern := regexp.MustCompile(`<dependencies>`)
	dmPattern := regexp.MustCompile(`<dependencyManagement>`)

	depsMatches := depsPattern.FindAllStringIndex(content, -1)
	dmMatches := dmPattern.FindAllStringIndex(content, -1)

	// If no dependencyManagement, return first <dependencies>
	if len(dmMatches) == 0 {
		if len(depsMatches) > 0 {
			return depsMatches[0][0]
		}
		return -1
	}

	// Find <dependencies> that's NOT inside any <dependencyManagement>...<build> block
	for _, depsMatch := range depsMatches {
		depsStart := depsMatch[0]
		insideDM := false
		for i := 0; i < len(dmMatches); i++ {
			dmStart := dmMatches[i][0]
			// Find the matching </dependencyManagement>
			dmEnd := strings.Index(content[dmStart:], "</dependencyManagement>")
			if dmEnd != -1 {
				dmEnd += dmStart + len("</dependencyManagement>")
				if depsStart > dmStart && depsStart < dmEnd {
					insideDM = true
					break
				}
			}
		}
		if !insideDM {
			return depsStart
		}
	}

	return -1
}

// addMavenPluginText adds springdoc maven plugin using pure text manipulation.
func (p *Patcher) addMavenPluginText(content, version string, pom *gopom.Project) (string, error) {
	// Create the plugin XML with proper indentation
	pluginLines := []string{
		`            <plugin>`,
		fmt.Sprintf(`                <groupId>%s</groupId>`, SpringdocGroupID),
		fmt.Sprintf(`                <artifactId>%s</artifactId>`, SpringdocMavenPluginArtifact),
		fmt.Sprintf(`                <version>%s</version>`, version),
		`                <executions>`,
		`                    <execution>`,
		`                        <goals>`,
		`                            <goal>generate</goal>`,
		`                        </goals>`,
		`                    </execution>`,
		`                </executions>`,
		`            </plugin>`,
	}
	pluginXML := "\n" + strings.Join(pluginLines, "\n")

	// Strategy 1: Find <build><plugins> that is NOT inside <pluginManagement>
	// Look for pattern: <build>...<plugins> where there's no <pluginManagement> between them
	buildStart := strings.Index(content, "<build>")
	if buildStart != -1 {
		afterBuild := content[buildStart:]

		// Check if there's a <plugins> directly under <build> (not in pluginManagement)
		// by looking for </pluginManagement> before <plugins> or no pluginManagement at all
		pmEnd := strings.Index(afterBuild, "</pluginManagement>")
		firstPlugins := strings.Index(afterBuild, "<plugins>")

		// Determine if the first <plugins> is inside or outside pluginManagement
		hasDirectPlugins := false
		var pluginsPos int

		if firstPlugins != -1 {
			if pmEnd == -1 || firstPlugins < pmEnd {
				// No pluginManagement or plugins comes before pluginManagement ends
				// This means plugins is inside pluginManagement
				// Look for plugins after pluginManagement
				if pmEnd != -1 {
					afterPM := afterBuild[pmEnd+len("</pluginManagement>"):]
					if nextPlugins := strings.Index(afterPM, "<plugins>"); nextPlugins != -1 {
						hasDirectPlugins = true
						pluginsPos = pmEnd + len("</pluginManagement>") + nextPlugins
					}
				}
			} else {
				// plugins comes after pluginManagement ends - it's a direct plugins section
				hasDirectPlugins = true
				pluginsPos = firstPlugins
			}
		}

		if hasDirectPlugins {
			insertPos := buildStart + pluginsPos + len("<plugins>")
			// Find end of line
			if eol := strings.Index(content[insertPos:], "\n"); eol != -1 {
				insertPos += eol + 1
			}
			return content[:insertPos] + pluginXML + content[insertPos:], nil
		}

		// Strategy 2: Has <build> with only <pluginManagement>, add <plugins> after </pluginManagement>
		if pmEnd != -1 {
			pluginsBlock := fmt.Sprintf(`

        <plugins>
%s
        </plugins>
`, pluginXML)

			insertPos := buildStart + pmEnd + len("</pluginManagement>")
			// Find end of line
			if eol := strings.Index(content[insertPos:], "\n"); eol != -1 {
				insertPos += eol + 1
			}
			return content[:insertPos] + pluginsBlock + content[insertPos:], nil
		}

		// Strategy 3: Has <build> but no <plugins> at all, add after <build> opening
		pluginsBlock := fmt.Sprintf(`
        <plugins>
%s
        </plugins>`, pluginXML)

		buildEndTag := strings.Index(content[buildStart:], ">")
		if buildEndTag != -1 {
			insertPos := buildStart + buildEndTag + 1
			// Find end of line
			if eol := strings.Index(content[insertPos:], "\n"); eol != -1 {
				insertPos += eol + 1
			}
			return content[:insertPos] + pluginsBlock + content[insertPos:], nil
		}
	}

	// Strategy 4: No <build> section - create one before </project>
	pluginsBlock := fmt.Sprintf(`
    <build>
        <plugins>
%s
        </plugins>
    </build>`, pluginXML)

	if projectEnd := strings.Index(content, "</project>"); projectEnd != -1 {
		return content[:projectEnd] + pluginsBlock + "\n" + content[projectEnd:], nil
	}

	return "", fmt.Errorf("could not find suitable location to add plugin")
}

// patchGradle patches a Gradle project using pure text manipulation.
func (p *Patcher) patchGradle(info *extractor.ProjectInfo, opts *extractor.PatchOptions) (*PatchResult, error) {
	result := &PatchResult{
		BuildFilePath: info.BuildFilePath,
	}

	// Read original content
	content, err := os.ReadFile(info.BuildFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read build.gradle: %w", err)
	}
	originalContent := string(content)

	// Check what needs to be added using parsed data
	parser := NewGradleParser()
	build, err := parser.Parse(info.BuildFilePath)
	if err != nil {
		return nil, err
	}

	modified := originalContent

	// Add dependency if needed
	if opts.Force || !info.HasSpringdocDeps {
		if !parser.HasSpringdocDependency(build) {
			modified = p.addGradleDependencyText(modified, opts.SpringdocVersion)
			result.DependencyAdded = true
		}
	}

	// Add plugin if needed
	if opts.Force || !info.HasSpringdocPlugin {
		if !parser.HasSpringdocPlugin(build) {
			modified = p.addGradlePluginText(modified, opts.GradlePluginVersion)
			result.PluginAdded = true
		}
	}

	// Write changes if not dry-run and content actually changed
	if !opts.DryRun && modified != originalContent {
		if err := os.WriteFile(info.BuildFilePath, []byte(modified), 0644); err != nil {
			return nil, fmt.Errorf("failed to write build.gradle: %w", err)
		}
	}

	return result, nil
}

// addGradleDependencyText adds springdoc dependency using text manipulation.
func (p *Patcher) addGradleDependencyText(content, version string) string {
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
	lineStart := bytes.LastIndexByte([]byte(content[:depsIdx]), '\n')
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
func (p *Patcher) addGradlePluginText(content, version string) string {
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
	lineStart := bytes.LastIndexByte([]byte(content[:pluginsIdx]), '\n')
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
