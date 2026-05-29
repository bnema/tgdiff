package component

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"ero/internal/adapters/in/tui/presenter"
	"ero/internal/adapters/in/tui/render"
	"ero/internal/adapters/in/tui/theme"
	"ero/internal/core"
)

type ReviewDocument struct {
	width         int
	enterKeyLabel string
}

type ReviewDocumentRender struct {
	Content string
	Lines   []string
	Rows    []ReviewRow
	Anchors ReviewAnchors
}

type ReviewRowKind = presenter.ReviewRowKind

const (
	ReviewRowKindBlank    = presenter.ReviewRowKindBlank
	ReviewRowKindFile     = presenter.ReviewRowKindFile
	ReviewRowKindRule     = presenter.ReviewRowKindRule
	ReviewRowKindLine     = presenter.ReviewRowKindLine
	ReviewRowKindExpander = presenter.ReviewRowKindExpander
	ReviewRowKindMessage  = presenter.ReviewRowKindMessage
)

type ReviewRow = presenter.ReviewRow
type ReviewAnchors = presenter.ReviewAnchors
type ReviewLineAnchor = presenter.ReviewLineAnchor

func NewReviewDocument(width int, enterKeyLabel string) ReviewDocument {
	return ReviewDocument{width: width, enterKeyLabel: enterKeyLabel}
}

func (c ReviewDocument) Render(files []core.ReviewFile, selectedContext int) string {
	return c.RenderWithAnchors(files, -1, selectedContext).Content
}

func (c ReviewDocument) RenderWithAnchors(files []core.ReviewFile, selectedFile, selectedContext int) ReviewDocumentRender {
	anchors := ReviewAnchors{FileRows: map[int]int{}, LineRows: map[ReviewLineAnchor]int{}}
	if len(files) == 0 {
		lines := []string{
			theme.PanelTitleStyle.Render("Review"),
			theme.MutedStyle.Render("No files to review"),
		}
		rows := []ReviewRow{
			{Kind: ReviewRowKindMessage, Text: lines[0]},
			{Kind: ReviewRowKindMessage, Text: lines[1]},
		}
		return ReviewDocumentRender{Content: strings.Join(lines, "\n"), Lines: lines, Rows: rows, Anchors: anchors}
	}

	lines := make([]string, 0)
	rows := make([]ReviewRow, 0)
	selectedContextOrdinal := 0
	for fileIndex, file := range files {
		if fileIndex > 0 {
			lines = append(lines, "")
			rows = append(rows, ReviewRow{Kind: ReviewRowKindBlank, FileIndex: fileIndex, FilePath: file.Path})
		}
		anchors.FileRows[fileIndex] = len(lines)
		header := renderFileHeader(file, c.width)
		lines = append(lines, header)
		rows = append(rows, ReviewRow{Kind: ReviewRowKindFile, FileIndex: fileIndex, FilePath: file.Path, Text: header})
		rule := theme.FileRuleStyle.Render(strings.Repeat("─", max(c.width, 1)))
		lines = append(lines, rule)
		rows = append(rows, ReviewRow{Kind: ReviewRowKindRule, FileIndex: fileIndex, FilePath: file.Path, Text: rule})

		numberWidth := render.LineNumberWidth(file)
		contextBar := NewContextBar(c.width, c.enterKeyLabel)
		for sectionIndex, section := range file.Sections {
			selected := false
			if fileIndex == selectedFile && section.Kind == core.SectionKindContext && section.HiddenLineCount() > 0 {
				selected = selectedContextOrdinal == selectedContext
				selectedContextOrdinal++
			}
			switch section.Kind {
			case core.SectionKindChanged:
				for lineIndex, line := range section.VisibleLines() {
					anchors.LineRows[ReviewLineAnchor{FileIndex: fileIndex, SectionIndex: sectionIndex, LineIndex: lineIndex}] = len(lines)
					renderedLine := render.ReviewLine(line, numberWidth)
					lines = append(lines, renderedLine)
					rows = append(rows, ReviewRow{Kind: ReviewRowKindLine, FileIndex: fileIndex, SectionIndex: sectionIndex, LineIndex: lineIndex, FilePath: file.Path, Line: line, Text: renderedLine, Selectable: true})
				}
			case core.SectionKindContext:
				aboveCount := min(section.ExpandedAbove, len(section.Lines))
				for lineIndex := range aboveCount {
					line := section.Lines[lineIndex]
					anchors.LineRows[ReviewLineAnchor{FileIndex: fileIndex, SectionIndex: sectionIndex, LineIndex: lineIndex}] = len(lines)
					renderedLine := render.ReviewLine(line, numberWidth)
					lines = append(lines, renderedLine)
					rows = append(rows, ReviewRow{Kind: ReviewRowKindLine, FileIndex: fileIndex, SectionIndex: sectionIndex, LineIndex: lineIndex, FilePath: file.Path, Line: line, Text: renderedLine, Selectable: true})
				}
				if section.HiddenLineCount() > 0 {
					renderedExpander := contextBar.View(NewContextBarViewModel(file, sectionIndex), selected)
					lines = append(lines, renderedExpander)
					rows = append(rows, ReviewRow{Kind: ReviewRowKindExpander, FileIndex: fileIndex, SectionIndex: sectionIndex, FilePath: file.Path, Text: renderedExpander, Selectable: true})
				}
				belowCount := min(section.ExpandedBelow, len(section.Lines)-aboveCount)
				belowStart := len(section.Lines) - belowCount
				for lineIndex := belowStart; lineIndex < len(section.Lines); lineIndex++ {
					line := section.Lines[lineIndex]
					anchors.LineRows[ReviewLineAnchor{FileIndex: fileIndex, SectionIndex: sectionIndex, LineIndex: lineIndex}] = len(lines)
					renderedLine := render.ReviewLine(line, numberWidth)
					lines = append(lines, renderedLine)
					rows = append(rows, ReviewRow{Kind: ReviewRowKindLine, FileIndex: fileIndex, SectionIndex: sectionIndex, LineIndex: lineIndex, FilePath: file.Path, Line: line, Text: renderedLine, Selectable: true})
				}
			}
		}
	}

	return ReviewDocumentRender{Content: strings.Join(lines, "\n"), Lines: lines, Rows: rows, Anchors: anchors}
}

func renderFileHeader(file core.ReviewFile, width int) string {
	stats := fileStats(file)
	left := theme.FileHeaderStyle.Render(file.Path)
	right := theme.MutedStyle.Render(stats)
	space := max(width-lipgloss.Width(left)-lipgloss.Width(right), 1)
	return left + strings.Repeat(" ", space) + right
}

func fileStats(file core.ReviewFile) string {
	added := 0
	deleted := 0
	for _, section := range file.Sections {
		for _, line := range section.Lines {
			switch line.Kind {
			case core.LineKindAdded:
				added++
			case core.LineKindDeleted:
				deleted++
			}
		}
	}
	return fmt.Sprintf("+%d -%d", added, deleted)
}
