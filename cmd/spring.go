// Package cmd contains all CLI commands for spec-forge.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// springCmd represents the spring command group
var springCmd = &cobra.Command{
	Use:   "spring",
	Short: "Spring framework specific commands",
	Long: `Commands for working with Spring (Java) projects.

These commands help you:
- Detect Spring project configuration
- Patch projects with springdoc dependencies
- Extract OpenAPI specs from Spring controllers`,
}

func init() {
	rootCmd.AddCommand(springCmd)
}

// springDetectCmd represents the spring detect command
var springDetectCmd = &cobra.Command{
	Use:   "detect [path]",
	Short: "Detect Spring project information",
	Long: `Analyze the current directory to detect Spring project type, build tool, and dependencies.

This command will identify:
- Build tool (Maven or Gradle)
- Spring Boot version
- springdoc-openapi dependency status`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}
		fmt.Printf("Detecting Spring project in %s...\n", path)
		fmt.Println("detect called - implementation coming soon")
		return nil
	},
}

var (
	patchDryRun bool
	patchForce  bool
)

// springPatchCmd represents the spring patch command
var springPatchCmd = &cobra.Command{
	Use:   "patch [path]",
	Short: "Add springdoc dependencies to Spring project",
	Long: `Add springdoc-openapi dependencies to the Spring project's build file.
Supports both Maven (pom.xml) and Gradle (build.gradle) projects.

This command will:
- Detect the build tool
- Add the appropriate springdoc dependency
- Optionally update existing dependencies`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}
		fmt.Printf("Patching Spring project in %s...\n", path)
		if patchDryRun {
			fmt.Println("Dry run mode - showing changes without modifying files")
		}
		if patchForce {
			fmt.Println("Force mode - will overwrite existing dependencies")
		}
		fmt.Println("patch called - implementation coming soon")
		return nil
	},
}

func init() {
	springCmd.AddCommand(springDetectCmd)
	springCmd.AddCommand(springPatchCmd)

	springPatchCmd.Flags().BoolVar(&patchDryRun, "dry-run", false, "show changes without modifying files")
	springPatchCmd.Flags().BoolVar(&patchForce, "force", false, "force overwrite existing dependencies")
}
