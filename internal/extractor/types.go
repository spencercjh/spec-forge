// Package extractor provides interfaces and types for extracting OpenAPI specs from projects.
package extractor

import (
	"context"
	"errors"
	"time"
)

// BuildTool represents the build tool type for a project.
type BuildTool string

// Framework type constants.
const (
	FrameworkSpringBoot = "springboot"
	FrameworkGoZero     = "gozero"
)

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

// registry holds all registered extractors.
var registry = make(map[string]Extractor)

// Register registers an extractor for a framework.
// This should be called in the init() function of each framework package.
func Register(name string, e Extractor) {
	registry[name] = e
}

// Get returns a registered extractor by name.
func Get(name string) (Extractor, bool) {
	e, ok := registry[name]
	return e, ok
}

// GetAll returns all registered extractors.
func GetAll() []Extractor {
	extractors := make([]Extractor, 0, len(registry))
	for _, e := range registry {
		extractors = append(extractors, e)
	}
	return extractors
}

// DetectFramework tries all registered extractors and returns the first matching framework.
func DetectFramework(projectPath string) (Extractor, *ProjectInfo, error) {
	for _, e := range GetAll() {
		info, err := e.Detect(projectPath)
		if err == nil {
			return e, info, nil
		}
	}
	return nil, nil, ErrNoFrameworkDetected
}

// ErrNoFrameworkDetected is returned when no framework can detect the project.
var ErrNoFrameworkDetected = errors.New("no supported framework detected in project")

// PatchResult contains the result of patching a project.
type PatchResult struct {
	BuildFilePath        string // Path to the patched build file
	OriginalContent      string // Original content for restoration
	DependencyAdded      bool   // Whether a new dependency was added
	PluginAdded          bool   // Whether a new plugin was added
	SpringBootConfigured bool   // Whether spring-boot plugin was configured
}

// ProjectInfo contains detected information about a project.
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

	// go-zero framework support
	Framework     string   // Framework type: "springboot" or "gozero"
	GoVersion     string   // Go version (for go-zero projects)
	GoModule      string   // Go module path (for go-zero projects)
	ModuleName    string   // Go module name (for go-zero projects)
	HasGoZeroDeps bool     // Whether go-zero dependencies exist
	GoZeroVersion string   // Existing go-zero version if any
	HasGoctl      bool     // Whether goctl is available (for go-zero projects)
	APIFiles      []string // List of .api file paths (for go-zero projects)
	MainPackage   string   // Main package path (for go-zero projects)
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
