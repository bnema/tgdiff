package app

import (
	"fmt"
	"io"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"tgdiff/internal/adapters/in/cli"
	"tgdiff/internal/adapters/in/tui"
	gitadapter "tgdiff/internal/adapters/out/git"
	chromatokenizer "tgdiff/internal/adapters/out/syntax/chroma"
	"tgdiff/internal/core"
)

type reviewLoader interface {
	Load(repoPath string, contextLines int) ([]core.ReviewFile, error)
}

type tuiRunner interface {
	Run(model tea.Model) error
}

type App struct {
	config       *viper.Viper
	reviewLoader reviewLoader
	runner       tuiRunner
	root         *cobra.Command
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
		files, err := loader.Load(cfg.GetString("repo-path"), cfg.GetInt("context-lines"))
		if err != nil {
			return err
		}
		return runner.Run(tui.NewModel(files))
	})
	if err != nil {
		return nil, fmt.Errorf("build root command: %w", err)
	}

	return &App{
		config:       cfg,
		reviewLoader: loader,
		runner:       runner,
		root:         root,
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
