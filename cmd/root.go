package cmd

import "github.com/spf13/cobra"

func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "wsx",
		Short:         "Manage Windows-first AI workspaces",
		Long: `Manage Windows-first AI workspaces.

Currently supported commands:
  init    Initialize a workspace in the current directory

Only implemented commands are shown below.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.CompletionOptions.DisableDefaultCmd = true
	root.AddCommand(newInitCommand())

	return root
}

func Execute() error {
	return NewRootCommand().Execute()
}
