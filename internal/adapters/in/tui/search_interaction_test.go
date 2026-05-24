package tui

import (
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tgdiff/internal/core"
)

func TestModelOpensFileFindAndFiltersResults(t *testing.T) {
	t.Parallel()

	model := NewModel([]core.ReviewFile{
		reviewFile("internal/adapters/in/tui/model.go", "package tui"),
		reviewFile("internal/core/review.go", "package core"),
	})
	updated, cmd := model.Update(keyPress("f"))
	model = updated.(Model)

	require.NotNil(t, cmd)
	assert.True(t, model.search.active())
	assert.Equal(t, searchModeFiles, model.search.mode)
	assert.Contains(t, stripANSI(model.View().Content), "Find file")

	updated, _ = model.Update(keyPress("t"))
	model = updated.(Model)
	updated, _ = model.Update(keyPress("u"))
	model = updated.(Model)
	view := stripANSI(model.View().Content)

	assert.Contains(t, model.search.query(), "tu")
	assert.Contains(t, view, "internal/adapters/in/tui/model.go")
}

func TestModelSearchInputSupportsBackspace(t *testing.T) {
	t.Parallel()

	model := NewModel([]core.ReviewFile{reviewFile("internal/adapters/in/tui/model.go", "package tui")})
	updated, _ := model.Update(keyPress("f"))
	model = updated.(Model)
	updated, _ = model.Update(keyPress("t"))
	model = updated.(Model)
	updated, _ = model.Update(keyPress("u"))
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	model = updated.(Model)

	assert.Equal(t, "t", model.search.query())
}

func TestModelSearchPaneFollowsSelectionBeyondVisibleResults(t *testing.T) {
	t.Parallel()

	files := make([]core.ReviewFile, 0, 22)
	for i := range 22 {
		files = append(files, reviewFile(fmt.Sprintf("file-%02d.go", i), "package demo"))
	}
	model := NewModel(files)
	updated, _ := model.Update(keyPress("f"))
	model = updated.(Model)
	for range 12 {
		updated, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
		model = updated.(Model)
	}

	view := stripANSI(model.View().Content)
	assert.Equal(t, 12, model.search.selected)
	assert.Contains(t, view, "file-12.go")
}

func TestModelOpensGrepSelectsAndCancels(t *testing.T) {
	t.Parallel()

	model := NewModel([]core.ReviewFile{
		reviewFile("a.go", "target one"),
		reviewFile("b.go", "target two"),
	})
	updated, _ := model.Update(keyPress("/"))
	model = updated.(Model)

	model = typeQuery(t, model, "target")
	updated, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	model = updated.(Model)

	assert.Equal(t, searchModeGrep, model.search.mode)
	assert.Equal(t, 1, model.search.selected)
	view := stripANSI(model.View().Content)
	assert.Contains(t, view, "Grep references")
	assert.Contains(t, view, "b.go:1 target two")

	updated, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	model = updated.(Model)
	assert.False(t, model.search.active())
	assert.NotContains(t, stripANSI(model.View().Content), "Grep references")
}

func TestModelSearchStatusHintsDoNotReplaceNormalHintsUntilActive(t *testing.T) {
	t.Parallel()

	model := NewModel([]core.ReviewFile{reviewFile("demo.go", "package main")})
	normal := stripANSI(model.View().Content)
	assert.Contains(t, normal, "find")
	assert.Contains(t, normal, "grep")
	assert.Contains(t, normal, "scroll")

	updated, _ := model.Update(keyPress("f"))
	model = updated.(Model)
	active := stripANSI(model.View().Content)
	assert.Contains(t, active, "select")
	assert.Contains(t, active, "jump")
	assert.Contains(t, active, "cancel")
}

func keyPress(text string) tea.KeyPressMsg {
	runes := []rune(text)
	if len(runes) != 1 {
		panic("keyPress requires exactly one rune")
	}
	return tea.KeyPressMsg{Text: text, Code: runes[0]}
}

func typeQuery(t *testing.T, model Model, query string) Model {
	t.Helper()
	for _, r := range query {
		updated, _ := model.Update(keyPress(string(r)))
		model = updated.(Model)
	}
	return model
}
