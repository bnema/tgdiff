package app

import (
	"fmt"
	"io"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
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
			DiffMode:     core.DiffModeBranch,
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
	a.root.SetOut(stdout)
	a.root.SetErr(stderr)
	a.root.SetArgs(args)
	return a.root.Execute()
}
