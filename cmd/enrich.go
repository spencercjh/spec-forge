// Package cmd contains all CLI commands for spec-forge.
package cmd

import (
	"errors"
	"fmt"
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
)

// enrichCmd represents the enrich command
var enrichCmd = &cobra.Command{
	Use:   "enrich \u003cspec-file\u003e",
	Short: "Enrich OpenAPI spec with AI-generated descriptions",
	Long: `Enrich OpenAPI specification by using LLM to generate missing descriptions
for APIs and fields.

Supports multiple LLM providers: OpenAI, Anthropic, Ollama, and custom OpenAI-compatible services.

Examples:
  # Enrich with OpenAI
  spec-forge enrich openapi.yaml --provider openai --model gpt-4o

  # Enrich with Chinese descriptions
  spec-forge enrich openapi.yaml --provider openai --language zh

  # Use custom internal AI service
  spec-forge enrich openapi.yaml \
    --provider custom \
    --custom-base-url https://ai.company.com/v1 \
    --custom-api-key-env COMPANY_AI_KEY`,
	Args: cobra.ExactArgs(1),
	RunE: runEnrich,
}

//nolint:gocyclo // CLI command runner with many branches
func runEnrich(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	specFile := args[0]

	cfg := config.Get()

	// Get flag values from command (isolated per command instance)
	//nolint:errcheck // flags are bound at command creation, errors not possible
	providerFlag, _ := cmd.Flags().GetString("provider")
	//nolint:errcheck
	modelFlag, _ := cmd.Flags().GetString("model")
	//nolint:errcheck
	languageFlag, _ := cmd.Flags().GetString("language")
	//nolint:errcheck
	concurrencyFlag, _ := cmd.Flags().GetInt("concurrency")
	//nolint:errcheck
	timeoutFlag, _ := cmd.Flags().GetDuration("timeout")
	//nolint:errcheck
	customBaseURLFlag, _ := cmd.Flags().GetString("custom-base-url")
	//nolint:errcheck
	customAPIKeyEnvFlag, _ := cmd.Flags().GetString("custom-api-key-env")
	//nolint:errcheck
	noStreamFlag, _ := cmd.Flags().GetBool("no-stream")
	//nolint:errcheck
	forceFlag, _ := cmd.Flags().GetBool("force")

	// Determine provider
	prov := providerFlag
	if prov == "" {
		prov = cfg.Enrich.Provider
	}
	if prov == "" {
		return errors.New("provider is required. Use --provider flag or configure in .spec-forge.yaml")
	}

	// Determine model
	model := modelFlag
	if model == "" {
		model = cfg.Enrich.Model
	}
	if model == "" {
		return errors.New("model is required. Use --model flag or configure in .spec-forge.yaml")
	}

	// Determine language
	lang := languageFlag
	if lang == "" {
		lang = cfg.Enrich.Language
	}
	if lang == "" {
		lang = "en"
	}

	// Create provider
	p, err := createProvider(prov, model, cfg.Enrich, customBaseURLFlag, customAPIKeyEnvFlag)
	if err != nil {
		return err
	}

	cli.Statusf(os.Stderr, "Enriching OpenAPI spec (provider: %s, model: %s, language: %s)", prov, model, lang)

	// Load spec
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	spec, err := loader.LoadFromFile(specFile)
	if err != nil {
		return fmt.Errorf("failed to load spec: %w", err)
	}

	// Determine output file
	//nolint:errcheck
	outputFlag, _ := cmd.Flags().GetString("output")
	outputFile := outputFlag
	if outputFile == "" {
		outputFile = specFile // Overwrite input by default
	}

	// Create enricher config
	customBaseURL := customBaseURLFlag
	if customBaseURL == "" {
		customBaseURL = cfg.Enrich.BaseURL
	}
	customAPIKeyEnv := customAPIKeyEnvFlag
	if customAPIKeyEnv == "" {
		customAPIKeyEnv = cfg.Enrich.APIKeyEnv
	}

	enricherCfg := enricher.Config{
		Provider:        prov,
		Model:           model,
		Language:        lang,
		Concurrency:     concurrencyFlag,
		Timeout:         timeoutFlag,
		CustomBaseURL:   customBaseURL,
		CustomAPIKeyEnv: customAPIKeyEnv,
	}
	enricherCfg = enricherCfg.MergeWithDefaults()

	// Create enricher
	e, err := enricher.NewEnricher(enricherCfg, p)
	if err != nil {
		return fmt.Errorf("failed to create enricher: %w", err)
	}

	// Enrich
	streamEnabled := !noStreamFlag // Streaming enabled by default
	result, err := e.Enrich(ctx, spec, &enricher.EnrichOptions{
		Language: lang,
		Stream:   &streamEnabled,
		Force:    forceFlag,
	})
	if err != nil {
		// Check if partial enrichment
		if partialErr, ok := errors.AsType[*processor.PartialEnrichmentError](err); ok {
			slog.WarnContext(ctx, "Partial enrichment completed",
				"failed_batches", partialErr.FailedBatches,
				"total_batches", partialErr.TotalBatches,
			)
			cli.Statusf(os.Stderr, "Partial enrichment: %d/%d batches succeeded", partialErr.TotalBatches-partialErr.FailedBatches, partialErr.TotalBatches)
		} else {
			return fmt.Errorf("enrichment failed: %w", err)
		}
	}

	// Save enriched spec to file
	var data []byte
	if strings.ToLower(filepath.Ext(outputFile)) == ".json" {
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

	// Ensure output directory exists
	if dir := filepath.Dir(outputFile); dir != "" && dir != "." {
		if mkdirErr := os.MkdirAll(dir, 0o755); mkdirErr != nil {
			return fmt.Errorf("failed to create output directory %q: %w", dir, mkdirErr)
		}
	}

	if writeErr := os.WriteFile(outputFile, data, 0o600); writeErr != nil {
		return fmt.Errorf("failed to write enriched spec: %w", writeErr)
	}

	cli.Successf(os.Stderr, "Enrichment complete: %s", outputFile)
	return nil
}

// createProvider creates a provider based on the provider type
func createProvider(providerType, model string, enrichCfg config.EnrichConfig, customBaseURL, customAPIKeyEnv string) (provider.Provider, error) { //nolint:gocritic // copying config is acceptable
	// Determine baseURL: flag > config > default
	baseURL := customBaseURL
	if baseURL == "" {
		baseURL = enrichCfg.BaseURL
	}

	cfg := provider.Config{
		Provider: providerType,
		Model:    model,
		BaseURL:  baseURL,
	}

	// Get API key based on provider type
	switch providerType {
	case "openai":
		cfg.APIKey = os.Getenv("OPENAI_API_KEY")
		if cfg.APIKey == "" {
			return nil, errors.New("OPENAI_API_KEY environment variable not set")
		}
	case "anthropic":
		cfg.APIKey = os.Getenv("ANTHROPIC_API_KEY")
		if cfg.APIKey == "" {
			return nil, errors.New("ANTHROPIC_API_KEY environment variable not set")
		}
	case "custom":
		cfg.APIKey = getCustomAPIKey(&enrichCfg, customAPIKeyEnv)
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("API key not found. Set %s environment variable", getCustomAPIKeyEnv(&enrichCfg, customAPIKeyEnv))
		}
	}

	return provider.NewProvider(cfg)
}

func getCustomAPIKeyEnv(enrichCfg *config.EnrichConfig, flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if enrichCfg.APIKeyEnv != "" {
		return enrichCfg.APIKeyEnv
	}
	return "LLM_API_KEY"
}

func getCustomAPIKey(enrichCfg *config.EnrichConfig, flagValue string) string {
	// Priority: env > config
	// First check environment variable
	if apiKey := os.Getenv(getCustomAPIKeyEnv(enrichCfg, flagValue)); apiKey != "" {
		return apiKey
	}
	// Then check config file
	return enrichCfg.APIKey
}

// newEnrichCmd creates a new enrich command instance for testing.
func newEnrichCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "enrich \u003cspec-file\u003e",
		Short: "Enrich OpenAPI spec with AI-generated descriptions",
		Long: `Enrich OpenAPI specification by using LLM to generate missing descriptions
for APIs and fields.

Supports multiple LLM providers: OpenAI, Anthropic, Ollama, and custom OpenAI-compatible services.

Examples:
  # Enrich with OpenAI
  spec-forge enrich openapi.yaml --provider openai --model gpt-4o

  # Enrich with Chinese descriptions
  spec-forge enrich openapi.yaml --provider openai --language zh

  # Use custom internal AI service
  spec-forge enrich openapi.yaml \
    --provider custom \
    --custom-base-url https://ai.company.com/v1 \
    --custom-api-key-env COMPANY_AI_KEY`,
		Args: cobra.ExactArgs(1),
		RunE: runEnrich,
	}

	c.Flags().String("provider", "", "LLM provider (openai, anthropic, ollama, custom)")
	c.Flags().String("model", "", "LLM model name")
	c.Flags().String("language", "en", "Output language for descriptions")
	c.Flags().StringP("output", "o", "", "Output file (default: overwrite input)")
	c.Flags().Int("concurrency", 3, "Max concurrent LLM calls (only effective with --no-stream)")
	c.Flags().Duration("timeout", 30*time.Second, "Timeout for single LLM call")
	c.Flags().String("custom-base-url", "", "Custom provider API URL")
	c.Flags().String("custom-api-key-env", "LLM_API_KEY", "Environment variable for custom API key")
	c.Flags().Bool("no-stream", false, "Disable streaming to enable concurrent processing (faster, but no real-time output)")
	c.Flags().Bool("force", false, "Force regeneration of all descriptions, ignoring existing ones")

	registerCompletion(c, "provider", []string{"openai", "anthropic", "ollama", "custom"})
	registerCompletion(c, "language", []string{"en", "zh"})

	return c
}

// enrich command flag variables for global rootCmd only
var (
	enrichProvider        string
	enrichModel           string
	enrichLanguage        string
	enrichOutput          string
	enrichConcurrency     int
	enrichTimeout         time.Duration
	enrichCustomBaseURL   string
	enrichCustomAPIKeyEnv string
	enrichNoStream        bool
	enrichForce           bool
)

func init() {
	rootCmd.AddCommand(enrichCmd)

	enrichCmd.Flags().StringVar(&enrichProvider, "provider", "", "LLM provider (openai, anthropic, ollama, custom)")
	enrichCmd.Flags().StringVar(&enrichModel, "model", "", "LLM model name")
	enrichCmd.Flags().StringVar(&enrichLanguage, "language", "en", "Output language for descriptions")
	enrichCmd.Flags().StringVarP(&enrichOutput, "output", "o", "", "Output file (default: overwrite input)")
	enrichCmd.Flags().IntVar(&enrichConcurrency, "concurrency", 3, "Max concurrent LLM calls (only with --no-stream)")
	enrichCmd.Flags().DurationVar(&enrichTimeout, "timeout", 30*time.Second, "Timeout for single LLM call")
	enrichCmd.Flags().StringVar(&enrichCustomBaseURL, "custom-base-url", "", "Custom provider API URL")
	enrichCmd.Flags().StringVar(&enrichCustomAPIKeyEnv, "custom-api-key-env", "LLM_API_KEY", "Environment variable for custom API key")
	enrichCmd.Flags().BoolVar(&enrichNoStream, "no-stream", false, "Disable streaming output to enable concurrent LLM calls (faster)")
	enrichCmd.Flags().BoolVar(&enrichForce, "force", false, "Force regeneration of all descriptions, ignoring existing ones")

	registerCompletion(enrichCmd, "provider", []string{"openai", "anthropic", "ollama", "custom"})
	registerCompletion(enrichCmd, "language", []string{"en", "zh"})
}
