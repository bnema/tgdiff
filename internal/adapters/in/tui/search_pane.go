package tui

import (
	"charm.land/bubbles/v2/textinput"

	"ero/internal/adapters/in/tui/component"
)

type searchMode string

const (
	searchModeInactive searchMode = ""
	searchModeFiles    searchMode = "files"
	searchModeGrep     searchMode = "grep"
	searchModeDiff     searchMode = "diff mode"
)

type searchState struct {
	mode     searchMode
	input    textinput.Model
	selected int
}

func newSearchState() searchState {
	input := textinput.New()
	input.Prompt = "› "
	input.Placeholder = "type to filter"
	input.SetWidth(40)
	return searchState{input: input}
}

func (s searchState) active() bool {
	return s.mode != searchModeInactive
}

func (s searchState) title() string {
	switch s.mode {
	case searchModeFiles:
		return "Find file"
	case searchModeGrep:
		return "Grep references"
	case searchModeDiff:
		return "Switch diff mode"
	default:
		return "Search"
	}
}

func (s searchState) query() string {
	return s.input.Value()
}

func renderSearchPane(width, height int, state searchState, results []SearchResult) string {
	views := make([]component.SearchResultView, 0, len(results))
	for _, result := range results {
		views = append(views, component.SearchResultView{
			Path:       result.Path,
			LineNumber: result.LineNumber,
			Preview:    result.Preview,
		})
	}
	return component.RenderSearchPane(width, height, state.title(), state.input, state.selected, views, state.mode == searchModeGrep)
}

func searchResultWindow(selected, total, maxVisible int) (int, int) {
	return component.SearchResultWindow(selected, total, maxVisible)
}
