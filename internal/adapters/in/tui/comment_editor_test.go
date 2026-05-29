package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommentEditorSupportsMultilineSubmitAndCancelKeys(t *testing.T) {
	t.Parallel()

	editor := NewCommentEditor(80)
	cmd := editor.Focus()
	require.NotNil(t, cmd)

	editor, action, _ := editor.Update(keyPress("h"))
	assert.Equal(t, CommentEditorActionNone, action)
	editor, action, _ = editor.Update(keyPress("i"))
	assert.Equal(t, CommentEditorActionNone, action)
	editor, action, _ = editor.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	assert.Equal(t, CommentEditorActionNone, action)
	editor, action, _ = editor.Update(keyPress("!"))
	assert.Equal(t, CommentEditorActionNone, action)

	assert.Equal(t, "hi\n!", editor.Value())

	editor, action, _ = editor.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModCtrl})
	assert.Equal(t, CommentEditorActionSubmit, action)

	editor, action, _ = editor.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	assert.Equal(t, CommentEditorActionCancel, action)
}

func TestCommentEditorSubmitsWhenTerminalEncodesCtrlEnterAsCtrlJ(t *testing.T) {
	t.Parallel()

	editor := NewCommentEditor(80)
	_, action, _ := editor.Update(tea.KeyPressMsg{Code: 'j', Mod: tea.ModCtrl})

	assert.Equal(t, CommentEditorActionSubmit, action)
}

func TestCommentEditorSubmitsWithCtrlSAsPortableFallback(t *testing.T) {
	t.Parallel()

	editor := NewCommentEditor(80)
	_, action, _ := editor.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})

	assert.Equal(t, CommentEditorActionSubmit, action)
}

func TestCommentEditorViewDocumentsSubmitAndCancel(t *testing.T) {
	t.Parallel()

	view := stripANSI(NewCommentEditor(80).View())

	assert.Contains(t, view, "Add review comment")
	assert.Contains(t, view, commentSubmitKeyLabel())
	assert.Contains(t, view, "submit")
	assert.Contains(t, view, "esc cancel")
}
