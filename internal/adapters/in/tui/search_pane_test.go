package tui

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSearchResultWindowKeepsSelectedResultVisible(t *testing.T) {
	t.Parallel()

	start, end := searchResultWindow(8, 12, 8)

	assert.LessOrEqual(t, start, 8)
	assert.Greater(t, end, 8)
	assert.Equal(t, 8, end-start)
}

func TestRenderSearchPaneShowsSelectedResultBeyondFirstPage(t *testing.T) {
	t.Parallel()

	results := make([]SearchResult, 12)
	for i := range results {
		results[i] = SearchResult{Path: fmt.Sprintf("file-%02d.go", i)}
	}
	state := newSearchState()
	state.mode = searchModeFiles
	state.selected = 8

	view := stripANSI(renderSearchPane(80, 24, state, results))

	assert.Contains(t, view, "file-08.go")
	assert.NotContains(t, view, "file-00.go")
}
