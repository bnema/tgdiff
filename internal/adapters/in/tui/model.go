package tui

import (
	"sort"

	viewport "charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"tgdiff/internal/core"
)

const (
	defaultWidth  = 100
	defaultHeight = 24
	contextStep   = 10
)

type Model struct {
	title           string
	files           []core.ReviewFile
	selectedFile    int
	selectedContext int
	width           int
	height          int
	reviewViewport  viewport.Model
}

func NewModel(files []core.ReviewFile) Model {
	m := Model{
		title:           "tgdiff",
		files:           sortedReviewFiles(files),
		selectedContext: -1,
		width:           defaultWidth,
		height:          defaultHeight,
		reviewViewport:  viewport.New(),
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
	case tea.WindowSizeMsg:
		m.width = max(msg.Width, 0)
		m.height = max(msg.Height, 0)
		m.syncReviewViewport()
		return m, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			m.reviewViewport.ScrollUp(1)
		case "down", "j":
			m.reviewViewport.ScrollDown(1)
		case "pgup":
			m.reviewViewport.PageUp()
		case "pgdown":
			m.reviewViewport.PageDown()
		case "home":
			m.reviewViewport.GotoTop()
		case "end":
			m.reviewViewport.GotoBottom()
		case "left", "h", "p":
			m.moveFile(-1)
		case "right", "l", "n":
			m.moveFile(1)
		case "a":
			m.expandSelectedContextAbove(contextStep)
		case "b":
			m.expandSelectedContextBelow(contextStep)
		case "enter":
			m.expandSelectedContextAll()
		default:
			var cmd tea.Cmd
			m.reviewViewport, cmd = m.reviewViewport.Update(msg)
			return m, cmd
		}
		return m, nil
	default:
		return m, nil
	}
}

func (m Model) View() tea.View {
	content := lipgloss.JoinVertical(lipgloss.Left,
		m.reviewViewport.View(),
		NewStatusBar(m.width).Render(StatusModel{
			AppName:       m.title,
			Mode:          "review",
			FileCount:     len(m.files),
			ScrollPercent: m.reviewViewport.ScrollPercent(),
		}),
	)
	view := tea.NewView(content)
	view.AltScreen = true
	return view
}

func (m *Model) moveFile(delta int) {
	if len(m.files) == 0 {
		return
	}

	m.selectedFile = min(max(m.selectedFile+delta, 0), len(m.files)-1)
	m.resetContextSelection()
	m.syncReviewViewport()
}

func (m *Model) moveContext(delta int) {
	indexes := m.selectableContextSectionIndexes()
	if len(indexes) == 0 {
		m.selectedContext = -1
		return
	}
	if m.selectedContext < 0 {
		m.selectedContext = 0
		return
	}

	m.selectedContext = min(max(m.selectedContext+delta, 0), len(indexes)-1)
}

func (m *Model) expandSelectedContextAbove(count int) {
	section := m.selectedContextSection()
	if section == nil {
		return
	}
	section.ExpandAbove(count)
	m.normalizeContextSelection()
	m.syncReviewViewport()
}

func (m *Model) expandSelectedContextBelow(count int) {
	section := m.selectedContextSection()
	if section == nil {
		return
	}
	section.ExpandBelow(count)
	m.normalizeContextSelection()
	m.syncReviewViewport()
}

func (m *Model) expandSelectedContextAll() {
	section := m.selectedContextSection()
	if section == nil {
		return
	}
	section.ExpandAll()
	m.normalizeContextSelection()
	m.syncReviewViewport()
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

func (m *Model) syncReviewViewport() {
	width := m.reviewWidth()
	height := max(m.height-1, 1)
	content := NewReviewDocument(width).Render(m.files, m.selectedContext)
	currentOffset := m.reviewViewport.YOffset()
	m.reviewViewport.SetWidth(width)
	m.reviewViewport.SetHeight(height)
	m.reviewViewport.SetContent(content)
	m.reviewViewport.SetYOffset(currentOffset)
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
