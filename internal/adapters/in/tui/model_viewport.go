package tui

import (
	viewport "charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"

	"ero/internal/adapters/in/tui/theme"
	"ero/internal/core"
)

func (m *Model) syncReviewViewport() {
	width := m.reviewWidth()
	height := max(m.height-1, 1)
	annotations := ReviewAnnotations{}
	if m.reviewDraft != nil {
		annotations.Comments = m.reviewDraft.Comments()
	}
	annotations.RemoteThreads = m.remoteThreads
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
	m.updateAfterCursorMove()
}

func (m *Model) moveCursorToStart() {
	m.cursorRow = m.firstSelectableRow()
	m.updateAfterCursorMoveWithOffset(0)
}

func (m *Model) moveCursorToEnd() {
	m.cursorRow = m.lastSelectableRow()
	m.updateAfterCursorMoveWithOffset(m.cursorRow - m.reviewViewport.Height() + 1)
}

func (m *Model) pageCursor(direction int) {
	if direction == 0 {
		return
	}
	height := max(m.reviewViewport.Height(), 1)
	m.cursorRow = m.selectableRowFrom(m.cursorRow, direction*height)
	m.updateAfterCursorMoveWithOffset(m.reviewViewport.YOffset() + direction*height)
}

func (m *Model) updateAfterCursorMove() {
	m.updateAfterCursorMoveWithOffset(m.reviewViewport.YOffset())
}

func (m *Model) updateAfterCursorMoveWithOffset(preferredOffset int) {
	previousFile := m.selectedFile
	previousContext := m.selectedContext
	m.selectNearestContextToCursor()
	if m.selectedFile != previousFile || m.selectedContext != previousContext {
		m.syncReviewViewport()
		m.reviewViewport.SetYOffset(preferredOffset)
		m.keepCursorVisible()
		return
	}
	m.reviewViewport.SetYOffset(preferredOffset)
	m.keepCursorVisible()
	m.updateActiveFileFromCursor()
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

func (m *Model) keepCursorVisible() {
	height := m.reviewViewport.Height()
	if height <= 0 {
		return
	}
	top := m.reviewViewport.YOffset()
	bottom := top + height - 1
	switch {
	case m.cursorRow < top:
		m.reviewViewport.SetYOffset(m.cursorRow)
	case m.cursorRow > bottom:
		m.reviewViewport.SetYOffset(m.cursorRow - height + 1)
	}
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
	if marker, ok := m.commentRangeGutter(info.Index); ok {
		return marker
	}
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
	if m.rowHasCommentRange(rowIndex) || m.rowHasActiveEditorRange(rowIndex) {
		return theme.CommentRangeRowStyle
	}
	return lipgloss.NewStyle()
}

func (m Model) commentRangeGutter(rowIndex int) (string, bool) {
	if rowIndex < 0 || rowIndex >= len(m.reviewRows) {
		return "", false
	}
	if m.rowHasActiveEditorRange(rowIndex) {
		return commentBlockMarker(m.reviewRows[rowIndex].Line, m.commentEditor.Range), true
	}
	if m.reviewDraft == nil {
		return "", false
	}
	row := m.reviewRows[rowIndex]
	if row.Kind != ReviewRowKindLine {
		return "", false
	}
	for _, comment := range m.reviewDraft.Comments() {
		if comment.FilePath == row.FilePath && lineInReviewRange(row.Line, comment.Range) {
			return commentBlockMarker(row.Line, comment.Range), true
		}
	}
	return "", false
}

func (m Model) rowHasCommentRange(rowIndex int) bool {
	if rowIndex < 0 || rowIndex >= len(m.reviewRows) || m.reviewDraft == nil {
		return false
	}
	row := m.reviewRows[rowIndex]
	if row.Kind != ReviewRowKindLine {
		return false
	}
	for _, comment := range m.reviewDraft.Comments() {
		if comment.FilePath == row.FilePath && lineInReviewRange(row.Line, comment.Range) {
			return true
		}
	}
	return false
}

func (m Model) rowHasActiveEditorRange(rowIndex int) bool {
	if rowIndex < 0 || rowIndex >= len(m.reviewRows) || m.commentEditor == nil {
		return false
	}
	row := m.reviewRows[rowIndex]
	return row.Kind == ReviewRowKindLine && row.FilePath == m.commentEditor.FilePath && lineInReviewRange(row.Line, m.commentEditor.Range)
}

func commentBlockMarker(line core.ReviewLine, lineRange core.ReviewLineRange) string {
	if reviewLineMatchesRef(line, lineRange.Start) {
		return inlineCommentIconStyle.Render("╭ ")
	}
	if reviewLineMatchesRef(line, lineRange.End) {
		return inlineCommentIconStyle.Render("╰ ")
	}
	return inlineCommentIconStyle.Render("│ ")
}

func lineInReviewRange(line core.ReviewLine, lineRange core.ReviewLineRange) bool {
	if line.NewLineNumber > 0 && lineRange.Start.NewLineNumber > 0 && lineRange.End.NewLineNumber > 0 {
		start := min(lineRange.Start.NewLineNumber, lineRange.End.NewLineNumber)
		end := max(lineRange.Start.NewLineNumber, lineRange.End.NewLineNumber)
		return line.NewLineNumber >= start && line.NewLineNumber <= end
	}
	if line.OldLineNumber > 0 && lineRange.Start.OldLineNumber > 0 && lineRange.End.OldLineNumber > 0 {
		start := min(lineRange.Start.OldLineNumber, lineRange.End.OldLineNumber)
		end := max(lineRange.Start.OldLineNumber, lineRange.End.OldLineNumber)
		return line.OldLineNumber >= start && line.OldLineNumber <= end
	}
	return reviewLineMatchesRef(line, lineRange.Start) || reviewLineMatchesRef(line, lineRange.End)
}
