// Package cmd contains all CLI commands for spec-forge.
package cmd

import (
	"log/slog"

	"github.com/spf13/cobra"
)

var (
	publishFormat string
	publishOutput string
)

// publishCmd represents the publish command
var publishCmd = &cobra.Command{
	Use:   "publish [spec-file]",
	Short: "Publish OpenAPI spec to target platform",
	Long: `Publish OpenAPI specification to local files or external platforms.

Supports:
- Local file system (YAML/JSON)
- Postman
- Apifox
- Swagger UI`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPublish,
}

func runPublish(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	specFile := "openapi.yaml"
	if len(args) > 0 {
		specFile = args[0]
	}
	slog.InfoContext(ctx, "Publishing spec", "file", specFile, "output", publishOutput)
	slog.InfoContext(ctx, "publish called - implementation coming soon")
	return nil
}

func init() {
	rootCmd.AddCommand(publishCmd)

	publishCmd.Flags().StringVarP(&publishFormat, "format", "f", "yaml", "output format (yaml or json)")
	publishCmd.Flags().StringVarP(&publishOutput, "output", "o", "./openapi", "output directory")
}
