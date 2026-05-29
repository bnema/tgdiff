package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"ero/internal/core"
	"ero/internal/ports/mocks"
)

func TestModelInlineCommentSubmitCopiesReviewJSON(t *testing.T) {
	t.Parallel()

	clipboard := mocks.NewMockClipboardWriter(t)
	model := NewModelWithClipboardWriter([]core.ReviewFile{reviewFileWithLines("demo.go", 3)}, nil, nil, core.ReviewRequest{DiffMode: core.DiffModeBranch}, clipboard)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	model = updated.(Model)

	updated, _ = model.Update(keyPress("s"))
	model = updated.(Model)
	updated, _ = model.Update(keyPress("j"))
	model = updated.(Model)
	updated, _ = model.Update(keyPress("c"))
	model = updated.(Model)
	require.NotNil(t, model.commentEditor)

	for _, r := range "review note" {
		updated, _ = model.Update(tea.KeyPressMsg{Text: string(r), Code: r})
		model = updated.(Model)
	}

	clipboard.EXPECT().WriteClipboard(mock.Anything, mock.MatchedBy(func(text string) bool {
		return strings.Contains(text, `"file": "demo.go"`) && strings.Contains(text, `"body": "review note"`) && strings.Contains(text, `"new": 1`) && strings.Contains(text, `"new": 2`)
	})).Return(nil).Once()
	updated, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModCtrl})
	model = updated.(Model)
	require.NotNil(t, cmd)
	updated, _ = model.Update(cmd())
	model = updated.(Model)

	assert.Nil(t, model.commentEditor)
	assert.Len(t, model.reviewDraft.Comments(), 1)
	assert.Contains(t, model.copyFeedback, "Copied review JSON with 1 comment")
	assert.Contains(t, model.lastCopiedText, "review note")
	assert.Contains(t, stripANSI(model.View().Content), "review note")
}

func TestModelInlineCommentCancelAndClear(t *testing.T) {
	t.Parallel()

	model := NewModel([]core.ReviewFile{reviewFileWithLines("demo.go", 1)})
	updated, _ := model.Update(keyPress("c"))
	model = updated.(Model)
	require.NotNil(t, model.commentEditor)

	updated, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	model = updated.(Model)
	assert.Nil(t, model.commentEditor)

	_, err := model.reviewDraft.AddComment(core.ReviewCommentInput{FilePath: "demo.go", Range: core.ReviewLineRange{Start: core.ReviewLineRef{NewLineNumber: 1, Kind: core.LineKindAdded}, End: core.ReviewLineRef{NewLineNumber: 1, Kind: core.LineKindAdded}}, Body: "old"})
	require.NoError(t, err)
	updated, _ = model.Update(keyPress("C"))
	model = updated.(Model)
	assert.Empty(t, model.reviewDraft.Comments())
	assert.Contains(t, model.copyFeedback, "Cleared review")
}
