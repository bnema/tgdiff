package tui

import (
	"sort"

	viewport "charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"tgdiff/internal/core"
	"tgdiff/internal/ports"
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

type Model struct {
	title           string
	files           []core.ReviewFile
	loader          reviewLoader
	request         core.ReviewRequest
	loading         bool
	loadError       string
	selectedFile    int
	selectedContext int
	width           int
	height          int
	reviewViewport  viewport.Model
	reviewAnchors   ReviewAnchors
	activeFilePath  string
	diffMode        core.DiffMode
	nerdFont        bool
	helpActive      bool
	search          searchState
}

func NewModel(files []core.ReviewFile) Model {
	return NewModelWithTerminal(files, nil)
}

func NewModelWithTerminal(files []core.ReviewFile, terminal ports.Terminal) Model {
	return NewModelWithLoader(files, terminal, nil, core.ReviewRequest{DiffMode: core.DiffModeBranch})
}

func NewModelWithLoader(files []core.ReviewFile, terminal ports.Terminal, loader reviewLoader, request core.ReviewRequest) Model {
	if request.DiffMode == "" {
		request.DiffMode = core.DiffModeBranch
	}
	m := Model{
		title:           "tgdiff",
		files:           sortedReviewFiles(files),
		loader:          loader,
		request:         request,
		selectedContext: -1,
		width:           defaultWidth,
		height:          defaultHeight,
		reviewViewport:  viewport.New(),
		diffMode:        request.DiffMode,
		nerdFont:        true,
		search:          newSearchState(),
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
			m.reviewViewport.ScrollUp(1)
			m.updateActiveFileFromViewport()
		case "down", "j":
			m.reviewViewport.ScrollDown(1)
			m.updateActiveFileFromViewport()
		case "pgup":
			m.reviewViewport.PageUp()
			m.updateActiveFileFromViewport()
		case "pgdown":
			m.reviewViewport.PageDown()
			m.updateActiveFileFromViewport()
		case "home":
			m.reviewViewport.GotoTop()
			m.updateActiveFileFromViewport()
		case "end":
			m.reviewViewport.GotoBottom()
			m.updateActiveFileFromViewport()
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
			m.expandSelectedContextAbove(contextStep)
		case "b":
			m.expandSelectedContextBelow(contextStep)
		case "enter":
			m.expandSelectedContextAll()
		default:
			var cmd tea.Cmd
			m.reviewViewport, cmd = m.reviewViewport.Update(msg)
			m.updateActiveFileFromViewport()
			return m, cmd
		}
		return m, nil
	default:
		return m, nil
	}
}

func (m Model) View() tea.View {
	review := m.reviewViewport.View()
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
			CurrentFile:   m.activeFilePath,
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
	rendered := NewReviewDocument(width).RenderWithAnchors(m.files, m.selectedFile, m.selectedContext)
	currentOffset := m.reviewViewport.YOffset()
	m.reviewViewport.SetWidth(width)
	m.reviewViewport.SetHeight(height)
	m.reviewViewport.SetContent(rendered.Content)
	m.reviewAnchors = rendered.Anchors
	m.reviewViewport.SetYOffset(currentOffset)
	m.updateActiveFileFromViewport()
}

func (m *Model) updateActiveFileFromViewport() {
	if len(m.files) == 0 || len(m.reviewAnchors.FileRows) == 0 {
		m.activeFilePath = ""
		return
	}

	activeIndex := 0
	activeRow := -1
	offset := m.reviewViewport.YOffset()
	for fileIndex, row := range m.reviewAnchors.FileRows {
		if row <= offset && row >= activeRow {
			activeIndex = fileIndex
			activeRow = row
		}
	}
	m.activeFilePath = m.files[activeIndex].Path
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
