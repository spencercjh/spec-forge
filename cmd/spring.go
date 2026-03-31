// Package cmd contains all CLI commands for spec-forge.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/spencercjh/spec-forge/internal/cli"
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
	cli.Statusf(os.Stderr, "Spring Project Detection Results")
	cli.Statusf(os.Stderr, "Build Tool: %s", info.BuildTool)
	cli.Statusf(os.Stderr, "Build File: %s", info.BuildFilePath)

	springInfo, ok := info.FrameworkData.(*spring.Info)
	if !ok || springInfo == nil {
		springInfo = &spring.Info{}
	}

	cli.Statusf(os.Stderr, "Spring Boot: %s", springInfo.SpringBootVersion)

	if springInfo.IsMultiModule {
		cli.Successf(os.Stderr, "Multi-Module: Yes")
		cli.Statusf(os.Stderr, "Modules: %v", springInfo.Modules)
		if springInfo.MainModule != "" {
			cli.Statusf(os.Stderr, "Main Module: %s", springInfo.MainModule)
			cli.Statusf(os.Stderr, "Main Module Path: %s", springInfo.MainModulePath)
		}
	} else {
		cli.Skipf(os.Stderr, "Multi-Module: No")
	}

	if springInfo.HasSpringdocDeps {
		cli.Successf(os.Stderr, "springdoc Dependency: Present (version: %s)", springInfo.SpringdocVersion)
	} else {
		cli.Skipf(os.Stderr, "springdoc Dependency: Not found")
	}

	if springInfo.HasSpringdocPlugin {
		cli.Successf(os.Stderr, "springdoc Plugin: Configured")
	} else {
		cli.Skipf(os.Stderr, "springdoc Plugin: Not configured")
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
		cli.Statusf(os.Stderr, "Dry run mode - showing changes without modifying files")
	}

	cli.Statusf(os.Stderr, "Build file: %s", result.BuildFilePath)

	if result.DependencyAdded {
		cli.Successf(os.Stderr, "springdoc dependency will be added")
	} else {
		cli.Skipf(os.Stderr, "springdoc dependency already present")
	}

	if result.PluginAdded {
		cli.Successf(os.Stderr, "springdoc plugin will be added")
	} else {
		cli.Skipf(os.Stderr, "springdoc plugin already configured")
	}

	if !opts.DryRun && (result.DependencyAdded || result.PluginAdded) {
		cli.Successf(os.Stderr, "Patch applied successfully!")
		cli.Statusf(os.Stderr, "Note: Build file format may differ from original due to XML serialization.")
		cli.Statusf(os.Stderr, "Use 'spec-forge generate' to extract specs while preserving original file format.")
	} else if !result.DependencyAdded && !result.PluginAdded {
		cli.Skipf(os.Stderr, "No changes needed.")
	}

	return nil
}

func init() {
	springCmd.AddCommand(springDetectCmd)
	springCmd.AddCommand(springPatchCmd)

	springPatchCmd.Flags().BoolVar(&patchDryRun, "dry-run", false, "show changes without modifying files")
	springPatchCmd.Flags().BoolVar(&patchForce, "force", false, "force overwrite existing dependencies")
}
