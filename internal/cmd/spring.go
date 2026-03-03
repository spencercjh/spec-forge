package cmd

import (
	"github.com/spencercjh/spec-forge/internal/cmd/spring"
	"github.com/spf13/cobra"
)

// springCmd represents the spring command group.
var springCmd = &cobra.Command{
	Use:   "spring",
	Short: "Spring framework specific commands",
	Long:  `Commands for working with Spring (Java) projects.`,
}

func init() {
	// springCmd is added to root in registerCommands
	springCmd.AddCommand(spring.GetDetectCmd())
}
