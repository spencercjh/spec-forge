// Package cmd contains all CLI commands for spec-forge.
package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/spf13/cobra"

	"github.com/spencercjh/spec-forge/internal/publisher"
)

var (
	publishFormat    string
	publishOutput    string
	publishTarget    string
	publishOverwrite bool

	// ReadMe-specific flags
	readMeAPIKey         string
	readMeBranch         string
	readMeSlug           string
	readMeUseSpecVersion bool
)

// publishCmd represents the publish command
var publishCmd = &cobra.Command{
	Use:   "publish [spec-file]",
	Short: "Publish OpenAPI spec to target platform",
	Long: `Publish OpenAPI specification to external platforms.

Supports:
- ReadMe (via rdme CLI)

Note: Local file output is handled by the generate command.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPublish,
}

func runPublish(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	specFile := "openapi.yaml"
	if len(args) > 0 {
		specFile = args[0]
	}

	slog.InfoContext(ctx, "Publishing spec", "file", specFile, "target", publishTarget)

	// Create publisher using factory
	pub, err := publisher.NewPublisher(publishTarget)
	if err != nil {
		return fmt.Errorf("failed to create publisher: %w", err)
	}

	slog.InfoContext(ctx, "Using publisher", "name", pub.Name())

	// Load spec file
	specData, err := os.ReadFile(specFile)
	if err != nil {
		return fmt.Errorf("failed to read spec file: %w", err)
	}

	spec, err := openapi3.NewLoader().LoadFromData(specData)
	if err != nil {
		return fmt.Errorf("failed to parse spec: %w", err)
	}

	// Build publish options
	opts := &publisher.PublishOptions{
		OutputPath: publishOutput,
		Format:     publishFormat,
		Overwrite:  publishOverwrite,
	}

	// Add ReadMe-specific options if using ReadMe publisher
	if pub.Name() == "readme" {
		// Resolve API key from flag or environment variable
		apiKey := readMeAPIKey
		if apiKey == "" {
			apiKey = os.Getenv("README_API_KEY")
		}
		opts.ReadMe = &publisher.ReadMeOptions{
			APIKey:         apiKey,
			Branch:         readMeBranch,
			Slug:           readMeSlug,
			UseSpecVersion: readMeUseSpecVersion,
		}
	}

	// Publish
	result, err := pub.Publish(ctx, spec, opts)
	if err != nil {
		return fmt.Errorf("failed to publish: %w", err)
	}

	slog.InfoContext(ctx, "Published successfully",
		"path", result.Path,
		"format", result.Format,
		"bytes", result.BytesWritten,
	)
	if result.Message != "" {
		slog.InfoContext(ctx, "Publisher output", "message", result.Message)
	}

	return nil
}

// newPublishCmd creates a new publish command instance for testing.
func newPublishCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "publish [spec-file]",
		Short: "Publish OpenAPI spec to target platform",
		Long: `Publish OpenAPI specification to external platforms.

Supports:
- ReadMe (via rdme CLI)

Note: Local file output is handled by the generate command.`,
		Args: cobra.MaximumNArgs(1),
		RunE: runPublish,
	}

	c.Flags().StringVarP(&publishFormat, "format", "f", "yaml", "output format (yaml or json)")
	c.Flags().StringVarP(&publishOutput, "output", "o", "", "output file path (currently unused for 'readme'; reserved for future publishers)")
	c.Flags().StringVarP(&publishTarget, "to", "t", "", "publish target (required: readme)")
	c.Flags().BoolVar(&publishOverwrite, "overwrite", false, "overwrite existing spec")

	// ReadMe-specific flags
	c.Flags().StringVar(&readMeAPIKey, "readme-api-key", "", "ReadMe API key (or set README_API_KEY env var)")
	c.Flags().StringVar(&readMeBranch, "readme-branch", "", "ReadMe version/branch (default: stable)")
	c.Flags().StringVar(&readMeSlug, "readme-slug", "", "ReadMe API slug/identifier")
	c.Flags().BoolVar(&readMeUseSpecVersion, "readme-use-spec-version", false, "use OpenAPI info.version as ReadMe version")

	// Mark target as required
	if err := c.MarkFlagRequired("to"); err != nil {
		panic(fmt.Sprintf("failed to mark flag 'to' as required: %v", err))
	}

	return c
}

func init() {
	rootCmd.AddCommand(publishCmd)

	publishCmd.Flags().StringVarP(&publishFormat, "format", "f", "yaml", "output format (yaml or json)")
	publishCmd.Flags().StringVarP(&publishOutput, "output", "o", "", "output file path (currently unused for 'readme'; reserved for future publishers)")
	publishCmd.Flags().StringVarP(&publishTarget, "to", "t", "", "publish target (required: readme)")
	publishCmd.Flags().BoolVar(&publishOverwrite, "overwrite", false, "overwrite existing spec")

	// ReadMe-specific flags
	publishCmd.Flags().StringVar(&readMeAPIKey, "readme-api-key", "", "ReadMe API key (or set README_API_KEY env var)")
	publishCmd.Flags().StringVar(&readMeBranch, "readme-branch", "", "ReadMe version/branch (default: stable)")
	publishCmd.Flags().StringVar(&readMeSlug, "readme-slug", "", "ReadMe API slug/identifier")
	publishCmd.Flags().BoolVar(&readMeUseSpecVersion, "readme-use-spec-version", false, "use OpenAPI info.version as ReadMe version")

	// Mark target as required
	if err := publishCmd.MarkFlagRequired("to"); err != nil {
		panic(fmt.Sprintf("failed to mark flag 'to' as required: %v", err))
	}
}
