package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ero/internal/core"
)

func TestModelRequestsKeyboardEnhancementsWhileCommentEditorIsActive(t *testing.T) {
	t.Parallel()

	model := NewModel([]core.ReviewFile{reviewFileWithLines("demo.go", 1)})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	model = updated.(Model)

	view := model.View()
	assert.False(t, view.KeyboardEnhancements.ReportAllKeysAsEscapeCodes)
	assert.False(t, view.KeyboardEnhancements.ReportAssociatedText)

	updated, _ = model.Update(keyPress("c"))
	model = updated.(Model)
	require.NotNil(t, model.commentEditor)

	view = model.View()
	assert.True(t, view.KeyboardEnhancements.ReportAllKeysAsEscapeCodes)
	assert.True(t, view.KeyboardEnhancements.ReportAssociatedText)
}
