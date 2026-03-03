// Package cmd contains all CLI commands for spec-forge.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// publishCmd represents the publish command
var publishCmd = &cobra.Command{
	Use:   "publish [spec-file]",
	Short: "Publish OpenAPI specification to registry",
	Long: `Publish OpenAPI specification to a registry.

This command publishes your OpenAPI specification to a supported registry
such as SwaggerHub, Backstage, or other API documentation platforms.

Examples:
  spec-forge publish openapi.yaml
  spec-forge publish --registry swaggerhub openapi.yaml
  spec-forge publish -r backstage -t production openapi.yaml`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		specFile := "openapi.yaml"
		if len(args) > 0 {
			specFile = args[0]
		}

		fmt.Printf("Publishing specification: %s\n", specFile)
		// TODO: Implement publish logic
		return nil
	},
}

func init() {
	// Flags would be added here if needed
}
