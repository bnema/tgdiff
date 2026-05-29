package app

import (
	"fmt"
	"io"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"ero/internal/adapters/in/cli"
	"ero/internal/adapters/in/terminal"
	"ero/internal/adapters/in/tui"
	clipboardadapter "ero/internal/adapters/out/clipboard"
	gitadapter "ero/internal/adapters/out/git"
	chromatokenizer "ero/internal/adapters/out/syntax/chroma"
	"ero/internal/core"
	"ero/internal/ports"
)

type reviewLoader interface {
	LoadReview(request core.ReviewRequest) ([]core.ReviewFile, error)
}

type startupStateReader interface {
	ReadStartupState(repoPath string) (core.StartupState, error)
}

type startupPrompt interface {
	PromptLocalChangeMode() (core.DiffMode, error)
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
	clipboardWriter := clipboardadapter.NewSystemWriter()
	return newAppWithClipboard(cfg, reviewLoader, runner, repositoryLoader, tui.NewStartupPrompt(), terminal.IsInteractive, clipboardWriter)
}

func newApp(cfg *viper.Viper, loader reviewLoader, runner tuiRunner) (*App, error) {
	return newAppWithStartup(cfg, loader, runner, nil, nil, func() bool { return false })
}

func newAppWithStartup(cfg *viper.Viper, loader reviewLoader, runner tuiRunner, startupReader startupStateReader, prompt startupPrompt, isInteractive func() bool) (*App, error) {
	return newAppWithClipboard(cfg, loader, runner, startupReader, prompt, isInteractive, nil)
}

func newAppWithClipboard(cfg *viper.Viper, loader reviewLoader, runner tuiRunner, startupReader startupStateReader, prompt startupPrompt, isInteractive func() bool, clipboardWriter ports.ClipboardWriter) (*App, error) {
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
		if cfg.GetBool("startup-detect") {
			request, err := resolveStartupRequest(initialRequest, startupReader, prompt, isInteractive)
			if err != nil {
				return err
			}
			initialRequest = request
		}
		files, err := loader.LoadReview(initialRequest)
		if err != nil {
			return err
		}
		return runner.Run(tui.NewModelWithClipboardWriter(files, terminal.NewCapabilities(), loader, initialRequest, clipboardWriter))
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
