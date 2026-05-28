package app

import (
	"bytes"
	"fmt"
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

func TestRunClearsHelpFlagBetweenExecutions(t *testing.T) {
	t.Parallel()

	cfg := viper.New()
	loader := &fakeReviewLoader{files: minimalReviewFiles()}
	runner := &fakeRunner{}
	application, err := newApp(cfg, loader, runner)
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err = application.Run([]string{"commit", "--help"}, &stdout, &stderr)
	require.NoError(t, err)
	assert.Empty(t, loader.requests)

	err = application.Run([]string{"commit", "HEAD~1"}, &stdout, &stderr)
	require.NoError(t, err)
	require.Len(t, loader.requests, 1)
	assert.Equal(t, core.DiffModeCommit, loader.requests[0].DiffMode)
	assert.Equal(t, "HEAD~1", loader.requests[0].Revision)

	err = application.Run([]string{"--help"}, &stdout, &stderr)
	require.NoError(t, err)
	require.Len(t, loader.requests, 1)

	err = application.Run(nil, &stdout, &stderr)
	require.NoError(t, err)
	require.Len(t, loader.requests, 2)
	assert.Equal(t, core.DiffModeBranch, loader.requests[1].DiffMode)
}

func TestRunDefaultsBackToBranchModeAndFlagDefaultsAfterModeCommand(t *testing.T) {
	t.Parallel()

	cfg := viper.New()
	loader := &fakeReviewLoader{files: minimalReviewFiles()}
	runner := &fakeRunner{}
	application, err := newApp(cfg, loader, runner)
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err = application.Run([]string{"--repo-path", "/tmp/repo", "--context-lines", "2", "commit", "HEAD~1"}, &stdout, &stderr)
	require.NoError(t, err)
	require.Len(t, loader.requests, 1)
	assert.Equal(t, "/tmp/repo", loader.requests[0].RepoPath)
	assert.Equal(t, 2, loader.requests[0].ContextLines)
	assert.Equal(t, core.DiffModeCommit, loader.requests[0].DiffMode)
	assert.Equal(t, "HEAD~1", loader.requests[0].Revision)

	err = application.Run(nil, &stdout, &stderr)
	require.NoError(t, err)
	require.Len(t, loader.requests, 2)
	assert.Equal(t, ".", loader.requests[1].RepoPath)
	assert.Equal(t, 3, loader.requests[1].ContextLines)
	assert.Equal(t, core.DiffModeBranch, loader.requests[1].DiffMode)
	assert.Empty(t, loader.requests[1].Revision)
}

func TestRunDetectsStartupModeWhenNoExplicitCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		state      core.StartupState
		promptMode core.DiffMode
		wantMode   core.DiffMode
		wantErr    string
	}{
		{name: "working changes", state: core.StartupState{HasUnstagedChanges: true}, wantMode: core.DiffModeWorking},
		{name: "staged changes", state: core.StartupState{HasStagedChanges: true}, wantMode: core.DiffModeStaged},
		{name: "ahead of upstream", state: core.StartupState{HasUpstream: true, Ahead: 1}, wantMode: core.DiffModeUpstream},
		{name: "mixed prompts and uses selected mode", state: core.StartupState{HasStagedChanges: true, HasUnstagedChanges: true}, promptMode: core.DiffModeLocal, wantMode: core.DiffModeLocal},
		{name: "mixed non-interactive errors", state: core.StartupState{HasStagedChanges: true, HasUnstagedChanges: true}, wantErr: "choose explicitly"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := viper.New()
			loader := &fakeReviewLoader{files: minimalReviewFiles()}
			runner := &fakeRunner{}
			stateReader := &fakeStartupStateReader{state: tt.state}
			prompt := &fakeStartupPrompt{mode: tt.promptMode}
			interactive := tt.wantErr == ""
			application, err := newAppWithStartup(cfg, loader, runner, stateReader, prompt, func() bool { return interactive })
			require.NoError(t, err)

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			err = application.Run(nil, &stdout, &stderr)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Empty(t, loader.requests)
				return
			}
			require.NoError(t, err)
			require.Len(t, loader.requests, 1)
			assert.Equal(t, tt.wantMode, loader.requests[0].DiffMode)
		})
	}
}

func TestRunExplicitCommandBypassesStartupDetection(t *testing.T) {
	t.Parallel()

	cfg := viper.New()
	loader := &fakeReviewLoader{files: minimalReviewFiles()}
	runner := &fakeRunner{}
	stateReader := &fakeStartupStateReader{err: fmt.Errorf("should not read startup state")}
	application, err := newAppWithStartup(cfg, loader, runner, stateReader, &fakeStartupPrompt{}, func() bool { return true })
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err = application.Run([]string{"staged"}, &stdout, &stderr)
	require.NoError(t, err)
	require.Len(t, loader.requests, 1)
	assert.Equal(t, core.DiffModeStaged, loader.requests[0].DiffMode)
}

func TestRunLoadsReviewAndRunsTUIWithConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		args          []string
		expectRepo    string
		expectContext int
		expectRequest core.ReviewRequest
	}{
		{
			name:          "explicit repo path and context lines",
			args:          []string{"--repo-path", "/tmp/repo", "--context-lines", "2"},
			expectRepo:    "/tmp/repo",
			expectContext: 2,
			expectRequest: core.ReviewRequest{DiffMode: core.DiffModeBranch},
		},
		{
			name:          "initial commit mode",
			args:          []string{"--repo-path", "/tmp/repo", "commit", "HEAD~1"},
			expectRepo:    "/tmp/repo",
			expectContext: 3,
			expectRequest: core.ReviewRequest{DiffMode: core.DiffModeCommit, Revision: "HEAD~1"},
		},
		{
			name:          "initial range mode",
			args:          []string{"range", "main", "feature"},
			expectRepo:    ".",
			expectContext: 3,
			expectRequest: core.ReviewRequest{DiffMode: core.DiffModeRange, BaseRevision: "main", HeadRevision: "feature"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := viper.New()
			loader := &fakeReviewLoader{files: minimalReviewFiles()}
			runner := &fakeRunner{}

			application, err := newApp(cfg, loader, runner)
			require.NoError(t, err)

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			err = application.Run(tt.args, &stdout, &stderr)
			require.NoError(t, err)

			assert.Equal(t, tt.expectRepo, loader.repoPath)
			assert.Equal(t, tt.expectContext, loader.contextLines)
			assert.Equal(t, tt.expectRequest.DiffMode, loader.request.DiffMode)
			assert.Equal(t, tt.expectRequest.Revision, loader.request.Revision)
			assert.Equal(t, tt.expectRequest.BaseRevision, loader.request.BaseRevision)
			assert.Equal(t, tt.expectRequest.HeadRevision, loader.request.HeadRevision)
			require.NotNil(t, runner.model)
		})
	}
}

func minimalReviewFiles() []core.ReviewFile {
	return []core.ReviewFile{{
		Path: "demo.go",
		Sections: []core.ReviewSection{{
			ID:    "changed-1",
			Kind:  core.SectionKindChanged,
			Lines: []core.ReviewLine{{NewLineNumber: 1, Content: "package main", Kind: core.LineKindAdded}},
		}},
	}}
}

type fakeStartupStateReader struct {
	state core.StartupState
	err   error
}

func (f *fakeStartupStateReader) ReadStartupState(string) (core.StartupState, error) {
	return f.state, f.err
}

type fakeStartupPrompt struct {
	mode core.DiffMode
	err  error
}

func (f *fakeStartupPrompt) PromptLocalChangeMode() (core.DiffMode, error) {
	if f.err != nil {
		return "", f.err
	}
	if f.mode == "" {
		return core.DiffModeStaged, nil
	}
	return f.mode, nil
}

type fakeReviewLoader struct {
	repoPath     string
	contextLines int
	request      core.ReviewRequest
	requests     []core.ReviewRequest
	files        []core.ReviewFile
}

func (f *fakeReviewLoader) LoadReview(request core.ReviewRequest) ([]core.ReviewFile, error) {
	f.repoPath = request.RepoPath
	f.contextLines = request.ContextLines
	f.request = request
	f.requests = append(f.requests, request)
	return f.files, nil
}

type fakeRunner struct {
	model tea.Model
}

func (f *fakeRunner) Run(model tea.Model) error {
	f.model = model
	return nil
}
