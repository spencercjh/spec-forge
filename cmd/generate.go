// Package cmd contains all CLI commands for spec-forge.
package cmd

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/spencercjh/spec-forge/internal/extractor"
	"github.com/spencercjh/spec-forge/internal/extractor/spring"
)

// generateKeepPatched controls whether to keep the patched pom/build file
// Default is false (restore original) for generate command
var generateKeepPatched bool

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

func runGenerate(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	slog.InfoContext(ctx, "Generating OpenAPI spec", "path", path)

	// Step 1: Detect project
	detector := spring.NewDetector()
	info, err := detector.Detect(path)
	if err != nil {
		return fmt.Errorf("detection failed: %w", err)
	}

	slog.InfoContext(ctx, "Detected project", "tool", info.BuildTool, "build_file", info.BuildFilePath)

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
			slog.InfoContext(ctx, "Restoring original build file...")
			if err := patcher.Restore(result.BuildFilePath, result.OriginalContent); err != nil {
				slog.WarnContext(ctx, "failed to restore original file", "error", err)
			} else {
				slog.InfoContext(ctx, "Original build file restored", "status", "✅")
			}
		}()
	}

	// Step 4: Extract OpenAPI spec (TODO: implement actual extraction)
	// For now, just print what would happen
	slog.InfoContext(ctx, "Extraction would run here...")

	if result.DependencyAdded {
		slog.InfoContext(ctx, "springdoc dependency added temporarily", "status", "✅")
	}
	if result.PluginAdded {
		slog.InfoContext(ctx, "springdoc plugin added temporarily", "status", "✅")
	}

	slog.InfoContext(ctx, "Generate command complete - extraction implementation coming soon")
	return nil
}

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.Flags().BoolVar(&generateKeepPatched, "keep-patched", false,
		"keep the patched pom.xml/build.gradle (default: restore original after extraction)")
}
