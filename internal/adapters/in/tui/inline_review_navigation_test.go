package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ero/internal/core"
)

func TestModelCancelsOpenCommentEditorWhenJumpingFiles(t *testing.T) {
	t.Parallel()

	model := NewModel([]core.ReviewFile{reviewFileWithLines("a.go", 1), reviewFileWithLines("b.go", 1)})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	model = updated.(Model)
	updated, _ = model.Update(keyPress("c"))
	model = updated.(Model)
	require.NotNil(t, model.commentEditor)

	model.jumpToFile(1)

	assert.Nil(t, model.commentEditor)
}

func TestModelReloadClearsDraftAndCommentEditor(t *testing.T) {
	t.Parallel()

	model := NewModel([]core.ReviewFile{reviewFileWithLines("a.go", 1)})
	updated, _ := model.Update(keyPress("c"))
	model = updated.(Model)
	require.NotNil(t, model.commentEditor)
	_, err := model.reviewDraft.AddComment(core.ReviewCommentInput{FilePath: "a.go", Range: core.ReviewLineRange{Start: core.ReviewLineRef{NewLineNumber: 1, Kind: core.LineKindAdded}, End: core.ReviewLineRef{NewLineNumber: 1, Kind: core.LineKindAdded}}, Body: "old"})
	require.NoError(t, err)

	updated, _ = model.Update(reviewLoadedMsg{mode: core.DiffModeWorking, files: []core.ReviewFile{reviewFileWithLines("b.go", 1)}})
	model = updated.(Model)

	assert.Nil(t, model.commentEditor)
	assert.Empty(t, model.reviewDraft.Comments())
}
