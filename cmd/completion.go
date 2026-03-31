package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `Generate shell completion script for spec-forge.

To load completions:

Bash:
  source <(spec-forge completion bash)

  # To load completions for each session, execute once:
  # Linux:
  spec-forge completion bash > /etc/bash_completion.d/spec-forge
  # macOS:
  spec-forge completion bash > $(brew --prefix)/etc/bash_completion.d/spec-forge

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. Add the following to your ~/.zshrc:
  autoload -Uz compinit
  compinit

  # Then load completions:
  spec-forge completion zsh > "${fpath[1]}/_spec-forge"

  # You will need to start a new shell for this setup to take effect.

Fish:
  spec-forge completion fish | source

  # To load completions for each session, execute once:
  spec-forge completion fish > ~/.config/fish/completions/spec-forge.fish

PowerShell:
  spec-forge completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  spec-forge completion powershell > spec-forge.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			return cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			return cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		default:
			return nil
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}

// newCompletionCmd creates a new completion command instance for testing.
func newCompletionCmd() *cobra.Command {
	return &cobra.Command{
		Use:                   "completion [bash|zsh|fish|powershell]",
		Short:                 "Generate shell completion script",
		Long:                  completionCmd.Long,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE:                  completionCmd.RunE,
	}
}
