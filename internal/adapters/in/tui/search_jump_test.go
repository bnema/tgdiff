package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ero/internal/core"
)

func TestSearchAcceptFileResultMovesCursorToFileHeader(t *testing.T) {
	t.Parallel()

	model := NewModel([]core.ReviewFile{
		reviewFileWithLines("alpha.go", 20),
		reviewFileWithLines("zeta.go", 20),
	})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 6})
	model = updated.(Model)
	updated, _ = model.Update(keyPress("f"))
	model = updated.(Model)
	model = typeQuery(t, model, "zeta")

	updated, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	model = updated.(Model)

	require.False(t, model.search.active())
	assert.Equal(t, 1, model.selectedFile)
	assert.Equal(t, model.reviewAnchors.FileRows[1], model.cursorRow)
	assert.LessOrEqual(t, model.reviewViewport.YOffset(), model.cursorRow)
	assert.Less(t, model.cursorRow, model.reviewViewport.YOffset()+model.reviewViewport.Height())
}

func TestSearchAcceptFileResultClampsOffsetForShortDocuments(t *testing.T) {
	t.Parallel()

	model := NewModel([]core.ReviewFile{
		reviewFile("alpha.go", "package alpha"),
		reviewFile("zeta.go", "package zeta"),
	})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	model = updated.(Model)
	updated, _ = model.Update(keyPress("f"))
	model = updated.(Model)
	model = typeQuery(t, model, "zeta")
	updated, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	model = updated.(Model)

	assert.Equal(t, 1, model.selectedFile)
	assert.Equal(t, 0, model.reviewViewport.YOffset())
	assert.Contains(t, stripANSI(model.View().Content), "zeta.go")
}

func TestSearchAcceptGrepResultMovesCursorToChangedLine(t *testing.T) {
	t.Parallel()

	model := NewModel([]core.ReviewFile{{
		Path: "demo.go",
		Sections: []core.ReviewSection{{ID: "changed", Kind: core.SectionKindChanged, Lines: []core.ReviewLine{
			{NewLineNumber: 1, Content: "one", Kind: core.LineKindAdded},
			{NewLineNumber: 2, Content: "needle", Kind: core.LineKindAdded},
			{NewLineNumber: 3, Content: "three", Kind: core.LineKindAdded},
			{NewLineNumber: 4, Content: "four", Kind: core.LineKindAdded},
			{NewLineNumber: 5, Content: "five", Kind: core.LineKindAdded},
		}}},
	}})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 4})
	model = updated.(Model)
	updated, _ = model.Update(keyPress("/"))
	model = updated.(Model)
	model = typeQuery(t, model, "needle")

	updated, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	model = updated.(Model)

	anchor := ReviewLineAnchor{FileIndex: 0, SectionIndex: 0, LineIndex: 1}
	assert.Equal(t, model.reviewAnchors.LineRows[anchor], model.cursorRow)
	assert.LessOrEqual(t, model.reviewViewport.YOffset(), model.cursorRow)
	assert.Less(t, model.cursorRow, model.reviewViewport.YOffset()+model.reviewViewport.Height())
}

func TestSearchAcceptGrepResultPreservesAlreadyVisibleContextExpansion(t *testing.T) {
	t.Parallel()

	model := NewModel([]core.ReviewFile{{
		Path: "demo.go",
		Sections: []core.ReviewSection{
			{ID: "context", Kind: core.SectionKindContext, ExpandedBelow: 1, Lines: []core.ReviewLine{
				{OldLineNumber: 1, NewLineNumber: 1, Content: "hidden other", Kind: core.LineKindUnchanged},
				{OldLineNumber: 2, NewLineNumber: 2, Content: "visible needle", Kind: core.LineKindUnchanged},
			}},
			{ID: "changed", Kind: core.SectionKindChanged, Lines: []core.ReviewLine{{NewLineNumber: 3, Content: "changed", Kind: core.LineKindAdded}}},
		},
	}})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 4})
	model = updated.(Model)
	updated, _ = model.Update(keyPress("/"))
	model = updated.(Model)
	model = typeQuery(t, model, "needle")
	updated, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	model = updated.(Model)

	assert.Equal(t, 1, model.files[0].Sections[0].HiddenLineCount())
	assert.Equal(t, 0, model.files[0].Sections[0].ExpandedAbove)
	assert.Equal(t, 1, model.files[0].Sections[0].ExpandedBelow)
}

func TestSearchAcceptGrepResultWithMissingAnchorLeavesOffsetUnchanged(t *testing.T) {
	t.Parallel()

	model := NewModel([]core.ReviewFile{reviewFileWithLines("demo.go", 12)})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 4})
	model = updated.(Model)
	model.reviewViewport.SetYOffset(3)

	model.jumpToLine(SearchResult{FileIndex: 0, SectionIndex: 0, LineIndex: 99})

	assert.Equal(t, 3, model.reviewViewport.YOffset())
}

func TestSearchAcceptGrepResultExpandsHiddenContextBeforeMovingCursor(t *testing.T) {
	t.Parallel()

	model := NewModel([]core.ReviewFile{{
		Path: "demo.go",
		Sections: []core.ReviewSection{
			{ID: "context", Kind: core.SectionKindContext, Lines: []core.ReviewLine{
				{OldLineNumber: 1, NewLineNumber: 1, Content: "hidden needle", Kind: core.LineKindUnchanged},
				{OldLineNumber: 2, NewLineNumber: 2, Content: "hidden other", Kind: core.LineKindUnchanged},
			}},
			{ID: "changed", Kind: core.SectionKindChanged, Lines: []core.ReviewLine{{NewLineNumber: 3, Content: "changed", Kind: core.LineKindAdded}}},
		},
	}})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 4})
	model = updated.(Model)
	_, anchoredBefore := model.reviewAnchors.LineRows[ReviewLineAnchor{FileIndex: 0, SectionIndex: 0, LineIndex: 0}]
	require.False(t, anchoredBefore)

	updated, _ = model.Update(keyPress("/"))
	model = updated.(Model)
	model = typeQuery(t, model, "needle")
	updated, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	model = updated.(Model)

	anchor := ReviewLineAnchor{FileIndex: 0, SectionIndex: 0, LineIndex: 0}
	row, anchoredAfter := model.reviewAnchors.LineRows[anchor]
	require.True(t, anchoredAfter)
	assert.Equal(t, 0, model.files[0].Sections[0].HiddenLineCount())
	assert.Equal(t, row, model.cursorRow)
	assert.LessOrEqual(t, model.reviewViewport.YOffset(), model.cursorRow)
	assert.Less(t, model.cursorRow, model.reviewViewport.YOffset()+model.reviewViewport.Height())
}
