/*
Copyright © 2026 Spencer Cjh <spencercjh@gmail.com>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate [path]",
	Short: "Generate OpenAPI spec from source code",
	Long: `Generate OpenAPI specification by running the complete pipeline:
extract -> enrich -> publish

This is the main command that orchestrates the entire workflow.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}
		fmt.Printf("Generating OpenAPI spec from %s...\n", path)
		fmt.Println("generate called - implementation coming soon")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
}
