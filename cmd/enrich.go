// Package cmd contains all CLI commands for spec-forge.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
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
	p, err := createProvider(prov, model)
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
	enricherCfg := enricher.Config{
		Provider:        prov,
		Model:           model,
		Language:        lang,
		Concurrency:     enrichConcurrency,
		Timeout:         enrichTimeout,
		CustomBaseURL:   enrichCustomBaseURL,
		CustomAPIKeyEnv: enrichCustomAPIKeyEnv,
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
		if partialErr, ok := err.(*processor.PartialEnrichmentError); ok {
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

	// Save result
	if err := saveSpec(result, outputFile); err != nil {
		return fmt.Errorf("failed to save spec: %w", err)
	}

	slog.InfoContext(ctx, "Enrichment complete", "output", outputFile)
	return nil
}

// createProvider creates a provider based on the provider type
func createProvider(providerType, model string) (provider.Provider, error) {
	switch providerType {
	case "openai":
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			return nil, errors.New("OPENAI_API_KEY environment variable not set")
		}
		return provider.NewOpenAIProvider(apiKey, model)

	case "anthropic":
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			return nil, errors.New("ANTHROPIC_API_KEY environment variable not set")
		}
		return provider.NewAnthropicProvider(apiKey, model)

	case "ollama":
		baseURL := enrichCustomBaseURL
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
		return provider.NewOllamaProvider(baseURL, model)

	case "custom":
		apiKey := getCustomAPIKey()
		if apiKey == "" {
			return nil, fmt.Errorf("API key not found. Set %s environment variable", getCustomAPIKeyEnv())
		}
		return provider.NewCustomProvider(provider.CustomProviderConfig{
			BaseURL: enrichCustomBaseURL,
			APIKey:  apiKey,
			Model:   model,
		})

	default:
		return nil, fmt.Errorf("unknown provider: %s", providerType)
	}
}

func getCustomAPIKeyEnv() string {
	if enrichCustomAPIKeyEnv != "" {
		return enrichCustomAPIKeyEnv
	}
	return "LLM_API_KEY"
}

func getCustomAPIKey() string {
	// First check explicit flag
	if enrichCustomAPIKey != "" {
		return enrichCustomAPIKey
	}
	// Then check environment variable
	return os.Getenv(getCustomAPIKeyEnv())
}

// saveSpec saves the spec to a file
func saveSpec(spec *openapi3.T, path string) error {
	// Validate before saving
	ctx := context.Background()
	if err := spec.Validate(ctx); err != nil {
		slog.Warn("Spec validation warning", "error", err)
	}

	// Determine format from extension
	var data []byte
	var err error
	switch {
	case len(path) > 5 && path[len(path)-5:] == ".json":
		data, err = spec.MarshalJSON()
	default:
		var yamlData any
		yamlData, err = spec.MarshalYAML()
		if err == nil {
			data, err = yaml.Marshal(yamlData)
		}
	}

	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
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
