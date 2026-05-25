package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
)

var (
	searchPaneStyle        = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62")).Padding(0, 1)
	searchPaneTitleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81"))
	searchSelectedRowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("62"))
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
	paneWidth := min(max(width-8, 24), 72)
	input := state.input
	input.SetWidth(max(paneWidth-6, 8))

	lines := []string{
		searchPaneTitleStyle.Render(fmt.Sprintf("%s · %d result%s", state.title(), len(results), pluralSuffix(len(results)))),
		input.View(),
	}
	if len(results) == 0 {
		lines = append(lines, mutedStyle.Render("No matches"))
	} else {
		start, end := searchResultWindow(state.selected, len(results), maxVisibleSearchResults(height))
		for i := start; i < end; i++ {
			row := formatSearchResult(results[i], state.mode, paneWidth-4)
			if i == state.selected {
				row = searchSelectedRowStyle.Width(paneWidth - 4).Render(row)
			}
			lines = append(lines, row)
		}
	}

	return searchPaneStyle.Width(paneWidth).Render(strings.Join(lines, "\n"))
}

func maxVisibleSearchResults(height int) int {
	return min(max(height-5, 1), 8)
}

func searchResultWindow(selected, total, maxVisible int) (int, int) {
	if total <= 0 || maxVisible <= 0 {
		return 0, 0
	}
	if total <= maxVisible {
		return 0, total
	}
	selected = min(max(selected, 0), total-1)
	start := max(selected-maxVisible/2, 0)
	start = min(start, total-maxVisible)
	return start, start + maxVisible
}

func formatSearchResult(result SearchResult, mode searchMode, width int) string {
	label := result.Path
	if mode == searchModeGrep {
		label = fmt.Sprintf("%s:%d %s", result.Path, result.LineNumber, result.Preview)
	}
	runes := []rune(label)
	if len(runes) > width && width > 1 {
		label = string(runes[:width-1]) + "…"
	}
	return label
}

func pluralSuffix(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
