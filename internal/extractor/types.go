// Package extractor provides interfaces and types for extracting OpenAPI specs from projects.
package extractor

import "time"

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
	DefaultSpringdocVersion      = "3.0.2"
	DefaultSpringdocMavenPlugin  = "1.5"
	DefaultSpringdocGradlePlugin = "1.9.0"
)

// ProjectInfo contains detected information about a Spring project.
type ProjectInfo struct {
	BuildTool          BuildTool // Maven or Gradle
	BuildFilePath      string    // pom.xml or build.gradle path
	SpringBootVersion  string    // Spring Boot version
	HasSpringdocDeps   bool      // Whether springdoc dependencies exist
	HasSpringdocPlugin bool      // Whether springdoc plugin is configured
	SpringdocVersion   string    // Existing springdoc version if any

	// Multi-module project support
	IsMultiModule  bool     // Whether this is a multi-module project
	Modules        []string // List of module names (for multi-module projects)
	MainModule     string   // The main application module (if detected)
	MainModulePath string   // Path to the main module's build file
}

// PatchOptions configures the patch behavior.
type PatchOptions struct {
	DryRun              bool   // Only print changes, don't write
	Force               bool   // Force overwrite existing dependencies
	SpringdocVersion    string // springdoc version (default: built-in)
	MavenPluginVersion  string // Maven plugin version (default: built-in)
	GradlePluginVersion string // Gradle plugin version (default: built-in)
	KeepPatched         bool   // If false (default for generate), restore original file after extraction
}

// GenerateOptions configures OpenAPI spec generation.
type GenerateOptions struct {
	OutputDir  string        // Output directory for generated spec (default: project target/build dir)
	OutputFile string        // Output file name without extension (default: "openapi")
	Format     string        // Output format: "json" or "yaml" (default: "json")
	Timeout    time.Duration // Command execution timeout (default: 5 minutes)
	SkipTests  bool          // Skip tests during build (default: true)
}

// GenerateResult contains the result of OpenAPI spec generation.
type GenerateResult struct {
	SpecFilePath string // Absolute path to the generated spec file
	Format       string // Output format: "json" or "yaml"
}

// ValidateResult contains the result of OpenAPI spec validation.
type ValidateResult struct {
	Valid  bool     // Whether the spec is valid
	Errors []string // Validation errors (if any)
}
