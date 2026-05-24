package tui

import (
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"tgdiff/internal/core"
)

func TestSearchOverlayUsesTerminalHeightForVisibleResultWindow(t *testing.T) {
	t.Parallel()

	files := make([]core.ReviewFile, 0, 22)
	for i := range 22 {
		files = append(files, reviewFile(fmt.Sprintf("file-%02d.go", i), "package demo"))
	}
	model := NewModel(files)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 7})
	model = updated.(Model)
	updated, _ = model.Update(keyPress("f"))
	model = updated.(Model)
	for range 12 {
		updated, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
		model = updated.(Model)
	}

	view := stripANSI(renderSearchPane(model.width, model.height, model.search, model.searchResults()))

	assert.Equal(t, 12, model.search.selected)
	assert.Contains(t, view, "file-12.go")
	assert.NotContains(t, view, "file-00.go")
}
