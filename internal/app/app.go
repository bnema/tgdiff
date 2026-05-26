package app

import (
	"fmt"
	"io"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"tgdiff/internal/adapters/in/cli"
	"tgdiff/internal/adapters/in/terminal"
	"tgdiff/internal/adapters/in/tui"
	gitadapter "tgdiff/internal/adapters/out/git"
	chromatokenizer "tgdiff/internal/adapters/out/syntax/chroma"
	"tgdiff/internal/core"
)

type reviewLoader interface {
	LoadReview(request core.ReviewRequest) ([]core.ReviewFile, error)
}

type tuiRunner interface {
	Run(model tea.Model) error
}

type App struct {
	config *viper.Viper
	root   *cobra.Command
}

func New() (*App, error) {
	cfg := viper.New()
	repositoryLoader := gitadapter.NewRepositoryLoader()
	syntaxTokenizer := chromatokenizer.NewTokenizer()
	reviewLoader := core.NewReviewLoader(repositoryLoader, repositoryLoader, syntaxTokenizer, repositoryLoader)
	runner := tui.NewRunner()
	return newApp(cfg, reviewLoader, runner)
}

func newApp(cfg *viper.Viper, loader reviewLoader, runner tuiRunner) (*App, error) {
	if cfg == nil {
		cfg = viper.New()
	}
	if loader == nil {
		return nil, fmt.Errorf("review loader is nil")
	}
	if runner == nil {
		return nil, fmt.Errorf("runner is nil")
	}

	root, err := cli.NewRootCommand(cfg, func() error {
		initialRequest := core.ReviewRequest{
			RepoPath:     cfg.GetString("repo-path"),
			ContextLines: cfg.GetInt("context-lines"),
			DiffMode:     core.DiffMode(cfg.GetString("diff-mode")),
			Revision:     cfg.GetString("revision"),
			BaseRevision: cfg.GetString("base-revision"),
			HeadRevision: cfg.GetString("head-revision"),
			UpstreamRef:  cfg.GetString("upstream-ref"),
		}
		files, err := loader.LoadReview(initialRequest)
		if err != nil {
			return err
		}
		return runner.Run(tui.NewModelWithLoader(files, terminal.NewCapabilities(), loader, initialRequest))
	})
	if err != nil {
		return nil, fmt.Errorf("build root command: %w", err)
	}

	return &App{
		config: cfg,
		root:   root,
	}, nil
}

func (a *App) RootCommand() *cobra.Command {
	return a.root
}

func (a *App) Run(args []string, stdout, stderr io.Writer) error {
	if err := resetCommandFlags(a.root); err != nil {
		return err
	}
	a.root.SetOut(stdout)
	a.root.SetErr(stderr)
	a.root.SetArgs(args)
	return a.root.Execute()
}

func resetCommandFlags(cmd *cobra.Command) error {
	if err := resetFlags(cmd.PersistentFlags()); err != nil {
		return err
	}
	if err := resetFlags(cmd.Flags()); err != nil {
		return err
	}
	for _, child := range cmd.Commands() {
		if err := resetCommandFlags(child); err != nil {
			return err
		}
	}
	return nil
}

func resetFlags(flags *pflag.FlagSet) error {
	var resetErr error
	flags.VisitAll(func(flag *pflag.Flag) {
		if resetErr != nil {
			return
		}
		if err := flag.Value.Set(flag.DefValue); err != nil {
			resetErr = fmt.Errorf("reset flag %s: %w", flag.Name, err)
			return
		}
		flag.Changed = false
	})
	return resetErr
}
