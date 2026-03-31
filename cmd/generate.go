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

	"github.com/spencercjh/spec-forge/internal/cli"
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

	slog.DebugContext(ctx, "Generating OpenAPI spec", "path", path)

	// Get all flag values from command (isolated per command instance)
	//nolint:errcheck // flags are bound at command creation, errors not possible
	keepPatched, _ := cmd.Flags().GetBool("keep-patched")
	//nolint:errcheck
	skipValidate, _ := cmd.Flags().GetBool("skip-validate")
	//nolint:errcheck
	timeout, _ := cmd.Flags().GetDuration("timeout")
	//nolint:errcheck
	skipEnrich, _ := cmd.Flags().GetBool("skip-enrich")
	//nolint:errcheck
	language, _ := cmd.Flags().GetString("language")
	//nolint:errcheck
	outputDirFlag, _ := cmd.Flags().GetString("output-dir")
	//nolint:errcheck
	outputFormatFlag, _ := cmd.Flags().GetString("output")
	//nolint:errcheck
	skipPublish, _ := cmd.Flags().GetBool("skip-publish")
	//nolint:errcheck
	publishTarget, _ := cmd.Flags().GetString("publish-target")
	//nolint:errcheck
	publishOverwrite, _ := cmd.Flags().GetBool("publish-overwrite")
	//nolint:errcheck
	overwriteOutput, _ := cmd.Flags().GetBool("overwrite-output")
	//nolint:errcheck
	protoImportPaths, _ := cmd.Flags().GetStringSlice("proto-import-path")

	// Step 1: Detect framework - try all registered extractors
	extractorImpl, info, err := builtin.DetectFramework(path)
	if err != nil {
		return errWrap("no supported framework detected", err)
	}

	cli.Statusf(os.Stderr, "Detected %s project (tool: %s, build: %s)", extractorImpl.Name(), info.BuildTool, info.BuildFilePath)

	// Step 2: Patch project if needed
	patchOpts := &extractor.PatchOptions{
		KeepPatched: keepPatched,
	}

	patchResult, err := extractorImpl.Patch(path, patchOpts)
	if err != nil {
		return errWrap("patch failed", err)
	}

	// Step 3: If we patched the file and should restore later, defer the restore
	if !keepPatched && patchResult.OriginalContent != "" {
		defer func() {
			slog.DebugContext(ctx, "Restoring original build file")
			if restoreErr := extractorImpl.Restore(patchResult.BuildFilePath, patchResult.OriginalContent); restoreErr != nil {
				slog.WarnContext(ctx, "failed to restore original file", "error", restoreErr)
			} else {
				slog.DebugContext(ctx, "Original build file restored")
			}
		}()
	}

	if patchResult.DependencyAdded {
		cli.Successf(os.Stderr, "Dependencies added temporarily")
	}
	if patchResult.PluginAdded {
		cli.Successf(os.Stderr, "Plugin added temporarily")
	}
	if patchResult.SpringBootConfigured {
		cli.Successf(os.Stderr, "spring-boot-maven-plugin configured with start/stop goals")
	}

	// Step 4: Generate OpenAPI spec

	// Determine output directory
	// Precedence: flag > config > default (project root)
	outputDir := outputDirFlag
	if outputDir == "" {
		outputDir = config.Get().Output.Dir
	}
	if outputDir == "" {
		outputDir = path // Default to project root
	}

	// Determine output format
	outputFormat := outputFormatFlag
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
		Timeout:          timeout,
		SkipTests:        true,
		ProtoImportPaths: protoImportPaths,
	}

	genResult, err := extractorImpl.Generate(ctx, path, info, genOpts)
	if err != nil {
		return errWrap("generation failed", err)
	}

	cli.Statusf(os.Stderr, "OpenAPI spec generated: %s (%s)", genResult.SpecFilePath, genResult.Format)

	// Step 5: Validate the generated spec
	if !skipValidate {
		v := validator.NewValidator()
		valResult, valErr := v.Validate(ctx, genResult.SpecFilePath)
		if valErr != nil {
			return errWrap("validation error", valErr)
		}

		if !valResult.Valid {
			cli.Errorf(os.Stderr, "OpenAPI spec validation failed")
			for _, validationErr := range valResult.Errors {
				cli.Errorf(os.Stderr, "  - %s", validationErr)
			}
			return errors.New("generated OpenAPI spec is invalid")
		}

		cli.Successf(os.Stderr, "OpenAPI spec validated")
	} else {
		cli.Skipf(os.Stderr, "Validation skipped")
	}

	// Step 6: Enrich with AI (optional)
	cfg := config.Get()
	if !skipEnrich && cfg.Enrich.Enabled && cfg.Enrich.Provider != "" && cfg.Enrich.Model != "" {
		if enrichErr := enrichGeneratedSpec(ctx, genResult.SpecFilePath, cfg, language); enrichErr != nil {
			// Log warning but don't fail - enrichment is optional
			slog.WarnContext(ctx, "Enrichment failed (non-fatal)", "error", enrichErr)
		}
	} else {
		cli.Skipf(os.Stderr, "Enrichment skipped")
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
		if err := copySpecToOutput(genResult.SpecFilePath, outputDir, overwriteOutput); err != nil {
			return errWrap("failed to copy spec to output directory", err)
		}
		cli.Successf(os.Stderr, "Spec saved: %s", targetPath)
		// Update genResult.SpecFilePath to point to the copied file for publishing
		genResult.SpecFilePath = targetPath
	} else {
		cli.Successf(os.Stderr, "Spec saved: %s", genResult.SpecFilePath)
	}

	// Step 8: Publish the spec to remote platforms (optional)
	if !skipPublish && publishTarget != "" {
		// Create publisher using factory
		pub, err := publisher.NewPublisher(publishTarget)
		if err != nil {
			return errWrap("failed to create publisher", err)
		}

		// Build publish options
		pubOpts := &publisher.PublishOptions{
			OutputPath: genResult.SpecFilePath,
			Format:     outputFormat,
			Overwrite:  publishOverwrite,
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

		cli.Successf(os.Stderr, "Spec published to %s", pub.Name())
		if pubResult.Message != "" {
			cli.Statusf(os.Stderr, "%s", pubResult.Message)
		}
	}

	// Step 8: Output final result
	cli.Successf(os.Stderr, "Generation complete")

	return nil
}

// newGenerateCmd creates a new generate command instance for testing.
func newGenerateCmd() *cobra.Command {
	c := &cobra.Command{
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

	c.Flags().Bool("keep-patched", false,
		"keep the patched pom.xml/build.gradle (default: restore original after extraction)")
	c.Flags().Bool("skip-validate", false,
		"skip validation of the generated OpenAPI spec")
	c.Flags().Duration("timeout", 5*time.Minute,
		"timeout for Maven/Gradle commands")
	c.Flags().Bool("skip-enrich", false,
		"skip AI enrichment of the generated OpenAPI spec")
	c.Flags().String("language", "en",
		"language for AI-generated descriptions (e.g., en, zh)")
	c.Flags().StringP("output", "o", "",
		"output format (yaml or json; defaults to yaml if not specified in config)")
	c.Flags().StringP("output-dir", "d", "",
		"output directory for generated spec (default: project root)")
	c.Flags().Bool("skip-publish", false,
		"skip publishing to remote platforms")
	c.Flags().String("publish-target", "",
		"publish target (readme). If empty, spec is only saved locally")
	c.Flags().Bool("publish-overwrite", false,
		"overwrite existing spec on remote platform")
	c.Flags().Bool("overwrite-output", false,
		"overwrite existing local spec file if it already exists")
	c.Flags().StringSlice("proto-import-path", nil,
		"additional import paths for protoc (-I flags), can be specified multiple times")

	registerCompletion(c, "output", []string{"yaml", "json"})
	registerCompletion(c, "language", []string{"en", "zh"})
	registerCompletion(c, "publish-target", []string{"readme"})

	return c
}

// generate command flag variables for global rootCmd only
var (
	generateKeepPatched      bool
	generateSkipValidate     bool
	generateTimeout          time.Duration
	generateSkipEnrich       bool
	generateLanguage         string
	generateOutputDir        string
	generateOutputFormat     string
	generateSkipPublish      bool
	generatePublishTarget    string
	generatePublishOverwrite bool
	generateOverwriteOutput  bool
	generateProtoImportPaths []string
)

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

	registerCompletion(generateCmd, "output", []string{"yaml", "json"})
	registerCompletion(generateCmd, "language", []string{"en", "zh"})
	registerCompletion(generateCmd, "publish-target", []string{"readme"})
}

// enrichGeneratedSpec enriches the generated spec with AI-generated descriptions
func enrichGeneratedSpec(ctx context.Context, specFilePath string, cfg *config.Config, language string) error {
	cli.Statusf(os.Stderr, "Enriching OpenAPI spec with AI descriptions...")

	// Determine language
	lang := language
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

	cli.Successf(os.Stderr, "OpenAPI spec enriched: %s", specFilePath)
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
