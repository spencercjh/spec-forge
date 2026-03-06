// Package extractor provides interfaces and types for extracting OpenAPI specs from projects.
package extractor

import (
	"context"
	"time"
)

// BuildTool represents the build tool type for a project.
type BuildTool string

// Framework type constants.
const ()

// Extractor is the interface for framework-specific OpenAPI spec extraction.
// Each framework (Spring Boot, go-zero, etc.) implements this interface.
type Extractor interface {
	// Name returns the extractor name (e.g., "springboot", "gozero")
	Name() string

	// Detect analyzes a project and returns its information if the framework is detected.
	// Returns an error if the project is not of this framework type.
	Detect(projectPath string) (*ProjectInfo, error)

	// Patch prepares the project for OpenAPI spec generation (e.g., add dependencies).
	Patch(projectPath string, opts *PatchOptions) (*PatchResult, error)

	// Generate produces the OpenAPI spec from the patched project.
	Generate(ctx context.Context, projectPath string, info *ProjectInfo, opts *GenerateOptions) (*GenerateResult, error)

	// Restore restores the original project files after generation.
	Restore(buildFilePath, originalContent string) error
}

// PatchResult contains the result of patching a project.
type PatchResult struct {
	BuildFilePath        string // Path to the patched build file
	OriginalContent      string // Original content for restoration
	DependencyAdded      bool   // Whether a new dependency was added
	PluginAdded          bool   // Whether a new plugin was added
	SpringBootConfigured bool   // Whether spring-boot plugin was configured
}

// ProjectInfo contains common detected information about a project.
// Framework-specific details are stored via type assertion by each framework package.
type ProjectInfo struct {
	Framework     string    // Framework type: "springboot" or "gozero"
	BuildTool     BuildTool // Maven, Gradle, or GoModules
	BuildFilePath string    // Path to build file (pom.xml, build.gradle, or go.mod)

	// FrameworkData holds framework-specific info as interface{}.
	// Each framework casts this to their own type (e.g., *spring.Info or *gozero.Info)
	FrameworkData any
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
