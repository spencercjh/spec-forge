// Package cmd contains all CLI commands for spec-forge.
package cmd

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/spencercjh/spec-forge/internal/extractor"
	"github.com/spencercjh/spec-forge/internal/extractor/spring"
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

func runSpringDetect(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
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
	printProjectInfo(ctx, info)
	return nil
}

func printProjectInfo(ctx context.Context, info *extractor.ProjectInfo) {
	slog.InfoContext(ctx, "Spring Project Detection Results")
	slog.InfoContext(ctx, "Build Tool", "tool", info.BuildTool)
	slog.InfoContext(ctx, "Build File", "path", info.BuildFilePath)

	springInfo, ok := info.FrameworkData.(*spring.Info)
	if !ok || springInfo == nil {
		springInfo = &spring.Info{}
	}

	slog.InfoContext(ctx, "Spring Boot", "version", springInfo.SpringBootVersion)

	if springInfo.IsMultiModule {
		slog.InfoContext(ctx, "Multi-Module", "enabled", "✅ Yes")
		slog.InfoContext(ctx, "Modules", "list", springInfo.Modules)
		if springInfo.MainModule != "" {
			slog.InfoContext(ctx, "Main Module", "name", springInfo.MainModule)
			slog.InfoContext(ctx, "Main Module Path", "path", springInfo.MainModulePath)
		}
	} else {
		slog.InfoContext(ctx, "Multi-Module", "enabled", "❌ No")
	}

	if springInfo.HasSpringdocDeps {
		slog.InfoContext(ctx, "springdoc Dependency", "status", "✅ Present", "version", springInfo.SpringdocVersion)
	} else {
		slog.InfoContext(ctx, "springdoc Dependency", "status", "❌ Not found")
	}

	if springInfo.HasSpringdocPlugin {
		slog.InfoContext(ctx, "springdoc Plugin", "status", "✅ Configured")
	} else {
		slog.InfoContext(ctx, "springdoc Plugin", "status", "❌ Not configured")
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

func runSpringPatch(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
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
		slog.InfoContext(ctx, "Dry run mode - showing changes without modifying files")
	}

	slog.InfoContext(ctx, "Build file", "path", result.BuildFilePath)

	if result.DependencyAdded {
		slog.InfoContext(ctx, "springdoc dependency will be added", "status", "✅")
	} else {
		slog.InfoContext(ctx, "springdoc dependency already present", "status", "⏭️")
	}

	if result.PluginAdded {
		slog.InfoContext(ctx, "springdoc plugin will be added", "status", "✅")
	} else {
		slog.InfoContext(ctx, "springdoc plugin already configured", "status", "⏭️")
	}

	if !opts.DryRun && (result.DependencyAdded || result.PluginAdded) {
		slog.InfoContext(ctx, "Patch applied successfully!")
		slog.InfoContext(ctx, "Note: Build file format may differ from original due to XML serialization.")
		slog.InfoContext(ctx, "Use 'spec-forge generate' to extract specs while preserving original file format.")
	} else if !result.DependencyAdded && !result.PluginAdded {
		slog.InfoContext(ctx, "No changes needed.")
	}

	return nil
}

func init() {
	springCmd.AddCommand(springDetectCmd)
	springCmd.AddCommand(springPatchCmd)

	springPatchCmd.Flags().BoolVar(&patchDryRun, "dry-run", false, "show changes without modifying files")
	springPatchCmd.Flags().BoolVar(&patchForce, "force", false, "force overwrite existing dependencies")
}
