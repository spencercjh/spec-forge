package spring

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spencercjh/spec-forge/internal/executor"
	"github.com/spencercjh/spec-forge/internal/extractor"
)

// wrapCommandError wraps executor errors with helpful hints.
func wrapCommandError(err error) error {
	var cmdNotFound *executor.CommandNotFoundError
	if errors.As(err, &cmdNotFound) && cmdNotFound.Hint != "" {
		return fmt.Errorf("%w\nHint: %s", err, cmdNotFound.Hint)
	}
	return err
}

const (
	defaultOutputFileName = "openapi"
	defaultFormat         = "json"
	mavenCmd              = "mvn"
	gradleCmd             = "gradle"
)

// Generator generates OpenAPI specs from Spring projects.
type Generator struct {
	detector *Detector
	executor executor.Interface
}

// NewGenerator creates a new Generator instance.
func NewGenerator() *Generator {
	return &Generator{
		detector: NewDetector(),
		executor: executor.NewExecutor(),
	}
}

// NewGeneratorWithExecutor creates a new Generator with a custom executor (for testing).
func NewGeneratorWithExecutor(exec executor.Interface) *Generator {
	return &Generator{
		detector: NewDetector(),
		executor: exec,
	}
}

// Generate generates OpenAPI spec by invoking Maven/Gradle springdoc plugin.
func (g *Generator) Generate(ctx context.Context, projectPath string, info *extractor.ProjectInfo, opts *extractor.GenerateOptions) (*extractor.GenerateResult, error) {
	if opts == nil {
		opts = &extractor.GenerateOptions{}
	}

	// Apply defaults
	if opts.Timeout <= 0 {
		opts.Timeout = 5 * time.Minute
	}
	if opts.Format == "" {
		opts.Format = defaultFormat
	}
	if opts.OutputFile == "" {
		opts.OutputFile = defaultOutputFileName
	}

	// Determine the working directory
	workDir := projectPath
	if info.IsMultiModule && info.MainModulePath != "" {
		// For multi-module projects, run from the main module directory
		workDir = filepath.Dir(info.MainModulePath)
	}

	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve working directory: %w", err)
	}

	// Generate based on build tool
	switch info.BuildTool {
	case BuildToolMaven:
		return g.generateMaven(ctx, absWorkDir, info, opts)
	case BuildToolGradle:
		return g.generateGradle(ctx, absWorkDir, info, opts)
	default:
		return nil, fmt.Errorf("unsupported build tool: %s", info.BuildTool)
	}
}

// resolveMavenCommand resolves the Maven command to use.
// Priority: mvnw in project root > mvnw in parent directories > mvn from PATH
func (g *Generator) resolveMavenCommand(workDir string) string {
	// Check for Maven wrapper in current directory
	mvnwPath := filepath.Join(workDir, "mvnw")
	if _, err := os.Stat(mvnwPath); err == nil {
		return "./mvnw"
	}

	// For multi-module projects, check parent directories
	// Walk up the directory tree looking for mvnw
	currentDir := workDir
	for {
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			// Reached root directory
			break
		}

		mvnwInParent := filepath.Join(parentDir, "mvnw")
		if _, err := os.Stat(mvnwInParent); err == nil {
			// Found wrapper in parent, return absolute path
			absPath, absErr := filepath.Abs(mvnwInParent)
			if absErr != nil {
				return mavenCmd // Fallback to system Maven on error
			}
			return absPath
		}

		// Check if we've gone too far (no pom.xml in parent means we left the project)
		pomInParent := filepath.Join(parentDir, "pom.xml")
		if _, err := os.Stat(pomInParent); os.IsNotExist(err) {
			break
		}

		currentDir = parentDir
	}

	// Fallback to system Maven
	return mavenCmd
}

// resolveGradleCommand resolves the Gradle command to use.
// Priority: gradlew in project root > gradlew in parent directories > gradle from PATH
func (g *Generator) resolveGradleCommand(workDir string) string {
	// Check for Gradle wrapper in current directory
	gradlewPath := filepath.Join(workDir, "gradlew")
	if _, err := os.Stat(gradlewPath); err == nil {
		return "./gradlew"
	}

	// For multi-module projects, check parent directories
	currentDir := workDir
	for {
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			// Reached root directory
			break
		}

		gradlewInParent := filepath.Join(parentDir, "gradlew")
		if _, err := os.Stat(gradlewInParent); err == nil {
			absPath, absErr := filepath.Abs(gradlewInParent)
			if absErr != nil {
				return gradleCmd // Fallback to system Gradle on error
			}
			return absPath
		}

		// Check if we've gone too far (no build.gradle in parent means we left the project)
		gradleInParent := filepath.Join(parentDir, "build.gradle")
		gradleKtsInParent := filepath.Join(parentDir, "build.gradle.kts")
		if _, err := os.Stat(gradleInParent); os.IsNotExist(err) {
			if _, err := os.Stat(gradleKtsInParent); os.IsNotExist(err) {
				break
			}
		}

		currentDir = parentDir
	}

	// Fallback to system Gradle
	return gradleCmd
}

// generateMaven generates OpenAPI spec using Maven springdoc plugin.
func (g *Generator) generateMaven(ctx context.Context, workDir string, _ *extractor.ProjectInfo, opts *extractor.GenerateOptions) (*extractor.GenerateResult, error) {
	// Resolve Maven command (wrapper or system)
	mavenCmd := g.resolveMavenCommand(workDir)

	// Build Maven command arguments
	// Per springdoc official documentation, use "verify" phase to trigger springdoc plugin
	args := []string{
		"verify",
	}

	// Skip tests by default
	if opts.SkipTests {
		args = append(args, "-DskipTests")
	}

	// Add output format configuration
	args = append(args, "-Dspringdoc.outputFormat="+opts.Format)

	// Add output file name if specified
	if opts.OutputFile != defaultOutputFileName {
		args = append(args, "-Dspringdoc.outputFileName="+opts.OutputFile)
	}

	// Execute Maven command
	result, err := g.executor.Execute(ctx, &executor.ExecuteOptions{
		Command:    mavenCmd,
		Args:       args,
		WorkingDir: workDir,
		Timeout:    opts.Timeout,
	})
	if err != nil {
		return nil, fmt.Errorf("maven generation failed: %w", wrapCommandError(err))
	}

	if result.ExitCode != 0 {
		return nil, fmt.Errorf("maven generation failed with exit code %d: %s", result.ExitCode, result.Stderr)
	}

	// Find the generated spec file
	specPath, err := g.findGeneratedSpec(workDir, nil, "target", opts)
	if err != nil {
		return nil, err
	}

	return &extractor.GenerateResult{
		SpecFilePath: specPath,
		Format:       opts.Format,
	}, nil
}

// generateGradle generates OpenAPI spec using Gradle springdoc plugin.
func (g *Generator) generateGradle(ctx context.Context, workDir string, _ *extractor.ProjectInfo, opts *extractor.GenerateOptions) (*extractor.GenerateResult, error) {
	// Resolve Gradle command (wrapper or system)
	gradleCmd := g.resolveGradleCommand(workDir)

	// Build Gradle command arguments
	// Per springdoc official documentation, use "generateOpenApiDocs" task
	args := []string{
		"generateOpenApiDocs",
	}

	// Skip tests by default
	if opts.SkipTests {
		args = append(args, "-x", "test")
	}

	// Execute Gradle command
	result, err := g.executor.Execute(ctx, &executor.ExecuteOptions{
		Command:    gradleCmd,
		Args:       args,
		WorkingDir: workDir,
		Timeout:    opts.Timeout,
	})
	if err != nil {
		return nil, fmt.Errorf("gradle generation failed: %w", wrapCommandError(err))
	}

	if result.ExitCode != 0 {
		return nil, fmt.Errorf("gradle generation failed with exit code %d: %s", result.ExitCode, result.Stderr)
	}

	// Find the generated spec file
	specPath, err := g.findGeneratedSpec(workDir, nil, "build", opts)
	if err != nil {
		return nil, err
	}

	return &extractor.GenerateResult{
		SpecFilePath: specPath,
		Format:       opts.Format,
	}, nil
}

// findGeneratedSpec locates the generated OpenAPI spec file.
func (g *Generator) findGeneratedSpec(workDir string, _ *extractor.ProjectInfo, outputDir string, opts *extractor.GenerateOptions) (string, error) {
	// Determine search directory
	searchDir := filepath.Join(workDir, outputDir)

	// Check for the spec file with the expected name
	fileName := opts.OutputFile
	if fileName == "" {
		fileName = defaultOutputFileName
	}

	// Try extensions based on format
	var extensions []string
	switch opts.Format {
	case "yaml":
		extensions = []string{".yaml", ".yml", ".json"}
	default: // "json" or any other format
		extensions = []string{".json", ".yaml", ".yml"}
	}

	for _, ext := range extensions {
		candidatePath := filepath.Join(searchDir, fileName+ext)
		if _, err := os.Stat(candidatePath); err == nil {
			absPath, absErr := filepath.Abs(candidatePath)
			if absErr != nil {
				return "", fmt.Errorf("failed to get absolute path: %w", absErr)
			}
			return absPath, nil
		}
	}

	// If not found in expected location, search recursively
	var foundPath string
	err := filepath.Walk(searchDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}

		name := strings.ToLower(info.Name())
		if strings.HasPrefix(name, fileName) && (strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml")) {
			foundPath = path
			return filepath.SkipAll
		}
		return nil
	})

	if err != nil && !errors.Is(err, filepath.SkipAll) {
		return "", fmt.Errorf("error searching for generated spec: %w", err)
	}

	if foundPath != "" {
		absPath, absErr := filepath.Abs(foundPath)
		if absErr != nil {
			return "", fmt.Errorf("failed to get absolute path: %w", absErr)
		}
		return absPath, nil
	}

	return "", fmt.Errorf("generated OpenAPI spec not found in %s (expected %s.{json|yaml})", searchDir, fileName)
}
