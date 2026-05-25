package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tgdiff/internal/core"
	"tgdiff/internal/ports/mocks"
)

func TestDiffModeLabelUsesNerdFontWhenDetected(t *testing.T) {
	t.Parallel()

	assert.Equal(t, nerdIconBranch+" branch", diffModeLabel(core.DiffModeBranch, true))
	assert.Equal(t, "branch diff", diffModeLabel(core.DiffModeBranch, false))
}

func TestModelStatusBarShowsDiffModeInsteadOfReview(t *testing.T) {
	t.Parallel()

	terminal := mocks.NewMockTerminal(t)
	terminal.EXPECT().SupportsNerdFont().Return(false)
	model := NewModelWithTerminal([]core.ReviewFile{reviewFile("demo.go", "package main")}, terminal)
	view := stripANSI(model.View().Content)

	assert.Contains(t, view, "branch diff")
	assert.NotContains(t, view, " review ")
}

func TestModelSwitchesDiffModeThroughSearchPane(t *testing.T) {
	t.Parallel()

	terminal := mocks.NewMockTerminal(t)
	terminal.EXPECT().SupportsNerdFont().Return(false)
	model := NewModelWithTerminal([]core.ReviewFile{reviewFile("demo.go", "package main")}, terminal)
	updated, _ := model.Update(keyPress("d"))
	updatedModel, ok := updated.(Model)
	require.True(t, ok, "expected Update to return Model")
	model = updatedModel

	assert.True(t, model.search.active())
	assert.Equal(t, searchModeDiff, model.search.mode)
	assert.Contains(t, stripANSI(model.View().Content), "Switch diff mode")

	results := model.searchResults()
	assert.Len(t, results, len(selectableDiffModes))
	assert.Equal(t, core.DiffModeBranch, results[0].DiffMode)

	updated, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	updatedModel, ok = updated.(Model)
	require.True(t, ok, "expected Update to return Model")
	model = updatedModel

	assert.False(t, model.search.active())
	assert.Equal(t, core.DiffModeBranch, model.diffMode)
	assert.Contains(t, stripANSI(model.View().Content), "branch diff")
}
