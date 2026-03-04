// Package cmd contains all CLI commands for spec-forge.
package cmd

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var (
	enrichProvider string
	enrichModel    string
	enrichEnabled  bool
)

// enrichCmd represents the enrich command
var enrichCmd = &cobra.Command{
	Use:   "enrich [spec-file]",
	Short: "Enrich OpenAPI spec with AI-generated descriptions",
	Long: `Enrich OpenAPI specification by using LLM to generate missing descriptions
for APIs and fields.

Supports multiple LLM providers: OpenAI, Anthropic, Ollama, and Zhipu GLM.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runEnrich,
}

func runEnrich(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	specFile := "openapi.yaml"
	if len(args) > 0 {
		specFile = args[0]
	}
	slog.InfoContext(ctx, "Enriching OpenAPI spec", "file", specFile)
	if !enrichEnabled {
		slog.InfoContext(ctx, "Enrichment disabled")
		return nil
	}
	slog.InfoContext(ctx, "Using provider", "provider", enrichProvider, "model", enrichModel)
	slog.InfoContext(ctx, "enrich called - implementation coming soon")
	return nil
}

func init() {
	rootCmd.AddCommand(enrichCmd)

	enrichCmd.Flags().StringVar(&enrichProvider, "provider", "", "LLM provider (openai, anthropic, ollama, zhipu)")
	enrichCmd.Flags().StringVar(&enrichModel, "model", "", "LLM model name")
	enrichCmd.Flags().BoolVar(&enrichEnabled, "enabled", true, "enable/disable enrichment")

	if err := enrichCmd.MarkFlagRequired("provider"); err != nil {
		slog.Error("failed to mark provider flag required", "error", err)
		os.Exit(1)
	}
}
