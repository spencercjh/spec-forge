// Package cmd contains all CLI commands for spec-forge.
package cmd

import (
	"fmt"

	"github.com/spencercjh/spec-forge/internal/extractor"
	"github.com/spencercjh/spec-forge/internal/extractor/spring"
	"github.com/spf13/cobra"
)

var (
	// generateKeepPatched controls whether to keep the patched pom/build file
	// Default is false (restore original) for generate command
	generateKeepPatched bool
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate [path]",
	Short: "Generate OpenAPI spec from source code",
	Long: `Generate OpenAPI specification by running the complete pipeline:
detect -> patch -> extract -> restore (optional)

This is the main command that orchestrates the entire workflow.

By default, the original pom.xml/build.gradle is restored after extraction
to preserve your project's formatting. Use --keep-patched to keep the changes.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runGenerate,
}

func runGenerate(_ *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	fmt.Printf("Generating OpenAPI spec from %s...\n", path)

	// Step 1: Detect project
	detector := spring.NewDetector()
	info, err := detector.Detect(path)
	if err != nil {
		return fmt.Errorf("detection failed: %w", err)
	}

	fmt.Printf("Detected %s project (%s)\n", info.BuildTool, info.BuildFilePath)

	// Step 2: Patch project if needed
	patcher := spring.NewPatcher()
	opts := &extractor.PatchOptions{
		KeepPatched: generateKeepPatched,
	}

	result, err := patcher.Patch(path, opts)
	if err != nil {
		return fmt.Errorf("patch failed: %w", err)
	}

	// Step 3: If we patched the file and should restore later, defer the restore
	if !generateKeepPatched && result.OriginalContent != "" {
		defer func() {
			fmt.Println("\nRestoring original build file...")
			if err := patcher.Restore(result.BuildFilePath, result.OriginalContent); err != nil {
				fmt.Printf("Warning: failed to restore original file: %v\n", err)
			} else {
				fmt.Println("✅ Original build file restored")
			}
		}()
	}

	// Step 4: Extract OpenAPI spec (TODO: implement actual extraction)
	// For now, just print what would happen
	fmt.Println("\nExtraction would run here...")

	if result.DependencyAdded {
		fmt.Println("✅ springdoc dependency added temporarily")
	}
	if result.PluginAdded {
		fmt.Println("✅ springdoc plugin added temporarily")
	}

	fmt.Println("\nGenerate command complete - extraction implementation coming soon")
	return nil
}

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.Flags().BoolVar(&generateKeepPatched, "keep-patched", false,
		"keep the patched pom.xml/build.gradle (default: restore original after extraction)")
}
