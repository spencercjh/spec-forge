// Package cmd contains all CLI commands for spec-forge.
package cmd

import (
	"fmt"
	"os"

	"github.com/spencercjh/spec-forge/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	verbose bool
	rootCmd *cobra.Command
)

// initRootCommand initializes the root command
func initRootCommand() *cobra.Command {
	rootCmd = &cobra.Command{
		Use:   "spec-forge",
		Short: "Generate OpenAPI specifications from source code",
		Long: `Spec Forge is a CLI tool that automatically generates OpenAPI specifications
from your source code. It supports multiple frameworks and uses AI to enhance
API descriptions.

Core workflow: Source Code -> Extract -> Enrich -> Publish`,
		Version: "0.1.0",
	}

	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is .spec-forge.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	if err := viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose")); err != nil {
		panic(err)
	}

	return rootCmd
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
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
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	}

	// Load configuration
	config.Load()
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	rootCmd = initRootCommand()
	registerCommands(rootCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// registerCommands adds all subcommands to the root command
func registerCommands(root *cobra.Command) {
	root.AddCommand(generateCmd)
	root.AddCommand(extractCmd)
	root.AddCommand(enrichCmd)
	root.AddCommand(publishCmd)
	root.AddCommand(springCmd)
}

// GetRootCommand returns the root command for testing purposes
func GetRootCommand() *cobra.Command {
	return rootCmd
}

// getConfig returns the global configuration
func getConfig() *config.Config {
	return config.Get()
}
