// Package cmd contains all CLI commands for spec-forge.
package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	forgeerrors "github.com/spencercjh/spec-forge/internal/errors"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "spec-forge",
	Short: "Generate OpenAPI specifications from source code",
	Long: `Spec Forge is a CLI tool that automatically generates OpenAPI specifications
from your source code. It supports multiple frameworks and uses AI to enhance
API descriptions.

Core workflow: Source Code -> Extract -> Enrich -> Publish`,
	Version: "0.1.0",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		slog.Error("command failed", "error", err)
		printHintAndExit(err)
	}
}

// printHintAndExit prints a recovery hint (if the error is classified) and exits
// with an appropriate exit code.
func printHintAndExit(err error) {
	var fe *forgeerrors.Error
	if errors.As(err, &fe) {
		hint := fe.Hint()
		if hint != "" {
			fmt.Fprintf(os.Stderr, "Hint: %s\n", hint)
		}
		os.Exit(exitCodeForCode(fe.Code))
	}
	os.Exit(1)
}

// exitCodeForCode maps an error category code to a shell exit code.
// User / configuration errors use exit code 2; all other errors use 1.
func exitCodeForCode(code string) int {
	switch code {
	case forgeerrors.CodeConfig, forgeerrors.CodeDetect, forgeerrors.CodePatch:
		return 2
	default:
		return 1
	}
}

// NewRootCommand creates a fresh root command instance for testing.
// This avoids global state pollution between tests.
func NewRootCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "spec-forge",
		Short: "Generate OpenAPI specifications from source code",
		Long: `Spec Forge is a CLI tool that automatically generates OpenAPI specifications
from your source code. It supports multiple frameworks and uses AI to enhance
API descriptions.

Core workflow: Source Code -> Extract -> Enrich -> Publish`,
		Version: "0.1.0",
	}

	// Persistent flags - use local variables bound directly to the command
	c.PersistentFlags().StringP("config", "c", "", "config file (default is .spec-forge.yaml)")
	c.PersistentFlags().BoolP("verbose", "v", false, "verbose output")

	// Bind verbose flag to viper for config file support
	if err := viper.BindPFlag("verbose", c.PersistentFlags().Lookup("verbose")); err != nil {
		slog.Error("failed to bind verbose flag", "error", err)
	}

	// Add all subcommands using factory functions
	c.AddCommand(newGenerateCmd())
	c.AddCommand(newEnrichCmd())
	c.AddCommand(newPublishCmd())

	return c
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is .spec-forge.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	if err := viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose")); err != nil {
		slog.Error("failed to bind verbose flag", "error", err)
		os.Exit(1)
	}
}

// cfgFile and verbose are only used by the global rootCmd (not by NewRootCommand)
// This maintains backward compatibility for the main CLI while allowing isolated testing.
var (
	cfgFile string
	verbose bool
)

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Initialize logging first
	setupLogging(verbose)

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".spec-forge")
	}

	viper.SetEnvPrefix("SPEC_FORGE")
	viper.AutomaticEnv()

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		if verbose {
			slog.Debug("using config file", "path", viper.ConfigFileUsed())
		}
	}
}

// setupLogging configures the default slog logger.
func setupLogging(verbose bool) {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	}))
	slog.SetDefault(logger)
}
