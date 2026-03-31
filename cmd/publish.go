// Package cmd contains all CLI commands for spec-forge.
package cmd

import (
	"fmt"
	"os"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/spf13/cobra"

	"github.com/spencercjh/spec-forge/internal/cli"
	"github.com/spencercjh/spec-forge/internal/publisher"
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

	// Get flag values from command (isolated per command instance)
	//nolint:errcheck // flags are bound at command creation, errors not possible
	format, _ := cmd.Flags().GetString("format")
	//nolint:errcheck
	output, _ := cmd.Flags().GetString("output")
	//nolint:errcheck
	target, _ := cmd.Flags().GetString("to")
	//nolint:errcheck
	overwrite, _ := cmd.Flags().GetBool("overwrite")
	//nolint:errcheck
	readMeAPIKey, _ := cmd.Flags().GetString("readme-api-key")
	//nolint:errcheck
	readMeBranch, _ := cmd.Flags().GetString("readme-branch")
	//nolint:errcheck
	readMeSlug, _ := cmd.Flags().GetString("readme-slug")
	//nolint:errcheck
	readMeUseSpecVersion, _ := cmd.Flags().GetBool("readme-use-spec-version")

	cli.Statusf(os.Stderr, "Publishing spec to %s", target)

	// Create publisher using factory
	pub, err := publisher.NewPublisher(target)
	if err != nil {
		return fmt.Errorf("failed to create publisher: %w", err)
	}

	cli.Statusf(os.Stderr, "Using publisher: %s", pub.Name())

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
		OutputPath: output,
		Format:     format,
		Overwrite:  overwrite,
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

	cli.Successf(os.Stderr, "Published successfully (%d bytes, %s)", result.BytesWritten, result.Format)
	if result.Message != "" {
		cli.Statusf(os.Stderr, "%s", result.Message)
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

	c.Flags().StringP("format", "f", "yaml", "output format (yaml or json)")
	c.Flags().StringP("output", "o", "", "output file path (currently unused for 'readme'; reserved for future publishers)")
	c.Flags().StringP("to", "t", "", "publish target (required: readme)")
	c.Flags().Bool("overwrite", false, "overwrite existing spec")

	// ReadMe-specific flags
	c.Flags().String("readme-api-key", "", "ReadMe API key (or set README_API_KEY env var)")
	c.Flags().String("readme-branch", "", "ReadMe version/branch (default: stable)")
	c.Flags().String("readme-slug", "", "ReadMe API slug/identifier")
	c.Flags().Bool("readme-use-spec-version", false, "use OpenAPI info.version as ReadMe version")

	// Mark target as required
	if err := c.MarkFlagRequired("to"); err != nil {
		panic(fmt.Sprintf("failed to mark flag 'to' as required: %v", err))
	}

	//nolint:errcheck // completion registration cannot fail with valid flag names
	c.RegisterFlagCompletionFunc("format", cobra.FixedCompletions(
		[]string{"yaml", "json"}, cobra.ShellCompDirectiveNoFileComp,
	))
	//nolint:errcheck
	c.RegisterFlagCompletionFunc("to", cobra.FixedCompletions(
		[]string{"readme"}, cobra.ShellCompDirectiveNoFileComp,
	))

	return c
}

// publish command flag variables for global rootCmd only
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

	//nolint:errcheck // completion registration cannot fail with valid flag names
	publishCmd.RegisterFlagCompletionFunc("format", cobra.FixedCompletions(
		[]string{"yaml", "json"}, cobra.ShellCompDirectiveNoFileComp,
	))
	//nolint:errcheck
	publishCmd.RegisterFlagCompletionFunc("to", cobra.FixedCompletions(
		[]string{"readme"}, cobra.ShellCompDirectiveNoFileComp,
	))
}
