# M2: Spring Detection and Patch Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement Spring project detection and patching functionality for Maven and Gradle projects.

**Architecture:** Layered architecture under `internal/extractor/spring/` with detector, patcher, maven, and gradle modules. Detector identifies project info, Patcher modifies build files using maven/gradle parsers.

**Tech Stack:** Go 1.26, `vifraa/gopom` for Maven, `scagogogo/gradle-parser` for Gradle

---

## Task 1: Add Dependencies

**Files:**
- Modify: `go.mod`

**Step 1: Add required dependencies**

Run:
```bash
go get github.com/vifraa/gopom@latest
go get github.com/scagogogo/gradle-parser@latest
```

**Step 2: Verify dependencies**

Run: `cat go.mod | grep -E "gopom|gradle-parser"`
Expected: Both dependencies listed

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add gopom and gradle-parser dependencies"
```

---

## Task 2: Create Types and Interfaces

**Files:**
- Create: `internal/extractor/types.go`
- Create: `internal/extractor/types_test.go`

**Step 1: Write the failing test**

```go
// Package extractor_test tests the extractor types and interfaces.
package extractor_test

import (
	"testing"

	"github.com/spencercjh/spec-forge/internal/extractor"
)

func TestBuildToolConstants(t *testing.T) {
	tests := []struct {
		name     string
		tool     extractor.BuildTool
		expected string
	}{
		{"maven", extractor.BuildToolMaven, "maven"},
		{"gradle", extractor.BuildToolGradle, "gradle"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.tool) != tt.expected {
				t.Errorf("BuildTool %s = %s, want %s", tt.name, tt.tool, tt.expected)
			}
		})
	}
}

func TestDefaultVersions(t *testing.T) {
	if extractor.DefaultSpringdocVersion == "" {
		t.Error("DefaultSpringdocVersion should not be empty")
	}
	if extractor.DefaultSpringdocMavenPlugin == "" {
		t.Error("DefaultSpringdocMavenPlugin should not be empty")
	}
	if extractor.DefaultSpringdocGradlePlugin == "" {
		t.Error("DefaultSpringdocGradlePlugin should not be empty")
	}
}

func TestProjectInfoDefaults(t *testing.T) {
	info := extractor.ProjectInfo{}
	if info.BuildTool != "" {
		t.Error("BuildTool should default to empty")
	}
	if info.HasSpringdocDeps {
		t.Error("HasSpringdocDeps should default to false")
	}
}

func TestPatchOptionsDefaults(t *testing.T) {
	opts := extractor.PatchOptions{}
	if opts.DryRun {
		t.Error("DryRun should default to false")
	}
	if opts.Force {
		t.Error("Force should default to false")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/extractor/... -v`
Expected: FAIL with "package not found"

**Step 3: Create the types file**

```go
// Package extractor provides interfaces and types for extracting OpenAPI specs from projects.
package extractor

// BuildTool represents the build tool type for a project.
type BuildTool string

const (
	// BuildToolMaven represents Maven build tool.
	BuildToolMaven BuildTool = "maven"
	// BuildToolGradle represents Gradle build tool.
	BuildToolGradle BuildTool = "gradle"
)

// Default version constants (convention over configuration).
const (
	DefaultSpringdocVersion       = "3.0.2"
	DefaultSpringdocMavenPlugin   = "1.5"
	DefaultSpringdocGradlePlugin  = "1.9.0"
)

// ProjectInfo contains detected information about a Spring project.
type ProjectInfo struct {
	BuildTool          BuildTool // Maven or Gradle
	BuildFilePath      string    // pom.xml or build.gradle path
	SpringBootVersion  string    // Spring Boot version
	HasSpringdocDeps   bool      // Whether springdoc dependencies exist
	HasSpringdocPlugin bool      // Whether springdoc plugin is configured
	SpringdocVersion   string    // Existing springdoc version if any
}

// PatchOptions configures the patch behavior.
type PatchOptions struct {
	DryRun              bool   // Only print changes, don't write
	Force               bool   // Force overwrite existing dependencies
	SpringdocVersion    string // springdoc version (default: built-in)
	MavenPluginVersion  string // Maven plugin version (default: built-in)
	GradlePluginVersion string // Gradle plugin version (default: built-in)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/extractor/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/extractor/types.go internal/extractor/types_test.go
git commit -m "feat(extractor): add types and constants for project detection"
```

---

## Task 3: Create Detector Interface and Implementation

**Files:**
- Create: `internal/extractor/spring/detector.go`
- Create: `internal/extractor/spring/detector_test.go`

**Step 1: Write the failing test for detector**

```go
// Package spring_test tests the Spring extractor implementation.
package spring_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spencercjh/spec-forge/internal/extractor"
	"github.com/spencercjh/spec-forge/internal/extractor/spring"
)

func TestDetector_Detect_NoBuildFile(t *testing.T) {
	// Create temp dir without build files
	tmpDir := t.TempDir()

	detector := spring.NewDetector()
	_, err := detector.Detect(tmpDir)

	if err == nil {
		t.Error("Expected error when no build file found")
	}
}

func TestDetector_Detect_MavenProject(t *testing.T) {
	// Use the integration test project
	projectPath := "../../../integration-tests/maven-springboot-openapi-demo"

	// Skip if project doesn't exist (CI environment)
	if _, err := os.Stat(filepath.Join(projectPath, "pom.xml")); os.IsNotExist(err) {
		t.Skip("Integration test project not found")
	}

	detector := spring.NewDetector()
	info, err := detector.Detect(projectPath)

	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if info.BuildTool != extractor.BuildToolMaven {
		t.Errorf("BuildTool = %s, want %s", info.BuildTool, extractor.BuildToolMaven)
	}

	if info.BuildFilePath == "" {
		t.Error("BuildFilePath should not be empty")
	}

	if !info.HasSpringdocDeps {
		t.Error("HasSpringdocDeps should be true for this project")
	}
}

func TestDetector_Detect_GradleProject(t *testing.T) {
	// Use the integration test project
	projectPath := "../../../integration-tests/gradle-springboot-openapi-demo"

	// Skip if project doesn't exist (CI environment)
	if _, err := os.Stat(filepath.Join(projectPath, "build.gradle")); os.IsNotExist(err) {
		t.Skip("Integration test project not found")
	}

	detector := spring.NewDetector()
	info, err := detector.Detect(projectPath)

	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if info.BuildTool != extractor.BuildToolGradle {
		t.Errorf("BuildTool = %s, want %s", info.BuildTool, extractor.BuildToolGradle)
	}

	if info.BuildFilePath == "" {
		t.Error("BuildFilePath should not be empty")
	}

	if !info.HasSpringdocDeps {
		t.Error("HasSpringdocDeps should be true for this project")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/extractor/spring/... -v`
Expected: FAIL with "package not found"

**Step 3: Create the detector implementation**

```go
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
func (d *Detector) detectMavenProject(projectPath, pomPath string) (*extractor.ProjectInfo, error) {
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
func (d *Detector) detectGradleProject(projectPath, gradlePath string) (*extractor.ProjectInfo, error) {
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
```

**Step 4: Run test to verify it fails (parser not implemented)**

Run: `go test ./internal/extractor/spring/... -v`
Expected: FAIL with "undefined: NewMavenParser" or similar

**Step 5: Commit detector (will fix in next task)**

```bash
git add internal/extractor/spring/detector.go internal/extractor/spring/detector_test.go
git commit -m "feat(spring): add detector skeleton for Spring project detection"
```

---

## Task 4: Implement Maven Parser

**Files:**
- Create: `internal/extractor/spring/maven.go`
- Create: `internal/extractor/spring/maven_test.go`

**Step 1: Write the failing test for Maven parser**

```go
// Package spring_test tests the Spring extractor implementation.
package spring_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spencercjh/spec-forge/internal/extractor/spring"
	"github.com/vifraa/gopom"
)

func TestMavenParser_Parse(t *testing.T) {
	pomPath := "../../../integration-tests/maven-springboot-openapi-demo/pom.xml"

	// Skip if project doesn't exist
	if _, err := os.Stat(pomPath); os.IsNotExist(err) {
		t.Skip("Integration test project not found")
	}

	parser := spring.NewMavenParser()
	pom, err := parser.Parse(pomPath)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if pom == nil {
		t.Fatal("Parsed pom should not be nil")
	}
}

func TestMavenParser_GetSpringBootVersion(t *testing.T) {
	pomPath := "../../../integration-tests/maven-springboot-openapi-demo/pom.xml"

	if _, err := os.Stat(pomPath); os.IsNotExist(err) {
		t.Skip("Integration test project not found")
	}

	parser := spring.NewMavenParser()
	pom, err := parser.Parse(pomPath)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	version := parser.GetSpringBootVersion(pom)
	if version == "" {
		t.Error("Spring Boot version should not be empty")
	}
	// The demo project uses Spring Boot 4.0.3
	if version != "4.0.3" {
		t.Logf("Warning: Expected Spring Boot 4.0.3, got %s", version)
	}
}

func TestMavenParser_HasSpringdocDependency(t *testing.T) {
	pomPath := "../../../integration-tests/maven-springboot-openapi-demo/pom.xml"

	if _, err := os.Stat(pomPath); os.IsNotExist(err) {
		t.Skip("Integration test project not found")
	}

	parser := spring.NewMavenParser()
	pom, err := parser.Parse(pomPath)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !parser.HasSpringdocDependency(pom) {
		t.Error("HasSpringdocDependency should be true for demo project")
	}
}

func TestMavenParser_GetSpringdocVersion(t *testing.T) {
	pomPath := "../../../integration-tests/maven-springboot-openapi-demo/pom.xml"

	if _, err := os.Stat(pomPath); os.IsNotExist(err) {
		t.Skip("Integration test project not found")
	}

	parser := spring.NewMavenParser()
	pom, err := parser.Parse(pomPath)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	version := parser.GetSpringdocVersion(pom)
	if version == "" {
		t.Error("Springdoc version should not be empty for demo project")
	}
}

func TestMavenParser_HasSpringdocPlugin(t *testing.T) {
	pomPath := "../../../integration-tests/maven-springboot-openapi-demo/pom.xml"

	if _, err := os.Stat(pomPath); os.IsNotExist(err) {
		t.Skip("Integration test project not found")
	}

	parser := spring.NewMavenParser()
	pom, err := parser.Parse(pomPath)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !parser.HasSpringdocPlugin(pom) {
		t.Error("HasSpringdocPlugin should be true for demo project")
	}
}

func TestMavenParser_FindDependency(t *testing.T) {
	parser := spring.NewMavenParser()

	// Create a minimal pom for testing
	pom := &gopom.Model{
		Dependencies: &gopom.Dependencies{
			Dependencies: []gopom.Dependency{
				{
					GroupId:    ptr("org.springframework.boot"),
					ArtifactId: ptr("spring-boot-starter-web"),
					Version:    ptr("4.0.3"),
				},
			},
		},
	}

	dep := parser.FindDependency(pom, "org.springframework.boot", "spring-boot-starter-web")
	if dep == nil {
		t.Fatal("Should find spring-boot-starter-web dependency")
	}
	if *dep.Version != "4.0.3" {
		t.Errorf("Version = %s, want 4.0.3", *dep.Version)
	}
}

// ptr is a helper to create string pointers.
func ptr(s string) *string {
	return &s
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/extractor/spring/... -v -run MavenParser`
Expected: FAIL with "undefined: NewMavenParser"

**Step 3: Implement Maven parser**

```go
package spring

import (
	"fmt"
	"os"
	"strings"

	"github.com/vifraa/gopom"
)

// Springdoc group and artifact constants.
const (
	SpringdocGroupID             = "org.springdoc"
	SpringdocWebMVCArtifactID    = "springdoc-openapi-starter-webmvc-ui"
	SpringdocMavenPluginArtifact = "springdoc-openapi-maven-plugin"
	SpringBootParentGroupID      = "org.springframework.boot"
	SpringBootParentArtifactID   = "spring-boot-starter-parent"
)

// MavenParser parses and modifies Maven pom.xml files.
type MavenParser struct{}

// NewMavenParser creates a new MavenParser instance.
func NewMavenParser() *MavenParser {
	return &MavenParser{}
}

// Parse reads and parses a pom.xml file.
func (p *MavenParser) Parse(pomPath string) (*gopom.Model, error) {
	file, err := os.Open(pomPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open pom.xml: %w", err)
	}
	defer file.Close()

	pom, err := gopom.Parse(file)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pom.xml: %w", err)
	}

	return pom, nil
}

// GetSpringBootVersion extracts the Spring Boot version from a pom.
func (p *MavenParser) GetSpringBootVersion(pom *gopom.Model) string {
	// Check parent version (most common case)
	if pom.Parent != nil && pom.Parent.Version != nil {
		if pom.Parent.GroupId != nil && *pom.Parent.GroupId == SpringBootParentGroupID {
			if pom.Parent.ArtifactId != nil && *pom.Parent.ArtifactId == SpringBootParentArtifactID {
				return *pom.Parent.Version
			}
		}
	}

	// Check dependency management
	if pom.DependencyManagement != nil && pom.DependencyManagement.Dependencies != nil {
		for _, dep := range pom.DependencyManagement.Dependencies.Dependencies {
			if dep.GroupId != nil && *dep.GroupId == SpringBootParentGroupID {
				if dep.ArtifactId != nil && *dep.ArtifactId == SpringBootParentArtifactID {
					if dep.Version != nil {
						return *dep.Version
					}
				}
			}
		}
	}

	return ""
}

// HasSpringdocDependency checks if the pom has springdoc dependencies.
func (p *MavenParser) HasSpringdocDependency(pom *gopom.Model) bool {
	return p.FindDependency(pom, SpringdocGroupID, SpringdocWebMVCArtifactID) != nil
}

// GetSpringdocVersion returns the springdoc version if present.
func (p *MavenParser) GetSpringdocVersion(pom *gopom.Model) string {
	dep := p.FindDependency(pom, SpringdocGroupID, SpringdocWebMVCArtifactID)
	if dep != nil && dep.Version != nil {
		// Handle ${springdoc.version} style references
		version := *dep.Version
		if strings.HasPrefix(version, "${") && strings.HasSuffix(version, "}") {
			propName := strings.Trim(version, "${}")
			if pom.Properties != nil {
				if val, ok := pom.Properties.Properties[propName]; ok {
					return val
				}
			}
		}
		return version
	}
	return ""
}

// HasSpringdocPlugin checks if the pom has the springdoc maven plugin.
func (p *MavenParser) HasSpringdocPlugin(pom *gopom.Model) bool {
	if pom.Build == nil || pom.Build.Plugins == nil {
		return false
	}

	for _, plugin := range pom.Build.Plugins.Plugins {
		if plugin.GroupId != nil && *plugin.GroupId == SpringdocGroupID {
			if plugin.ArtifactId != nil && *plugin.ArtifactId == SpringdocMavenPluginArtifact {
				return true
			}
		}
	}

	return false
}

// FindDependency finds a dependency by groupId and artifactId.
func (p *MavenParser) FindDependency(pom *gopom.Model, groupID, artifactID string) *gopom.Dependency {
	if pom.Dependencies == nil {
		return nil
	}

	for i := range pom.Dependencies.Dependencies {
		dep := &pom.Dependencies.Dependencies[i]
		if dep.GroupId != nil && *dep.GroupId == groupID {
			if dep.ArtifactId != nil && *dep.ArtifactId == artifactID {
				return dep
			}
		}
	}

	return nil
}

// AddDependency adds a dependency to the pom.
func (p *MavenParser) AddDependency(pom *gopom.Model, groupID, artifactID, version string) {
	if pom.Dependencies == nil {
		pom.Dependencies = &gopom.Dependencies{}
	}

	pom.Dependencies.Dependencies = append(pom.Dependencies.Dependencies, gopom.Dependency{
		GroupId:    &groupID,
		ArtifactId: &artifactID,
		Version:    &version,
	})
}

// AddPlugin adds a plugin to the pom build section.
func (p *MavenParser) AddPlugin(pom *gopom.Model, groupID, artifactID, version string) {
	if pom.Build == nil {
		pom.Build = &gopom.Build{}
	}
	if pom.Build.Plugins == nil {
		pom.Build.Plugins = &gopom.Plugins{}
	}

	pom.Build.Plugins.Plugins = append(pom.Build.Plugins.Plugins, gopom.Plugin{
		GroupId:    &groupID,
		ArtifactId: &artifactID,
		Version:    &version,
		Executions: &gopom.Executions{
			Executions: []gopom.Execution{
				{
					Goals: &gopom.Goals{
						Goals: []string{"generate"},
					},
				},
			},
		},
	})
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/extractor/spring/... -v -run MavenParser`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/extractor/spring/maven.go internal/extractor/spring/maven_test.go
git commit -m "feat(spring): implement Maven parser with gopom"
```

---

## Task 5: Implement Gradle Parser

**Files:**
- Create: `internal/extractor/spring/gradle.go`
- Create: `internal/extractor/spring/gradle_test.go`

**Step 1: Write the failing test for Gradle parser**

```go
// Package spring_test tests the Spring extractor implementation.
package spring_test

import (
	"os"
	"testing"

	"github.com/spencercjh/spec-forge/internal/extractor/spring"
)

func TestGradleParser_Parse(t *testing.T) {
	gradlePath := "../../../integration-tests/gradle-springboot-openapi-demo/build.gradle"

	if _, err := os.Stat(gradlePath); os.IsNotExist(err) {
		t.Skip("Integration test project not found")
	}

	parser := spring.NewGradleParser()
	build, err := parser.Parse(gradlePath)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if build == nil {
		t.Fatal("Parsed build.gradle should not be nil")
	}
}

func TestGradleParser_HasSpringdocDependency(t *testing.T) {
	gradlePath := "../../../integration-tests/gradle-springboot-openapi-demo/build.gradle"

	if _, err := os.Stat(gradlePath); os.IsNotExist(err) {
		t.Skip("Integration test project not found")
	}

	parser := spring.NewGradleParser()
	build, err := parser.Parse(gradlePath)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !parser.HasSpringdocDependency(build) {
		t.Error("HasSpringdocDependency should be true for demo project")
	}
}

func TestGradleParser_HasSpringdocPlugin(t *testing.T) {
	gradlePath := "../../../integration-tests/gradle-springboot-openapi-demo/build.gradle"

	if _, err := os.Stat(gradlePath); os.IsNotExist(err) {
		t.Skip("Integration test project not found")
	}

	parser := spring.NewGradleParser()
	build, err := parser.Parse(gradlePath)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !parser.HasSpringdocPlugin(build) {
		t.Error("HasSpringdocPlugin should be true for demo project")
	}
}

func TestGradleParser_GetSpringdocVersion(t *testing.T) {
	gradlePath := "../../../integration-tests/gradle-springboot-openapi-demo/build.gradle"

	if _, err := os.Stat(gradlePath); os.IsNotExist(err) {
		t.Skip("Integration test project not found")
	}

	parser := spring.NewGradleParser()
	build, err := parser.Parse(gradlePath)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	version := parser.GetSpringdocVersion(build)
	if version == "" {
		t.Error("Springdoc version should not be empty for demo project")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/extractor/spring/... -v -run GradleParser`
Expected: FAIL with "undefined: NewGradleParser"

**Step 3: Implement Gradle parser**

```go
package spring

import (
	"fmt"
	"os"
	"strings"

	gradleparser "github.com/scagogogo/gradle-parser"
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
func (p *GradleParser) Parse(gradlePath string) (*gradleparser.GradleModule, error) {
	content, err := os.ReadFile(gradlePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read build.gradle: %w", err)
	}

	module, err := gradleparser.Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse build.gradle: %w", err)
	}

	return module, nil
}

// GetSpringBootVersion extracts the Spring Boot version from build.gradle.
func (p *GradleParser) GetSpringBootVersion(build *gradleparser.GradleModule) string {
	// Check plugins block for spring-boot
	for _, plugin := range build.Plugins {
		if plugin.ID == "org.springframework.boot" || plugin.ID == "spring-boot" {
			if plugin.Version != "" {
				return plugin.Version
			}
		}
	}

	// Check dependencies for spring-boot-starter
	for _, dep := range build.Dependencies {
		if strings.Contains(dep.Name, "spring-boot-starter") {
			if dep.Version != "" {
				return dep.Version
			}
		}
	}

	return ""
}

// HasSpringdocDependency checks if build.gradle has springdoc dependencies.
func (p *GradleParser) HasSpringdocDependency(build *gradleparser.GradleModule) bool {
	for _, dep := range build.Dependencies {
		if strings.Contains(dep.Name, "springdoc-openapi") {
			return true
		}
	}
	return false
}

// GetSpringdocVersion returns the springdoc version if present.
func (p *GradleParser) GetSpringdocVersion(build *gradleparser.GradleModule) string {
	for _, dep := range build.Dependencies {
		if strings.Contains(dep.Name, "springdoc-openapi-starter-webmvc-ui") {
			return dep.Version
		}
	}
	return ""
}

// HasSpringdocPlugin checks if build.gradle has the springdoc gradle plugin.
func (p *GradleParser) HasSpringdocPlugin(build *gradleparser.GradleModule) bool {
	for _, plugin := range build.Plugins {
		if plugin.ID == SpringdocGradlePluginID {
			return true
		}
	}
	return false
}

// FindDependency finds a dependency by name pattern.
func (p *GradleParser) FindDependency(build *gradleparser.GradleModule, namePattern string) *gradleparser.Dependency {
	for i := range build.Dependencies {
		if strings.Contains(build.Dependencies[i].Name, namePattern) {
			return &build.Dependencies[i]
		}
	}
	return nil
}

// FindPlugin finds a plugin by ID.
func (p *GradleParser) FindPlugin(build *gradleparser.GradleModule, pluginID string) *gradleparser.Plugin {
	for i := range build.Plugins {
		if build.Plugins[i].ID == pluginID {
			return &build.Plugins[i]
		}
	}
	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/extractor/spring/... -v -run GradleParser`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/extractor/spring/gradle.go internal/extractor/spring/gradle_test.go
git commit -m "feat(spring): implement Gradle parser with gradle-parser"
```

---

## Task 6: Implement Patcher

**Files:**
- Create: `internal/extractor/spring/patcher.go`
- Create: `internal/extractor/spring/patcher_test.go`

**Step 1: Write the failing test for Patcher**

```go
// Package spring_test tests the Spring extractor implementation.
package spring_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spencercjh/spec-forge/internal/extractor"
	"github.com/spencercjh/spec-forge/internal/extractor/spring"
)

func TestPatcher_Patch_MavenDryRun(t *testing.T) {
	// Read original pom.xml content
	origContent, err := os.ReadFile("../../../integration-tests/maven-springboot-openapi-demo/pom.xml")
	if err != nil {
		t.Skip("Integration test project not found")
	}

	// Create a temp copy for testing
	tmpDir := t.TempDir()
	tmpPom := filepath.Join(tmpDir, "pom.xml")
	if err := os.WriteFile(tmpPom, origContent, 0644); err != nil {
		t.Fatalf("Failed to create temp pom.xml: %v", err)
	}

	// First, remove springdoc from the copy to test patching
	// (This would require implementing a remove function - skip for now)

	patcher := spring.NewPatcher()
	opts := &extractor.PatchOptions{
		DryRun:           true,
		SpringdocVersion: extractor.DefaultSpringdocVersion,
		MavenPluginVersion: extractor.DefaultSpringdocMavenPlugin,
	}

	changes, err := patcher.Patch(tmpDir, opts)
	if err != nil {
		t.Fatalf("Patch failed: %v", err)
	}

	// In dry-run mode, changes should be reported
	t.Logf("Changes: %+v", changes)
}

func TestPatcher_NeedsPatch(t *testing.T) {
	patcher := spring.NewPatcher()

	t.Run("needs patch when missing deps", func(t *testing.T) {
		info := &extractor.ProjectInfo{
			HasSpringdocDeps:   false,
			HasSpringdocPlugin: false,
		}
		if !patcher.NeedsPatch(info, false) {
			t.Error("Should need patch when missing deps")
		}
	})

	t.Run("needs patch when force is true", func(t *testing.T) {
		info := &extractor.ProjectInfo{
			HasSpringdocDeps:   true,
			HasSpringdocPlugin: true,
		}
		if !patcher.NeedsPatch(info, true) {
			t.Error("Should need patch when force is true")
		}
	})

	t.Run("no patch needed when already configured", func(t *testing.T) {
		info := &extractor.ProjectInfo{
			HasSpringdocDeps:   true,
			HasSpringdocPlugin: true,
		}
		if patcher.NeedsPatch(info, false) {
			t.Error("Should not need patch when already configured")
		}
	})
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/extractor/spring/... -v -run Patcher`
Expected: FAIL with "undefined: NewPatcher"

**Step 3: Implement Patcher**

```go
package spring

import (
	"fmt"

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

// writePom writes the pom model back to file.
func (p *Patcher) writePom(pomPath string, pom *gopom.Model) error {
	content, err := gopom.Marshal(pom)
	if err != nil {
		return fmt.Errorf("failed to marshal pom.xml: %w", err)
	}

	return writeFile(pomPath, content)
}

// patchGradleFile modifies the build.gradle file using text manipulation.
func (p *Patcher) patchGradleFile(gradlePath string, opts *extractor.PatchOptions, result *PatchResult) error {
	content, err := readFile(gradlePath)
	if err != nil {
		return err
	}

	// Add plugin if needed
	if result.PluginAdded {
		pluginLine := fmt.Sprintf("id '%s' version \"%s\"", SpringdocGradlePluginID, opts.GradlePluginVersion)
		content = insertAfter(content, "plugins {", pluginLine)
	}

	// Add dependency if needed
	if result.DependencyAdded {
		depLine := fmt.Sprintf("implementation 'org.springdoc:springdoc-openapi-starter-webmvc-ui:%s'", opts.SpringdocVersion)
		content = insertAfter(content, "dependencies {", depLine)
	}

	return writeFile(gradlePath, []byte(content))
}
```

**Step 4: Add helper functions file**

Create `internal/extractor/spring/helpers.go`:

```go
package spring

import (
	"fmt"
	"os"
	"strings"
)

// readFile reads file content as string.
func readFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	return string(content), nil
}

// writeFile writes content to file.
func writeFile(path string, content []byte) error {
	return os.WriteFile(path, content, 0644)
}

// insertAfter inserts a line after the target line in content.
func insertAfter(content, target, newLine string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inserted := false

	for _, line := range lines {
		result = append(result, line)
		if !inserted && strings.Contains(line, target) {
			// Calculate indentation from the next line if available
			indent := ""
			if len(result) > 1 {
				prevLine := result[len(result)-2]
				for _, c := range prevLine {
					if c == ' ' || c == '\t' {
						indent += string(c)
					} else {
						break
					}
				}
				// Add one level of indentation
				indent += "    "
			}
			result = append(result, indent+newLine)
			inserted = true
		}
	}

	return strings.Join(result, "\n")
}
```

**Step 5: Run tests**

Run: `go test ./internal/extractor/spring/... -v -run Patcher`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/extractor/spring/patcher.go internal/extractor/spring/patcher_test.go internal/extractor/spring/helpers.go
git commit -m "feat(spring): implement Patcher for adding springdoc dependencies"
```

---

## Task 7: Update CLI Commands

**Files:**
- Modify: `cmd/spring.go`

**Step 1: Update spring detect command**

```go
// Package cmd contains all CLI commands for spec-forge.
package cmd

import (
	"fmt"
	"os"

	"github.com/spencercjh/spec-forge/internal/extractor"
	"github.com/spencercjh/spec-forge/internal/extractor/spring"
	"github.com/spf13/cobra"
)

// springCmd represents the spring command group
var springCmd = &cobra.Command{
	Use:   "spring",
	Short: "Spring framework specific commands",
	Long: `Commands for working with Spring (Java) projects.

These commands help you:
- Detect Spring project configuration
- Patch projects with springdoc dependencies
- Extract OpenAPI specs from Spring controllers`,
}

func init() {
	rootCmd.AddCommand(springCmd)
}

// springDetectCmd represents the spring detect command
var springDetectCmd = &cobra.Command{
	Use:   "detect [path]",
	Short: "Detect Spring project information",
	Long: `Analyze the current directory to detect Spring project type, build tool, and dependencies.

This command will identify:
- Build tool (Maven or Gradle)
- Spring Boot version
- springdoc-openapi dependency status`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSpringDetect,
}

func runSpringDetect(_ *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	detector := spring.NewDetector()
	info, err := detector.Detect(path)
	if err != nil {
		return fmt.Errorf("detection failed: %w", err)
	}

	// Print human-readable output
	printProjectInfo(info)
	return nil
}

func printProjectInfo(info *extractor.ProjectInfo) {
	fmt.Println("Spring Project Detection Results")
	fmt.Println("================================")
	fmt.Printf("Build Tool:           %s\n", info.BuildTool)
	fmt.Printf("Build File:           %s\n", info.BuildFilePath)
	fmt.Printf("Spring Boot:          %s\n", info.SpringBootVersion)

	if info.HasSpringdocDeps {
		fmt.Printf("springdoc Dependency: ✅ Present (%s)\n", info.SpringdocVersion)
	} else {
		fmt.Println("springdoc Dependency: ❌ Not found")
	}

	if info.HasSpringdocPlugin {
		fmt.Println("springdoc Plugin:     ✅ Configured")
	} else {
		fmt.Println("springdoc Plugin:     ❌ Not configured")
	}
}

var (
	patchDryRun bool
	patchForce  bool
)

// springPatchCmd represents the spring patch command
var springPatchCmd = &cobra.Command{
	Use:   "patch [path]",
	Short: "Add springdoc dependencies to Spring project",
	Long: `Add springdoc-openapi dependencies to the Spring project's build file.
Supports both Maven (pom.xml) and Gradle (build.gradle) projects.

This command will:
- Detect the build tool
- Add the appropriate springdoc dependency
- Optionally update existing dependencies`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSpringPatch,
}

func runSpringPatch(_ *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	opts := &extractor.PatchOptions{
		DryRun: patchDryRun,
		Force:  patchForce,
	}

	patcher := spring.NewPatcher()
	result, err := patcher.Patch(path, opts)
	if err != nil {
		return fmt.Errorf("patch failed: %w", err)
	}

	// Print results
	if opts.DryRun {
		fmt.Println("Dry run mode - showing changes without modifying files")
	}

	fmt.Printf("Build file: %s\n", result.BuildFilePath)

	if result.DependencyAdded {
		fmt.Println("✅ springdoc dependency will be added")
	} else {
		fmt.Println("⏭️  springdoc dependency already present")
	}

	if result.PluginAdded {
		fmt.Println("✅ springdoc plugin will be added")
	} else {
		fmt.Println("⏭️  springdoc plugin already configured")
	}

	if !opts.DryRun && (result.DependencyAdded || result.PluginAdded) {
		fmt.Println("\nPatch applied successfully!")
	} else if !result.DependencyAdded && !result.PluginAdded {
		fmt.Println("\nNo changes needed.")
	}

	return nil
}

func init() {
	springCmd.AddCommand(springDetectCmd)
	springCmd.AddCommand(springPatchCmd)

	springPatchCmd.Flags().BoolVar(&patchDryRun, "dry-run", false, "show changes without modifying files")
	springPatchCmd.Flags().BoolVar(&patchForce, "force", false, "force overwrite existing dependencies")
}
```

**Step 2: Run build and test**

Run: `make all`
Expected: Build succeeds, all tests pass

**Step 3: Manual test**

```bash
./bin/spec-forge spring detect integration-tests/maven-springboot-openapi-demo
./bin/spec-forge spring detect integration-tests/gradle-springboot-openapi-demo
```

**Step 4: Commit**

```bash
git add cmd/spring.go
git commit -m "feat(cli): wire up spring detect and patch commands"
```

---

## Task 8: Run All Tests and Final Verification

**Step 1: Run all tests**

Run: `make all`
Expected: All tests pass, build succeeds, lint passes

**Step 2: Test detect on integration projects**

```bash
./bin/spec-forge spring detect integration-tests/maven-springboot-openapi-demo
./bin/spec-forge spring detect integration-tests/gradle-springboot-openapi-demo
```

**Step 3: Test patch dry-run**

```bash
./bin/spec-forge spring patch --dry-run integration-tests/maven-springboot-openapi-demo
./bin/spec-forge spring patch --dry-run integration-tests/gradle-springboot-openapi-demo
```

**Step 4: Final commit**

```bash
git status
# Ensure all changes are committed
```

---

## Summary

This plan implements M2: Spring Detection and Patch with:

1. **Types and constants** - BuildTool, ProjectInfo, PatchOptions
2. **Detector** - Detects Maven/Gradle projects and their springdoc status
3. **Maven Parser** - Uses gopom to parse and modify pom.xml
4. **Gradle Parser** - Uses gradle-parser to parse build.gradle
5. **Patcher** - Adds springdoc dependencies and plugins
6. **CLI integration** - Wires up detect and patch commands

Each task follows TDD: write test first, implement, verify, commit.
