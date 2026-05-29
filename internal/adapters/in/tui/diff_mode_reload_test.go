package tui

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ero/internal/core"
)

func TestModelReloadsReviewWhenDiffModeIsSelected(t *testing.T) {
	t.Parallel()

	loader := &fakeTUILoader{files: []core.ReviewFile{reviewFile("working.go", "package working")}}
	model := NewModelWithLoader([]core.ReviewFile{reviewFile("branch.go", "package branch")}, nil, loader, core.ReviewRequest{RepoPath: "/repo", ContextLines: 2, DiffMode: core.DiffModeBranch})
	updated, _ := model.Update(keyPress("d"))
	model = requireTUIModel(t, updated)
	updated, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	model = requireTUIModel(t, updated)

	updated, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	model = requireTUIModel(t, updated)
	require.NotNil(t, cmd)
	assert.True(t, model.loading)
	assert.Equal(t, core.DiffModeWorking, model.diffMode)

	updated, _ = model.Update(cmd())
	model = requireTUIModel(t, updated)

	assert.False(t, model.loading)
	assert.Empty(t, model.loadError)
	assert.Equal(t, core.DiffModeWorking, loader.request.DiffMode)
	assert.Equal(t, "/repo", loader.request.RepoPath)
	assert.Contains(t, stripANSI(model.View().Content), "working.go")
}

func TestModelShowsDiffModeLoadError(t *testing.T) {
	t.Parallel()

	loader := &fakeTUILoader{err: errors.New("boom")}
	model := NewModelWithLoader([]core.ReviewFile{reviewFile("branch.go", "package branch")}, nil, loader, core.ReviewRequest{RepoPath: "/repo", ContextLines: 2, DiffMode: core.DiffModeBranch})
	updated, _ := model.Update(keyPress("d"))
	model = requireTUIModel(t, updated)
	updated, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	model = requireTUIModel(t, updated)
	updated, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	model = requireTUIModel(t, updated)
	require.NotNil(t, cmd)

	updated, _ = model.Update(cmd())
	model = requireTUIModel(t, updated)

	assert.False(t, model.loading)
	assert.Equal(t, core.DiffModeBranch, model.diffMode)
	assert.Equal(t, "boom", model.loadError)
	assert.Contains(t, stripANSI(model.View().Content), "Failed to load diff: boom")
}

func requireTUIModel(t *testing.T, model tea.Model) Model {
	t.Helper()
	updated, ok := model.(Model)
	require.True(t, ok, "expected Update to return Model")
	return updated
}

type fakeTUILoader struct {
	request core.ReviewRequest
	files   []core.ReviewFile
	err     error
}

func (f *fakeTUILoader) LoadReview(request core.ReviewRequest) ([]core.ReviewFile, error) {
	f.request = request
	if f.err != nil {
		return nil, f.err
	}
	return f.files, nil
}
