package tui

import (
	"charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"tgdiff/internal/core"
)

func (m Model) openSearch(mode searchMode) (Model, tea.Cmd) {
	m.search.mode = mode
	m.search.selected = 0
	m.search.input.Reset()
	m.search.input.Placeholder = "type to filter"
	if mode == searchModeGrep {
		m.search.input.Placeholder = "grep changed files"
	}
	if mode == searchModeDiff {
		m.search.input.Placeholder = "filter diff modes"
		m.search.selected = m.currentDiffModeIndex()
	}
	return m, m.search.input.Focus()
}

func (m Model) updateSearch(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.closeSearch()
		return m, nil
	case "up", "ctrl+p":
		m.moveSearchSelection(-1)
		return m, nil
	case "down", "ctrl+n":
		m.moveSearchSelection(1)
		return m, nil
	case "enter":
		m.acceptSearchResult()
		return m, nil
	}

	var cmd tea.Cmd
	m.search.input, cmd = m.search.input.Update(msg)
	m.clampSearchSelection()
	return m, cmd
}

func (m *Model) acceptSearchResult() {
	results := m.searchResults()
	if len(results) == 0 {
		return
	}
	result := results[min(max(m.search.selected, 0), len(results)-1)]
	switch m.search.mode {
	case searchModeFiles:
		m.jumpToFile(result.FileIndex)
	case searchModeGrep:
		m.jumpToLine(result)
	case searchModeDiff:
		m.diffMode = result.DiffMode
	}
	m.closeSearch()
}

func (m *Model) jumpToFile(fileIndex int) {
	if fileIndex < 0 || fileIndex >= len(m.files) {
		return
	}
	m.selectedFile = fileIndex
	m.resetContextSelection()
	m.syncReviewViewport()
	m.reviewViewport.SetYOffset(m.reviewAnchors.FileRows[fileIndex])
	m.updateActiveFileFromViewport()
}

func (m *Model) jumpToLine(result SearchResult) {
	if result.FileIndex < 0 || result.FileIndex >= len(m.files) {
		return
	}
	if result.SectionIndex < 0 || result.SectionIndex >= len(m.files[result.FileIndex].Sections) {
		return
	}
	section := &m.files[result.FileIndex].Sections[result.SectionIndex]
	if section.Kind == core.SectionKindContext && !contextLineVisible(*section, result.LineIndex) {
		section.ExpandAll()
	}
	m.selectedFile = result.FileIndex
	m.resetContextSelection()
	m.syncReviewViewport()
	anchor := ReviewLineAnchor{FileIndex: result.FileIndex, SectionIndex: result.SectionIndex, LineIndex: result.LineIndex}
	if row, ok := m.reviewAnchors.LineRows[anchor]; ok {
		m.reviewViewport.SetYOffset(row)
	}
	m.updateActiveFileFromViewport()
}

func contextLineVisible(section core.ReviewSection, lineIndex int) bool {
	if section.Kind != core.SectionKindContext {
		return true
	}
	if lineIndex < 0 || lineIndex >= len(section.Lines) {
		return false
	}
	aboveCount := min(section.ExpandedAbove, len(section.Lines))
	if lineIndex < aboveCount {
		return true
	}
	belowCount := min(section.ExpandedBelow, len(section.Lines)-aboveCount)
	return lineIndex >= len(section.Lines)-belowCount
}

func (m *Model) closeSearch() {
	m.search.input.Blur()
	m.search.input.Reset()
	m.search.mode = searchModeInactive
	m.search.selected = 0
}

func (m *Model) moveSearchSelection(delta int) {
	results := m.searchResults()
	if len(results) == 0 {
		m.search.selected = 0
		return
	}
	m.search.selected = min(max(m.search.selected+delta, 0), len(results)-1)
}

func (m *Model) clampSearchSelection() {
	results := m.searchResults()
	if len(results) == 0 {
		m.search.selected = 0
		return
	}
	m.search.selected = min(max(m.search.selected, 0), len(results)-1)
}

func (m Model) searchResults() []SearchResult {
	switch m.search.mode {
	case searchModeFiles:
		return fuzzyFileResults(m.files, m.search.query())
	case searchModeGrep:
		return grepResults(m.files, m.search.query())
	case searchModeDiff:
		return diffModeResults(m.search.query(), m.nerdFont)
	default:
		return nil
	}
}

func (m Model) currentDiffModeIndex() int {
	for i, mode := range selectableDiffModes {
		if mode == m.diffMode {
			return i
		}
	}
	return 0
}

func (m Model) renderSearchOverlay(content string) string {
	width := max(m.width, 1)
	height := max(m.height, 1)
	pane := renderSearchPane(width, height, m.search, m.searchResults())
	return renderCenteredOverlay(content, pane, width, height, min(1, max(height-lipgloss.Height(pane), 0)))
}

func (m Model) renderHelpOverlay(content string) string {
	width := max(m.width, 1)
	height := max(m.height, 1)
	pane := renderHelpPane(width, height)
	return renderCenteredOverlay(content, pane, width, height, max((height-lipgloss.Height(pane))/2, 0))
}

func renderCenteredOverlay(content, pane string, width, height, y int) string {
	canvas := lipgloss.NewCanvas(width, height)
	compositor := lipgloss.NewCompositor(
		lipgloss.NewLayer(content),
		lipgloss.NewLayer(pane).X(max((width-lipgloss.Width(pane))/2, 0)).Y(y).Z(1),
	)
	canvas.Compose(compositor)
	return canvas.Render()
}
