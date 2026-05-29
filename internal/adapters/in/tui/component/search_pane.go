package component

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"

	"ero/internal/adapters/in/tui/theme"
)

type SearchResultView struct {
	Path       string
	LineNumber int
	Preview    string
}

func RenderSearchPane(width, height int, title string, input textinput.Model, selected int, results []SearchResultView, grepMode bool) string {
	paneWidth := min(max(width-8, 24), 72)
	input.SetWidth(max(paneWidth-6, 8))

	lines := []string{
		theme.SearchPaneTitleStyle.Render(fmt.Sprintf("%s · %d result%s", title, len(results), pluralSuffix(len(results)))),
		input.View(),
	}
	if len(results) == 0 {
		lines = append(lines, theme.MutedStyle.Render("No matches"))
	} else {
		start, end := SearchResultWindow(selected, len(results), maxVisibleSearchResults(height))
		for i := start; i < end; i++ {
			row := FormatSearchResult(results[i], grepMode, paneWidth-4)
			if i == selected {
				row = theme.SearchSelectedRowStyle.Width(paneWidth - 4).Render(row)
			}
			lines = append(lines, row)
		}
	}

	return theme.SearchPaneStyle.Width(paneWidth).Render(strings.Join(lines, "\n"))
}

func maxVisibleSearchResults(height int) int {
	return min(max(height-5, 1), 8)
}

func SearchResultWindow(selected, total, maxVisible int) (int, int) {
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

func FormatSearchResult(result SearchResultView, grepMode bool, width int) string {
	label := result.Path
	if grepMode {
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
