// Package cmd contains all CLI commands for spec-forge.
package cmd

import (
	"log/slog"

	"github.com/spf13/cobra"
)

var extractStrict bool

// extractCmd represents the extract command
var extractCmd = &cobra.Command{
	Use:   "extract [path]",
	Short: "Extract OpenAPI spec from source code",
	Long: `Extract OpenAPI specification from the source code using framework-specific extractors.

Supports multiple frameworks including Spring (Java), Go frameworks, and more.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runExtract,
}

func runExtract(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	path := "."
	if len(args) > 0 {
		path = args[0]
	}
	slog.InfoContext(ctx, "Extracting OpenAPI spec", "path", path)
	if extractStrict {
		slog.InfoContext(ctx, "Strict mode enabled")
	}
	slog.InfoContext(ctx, "extract called - implementation coming soon")
	return nil
}

func init() {
	rootCmd.AddCommand(extractCmd)

	extractCmd.Flags().BoolVar(&extractStrict, "strict", false, "fail if validation errors occur")
}
