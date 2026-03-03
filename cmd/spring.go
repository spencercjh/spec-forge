// Package cmd contains all CLI commands for spec-forge.
package cmd

import (
	"fmt"

	"github.com/spencercjh/spec-forge/internal/extractor"
	"github.com/spencercjh/spec-forge/internal/extractor/spring"
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
	RunE: runSpringDetect,
}

func runSpringDetect(_ *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	detector := spring.NewDetector()
	info, err := detector.Detect(path)
	if err != nil {
		return fmt.Errorf("detection failed: %w", err)
	}

	// Print human-readable output
	printProjectInfo(info)
	return nil
}

func printProjectInfo(info *extractor.ProjectInfo) {
	fmt.Println("Spring Project Detection Results")
	fmt.Println("================================")
	fmt.Printf("Build Tool:           %s\n", info.BuildTool)
	fmt.Printf("Build File:           %s\n", info.BuildFilePath)
	fmt.Printf("Spring Boot:          %s\n", info.SpringBootVersion)

	if info.HasSpringdocDeps {
		fmt.Printf("springdoc Dependency: ✅ Present (%s)\n", info.SpringdocVersion)
	} else {
		fmt.Println("springdoc Dependency: ❌ Not found")
	}

	if info.HasSpringdocPlugin {
		fmt.Println("springdoc Plugin:     ✅ Configured")
	} else {
		fmt.Println("springdoc Plugin:     ❌ Not configured")
	}
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
	RunE: runSpringPatch,
}

func runSpringPatch(_ *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	// spring patch defaults to keeping the patched file (KeepPatched=true)
	// unlike generate command which defaults to restoring
	opts := &extractor.PatchOptions{
		DryRun:      patchDryRun,
		Force:       patchForce,
		KeepPatched: true, // spring patch keeps changes by default
	}

	patcher := spring.NewPatcher()
	result, err := patcher.Patch(path, opts)
	if err != nil {
		return fmt.Errorf("patch failed: %w", err)
	}

	// Print results
	if opts.DryRun {
		fmt.Println("Dry run mode - showing changes without modifying files")
	}

	fmt.Printf("Build file: %s\n", result.BuildFilePath)

	if result.DependencyAdded {
		fmt.Println("✅ springdoc dependency will be added")
	} else {
		fmt.Println("⏭️  springdoc dependency already present")
	}

	if result.PluginAdded {
		fmt.Println("✅ springdoc plugin will be added")
	} else {
		fmt.Println("⏭️  springdoc plugin already configured")
	}

	if !opts.DryRun && (result.DependencyAdded || result.PluginAdded) {
		fmt.Println("\nPatch applied successfully!")
		fmt.Println("Note: Build file format may differ from original due to XML serialization.")
		fmt.Println("Use 'spec-forge generate' to extract specs while preserving original file format.")
	} else if !result.DependencyAdded && !result.PluginAdded {
		fmt.Println("\nNo changes needed.")
	}

	return nil
}

func init() {
	springCmd.AddCommand(springDetectCmd)
	springCmd.AddCommand(springPatchCmd)

	springPatchCmd.Flags().BoolVar(&patchDryRun, "dry-run", false, "show changes without modifying files")
	springPatchCmd.Flags().BoolVar(&patchForce, "force", false, "force overwrite existing dependencies")
}
