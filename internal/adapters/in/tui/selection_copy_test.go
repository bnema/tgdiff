package tui

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"ero/internal/core"
	"ero/internal/ports/mocks"
)

func TestModelSelectionShortcutsToggleAndClearRange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		startKey  tea.KeyPressMsg
		clearKey  tea.KeyPressMsg
		wantStart int
	}{
		{name: "s starts selection and esc clears it", startKey: keyPress("s"), clearKey: tea.KeyPressMsg{Code: tea.KeyEsc}, wantStart: 2},
		{name: "space starts selection and s toggles it off", startKey: keyPress(" "), clearKey: keyPress("s"), wantStart: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			model := NewModel([]core.ReviewFile{reviewFileWithLines("demo.go", 5)})
			updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 6})
			model = updated.(Model)

			updated, _ = model.Update(tt.startKey)
			model = updated.(Model)

			require.NotNil(t, model.selectionAnchorRow)
			assert.Equal(t, tt.wantStart, *model.selectionAnchorRow)

			updated, _ = model.Update(keyPress("j"))
			model = updated.(Model)
			start, end, ok := model.selectedRange()
			require.True(t, ok)
			assert.Equal(t, tt.wantStart, start)
			assert.Equal(t, tt.wantStart+1, end)
			view := stripANSI(model.View().Content)
			assert.Contains(t, view, "➜")
			assert.Contains(t, view, "┃")

			updated, _ = model.Update(tt.clearKey)
			model = updated.(Model)
			assert.Nil(t, model.selectionAnchorRow)
		})
	}
}

func TestModelStatusBarShowsActiveFileLine(t *testing.T) {
	t.Parallel()

	model := NewModel([]core.ReviewFile{reviewFileWithLines("demo.go", 5)})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 6})
	model = updated.(Model)
	updated, _ = model.Update(keyPress("j"))
	model = updated.(Model)

	assert.Contains(t, stripANSI(model.View().Content), "demo.go:2")
}

func TestModelCopyPlainUsesSelectionOrCurrentLine(t *testing.T) {
	t.Parallel()

	clipboard := mocks.NewMockClipboardWriter(t)
	model := NewModelWithClipboardWriter([]core.ReviewFile{{
		Path: "demo.go",
		Sections: []core.ReviewSection{{ID: "changed", Kind: core.SectionKindChanged, Lines: []core.ReviewLine{
			{NewLineNumber: 1, Content: "one", Kind: core.LineKindAdded},
			{NewLineNumber: 2, Content: "two", Kind: core.LineKindDeleted},
			{NewLineNumber: 3, Content: "three", Kind: core.LineKindUnchanged},
		}}},
	}}, nil, nil, core.ReviewRequest{DiffMode: core.DiffModeBranch}, clipboard)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 6})
	model = updated.(Model)

	clipboard.EXPECT().WriteClipboard(mock.Anything, "+ one").Return(nil).Once()
	updated, cmd := model.Update(keyPress("y"))
	model = updated.(Model)
	require.NotNil(t, cmd)
	updated, _ = model.Update(cmd())
	model = updated.(Model)
	assert.Equal(t, "+ one", model.lastCopiedText)
	assert.Equal(t, "Copied 1 line", model.copyFeedback)

	updated, _ = model.Update(keyPress("s"))
	model = updated.(Model)
	updated, _ = model.Update(keyPress("j"))
	model = updated.(Model)
	clipboard.EXPECT().WriteClipboard(mock.Anything, "+ one\n- two").Return(nil).Once()
	updated, cmd = model.Update(keyPress("y"))
	model = updated.(Model)
	require.NotNil(t, cmd)
	updated, _ = model.Update(cmd())
	model = updated.(Model)
	assert.Equal(t, "+ one\n- two", model.lastCopiedText)
	assert.Equal(t, "Copied 2 lines", model.copyFeedback)

	updated, _ = model.Update(copyFeedbackExpiredMsg{id: model.copyFeedbackID})
	model = updated.(Model)
	assert.Empty(t, model.copyFeedback)
}

func TestModelCopyWithMetadataGroupsSelectionByFile(t *testing.T) {
	t.Parallel()

	clipboard := mocks.NewMockClipboardWriter(t)
	model := NewModelWithClipboardWriter([]core.ReviewFile{{
		Path: "demo.go",
		Sections: []core.ReviewSection{{ID: "changed", Kind: core.SectionKindChanged, Lines: []core.ReviewLine{
			{NewLineNumber: 10, Content: "added", Kind: core.LineKindAdded},
			{OldLineNumber: 11, Content: "deleted", Kind: core.LineKindDeleted},
		}}},
	}}, nil, nil, core.ReviewRequest{DiffMode: core.DiffModeBranch}, clipboard)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 6})
	model = updated.(Model)
	updated, _ = model.Update(keyPress("s"))
	model = updated.(Model)
	updated, _ = model.Update(keyPress("j"))
	model = updated.(Model)

	clipboard.EXPECT().WriteClipboard(mock.Anything, mock.MatchedBy(func(text string) bool {
		return strings.Contains(text, "File: demo.go") && strings.Contains(text, "+ added")
	})).Return(nil).Once()
	updated, cmd := model.Update(keyPress("Y"))
	model = updated.(Model)
	require.NotNil(t, cmd)
	updated, _ = model.Update(cmd())
	model = updated.(Model)

	assert.Equal(t, "Copied 2 lines with metadata", model.copyFeedback)
	copied := model.lastCopiedText
	assert.Contains(t, copied, "File: demo.go")
	assert.Contains(t, copied, "Lines: +10 to -11")
	assert.Contains(t, copied, "```diff")
	assert.Contains(t, copied, "+ added")
	assert.Contains(t, copied, "- deleted")
	assert.True(t, strings.HasSuffix(copied, "```"))
}

func TestModelCopyReportsClipboardFailure(t *testing.T) {
	t.Parallel()

	clipboard := mocks.NewMockClipboardWriter(t)
	model := NewModelWithClipboardWriter([]core.ReviewFile{reviewFile("demo.go", "package main")}, nil, nil, core.ReviewRequest{DiffMode: core.DiffModeBranch}, clipboard)
	clipboard.EXPECT().WriteClipboard(mock.Anything, "+ package main").Return(errors.New("wl-copy failed")).Once()

	updated, cmd := model.Update(keyPress("y"))
	model = updated.(Model)
	require.NotNil(t, cmd)
	updated, _ = model.Update(cmd())
	model = updated.(Model)

	assert.Empty(t, model.lastCopiedText)
	assert.Contains(t, model.copyFeedback, "Copy failed")
	assert.Contains(t, model.copyFeedback, "wl-copy failed")
}

func TestModelCopySkipsClipboardWhenNoDiffLinesAreSelected(t *testing.T) {
	t.Parallel()

	clipboard := mocks.NewMockClipboardWriter(t)
	model := NewModelWithClipboardWriter([]core.ReviewFile{{
		Path: "demo.go",
		Sections: []core.ReviewSection{{ID: "context", Kind: core.SectionKindContext, Lines: []core.ReviewLine{
			{NewLineNumber: 1, Content: "hidden", Kind: core.LineKindUnchanged},
		}}},
	}}, nil, nil, core.ReviewRequest{DiffMode: core.DiffModeBranch}, clipboard)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 6})
	model = updated.(Model)

	updated, cmd := model.Update(keyPress("y"))
	model = updated.(Model)

	assert.Nil(t, cmd)
	assert.Empty(t, model.lastCopiedText)
	assert.Equal(t, "No diff lines to copy", model.copyFeedback)
}

func TestModelClearsSelectionWhenContextExpansionChangesRenderedRows(t *testing.T) {
	t.Parallel()

	model := NewModel([]core.ReviewFile{{
		Path: "demo.go",
		Sections: []core.ReviewSection{
			{ID: "context", Kind: core.SectionKindContext, Lines: []core.ReviewLine{{NewLineNumber: 1, Content: "hidden", Kind: core.LineKindUnchanged}}},
			{ID: "changed", Kind: core.SectionKindChanged, Lines: []core.ReviewLine{{NewLineNumber: 2, Content: "changed", Kind: core.LineKindAdded}}},
		},
	}})
	updated, _ := model.Update(keyPress("s"))
	model = updated.(Model)
	require.NotNil(t, model.selectionAnchorRow)

	updated, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	model = updated.(Model)

	assert.Nil(t, model.selectionAnchorRow)
}

func TestHelpPaneDocumentsSelectionAndCopyShortcuts(t *testing.T) {
	t.Parallel()

	view := stripANSI(renderHelpPane(80, 30))

	assert.Contains(t, view, "s/space")
	assert.Contains(t, view, "select lines")
	assert.Contains(t, view, "y/Y")
	assert.Contains(t, view, "copy")
}
