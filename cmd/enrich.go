/*
Copyright © 2026 Spencer Cjh <spencercjh@gmail.com>
*/
package cmd

import (
	"fmt"
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
	RunE: func(cmd *cobra.Command, args []string) error {
		specFile := "openapi.yaml"
		if len(args) > 0 {
			specFile = args[0]
		}
		fmt.Printf("Enriching OpenAPI spec from %s...\n", specFile)
		if !enrichEnabled {
			fmt.Println("Enrichment disabled")
			return nil
		}
		fmt.Printf("Provider: %s, Model: %s\n", enrichProvider, enrichModel)
		fmt.Println("enrich called - implementation coming soon")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(enrichCmd)

	enrichCmd.Flags().StringVar(&enrichProvider, "provider", "", "LLM provider (openai, anthropic, ollama, zhipu)")
	enrichCmd.Flags().StringVar(&enrichModel, "model", "", "LLM model name")
	enrichCmd.Flags().BoolVar(&enrichEnabled, "enabled", true, "enable/disable enrichment")

	if err := enrichCmd.MarkFlagRequired("provider"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking provider flag required: %v\n", err)
		os.Exit(1)
	}
}
