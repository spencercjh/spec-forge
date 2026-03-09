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
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/spencercjh/spec-forge/internal/config"
	"github.com/spencercjh/spec-forge/internal/enricher"
	"github.com/spencercjh/spec-forge/internal/enricher/processor"
	"github.com/spencercjh/spec-forge/internal/enricher/provider"
	"github.com/spencercjh/spec-forge/internal/extractor"
	"github.com/spencercjh/spec-forge/internal/extractor/builtin" // registers built-in extractors
	"github.com/spencercjh/spec-forge/internal/publisher"
	"github.com/spencercjh/spec-forge/internal/validator"
)

const (
	outputFormatYAML = "yaml"
	outputFormatJSON = "json"
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
	// generateOutputDir is the output directory for generated spec
	generateOutputDir string
	// generateOutputFormat is the output format (yaml or json)
	generateOutputFormat string
	// generateSkipPublish controls whether to skip publishing
	generateSkipPublish bool
	// generatePublishTarget is the publish target (readme)
	generatePublishTarget string
	// generatePublishOverwrite controls whether to overwrite existing remote spec
	generatePublishOverwrite bool
	// generateOverwriteOutput controls whether to overwrite existing local spec file
	generateOverwriteOutput bool
	// generateProtoImportPaths are additional import paths for protoc (-I flags)
	generateProtoImportPaths []string
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

	// Step 1: Detect framework - try all registered extractors
	extractorImpl, info, err := builtin.DetectFramework(path)
	if err != nil {
		return errWrap("no supported framework detected", err)
	}

	slog.InfoContext(ctx, "Detected project",
		"framework", extractorImpl.Name(),
		"tool", info.BuildTool,
		"build_file", info.BuildFilePath,
	)

	// Step 2: Patch project if needed
	patchOpts := &extractor.PatchOptions{
		KeepPatched: generateKeepPatched,
	}

	patchResult, err := extractorImpl.Patch(path, patchOpts)
	if err != nil {
		return errWrap("patch failed", err)
	}

	// Step 3: If we patched the file and should restore later, defer the restore
	if !generateKeepPatched && patchResult.OriginalContent != "" {
		defer func() {
			slog.InfoContext(ctx, "Restoring original build file...")
			if restoreErr := extractorImpl.Restore(patchResult.BuildFilePath, patchResult.OriginalContent); restoreErr != nil {
				slog.WarnContext(ctx, "failed to restore original file", "error", restoreErr)
			} else {
				slog.InfoContext(ctx, "Original build file restored", "status", "✅")
			}
		}()
	}

	if patchResult.DependencyAdded {
		slog.InfoContext(ctx, "dependencies added temporarily", "status", "✅")
	}
	if patchResult.PluginAdded {
		slog.InfoContext(ctx, "plugin added temporarily", "status", "✅")
	}
	if patchResult.SpringBootConfigured {
		slog.InfoContext(ctx, "spring-boot-maven-plugin configured with start/stop goals", "status", "✅")
	}

	// Step 4: Generate OpenAPI spec

	// Determine output directory
	// Precedence: flag > config > default (project root)
	outputDir := generateOutputDir
	if outputDir == "" {
		outputDir = config.Get().Output.Dir
	}
	if outputDir == "" {
		outputDir = path // Default to project root
	}

	// Determine output format
	outputFormat := generateOutputFormat
	if outputFormat == "" {
		outputFormat = config.Get().Output.Format
	}
	if outputFormat == "" {
		outputFormat = outputFormatYAML
	}
	// Normalize format value for consistent handling across extractors
	outputFormat, err = normalizeOutputFormat(outputFormat)
	if err != nil {
		return errWrap("invalid output format", err)
	}

	genOpts := &extractor.GenerateOptions{
		OutputDir:        outputDir,
		Format:           outputFormat,
		Timeout:          generateTimeout,
		SkipTests:        true,
		ProtoImportPaths: generateProtoImportPaths,
	}

	genResult, err := extractorImpl.Generate(ctx, path, info, genOpts)
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
		valResult, valErr := v.Validate(ctx, genResult.SpecFilePath)
		if valErr != nil {
			return errWrap("validation error", valErr)
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
		if enrichErr := enrichGeneratedSpec(ctx, genResult.SpecFilePath, cfg); enrichErr != nil {
			// Log warning but don't fail - enrichment is optional
			slog.WarnContext(ctx, "Enrichment failed (non-fatal)", "error", enrichErr)
		}
	} else {
		slog.InfoContext(ctx, "Enrichment skipped", "status", "⏭️")
	}

	// Step 7: Ensure spec is in the output directory
	// Some extractors (Spring) generate to target/build and need copying
	// Others (Gin, go-zero, gRPC) may already have written to outputDir
	genDir := filepath.Dir(genResult.SpecFilePath)
	targetPath := filepath.Join(outputDir, filepath.Base(genResult.SpecFilePath))

	// Clean paths for comparison
	absGenDir, err := filepath.Abs(genDir)
	if err != nil {
		absGenDir = genDir // Fallback to relative path
	}
	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		absOutputDir = outputDir // Fallback to relative path
	}

	if absGenDir != absOutputDir {
		if err := copySpecToOutput(genResult.SpecFilePath, outputDir, generateOverwriteOutput); err != nil {
			return errWrap("failed to copy spec to output directory", err)
		}
		slog.InfoContext(ctx, "Spec saved", "path", targetPath)
		// Update genResult.SpecFilePath to point to the copied file for publishing
		genResult.SpecFilePath = targetPath
	} else {
		slog.InfoContext(ctx, "Spec saved", "path", genResult.SpecFilePath)
	}

	// Step 8: Publish the spec to remote platforms (optional)
	if !generateSkipPublish && generatePublishTarget != "" {
		// Create publisher using factory
		pub, err := publisher.NewPublisher(generatePublishTarget)
		if err != nil {
			return errWrap("failed to create publisher", err)
		}

		// Build publish options
		pubOpts := &publisher.PublishOptions{
			OutputPath: genResult.SpecFilePath,
			Format:     outputFormat,
			Overwrite:  generatePublishOverwrite,
		}

		// Add ReadMe-specific options if using ReadMe publisher
		if pub.Name() == "readme" {
			cfg := config.Get()
			pubOpts.ReadMe = &publisher.ReadMeOptions{
				APIKey:         resolveReadMeAPIKey(cfg.ReadMe),
				Branch:         cfg.ReadMe.Branch,
				Slug:           cfg.ReadMe.Slug,
				UseSpecVersion: cfg.ReadMe.UseSpecVersion,
			}
		}

		// Load spec for publishing
		loader := openapi3.NewLoader()
		spec, err := loader.LoadFromFile(genResult.SpecFilePath)
		if err != nil {
			return errWrap("failed to load spec for publishing", err)
		}

		// Publish
		pubResult, err := pub.Publish(ctx, spec, pubOpts)
		if err != nil {
			return errWrap("failed to publish spec", err)
		}

		slog.InfoContext(ctx, "Spec published",
			"target", pub.Name(),
			"path", pubResult.Path,
			"format", pubResult.Format,
			"bytes", pubResult.BytesWritten,
		)
		if pubResult.Message != "" {
			slog.InfoContext(ctx, "Publisher output", "message", pubResult.Message)
		}
	}

	// Step 8: Output final result
	slog.InfoContext(ctx, "Generation complete")

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
	generateCmd.Flags().StringVarP(&generateOutputFormat, "output", "o", "",
		"output format (yaml or json, default: yaml)")
	generateCmd.Flags().StringVarP(&generateOutputDir, "output-dir", "d", "",
		"output directory for generated spec (default: project root)")
	generateCmd.Flags().BoolVar(&generateSkipPublish, "skip-publish", false,
		"skip publishing to remote platforms")
	generateCmd.Flags().StringVar(&generatePublishTarget, "publish-target", "",
		"publish target (readme). If empty, spec is only saved locally")
	generateCmd.Flags().BoolVar(&generatePublishOverwrite, "publish-overwrite", false,
		"overwrite existing spec on remote platform")
	generateCmd.Flags().BoolVar(&generateOverwriteOutput, "overwrite-output", false,
		"overwrite existing local spec file if it already exists")
	generateCmd.Flags().StringSliceVar(&generateProtoImportPaths, "proto-import-path", nil,
		"additional import paths for protoc (-I flags), can be specified multiple times")
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

	// Save enriched spec to file
	var data []byte
	if strings.ToLower(filepath.Ext(specFilePath)) == ".json" {
		data, err = result.MarshalJSON()
	} else {
		var yamlData any
		yamlData, err = result.MarshalYAML()
		if err == nil {
			data, err = yaml.Marshal(yamlData)
		}
	}
	if err != nil {
		return fmt.Errorf("failed to marshal enriched spec: %w", err)
	}

	if writeErr := os.WriteFile(specFilePath, data, 0o600); writeErr != nil {
		return fmt.Errorf("failed to write enriched spec: %w", writeErr)
	}

	slog.InfoContext(ctx, "OpenAPI spec enriched", "output", specFilePath)
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

// normalizeOutputFormat normalizes and validates the output format value.
// Accepts: "yaml", "yml", "YAML", "json", "JSON" -> returns "yaml" or "json".
// Returns an error for unsupported formats to avoid silent fallback behavior.
func normalizeOutputFormat(format string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(format))
	switch normalized {
	case "yaml", "yml":
		return outputFormatYAML, nil
	case "json":
		return outputFormatJSON, nil
	default:
		return "", fmt.Errorf("unsupported output format %q; allowed values are %q and %q", format, outputFormatYAML, outputFormatJSON)
	}
}

// copySpecToOutput copies the generated spec to the specified output directory.
// If overwrite is false and the destination file already exists, an error is returned.
func copySpecToOutput(srcPath, outputDir string, overwrite bool) error {
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

	// Check if destination file already exists
	if _, statErr := os.Stat(dstPath); statErr == nil {
		if !overwrite {
			return fmt.Errorf("destination file already exists: %s (use --overwrite-output to overwrite)", dstPath)
		}
		// Overwrite is allowed, continue
	} else if !os.IsNotExist(statErr) {
		return fmt.Errorf("failed to check destination file: %w", statErr)
	}

	// Create destination file (truncates if exists) with restrictive permissions
	dstFile, createErr := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if createErr != nil {
		return fmt.Errorf("failed to create destination file: %w", createErr)
	}
	defer dstFile.Close()

	// Copy content
	if _, copyErr := io.Copy(dstFile, srcFile); copyErr != nil {
		return fmt.Errorf("failed to copy file: %w", copyErr)
	}

	return nil
}

// resolveReadMeAPIKey resolves ReadMe API key from config or environment.
// Priority: 1) cfg.APIKey, 2) cfg.APIKeyEnv (or README_API_KEY as default)
func resolveReadMeAPIKey(cfg config.ReadMeConfig) string {
	// First check explicit config
	if cfg.APIKey != "" {
		return cfg.APIKey
	}
	// Then check environment variable
	envName := cfg.APIKeyEnv
	if envName == "" {
		envName = "README_API_KEY"
	}
	return os.Getenv(envName)
}
