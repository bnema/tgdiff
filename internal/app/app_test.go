package app

import (
	"bytes"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tgdiff/internal/core"
)

func TestNewBuildsRootCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		expectUse string
	}{
		{name: "default app", expectUse: "tgdiff"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			application, err := New()
			require.NoError(t, err)
			require.NotNil(t, application)
			require.NotNil(t, application.RootCommand())
			assert.Equal(t, tt.expectUse, application.RootCommand().Use)
		})
	}
}

func TestRunLoadsReviewAndRunsTUIWithConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		args          []string
		expectRepo    string
		expectContext int
	}{
		{
			name:          "explicit repo path and context lines",
			args:          []string{"--repo-path", "/tmp/repo", "--context-lines", "2"},
			expectRepo:    "/tmp/repo",
			expectContext: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := viper.New()
			loader := &fakeReviewLoader{
				files: []core.ReviewFile{{
					Path: "demo.go",
					Sections: []core.ReviewSection{{
						ID:    "changed-1",
						Kind:  core.SectionKindChanged,
						Lines: []core.ReviewLine{{NewLineNumber: 1, Content: "package main", Kind: core.LineKindAdded}},
					}},
				}},
			}
			runner := &fakeRunner{}

			application, err := newApp(cfg, loader, runner)
			require.NoError(t, err)

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			err = application.Run(tt.args, &stdout, &stderr)
			require.NoError(t, err)

			assert.Equal(t, tt.expectRepo, loader.repoPath)
			assert.Equal(t, tt.expectContext, loader.contextLines)
			require.NotNil(t, runner.model)
		})
	}
}

type fakeReviewLoader struct {
	repoPath     string
	contextLines int
	files        []core.ReviewFile
}

func (f *fakeReviewLoader) Load(repoPath string, contextLines int) ([]core.ReviewFile, error) {
	f.repoPath = repoPath
	f.contextLines = contextLines
	return f.files, nil
}

type fakeRunner struct {
	model tea.Model
}

func (f *fakeRunner) Run(model tea.Model) error {
	f.model = model
	return nil
}
