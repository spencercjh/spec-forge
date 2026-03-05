// Package cmd contains all CLI commands for spec-forge.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/spf13/cobra"

	"github.com/spencercjh/spec-forge/internal/config"
	"github.com/spencercjh/spec-forge/internal/enricher"
	"github.com/spencercjh/spec-forge/internal/enricher/processor"
	"github.com/spencercjh/spec-forge/internal/enricher/provider"
	"github.com/spencercjh/spec-forge/internal/extractor"
	"github.com/spencercjh/spec-forge/internal/extractor/spring"
	"github.com/spencercjh/spec-forge/internal/publisher"
	"github.com/spencercjh/spec-forge/internal/validator"
)

var (
	// generateKeepPatched controls whether to keep the patched pom/build file
	// Default is false (restore original) for generate command
	generateKeepPatched bool
	// generateSkipValidate controls whether to skip validation
	generateSkipValidate bool
	// generateTimeout is the timeout for generation commands
	generateTimeout time.Duration
	// generateSkipEnrich controls whether to skip AI enrichment
	generateSkipEnrich bool
	// generateLanguage is the language for AI-generated descriptions
	generateLanguage string
	// generateOutput is the output directory for generated spec
	generateOutput string
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate [path]",
	Short: "Generate OpenAPI spec from source code",
	Long: `Generate OpenAPI specification by running the complete pipeline:
detect -> patch -> generate -> validate -> restore (optional)

This is the main command that orchestrates the entire workflow.

By default, the original pom.xml/build.gradle is restored after extraction
to preserve your project's formatting. Use --keep-patched to keep the changes.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runGenerate,
}

func runGenerate(cmd *cobra.Command, args []string) error { //nolint:gocyclo // CLI command runner requires many branches
	ctx := cmd.Context()
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	slog.InfoContext(ctx, "Generating OpenAPI spec", "path", path)

	// Step 1: Detect project
	detector := spring.NewDetector()
	info, err := detector.Detect(path)
	if err != nil {
		return errWrap("detection failed", err)
	}

	slog.InfoContext(ctx, "Detected project",
		"tool", info.BuildTool,
		"build_file", info.BuildFilePath,
		"multi_module", info.IsMultiModule,
	)

	// Step 2: Patch project if needed
	patcher := spring.NewPatcher()
	patchOpts := &extractor.PatchOptions{
		KeepPatched: generateKeepPatched,
	}

	result, err := patcher.Patch(path, patchOpts)
	if err != nil {
		return errWrap("patch failed", err)
	}

	// Step 3: If we patched the file and should restore later, defer the restore
	if !generateKeepPatched && result.OriginalContent != "" {
		defer func() {
			slog.InfoContext(ctx, "Restoring original build file...")
			if restoreErr := patcher.Restore(result.BuildFilePath, result.OriginalContent); restoreErr != nil {
				slog.WarnContext(ctx, "failed to restore original file", "error", restoreErr)
			} else {
				slog.InfoContext(ctx, "Original build file restored", "status", "✅")
			}
		}()
	}

	if result.DependencyAdded {
		slog.InfoContext(ctx, "springdoc dependency added temporarily", "status", "✅")
	}
	if result.PluginAdded {
		slog.InfoContext(ctx, "springdoc plugin added temporarily", "status", "✅")
	}
	if result.SpringBootConfigured {
		slog.InfoContext(ctx, "spring-boot-maven-plugin configured with start/stop goals", "status", "✅")
	}

	// Step 4: Generate OpenAPI spec
	generator := spring.NewGenerator()

	// Determine output directory
	outputDir := generateOutput
	if outputDir == "" {
		outputDir = config.Get().Output.Dir
	}

	genOpts := &extractor.GenerateOptions{
		OutputDir: outputDir,
		Format:    config.Get().Output.Format,
		Timeout:   generateTimeout,
		SkipTests: true,
	}

	genResult, err := generator.Generate(ctx, path, info, genOpts)
	if err != nil {
		return errWrap("generation failed", err)
	}

	slog.InfoContext(ctx, "OpenAPI spec generated",
		"path", genResult.SpecFilePath,
		"format", genResult.Format,
	)

	// Step 5: Validate the generated spec
	if !generateSkipValidate {
		v := validator.NewValidator()
		valResult, err := v.Validate(ctx, genResult.SpecFilePath)
		if err != nil {
			return errWrap("validation error", err)
		}

		if !valResult.Valid {
			slog.ErrorContext(ctx, "OpenAPI spec validation failed")
			for _, validationErr := range valResult.Errors {
				slog.ErrorContext(ctx, "  - "+validationErr)
			}
			return errors.New("generated OpenAPI spec is invalid")
		}

		slog.InfoContext(ctx, "OpenAPI spec validated", "status", "✅")
	} else {
		slog.InfoContext(ctx, "Validation skipped", "status", "⏭️")
	}

	// Step 6: Enrich with AI (optional)
	cfg := config.Get()
	if !generateSkipEnrich && cfg.Enrich.Enabled && cfg.Enrich.Provider != "" && cfg.Enrich.Model != "" {
		if err := enrichGeneratedSpec(ctx, genResult.SpecFilePath, cfg); err != nil {
			// Log warning but don't fail - enrichment is optional
			slog.WarnContext(ctx, "Enrichment failed (non-fatal)", "error", err)
		}
	} else {
		slog.InfoContext(ctx, "Enrichment skipped", "status", "⏭️")
	}

	// Step 7: Copy to output directory if specified
	finalSpecPath := genResult.SpecFilePath
	if outputDir != "" {
		if err := copySpecToOutput(genResult.SpecFilePath, outputDir); err != nil {
			return errWrap("failed to copy spec to output directory", err)
		}
		finalSpecPath = filepath.Join(outputDir, filepath.Base(genResult.SpecFilePath))
		slog.InfoContext(ctx, "Spec copied to output directory", "path", finalSpecPath)
	}

	// Step 8: Output final result
	slog.InfoContext(ctx, "Generation complete",
		"spec_file", finalSpecPath,
	)

	return nil
}

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.Flags().BoolVar(&generateKeepPatched, "keep-patched", false,
		"keep the patched pom.xml/build.gradle (default: restore original after extraction)")
	generateCmd.Flags().BoolVar(&generateSkipValidate, "skip-validate", false,
		"skip validation of the generated OpenAPI spec")
	generateCmd.Flags().DurationVar(&generateTimeout, "timeout", 5*time.Minute,
		"timeout for Maven/Gradle commands")
	generateCmd.Flags().BoolVar(&generateSkipEnrich, "skip-enrich", false,
		"skip AI enrichment of the generated OpenAPI spec")
	generateCmd.Flags().StringVar(&generateLanguage, "language", "en",
		"language for AI-generated descriptions (e.g., en, zh)")
	generateCmd.Flags().StringVarP(&generateOutput, "output", "o", "",
		"output directory for generated spec (default: project's target/build dir)")
}

// enrichGeneratedSpec enriches the generated spec with AI-generated descriptions
func enrichGeneratedSpec(ctx context.Context, specFilePath string, cfg *config.Config) error {
	slog.InfoContext(ctx, "Enriching OpenAPI spec with AI descriptions...")

	// Determine language
	lang := generateLanguage
	if lang == "" {
		lang = cfg.Enrich.Language
	}
	if lang == "" {
		lang = "en"
	}

	// Create provider
	p, err := createProviderFromConfig(cfg.Enrich)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	// Load spec
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	spec, err := loader.LoadFromFile(specFilePath)
	if err != nil {
		return fmt.Errorf("failed to load spec: %w", err)
	}

	// Parse timeout
	timeout := 30 * time.Second
	if cfg.Enrich.Timeout != "" {
		if parsed, parseErr := time.ParseDuration(cfg.Enrich.Timeout); parseErr == nil {
			timeout = parsed
		}
	}

	// Create enricher config
	enricherCfg := enricher.Config{
		Provider:      cfg.Enrich.Provider,
		Model:         cfg.Enrich.Model,
		Language:      lang,
		Timeout:       timeout,
		CustomBaseURL: cfg.Enrich.BaseURL,
	}
	enricherCfg = enricherCfg.MergeWithDefaults()

	// Create enricher
	e, err := enricher.NewEnricher(enricherCfg, p)
	if err != nil {
		return fmt.Errorf("failed to create enricher: %w", err)
	}

	// Enrich
	result, err := e.Enrich(ctx, spec, &enricher.EnrichOptions{Language: lang})
	if err != nil {
		// Check if partial enrichment
		if partialErr, ok := errors.AsType[*processor.PartialEnrichmentError](err); ok {
			slog.WarnContext(ctx, "Partial enrichment completed",
				"failed_batches", partialErr.FailedBatches,
				"total_batches", partialErr.TotalBatches,
			)
		} else {
			return fmt.Errorf("enrichment failed: %w", err)
		}
	}

	// Publish result using Publisher
	pub := publisher.NewLocalPublisher()
	pubResult, err := pub.Publish(ctx, result, &publisher.PublishOptions{
		OutputPath: specFilePath,
		Overwrite:  true,
	})
	if err != nil {
		return fmt.Errorf("failed to save enriched spec: %w", err)
	}

	slog.InfoContext(ctx, "OpenAPI spec enriched", "output", pubResult.Path)
	return nil
}

// createProviderFromConfig creates a provider from config settings
func createProviderFromConfig(cfg config.EnrichConfig) (provider.Provider, error) { //nolint:gocritic // copying config is acceptable
	// Get API key based on provider type
	var apiKey string
	switch cfg.Provider {
	case "openai":
		apiKey = os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			return nil, errors.New("OPENAI_API_KEY environment variable not set")
		}
	case "anthropic":
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			return nil, errors.New("ANTHROPIC_API_KEY environment variable not set")
		}
	case "custom":
		apiKey = getAPIKeyFromConfig(cfg)
		if apiKey == "" {
			return nil, errors.New("API key not found for custom provider")
		}
	}

	return provider.NewProvider(provider.Config{
		Provider: cfg.Provider,
		Model:    cfg.Model,
		APIKey:   apiKey,
		BaseURL:  cfg.BaseURL,
	})
}

// getAPIKeyFromConfig gets API key from config or environment
func getAPIKeyFromConfig(cfg config.EnrichConfig) string { //nolint:gocritic // copying config is acceptable
	// First check explicit config
	if cfg.APIKey != "" {
		return cfg.APIKey
	}
	// Then check environment variable
	envName := cfg.APIKeyEnv
	if envName == "" {
		envName = "LLM_API_KEY"
	}
	return os.Getenv(envName)
}

// errWrap wraps an error with a message.
func errWrap(msg string, err error) error {
	return errors.New(msg + ": " + err.Error())
}

// copySpecToOutput copies the generated spec to the specified output directory
func copySpecToOutput(srcPath, outputDir string) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Open source file
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Determine destination filename
	filename := filepath.Base(srcPath)
	dstPath := filepath.Join(outputDir, filename)

	// Create destination file
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	// Copy content
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}
