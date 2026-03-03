// Package cmd contains all CLI commands for spec-forge.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	extractStrict bool
)

// extractCmd represents the extract command
var extractCmd = &cobra.Command{
	Use:   "extract [path]",
	Short: "Extract API endpoints from source code",
	Long: `Extract API endpoints from source code.

This command parses your source code and extracts API endpoint information
including paths, methods, parameters, and response types.

Examples:
  spec-forge extract ./src/main/java
  spec-forge extract --strict ./...
  spec-forge extract -o endpoints.json ./...`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}

		cfg := getConfig()
		// Command-line flag overrides config
		if extractStrict {
			cfg.Extract.Strict = true
		}

		fmt.Printf("Extracting API endpoints from: %s (strict: %v)\n", path, cfg.Extract.Strict)
		// TODO: Implement extraction logic
		return nil
	},
}

func init() {
	extractCmd.Flags().BoolVar(&extractStrict, "strict", false, "enable strict mode for extraction (fail on warnings)")
}
