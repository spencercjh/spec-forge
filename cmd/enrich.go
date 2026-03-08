// Package cmd contains all CLI commands for spec-forge.
package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/spencercjh/spec-forge/internal/config"
	"github.com/spencercjh/spec-forge/internal/enricher"
	"github.com/spencercjh/spec-forge/internal/enricher/processor"
	"github.com/spencercjh/spec-forge/internal/enricher/provider"
)

var (
	enrichProvider        string
	enrichModel           string
	enrichLanguage        string
	enrichOutput          string
	enrichConcurrency     int
	enrichTimeout         time.Duration
	enrichCustomBaseURL   string
	enrichCustomAPIKey    string
	enrichCustomAPIKeyEnv string
)

// enrichCmd represents the enrich command
var enrichCmd = &cobra.Command{
	Use:   "enrich <spec-file>",
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

	// Determine provider
	prov := enrichProvider
	if prov == "" {
		prov = cfg.Enrich.Provider
	}
	if prov == "" {
		return errors.New("provider is required. Use --provider flag or configure in .spec-forge.yaml")
	}

	// Determine model
	model := enrichModel
	if model == "" {
		model = cfg.Enrich.Model
	}
	if model == "" {
		return errors.New("model is required. Use --model flag or configure in .spec-forge.yaml")
	}

	// Determine language
	lang := enrichLanguage
	if lang == "" {
		lang = cfg.Enrich.Language
	}
	if lang == "" {
		lang = "en"
	}

	// Create provider
	p, err := createProvider(prov, model, cfg.Enrich)
	if err != nil {
		return err
	}

	slog.InfoContext(ctx, "Enriching OpenAPI spec",
		"file", specFile,
		"provider", prov,
		"model", model,
		"language", lang,
	)

	// Load spec
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	spec, err := loader.LoadFromFile(specFile)
	if err != nil {
		return fmt.Errorf("failed to load spec: %w", err)
	}

	// Create enricher config
	customBaseURL := enrichCustomBaseURL
	if customBaseURL == "" {
		customBaseURL = cfg.Enrich.BaseURL
	}
	customAPIKeyEnv := enrichCustomAPIKeyEnv
	if customAPIKeyEnv == "" {
		customAPIKeyEnv = cfg.Enrich.APIKeyEnv
	}

	enricherCfg := enricher.Config{
		Provider:        prov,
		Model:           model,
		Language:        lang,
		Concurrency:     enrichConcurrency,
		Timeout:         enrichTimeout,
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

	// Determine output file
	outputFile := enrichOutput
	if outputFile == "" {
		outputFile = specFile // Overwrite input by default
	}

	// Save enriched spec to file
	var data []byte
	if filepath.Ext(outputFile) == ".json" {
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

	slog.InfoContext(ctx, "Enrichment complete", "output", outputFile)
	return nil
}

// createProvider creates a provider based on the provider type
func createProvider(providerType, model string, enrichCfg config.EnrichConfig) (provider.Provider, error) { //nolint:gocritic // copying config is acceptable
	// Determine baseURL: flag > config > default
	baseURL := enrichCustomBaseURL
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
		cfg.APIKey = getCustomAPIKey(enrichCfg)
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("API key not found. Set %s environment variable", getCustomAPIKeyEnv(enrichCfg))
		}
	}

	return provider.NewProvider(cfg)
}

func getCustomAPIKeyEnv(enrichCfg config.EnrichConfig) string { //nolint:gocritic // copying config is acceptable
	if enrichCustomAPIKeyEnv != "" {
		return enrichCustomAPIKeyEnv
	}
	if enrichCfg.APIKeyEnv != "" {
		return enrichCfg.APIKeyEnv
	}
	return "LLM_API_KEY"
}

func getCustomAPIKey(enrichCfg config.EnrichConfig) string { //nolint:gocritic // copying config is acceptable
	// First check explicit flag
	if enrichCustomAPIKey != "" {
		return enrichCustomAPIKey
	}
	// Then check config file
	if enrichCfg.APIKey != "" {
		return enrichCfg.APIKey
	}
	// Then check environment variable
	return os.Getenv(getCustomAPIKeyEnv(enrichCfg))
}

func init() {
	rootCmd.AddCommand(enrichCmd)

	enrichCmd.Flags().StringVar(&enrichProvider, "provider", "", "LLM provider (openai, anthropic, ollama, custom)")
	enrichCmd.Flags().StringVar(&enrichModel, "model", "", "LLM model name")
	enrichCmd.Flags().StringVar(&enrichLanguage, "language", "en", "Output language for descriptions")
	enrichCmd.Flags().StringVarP(&enrichOutput, "output", "o", "", "Output file (default: overwrite input)")
	enrichCmd.Flags().IntVar(&enrichConcurrency, "concurrency", 3, "Number of concurrent LLM calls")
	enrichCmd.Flags().DurationVar(&enrichTimeout, "timeout", 30*time.Second, "Timeout for single LLM call")
	enrichCmd.Flags().StringVar(&enrichCustomBaseURL, "custom-base-url", "", "Custom provider API URL")
	enrichCmd.Flags().StringVar(&enrichCustomAPIKey, "custom-api-key", "", "Custom provider API key (or use env var)")
	enrichCmd.Flags().StringVar(&enrichCustomAPIKeyEnv, "custom-api-key-env", "LLM_API_KEY", "Environment variable for custom API key")
}
