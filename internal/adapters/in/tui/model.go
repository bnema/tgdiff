package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	viewport "charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"ero/internal/core"
	"ero/internal/ports"
)

const (
	defaultWidth  = 100
	defaultHeight = 24
	contextStep   = 10
)

type reviewLoader interface {
	LoadReview(request core.ReviewRequest) ([]core.ReviewFile, error)
}

type reviewLoadedMsg struct {
	mode  core.DiffMode
	files []core.ReviewFile
}

type reviewLoadFailedMsg struct {
	previousMode core.DiffMode
	err          error
}

type copyFeedbackExpiredMsg struct {
	id int
}

type clipboardCopiedMsg struct {
	text         string
	lineCount    int
	withMetadata bool
	reviewJSON   bool
	commentCount int
}

type clipboardCopyFailedMsg struct {
	err error
}

type Model struct {
	title              string
	files              []core.ReviewFile
	loader             reviewLoader
	request            core.ReviewRequest
	loading            bool
	loadError          string
	selectedFile       int
	selectedContext    int
	width              int
	height             int
	reviewViewport     viewport.Model
	reviewAnchors      ReviewAnchors
	activeFilePath     string
	cursorRow          int
	selectionAnchorRow *int
	reviewRows         []ReviewRow
	clipboardWriter    ports.ClipboardWriter
	lastCopiedText     string
	copyFeedback       string
	copyFeedbackID     int
	diffMode           core.DiffMode
	nerdFont           bool
	helpActive         bool
	search             searchState
	reviewDraft        *core.ReviewDraft
	commentEditor      *InlineCommentEditor
}

func NewModel(files []core.ReviewFile) Model {
	return NewModelWithTerminal(files, nil)
}

func NewModelWithTerminal(files []core.ReviewFile, terminal ports.Terminal) Model {
	return NewModelWithLoader(files, terminal, nil, core.ReviewRequest{DiffMode: core.DiffModeBranch})
}

func NewModelWithLoader(files []core.ReviewFile, terminal ports.Terminal, loader reviewLoader, request core.ReviewRequest) Model {
	return NewModelWithClipboardWriter(files, terminal, loader, request, nil)
}

func NewModelWithClipboardWriter(files []core.ReviewFile, terminal ports.Terminal, loader reviewLoader, request core.ReviewRequest, clipboardWriter ports.ClipboardWriter) Model {
	if request.DiffMode == "" {
		request.DiffMode = core.DiffModeBranch
	}
	m := Model{
		title:           "ero",
		files:           sortedReviewFiles(files),
		loader:          loader,
		request:         request,
		selectedContext: -1,
		width:           defaultWidth,
		height:          defaultHeight,
		reviewViewport:  viewport.New(),
		clipboardWriter: clipboardWriter,
		diffMode:        request.DiffMode,
		nerdFont:        true,
		search:          newSearchState(),
		reviewDraft:     core.NewReviewDraft(),
	}
	if terminal != nil {
		m.nerdFont = terminal.SupportsNerdFont()
	}
	m.reviewViewport.SoftWrap = false
	m.resetContextSelection()
	m.syncReviewViewport()
	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case reviewLoadedMsg:
		m.loading = false
		m.loadError = ""
		m.diffMode = msg.mode
		m.request.DiffMode = msg.mode
		m.files = sortedReviewFiles(msg.files)
		m.selectedFile = 0
		m.cursorRow = 0
		m.clearSelection()
		m.commentEditor = nil
		m.reviewDraft = core.NewReviewDraft()
		m.resetContextSelection()
		m.reviewViewport.GotoTop()
		m.syncReviewViewport()
		return m, nil
	case reviewLoadFailedMsg:
		m.loading = false
		m.loadError = msg.err.Error()
		m.diffMode = msg.previousMode
		m.request.DiffMode = msg.previousMode
		return m, nil
	case copyFeedbackExpiredMsg:
		if msg.id == m.copyFeedbackID {
			m.copyFeedback = ""
		}
		return m, nil
	case clipboardCopiedMsg:
		m.lastCopiedText = msg.text
		if msg.reviewJSON {
			m.setCopyFeedback(fmt.Sprintf("Review JSON copied (%d %s)", msg.commentCount, pluralize("comment", msg.commentCount)))
			return m, m.expireCopyFeedbackCmd()
		}
		feedback := fmt.Sprintf("Copied %d %s", msg.lineCount, pluralize("line", msg.lineCount))
		if msg.withMetadata {
			feedback += " with metadata"
		}
		m.setCopyFeedback(feedback)
		return m, m.expireCopyFeedbackCmd()
	case clipboardCopyFailedMsg:
		m.setCopyFeedback("Copy failed: " + msg.err.Error())
		return m, m.expireCopyFeedbackCmd()
	case tea.WindowSizeMsg:
		m.width = max(msg.Width, 0)
		m.height = max(msg.Height, 0)
		m.syncReviewViewport()
		return m, nil
	case tea.KeyPressMsg:
		if m.helpActive {
			switch msg.String() {
			case "?", "esc":
				m.helpActive = false
				return m, nil
			case "q", "ctrl+c":
				return m, tea.Quit
			default:
				return m, nil
			}
		}
		if m.commentEditor != nil {
			return m.updateCommentEditor(msg)
		}
		if m.search.active() {
			return m.updateSearch(msg)
		}
		if msg.String() == "?" {
			m.helpActive = true
			return m, nil
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			m.moveCursor(-1)
		case "down", "j":
			m.moveCursor(1)
		case "pgup":
			m.moveCursor(-m.reviewViewport.Height())
		case "pgdown":
			m.moveCursor(m.reviewViewport.Height())
		case "home":
			m.moveCursorToStart()
		case "end":
			m.moveCursorToEnd()
		case "s", "space":
			m.toggleSelection()
		case "esc":
			m.clearSelection()
		case "c":
			return m.openCommentEditor()
		case "C":
			m.clearReviewDraft()
		case "R":
			return m.copyReviewJSONToClipboard()
		case "y":
			return m.copyToClipboard(false)
		case "Y":
			return m.copyToClipboard(true)
		case "f":
			return m.openSearch(searchModeFiles)
		case "/":
			return m.openSearch(searchModeGrep)
		case "d":
			return m.openSearch(searchModeDiff)
		case "left", "h", "p":
			m.moveFile(-1)
		case "right", "l", "n":
			m.moveFile(1)
		case "a":
			m.showAllContext()
		case "enter":
			m.showMoreContext(contextStep)
		default:
			var cmd tea.Cmd
			m.reviewViewport, cmd = m.reviewViewport.Update(msg)
			m.cursorRow = m.clampCursorRow(m.reviewViewport.YOffset())
			m.updateActiveFileFromCursor()
			return m, cmd
		}
		return m, nil
	default:
		return m, nil
	}
}

func (m Model) View() tea.View {
	reviewViewport := m.reviewViewport
	reviewViewport.LeftGutterFunc = m.reviewGutter
	reviewViewport.StyleLineFunc = m.reviewLineStyle
	review := reviewViewport.View()
	if m.loading {
		review = mutedStyle.Render("Loading diff…") + "\n" + review
	} else if m.loadError != "" {
		review = mutedStyle.Render("Failed to load diff: "+m.loadError) + "\n" + review
	}
	content := lipgloss.JoinVertical(lipgloss.Left,
		review,
		NewStatusBar(m.width).Render(StatusModel{
			AppName:       m.title,
			Mode:          diffModeLabel(m.diffMode, m.nerdFont),
			FileCount:     len(m.files),
			CurrentFile:   m.activeLocation(),
			Message:       m.copyFeedback,
			ScrollPercent: m.reviewViewport.ScrollPercent(),
		}),
	)
	if m.search.active() {
		content = m.renderSearchOverlay(content)
	}
	if m.helpActive {
		content = m.renderHelpOverlay(content)
	}
	view := tea.NewView(content)
	view.AltScreen = true
	if m.commentEditor != nil {
		view.KeyboardEnhancements.ReportAllKeysAsEscapeCodes = true
		view.KeyboardEnhancements.ReportAssociatedText = true
	}
	return view
}

func (m *Model) moveFile(delta int) {
	if len(m.files) == 0 {
		return
	}

	m.selectedFile = min(max(m.selectedFile+delta, 0), len(m.files)-1)
	m.clearSelection()
	m.resetContextSelection()
	m.syncReviewViewport()
}

func (m *Model) showMoreContext(count int) {
	fileIndex, sectionIndex, ok := m.contextSectionLocationForExpansion()
	if !ok || !m.contextBarActionAllowed(fileIndex, sectionIndex, ContextBarActionShowMore) {
		return
	}
	m.expandContextSectionNearCursor(fileIndex, sectionIndex, count)
	m.clearSelection()
	m.normalizeContextSelection()
	m.syncReviewViewport()
}

func (m *Model) showAllContext() {
	fileIndex, sectionIndex, ok := m.contextSectionLocationForExpansion()
	if !ok || !m.contextBarActionAllowed(fileIndex, sectionIndex, ContextBarActionShowAll) {
		return
	}
	m.files[fileIndex].Sections[sectionIndex].ExpandAll()
	m.clearSelection()
	m.normalizeContextSelection()
	m.syncReviewViewport()
}

func (m *Model) expandContextSectionNearCursor(fileIndex, sectionIndex, count int) {
	section := &m.files[fileIndex].Sections[sectionIndex]
	switch m.resolveShowMoreDirection(fileIndex, sectionIndex) {
	case contextExpansionAbove:
		section.ExpandAbove(count)
	case contextExpansionBelow:
		section.ExpandBelow(count)
	}
}

func (m *Model) resetContextSelection() {
	if len(m.files) == 0 {
		m.selectedFile = 0
		m.selectedContext = -1
		return
	}

	m.selectedFile = min(max(m.selectedFile, 0), len(m.files)-1)
	if len(m.selectableContextSectionIndexes()) == 0 {
		m.selectedContext = -1
		return
	}
	m.selectedContext = 0
}

func (m *Model) normalizeContextSelection() {
	indexes := m.selectableContextSectionIndexes()
	if len(indexes) == 0 {
		m.selectedContext = -1
		return
	}
	if m.selectedContext < 0 {
		m.selectedContext = 0
		return
	}
	m.selectedContext = min(m.selectedContext, len(indexes)-1)
}

func (m Model) currentFile() *core.ReviewFile {
	if len(m.files) == 0 || m.selectedFile < 0 || m.selectedFile >= len(m.files) {
		return nil
	}
	return &m.files[m.selectedFile]
}

func (m Model) selectableContextSectionIndexes() []int {
	file := m.currentFile()
	if file == nil {
		return nil
	}

	var indexes []int
	for i, section := range file.Sections {
		if section.Kind == core.SectionKindContext && section.HiddenLineCount() > 0 {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

func (m *Model) selectedContextSection() *core.ReviewSection {
	indexes := m.selectableContextSectionIndexes()
	if len(indexes) == 0 || m.selectedContext < 0 || m.selectedContext >= len(indexes) {
		return nil
	}
	return &m.files[m.selectedFile].Sections[indexes[m.selectedContext]]
}

func (m *Model) contextSectionLocationForExpansion() (int, int, bool) {
	if m.cursorRow >= 0 && m.cursorRow < len(m.reviewRows) {
		row := m.reviewRows[m.cursorRow]
		if row.Kind == ReviewRowKindExpander && m.selectContextSection(row.FileIndex, row.SectionIndex) {
			return row.FileIndex, row.SectionIndex, true
		}
	}
	return m.selectedContextLocation()
}

func (m *Model) selectedContextLocation() (int, int, bool) {
	indexes := m.selectableContextSectionIndexes()
	if len(indexes) == 0 || m.selectedContext < 0 || m.selectedContext >= len(indexes) {
		return 0, 0, false
	}
	return m.selectedFile, indexes[m.selectedContext], true
}

type contextExpansionDirection int

const (
	contextExpansionAbove contextExpansionDirection = iota
	contextExpansionBelow
)

func (m Model) contextBarActionAllowed(fileIndex, sectionIndex int, action ContextBarAction) bool {
	if fileIndex < 0 || fileIndex >= len(m.files) || sectionIndex < 0 || sectionIndex >= len(m.files[fileIndex].Sections) {
		return false
	}

	model := NewContextBarViewModel(m.files[fileIndex], sectionIndex)
	if model.HiddenLines == 0 {
		return false
	}

	switch action {
	case ContextBarActionShowMore, ContextBarActionShowAll:
		return true
	default:
		return false
	}
}

func (m Model) resolveShowMoreDirection(fileIndex, sectionIndex int) contextExpansionDirection {
	position := NewContextBarViewModel(m.files[fileIndex], sectionIndex).Position
	switch position {
	case ContextBarAtFileStart:
		return contextExpansionBelow
	case ContextBarAtFileEnd:
		return contextExpansionAbove
	}

	expanderRow := -1
	for rowIndex, row := range m.reviewRows {
		if row.FileIndex == fileIndex && row.SectionIndex == sectionIndex && row.Kind == ReviewRowKindExpander {
			expanderRow = rowIndex
			break
		}
	}
	if expanderRow >= 0 && m.cursorRow > expanderRow {
		return contextExpansionBelow
	}
	return contextExpansionAbove
}

func (m *Model) selectContextSection(fileIndex, sectionIndex int) bool {
	if fileIndex < 0 || fileIndex >= len(m.files) || sectionIndex < 0 || sectionIndex >= len(m.files[fileIndex].Sections) {
		return false
	}
	section := m.files[fileIndex].Sections[sectionIndex]
	if section.Kind != core.SectionKindContext || section.HiddenLineCount() == 0 {
		return false
	}

	ordinal := 0
	for i, candidate := range m.files[fileIndex].Sections {
		if candidate.Kind != core.SectionKindContext || candidate.HiddenLineCount() == 0 {
			continue
		}
		if i == sectionIndex {
			m.selectedFile = fileIndex
			m.selectedContext = ordinal
			return true
		}
		ordinal++
	}
	return false
}

func (m *Model) selectNearestContextToCursor() {
	if m.cursorRow < 0 || m.cursorRow >= len(m.reviewRows) {
		return
	}

	fileIndex := m.reviewRows[m.cursorRow].FileIndex
	if fileIndex < 0 || fileIndex >= len(m.files) {
		return
	}

	nearestRow := -1
	nearestSection := -1
	nearestDistance := 0
	for rowIndex, row := range m.reviewRows {
		if row.FileIndex != fileIndex || row.Kind != ReviewRowKindExpander {
			continue
		}
		distance := rowIndex - m.cursorRow
		if distance < 0 {
			distance = -distance
		}
		if nearestRow == -1 || distance < nearestDistance {
			nearestRow = rowIndex
			nearestSection = row.SectionIndex
			nearestDistance = distance
		}
	}
	if nearestRow == -1 {
		m.selectedFile = fileIndex
		m.selectedContext = -1
		return
	}
	m.selectContextSection(fileIndex, nearestSection)
}

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
	first := m.firstSelectableRow()
	last := m.lastSelectableRow()
	if last < first {
		return 0
	}
	return min(max(row, first), last)
}

func (m Model) selectableRowFrom(row, delta int) int {
	if delta == 0 {
		return m.clampCursorRow(row)
	}
	step := 1
	if delta < 0 {
		step = -1
	}
	remaining := delta
	current := row
	for remaining != 0 {
		next := current + step
		if next < m.firstSelectableRow() || next > m.lastSelectableRow() {
			return m.clampCursorRow(next)
		}
		current = next
		if current >= 0 && current < len(m.reviewRows) && m.reviewRows[current].Selectable {
			remaining -= step
		}
	}
	return m.clampCursorRow(current)
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
		return statusKeyStyle.Render(nerdIconArrowRight + " ")
	}
	if selected && info.Index >= start && info.Index <= end {
		return selectedExpander.Render("┃ ")
	}
	return "  "
}

func (m Model) reviewLineStyle(rowIndex int) lipgloss.Style {
	start, end, selected := m.selectedRange()
	if selected && rowIndex >= start && rowIndex <= end {
		return selectedRowStyle
	}
	if rowIndex == m.cursorRow {
		return cursorRowStyle
	}
	return lipgloss.NewStyle()
}

func (m *Model) toggleSelection() {
	if m.selectionAnchorRow != nil {
		m.clearSelection()
		return
	}
	anchor := m.cursorRow
	m.selectionAnchorRow = &anchor
}

func (m *Model) clearSelection() {
	m.selectionAnchorRow = nil
}

func (m *Model) cancelCommentEditor() {
	m.commentEditor = nil
}

func (m Model) copyToClipboard(withMetadata bool) (Model, tea.Cmd) {
	rows := m.selectedRows()
	if len(rows) == 0 {
		m.setCopyFeedback("No diff lines to copy")
		return m, nil
	}
	if m.clipboardWriter == nil {
		m.setCopyFeedback("Copy failed: clipboard unavailable")
		return m, m.expireCopyFeedbackCmd()
	}

	text := m.copyPlainText()
	if withMetadata {
		text = m.copyMetadataText()
	}
	lineCount := len(rows)
	writer := m.clipboardWriter
	return m, func() tea.Msg {
		if err := writer.WriteClipboard(context.Background(), text); err != nil {
			return clipboardCopyFailedMsg{err: err}
		}
		return clipboardCopiedMsg{text: text, lineCount: lineCount, withMetadata: withMetadata}
	}
}

func (m *Model) setCopyFeedback(message string) {
	m.copyFeedbackID++
	m.copyFeedback = message
}

func (m Model) expireCopyFeedbackCmd() tea.Cmd {
	id := m.copyFeedbackID
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return copyFeedbackExpiredMsg{id: id}
	})
}

func (m Model) selectedRange() (int, int, bool) {
	if m.selectionAnchorRow == nil {
		return m.cursorRow, m.cursorRow, false
	}
	start := min(*m.selectionAnchorRow, m.cursorRow)
	end := max(*m.selectionAnchorRow, m.cursorRow)
	return start, end, true
}

func (m Model) selectedRows() []ReviewRow {
	start, end, ok := m.selectedRange()
	if !ok {
		start, end = m.cursorRow, m.cursorRow
	}
	if len(m.reviewRows) == 0 {
		return nil
	}
	start = min(max(start, 0), len(m.reviewRows)-1)
	end = min(max(end, 0), len(m.reviewRows)-1)
	rows := make([]ReviewRow, 0, end-start+1)
	for _, row := range m.reviewRows[start : end+1] {
		if row.Kind == ReviewRowKindLine {
			rows = append(rows, row)
		}
	}
	return rows
}

func (m Model) copyPlainText() string {
	rows := m.selectedRows()
	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		lines = append(lines, plainDiffLine(row.Line))
	}
	return strings.Join(lines, "\n")
}

func (m Model) copyMetadataText() string {
	rows := m.selectedRows()
	if len(rows) == 0 {
		return ""
	}

	var b strings.Builder
	for i := 0; i < len(rows); {
		filePath := rows[i].FilePath
		j := i + 1
		for j < len(rows) && rows[j].FilePath == filePath {
			j++
		}
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString("File: ")
		b.WriteString(filePath)
		b.WriteString("\n")
		b.WriteString("Lines: ")
		b.WriteString(lineRef(rows[i].Line))
		if j-i > 1 {
			b.WriteString(" to ")
			b.WriteString(lineRef(rows[j-1].Line))
		}
		b.WriteString("\n\n```diff\n")
		for _, row := range rows[i:j] {
			b.WriteString(plainDiffLine(row.Line))
			b.WriteString("\n")
		}
		b.WriteString("```")
		i = j
	}
	return b.String()
}

func plainDiffLine(line core.ReviewLine) string {
	prefix := " "
	switch line.Kind {
	case core.LineKindAdded:
		prefix = "+"
	case core.LineKindDeleted:
		prefix = "-"
	}
	return prefix + " " + line.Content
}

func pluralize(word string, count int) string {
	if count == 1 {
		return word
	}
	return word + "s"
}

func lineRef(line core.ReviewLine) string {
	parts := make([]string, 0, 2)
	if line.OldLineNumber > 0 {
		parts = append(parts, fmt.Sprintf("-%d", line.OldLineNumber))
	}
	if line.NewLineNumber > 0 {
		parts = append(parts, fmt.Sprintf("+%d", line.NewLineNumber))
	}
	return strings.Join(parts, "/")
}

func (m Model) activeLocation() string {
	if m.activeFilePath == "" {
		return ""
	}
	if m.cursorRow < 0 || m.cursorRow >= len(m.reviewRows) {
		return m.activeFilePath
	}
	row := m.reviewRows[m.cursorRow]
	if row.Kind != ReviewRowKindLine {
		return m.activeFilePath
	}
	lineNumber := displayLineNumber(row.Line)
	if lineNumber <= 0 {
		return m.activeFilePath
	}
	return fmt.Sprintf("%s:%d", m.activeFilePath, lineNumber)
}

func (m Model) reviewWidth() int {
	if m.width <= 0 {
		return defaultWidth
	}
	return max(m.width, 1)
}

func sortedReviewFiles(files []core.ReviewFile) []core.ReviewFile {
	result := append([]core.ReviewFile(nil), files...)
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].Path < result[j].Path
	})
	return result
}
