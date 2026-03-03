// Package spring contains Spring-specific CLI commands.
package spring

import (
	"fmt"

	"github.com/spencercjh/spec-forge/internal/config"
	"github.com/spf13/cobra"
)

// detectCmd represents the spring detect command.
var detectCmd = &cobra.Command{
	Use:   "detect",
	Short: "Detect Spring project information",
	Long:  `Analyze the current directory to detect Spring project type, build tool, and dependencies.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		cfg := config.Get()
		fmt.Printf("Detecting Spring project...\n")
		fmt.Printf("Config: %+v\n", cfg)
		fmt.Println("detect called - implementation coming soon")
		return nil
	},
}

func init() {
	// springCmd is added in spring.go
}

// GetDetectCmd returns the detect command for registration.
func GetDetectCmd() *cobra.Command {
	return detectCmd
}
