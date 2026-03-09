// Package cmd contains all CLI commands for spec-forge.
package cmd

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	verbose bool
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
		os.Exit(1)
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

	// Persistent flags
	c.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is .spec-forge.yaml)")
	c.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Initialize viper binding
	cobra.OnInitialize(initConfig)

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
