package app

import (
	"fmt"

	"github.com/spf13/cobra"
)

func versionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "ero %s\ncommit: %s\ndate: %s\nbuiltBy: %s\n", version, commit, date, builtBy)
			return err
		},
	}
}
