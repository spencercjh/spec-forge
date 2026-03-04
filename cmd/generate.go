// Package cmd contains all CLI commands for spec-forge.
package cmd

import (
	"errors"
	"log/slog"
	"time"

	"github.com/spf13/cobra"

	"github.com/spencercjh/spec-forge/internal/config"
	"github.com/spencercjh/spec-forge/internal/extractor"
	"github.com/spencercjh/spec-forge/internal/extractor/spring"
	"github.com/spencercjh/spec-forge/internal/validator"
)

// generateKeepPatched controls whether to keep the patched pom/build file
// Default is false (restore original) for generate command
var generateKeepPatched bool

// generateSkipValidate controls whether to skip validation
var generateSkipValidate bool

// generateTimeout is the timeout for generation commands
var generateTimeout time.Duration

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate [path]",
	Short: "Generate OpenAPI spec from source code",
	Long: `Generate OpenAPI specification by running the complete pipeline:
detect -> patch -> generate -> validate -> restore (optional)

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
		return errWrap("detection failed", err)
	}

	slog.InfoContext(ctx, "Detected project",
		"tool", info.BuildTool,
		"build_file", info.BuildFilePath,
		"multi_module", info.IsMultiModule,
	)

	// Step 2: Patch project if needed
	patcher := spring.NewPatcher()
	patchOpts := &extractor.PatchOptions{
		KeepPatched: generateKeepPatched,
	}

	result, err := patcher.Patch(path, patchOpts)
	if err != nil {
		return errWrap("patch failed", err)
	}

	// Step 3: If we patched the file and should restore later, defer the restore
	if !generateKeepPatched && result.OriginalContent != "" {
		defer func() {
			slog.InfoContext(ctx, "Restoring original build file...")
			if restoreErr := patcher.Restore(result.BuildFilePath, result.OriginalContent); restoreErr != nil {
				slog.WarnContext(ctx, "failed to restore original file", "error", restoreErr)
			} else {
				slog.InfoContext(ctx, "Original build file restored", "status", "✅")
			}
		}()
	}

	if result.DependencyAdded {
		slog.InfoContext(ctx, "springdoc dependency added temporarily", "status", "✅")
	}
	if result.PluginAdded {
		slog.InfoContext(ctx, "springdoc plugin added temporarily", "status", "✅")
	}
	if result.SpringBootConfigured {
		slog.InfoContext(ctx, "spring-boot-maven-plugin configured with start/stop goals", "status", "✅")
	}

	// Step 4: Generate OpenAPI spec
	generator := spring.NewGenerator()
	genOpts := &extractor.GenerateOptions{
		Format:    config.Get().Output.Format,
		Timeout:   generateTimeout,
		SkipTests: true,
	}

	genResult, err := generator.Generate(ctx, path, info, genOpts)
	if err != nil {
		return errWrap("generation failed", err)
	}

	slog.InfoContext(ctx, "OpenAPI spec generated",
		"path", genResult.SpecFilePath,
		"format", genResult.Format,
	)

	// Step 5: Validate the generated spec
	if !generateSkipValidate {
		v := validator.NewValidator()
		valResult, err := v.Validate(ctx, genResult.SpecFilePath)
		if err != nil {
			return errWrap("validation error", err)
		}

		if !valResult.Valid {
			slog.ErrorContext(ctx, "OpenAPI spec validation failed")
			for _, validationErr := range valResult.Errors {
				slog.ErrorContext(ctx, "  - "+validationErr)
			}
			return errors.New("generated OpenAPI spec is invalid")
		}

		slog.InfoContext(ctx, "OpenAPI spec validated", "status", "✅")
	} else {
		slog.InfoContext(ctx, "Validation skipped", "status", "⏭️")
	}

	// Step 6: Output final result
	slog.InfoContext(ctx, "Generation complete",
		"spec_file", genResult.SpecFilePath,
	)

	return nil
}

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.Flags().BoolVar(&generateKeepPatched, "keep-patched", false,
		"keep the patched pom.xml/build.gradle (default: restore original after extraction)")
	generateCmd.Flags().BoolVar(&generateSkipValidate, "skip-validate", false,
		"skip validation of the generated OpenAPI spec")
	generateCmd.Flags().DurationVar(&generateTimeout, "timeout", 5*time.Minute,
		"timeout for Maven/Gradle commands")
}

// errWrap wraps an error with a message.
func errWrap(msg string, err error) error {
	return errors.New(msg + ": " + err.Error())
}
