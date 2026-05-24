package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type RunFunc func() error

func NewRootCommand(cfg *viper.Viper, run RunFunc) (*cobra.Command, error) {
	if cfg == nil {
		cfg = viper.New()
	}

	cmd := &cobra.Command{
		Use:          "tgdiff",
		Short:        "Review branch diffs in a GitHub-style TUI",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if run == nil {
				return nil
			}
			return run()
		},
	}

	flags := cmd.PersistentFlags()
	flags.String("repo-path", ".", "Path to the git repository to review")
	flags.Int("context-lines", 3, "Number of unchanged context lines to keep around changes")
	cfg.SetDefault("repo-path", ".")
	cfg.SetDefault("context-lines", 3)
	if err := cfg.BindPFlag("repo-path", flags.Lookup("repo-path")); err != nil {
		return nil, fmt.Errorf("bind repo-path flag: %w", err)
	}
	if err := cfg.BindPFlag("context-lines", flags.Lookup("context-lines")); err != nil {
		return nil, fmt.Errorf("bind context-lines flag: %w", err)
	}

	return cmd, nil
}
