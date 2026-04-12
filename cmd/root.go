package cmd

import "github.com/spf13/cobra"

func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "wsx",
		Short:         "Manage Windows-first AI workspaces",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(newInitCommand())

	return root
}

func Execute() error {
	return NewRootCommand().Execute()
}
