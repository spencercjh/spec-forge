/*
Copyright © 2026 Spencer Cjh <spencercjh@gmail.com>
*/
package cmd

import (
	"fmt"

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
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}
		fmt.Printf("Extracting OpenAPI spec from %s...\n", path)
		if extractStrict {
			fmt.Println("Strict mode enabled")
		}
		fmt.Println("extract called - implementation coming soon")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(extractCmd)

	extractCmd.Flags().BoolVar(&extractStrict, "strict", false, "fail if validation errors occur")
}
