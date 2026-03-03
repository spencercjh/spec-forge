// Package cmd contains all CLI commands for spec-forge.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	enrichProvider string
	enrichModel    string
	enrichAPIKey   string
	enrichEnabled  bool
)

// enrichCmd represents the enrich command
var enrichCmd = &cobra.Command{
	Use:   "enrich [spec-file]",
	Short: "Enrich OpenAPI specification using LLM",
	Long: `Enrich OpenAPI specification using LLM.

This command takes an existing OpenAPI specification and uses an LLM
to enhance the descriptions, add examples, and improve documentation.

Supported providers: openai, anthropic, azure

Examples:
  spec-forge enrich openapi.yaml
  spec-forge enrich --provider openai --model gpt-4 openapi.yaml
  spec-forge enrich --provider anthropic --model claude-3 openapi.yaml`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		specFile := "openapi.yaml"
		if len(args) > 0 {
			specFile = args[0]
		}

		cfg := getConfig()
		// Command-line flags override config
		if enrichProvider != "" {
			cfg.Enrich.Provider = enrichProvider
		}
		if enrichModel != "" {
			cfg.Enrich.Model = enrichModel
		}
		if enrichAPIKey != "" {
			cfg.Enrich.APIKey = enrichAPIKey
		}
		if cmd.Flags().Changed("enabled") {
			cfg.Enrich.Enabled = enrichEnabled
		}

		fmt.Printf("Enriching specification: %s\n", specFile)
		fmt.Printf("  Provider: %s\n", cfg.Enrich.Provider)
		fmt.Printf("  Model: %s\n", cfg.Enrich.Model)
		fmt.Printf("  Enabled: %v\n", cfg.Enrich.Enabled)
		// TODO: Implement enrichment logic
		return nil
	},
}

func init() {
	enrichCmd.Flags().StringVar(&enrichProvider, "provider", "", "LLM provider (openai, anthropic, azure)")
	enrichCmd.Flags().StringVar(&enrichModel, "model", "", "LLM model to use")
	enrichCmd.Flags().StringVar(&enrichAPIKey, "api-key", "", "API key for the LLM provider")
	enrichCmd.Flags().BoolVar(&enrichEnabled, "enabled", true, "enable/disable LLM enrichment")
}
