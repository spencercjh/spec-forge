// Package cmd contains all CLI commands for spec-forge.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// springCmd represents the spring command
var springCmd = &cobra.Command{
	Use:   "spring",
	Short: "Spring Boot specific commands",
	Long: `Spring Boot specific commands for detecting and patching
Spring Boot applications for OpenAPI spec generation.`,
}

// springDetectCmd represents the spring detect command
var springDetectCmd = &cobra.Command{
	Use:   "detect [path]",
	Short: "Detect Spring Boot application structure",
	Long: `Detect Spring Boot application structure and API endpoints.

This command analyzes a Spring Boot application and detects:
- Controller classes and their endpoints
- Request/Response DTOs
- Spring configuration
- Springdoc/Swagger annotations

Examples:
  spec-forge spring detect ./src/main/java
  spec-forge spring detect ./...`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}

		fmt.Printf("Detecting Spring Boot application structure in: %s\n", path)
		// TODO: Implement detection logic
		return nil
	},
}

// springPatchCmd represents the spring patch command
var springPatchCmd = &cobra.Command{
	Use:   "patch [path]",
	Short: "Patch Spring Boot application with OpenAPI annotations",
	Long: `Patch Spring Boot application with OpenAPI annotations.

This command analyzes a Spring Boot application and automatically adds
or updates OpenAPI annotations (Springdoc) to improve API documentation.

Examples:
  spec-forge spring patch ./src/main/java
  spec-forge spring patch --dry-run ./src/main/java
  spec-forge spring patch --add-springdoc-deps ./...`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")

		fmt.Printf("Patching Spring Boot application in: %s (dry-run: %v)\n", path, dryRun)
		// TODO: Implement patch logic
		return nil
	},
}

func init() {
	springPatchCmd.Flags().Bool("dry-run", false, "preview changes without modifying files")
}
