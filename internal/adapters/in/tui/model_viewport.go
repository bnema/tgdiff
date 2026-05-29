package tui

import (
	viewport "charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"

	"ero/internal/adapters/in/tui/theme"
)

func (m *Model) syncReviewViewport() {
	width := m.reviewWidth()
	height := max(m.height-1, 1)
	annotations := ReviewAnnotations{}
	if m.reviewDraft != nil {
		annotations.Comments = m.reviewDraft.Comments()
	}
	annotations.Editor = m.commentEditor
	rendered := NewReviewDocument(width).RenderWithAnnotations(m.files, m.selectedFile, m.selectedContext, annotations)
	currentCursor := m.cursorRow
	m.reviewViewport.SetWidth(width)
	m.reviewViewport.SetHeight(height)
	m.reviewViewport.SetContentLines(rendered.Lines)
	m.reviewAnchors = rendered.Anchors
	m.reviewRows = rendered.Rows
	m.cursorRow = m.clampCursorRow(currentCursor)
	m.centerViewportOnCursor()
	m.updateActiveFileFromCursor()
}

func (m *Model) moveCursor(delta int) {
	m.cursorRow = m.selectableRowFrom(m.cursorRow, delta)
	m.selectNearestContextToCursor()
	m.syncReviewViewport()
}

func (m *Model) moveCursorToStart() {
	m.cursorRow = m.firstSelectableRow()
	m.selectNearestContextToCursor()
	m.syncReviewViewport()
}

func (m *Model) moveCursorToEnd() {
	m.cursorRow = m.lastSelectableRow()
	m.selectNearestContextToCursor()
	m.syncReviewViewport()
}

func (m Model) clampCursorRow(row int) int {
	return clampRowWithBounds(row, m.firstSelectableRow(), m.lastSelectableRow())
}

func (m Model) selectableRowFrom(row, delta int) int {
	if delta == 0 {
		return m.clampCursorRow(row)
	}
	step := 1
	if delta < 0 {
		step = -1
	}
	first := m.firstSelectableRow()
	last := m.lastSelectableRow()
	remaining := delta
	current := row
	for remaining != 0 {
		next := current + step
		if next < first || next > last {
			return clampRowWithBounds(next, first, last)
		}
		current = next
		if current >= 0 && current < len(m.reviewRows) && m.reviewRows[current].Selectable {
			remaining -= step
		}
	}
	return clampRowWithBounds(current, first, last)
}

func clampRowWithBounds(row, first, last int) int {
	if last < first {
		return 0
	}
	return min(max(row, first), last)
}

func (m Model) firstSelectableRow() int {
	for i, row := range m.reviewRows {
		if row.Selectable {
			return i
		}
	}
	return 0
}

func (m Model) lastSelectableRow() int {
	for i := len(m.reviewRows) - 1; i >= 0; i-- {
		if m.reviewRows[i].Selectable {
			return i
		}
	}
	return 0
}

func (m *Model) centerViewportOnCursor() {
	m.reviewViewport.SetYOffset(m.cursorRow - m.reviewViewport.Height()/2)
}

func (m *Model) updateActiveFileFromCursor() {
	if len(m.files) == 0 || len(m.reviewRows) == 0 {
		m.activeFilePath = ""
		return
	}

	rowIndex := min(max(m.cursorRow, 0), len(m.reviewRows)-1)
	fileIndex := m.reviewRows[rowIndex].FileIndex
	if fileIndex < 0 || fileIndex >= len(m.files) {
		m.activeFilePath = ""
		return
	}
	m.activeFilePath = m.files[fileIndex].Path
}

func (m Model) reviewGutter(info viewport.GutterContext) string {
	if info.Soft || info.Index >= len(m.reviewRows) {
		return "  "
	}
	start, end, selected := m.selectedRange()
	if info.Index == m.cursorRow {
		return theme.StatusKeyStyle.Render(nerdIconArrowRight + " ")
	}
	if selected && info.Index >= start && info.Index <= end {
		return theme.SelectedExpander.Render("┃ ")
	}
	return "  "
}

func (m Model) reviewLineStyle(rowIndex int) lipgloss.Style {
	start, end, selected := m.selectedRange()
	if selected && rowIndex >= start && rowIndex <= end {
		return theme.SelectedRowStyle
	}
	if rowIndex == m.cursorRow {
		return theme.CursorRowStyle
	}
	return lipgloss.NewStyle()
}
