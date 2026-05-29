package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ero/internal/core"
)

func TestModelJumpToLineAccountsForInsertedCommentRows(t *testing.T) {
	t.Parallel()

	model := NewModel([]core.ReviewFile{reviewFileWithLines("demo.go", 3)})
	_, err := model.reviewDraft.AddComment(core.ReviewCommentInput{
		FilePath: "demo.go",
		Range: core.ReviewLineRange{
			Start: core.ReviewLineRef{NewLineNumber: 1, Kind: core.LineKindAdded},
			End:   core.ReviewLineRef{NewLineNumber: 1, Kind: core.LineKindAdded},
		},
		Body: "comment above target",
	})
	require.NoError(t, err)
	model.syncReviewViewport()

	model.jumpToLine(SearchResult{FileIndex: 0, SectionIndex: 0, LineIndex: 2})

	require.GreaterOrEqual(t, model.cursorRow, 0)
	assert.Equal(t, 3, model.reviewRows[model.cursorRow].Line.NewLineNumber)
}

func TestModelDoesNotOpenCommentEditorForSelectionAcrossExpander(t *testing.T) {
	t.Parallel()

	model := NewModel([]core.ReviewFile{{
		Path: "demo.go",
		Sections: []core.ReviewSection{
			{ID: "changed-1", Kind: core.SectionKindChanged, Lines: []core.ReviewLine{{NewLineNumber: 1, Content: "one", Kind: core.LineKindAdded}}},
			{ID: "context", Kind: core.SectionKindContext, Lines: []core.ReviewLine{{NewLineNumber: 2, Content: "hidden", Kind: core.LineKindUnchanged}}},
			{ID: "changed-2", Kind: core.SectionKindChanged, Lines: []core.ReviewLine{{NewLineNumber: 3, Content: "three", Kind: core.LineKindAdded}}},
		},
	}})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	model = updated.(Model)
	updated, _ = model.Update(keyPress("s"))
	model = updated.(Model)
	updated, _ = model.Update(keyPress("j"))
	model = updated.(Model)
	updated, _ = model.Update(keyPress("j"))
	model = updated.(Model)
	updated, _ = model.Update(keyPress("c"))
	model = updated.(Model)

	assert.Nil(t, model.commentEditor)
	assert.Contains(t, model.copyFeedback, "Select contiguous lines")
}
