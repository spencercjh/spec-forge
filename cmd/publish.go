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
	publishFormat string
	publishOutput string
	publishTarget string

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
	Long: `Publish OpenAPI specification to local files or external platforms.

Supports:
- Local file system (YAML/JSON)
- Postman
- Apifox
- Swagger UI`,
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
	}

	// Add ReadMe-specific options if using ReadMe publisher
	if pub.Name() == "readme" {
		opts.ReadMe = &publisher.ReadMeOptions{
			APIKey:         readMeAPIKey,
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

func init() {
	rootCmd.AddCommand(publishCmd)

	publishCmd.Flags().StringVarP(&publishFormat, "format", "f", "yaml", "output format (yaml or json)")
	publishCmd.Flags().StringVarP(&publishOutput, "output", "o", "./openapi", "output path or directory")
	publishCmd.Flags().StringVarP(&publishTarget, "to", "t", "local", "publish target (local, readme)")

	// ReadMe-specific flags
	publishCmd.Flags().StringVar(&readMeAPIKey, "readme-api-key", "", "ReadMe API key (or set README_API_KEY env var)")
	publishCmd.Flags().StringVar(&readMeBranch, "readme-branch", "", "ReadMe version/branch (default: stable)")
	publishCmd.Flags().StringVar(&readMeSlug, "readme-slug", "", "ReadMe API slug/identifier")
	publishCmd.Flags().BoolVar(&readMeUseSpecVersion, "readme-use-spec-version", false, "use OpenAPI info.version as ReadMe version")
}
