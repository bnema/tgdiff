package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	basechroma "github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/styles"

	"ero/internal/core"
)

var (
	fileHeaderStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	fileRuleStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	panelTitleStyle     = lipgloss.NewStyle().Bold(true).Underline(true)
	mutedStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	addedLineStyle      = lipgloss.NewStyle().Background(lipgloss.Color("#011209")).Foreground(lipgloss.Color("#c9d1d9"))
	deletedLineStyle    = lipgloss.NewStyle().Background(lipgloss.Color("#1f0101")).Foreground(lipgloss.Color("#c9d1d9"))
	addedMarkerStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#3fb950")).Bold(true)
	deletedMarkerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff7b72")).Bold(true)
	lineNumberStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#8b949e"))
	selectedExpander    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#58a6ff"))
	cursorRowStyle      = lipgloss.NewStyle().Background(lipgloss.Color("#1f2a44"))
	selectedRowStyle    = lipgloss.NewStyle().Background(lipgloss.Color("#25351f"))
	keywordStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff7b72"))
	functionStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#d2a8ff"))
	typeStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffa657"))
	nameStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("#c9d1d9"))
	stringStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("#a5d6ff"))
	numberStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("#79c0ff"))
	commentStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#8b949e")).Italic(true)
	operatorStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff7b72"))
	punctuationStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#c9d1d9"))
	statusBaseStyle     = lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("252"))
	statusAppStyle      = statusBaseStyle.Bold(true).Background(lipgloss.Color("62")).Foreground(lipgloss.Color("230")).Padding(0, 1)
	statusModeStyle     = statusBaseStyle.Foreground(lipgloss.Color("229")).Padding(0, 1)
	statusInfoStyle     = statusBaseStyle.Foreground(lipgloss.Color("248")).Padding(0, 1)
	statusKeyStyle      = statusBaseStyle.Foreground(lipgloss.Color("81")).Bold(true)
	statusHintTextStyle = statusBaseStyle.Foreground(lipgloss.Color("244"))
)

type ReviewDocument struct {
	width int
}

type ReviewDocumentRender struct {
	Content string
	Lines   []string
	Rows    []ReviewRow
	Anchors ReviewAnchors
}

type ReviewRowKind string

const (
	ReviewRowKindBlank    ReviewRowKind = "blank"
	ReviewRowKindFile     ReviewRowKind = "file"
	ReviewRowKindRule     ReviewRowKind = "rule"
	ReviewRowKindLine     ReviewRowKind = "line"
	ReviewRowKindExpander ReviewRowKind = "expander"
	ReviewRowKindMessage  ReviewRowKind = "message"
)

type ReviewRow struct {
	Kind         ReviewRowKind
	FileIndex    int
	SectionIndex int
	LineIndex    int
	FilePath     string
	Line         core.ReviewLine
	Text         string
	Selectable   bool
}

type ReviewAnchors struct {
	FileRows map[int]int
	LineRows map[ReviewLineAnchor]int
}

type ReviewLineAnchor struct {
	FileIndex    int
	SectionIndex int
	LineIndex    int
}

func NewReviewDocument(width int) ReviewDocument {
	return ReviewDocument{width: width}
}

func (c ReviewDocument) Render(files []core.ReviewFile, selectedContext int) string {
	return c.RenderWithAnchors(files, -1, selectedContext).Content
}

func (c ReviewDocument) RenderWithAnchors(files []core.ReviewFile, selectedFile, selectedContext int) ReviewDocumentRender {
	anchors := ReviewAnchors{FileRows: map[int]int{}, LineRows: map[ReviewLineAnchor]int{}}
	if len(files) == 0 {
		lines := []string{
			panelTitleStyle.Render("Review"),
			mutedStyle.Render("No files to review"),
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
		rule := fileRuleStyle.Render(strings.Repeat("─", max(c.width, 1)))
		lines = append(lines, rule)
		rows = append(rows, ReviewRow{Kind: ReviewRowKindRule, FileIndex: fileIndex, FilePath: file.Path, Text: rule})

		numberWidth := lineNumberWidth(file)
		contextBar := NewContextBar(c.width)
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
					renderedLine := formatReviewLine(line, numberWidth)
					lines = append(lines, renderedLine)
					rows = append(rows, ReviewRow{Kind: ReviewRowKindLine, FileIndex: fileIndex, SectionIndex: sectionIndex, LineIndex: lineIndex, FilePath: file.Path, Line: line, Text: renderedLine, Selectable: true})
				}
			case core.SectionKindContext:
				// Context anchors use section.Lines indexes. Hidden lines intentionally
				// have no row until a jump expands their section and re-renders anchors.
				aboveCount := min(section.ExpandedAbove, len(section.Lines))
				for lineIndex := range aboveCount {
					line := section.Lines[lineIndex]
					anchors.LineRows[ReviewLineAnchor{FileIndex: fileIndex, SectionIndex: sectionIndex, LineIndex: lineIndex}] = len(lines)
					renderedLine := formatReviewLine(line, numberWidth)
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
					renderedLine := formatReviewLine(line, numberWidth)
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
	left := fileHeaderStyle.Render(file.Path)
	right := mutedStyle.Render(stats)
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

type StatusModel struct {
	AppName       string
	Mode          string
	FileCount     int
	CurrentFile   string
	Message       string
	ScrollPercent float64
}

type StatusBar struct {
	width int
}

func NewStatusBar(width int) StatusBar {
	return StatusBar{width: width}
}

func (c StatusBar) Render(model StatusModel) string {
	width := max(c.width, 1)
	right := renderStatusHint(width)
	leftWidth := max(width-lipgloss.Width(right)-1, 0)

	prefix := renderStatusSegments(leftWidth,
		statusSegment{style: statusAppStyle, label: model.AppName},
		statusSegment{style: statusModeStyle, label: model.Mode},
		statusSegment{style: statusInfoStyle, label: fileCountLabel(model.FileCount)},
	)
	percent := renderStatusSegments(leftWidth-lipgloss.Width(prefix), statusSegment{style: statusInfoStyle, label: fmt.Sprintf("%3.0f%%", model.ScrollPercent*100)})

	middleLabel := model.CurrentFile
	if model.Message != "" {
		middleLabel = nerdIconArrowRight + " " + model.Message
	}
	middleWidth := leftWidth - lipgloss.Width(prefix) - lipgloss.Width(percent)
	middle := ""
	if middleLabel != "" && middleWidth > 0 {
		middle = renderStatusSegments(middleWidth, statusSegment{style: statusInfoStyle, label: middleLabel})
	}
	left := prefix + middle + percent
	gap := max(width-lipgloss.Width(left)-lipgloss.Width(right), 0)
	bar := left + statusBaseStyle.Render(strings.Repeat(" ", gap)) + right
	return statusBaseStyle.Width(width).Render(bar)
}

type statusSegment struct {
	style lipgloss.Style
	label string
}

type KeyHint struct {
	Key   string
	Label string
}

func renderStatusHint(width int) string {
	label := "? help"
	full := renderKeyHints([]KeyHint{{Key: "?", Label: "help"}})
	if lipgloss.Width(full) <= width {
		return full
	}
	return statusInfoStyle.Render(truncateRunes(label, max(width-statusInfoStyle.GetHorizontalPadding(), 0)))
}

func renderStatusSegments(width int, segments ...statusSegment) string {
	var rendered strings.Builder
	for _, segment := range segments {
		used := lipgloss.Width(rendered.String())
		remaining := width - used
		if remaining <= 0 {
			break
		}
		padding := segment.style.GetHorizontalPadding()
		labelWidth := remaining - padding
		if labelWidth <= 0 {
			continue
		}
		rendered.WriteString(segment.style.Render(truncateRunes(segment.label, labelWidth)))
	}
	return rendered.String()
}

func renderKeyHints(hints []KeyHint) string {
	parts := make([]string, 0, len(hints))
	for _, hint := range hints {
		parts = append(parts, statusKeyStyle.Render(hint.Key)+statusHintTextStyle.Render(" "+hint.Label))
	}
	return statusBaseStyle.Render(strings.Join(parts, statusHintTextStyle.Render("  ")))
}

func fileCountLabel(count int) string {
	if count == 1 {
		return "1 file"
	}
	return fmt.Sprintf("%d files", count)
}

func truncateRunes(value string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	if width == 1 {
		return "…"
	}
	return string(runes[:width-1]) + "…"
}

func formatReviewLine(line core.ReviewLine, lineNumberWidth int) string {
	oldNum := lineNumberStyle.Render(formatLineNumber(line.OldLineNumber, lineNumberWidth))
	newNum := lineNumberStyle.Render(formatLineNumber(line.NewLineNumber, lineNumberWidth))
	marker := " "
	markerStyle := lipgloss.NewStyle()
	lineStyle := lipgloss.NewStyle()

	switch line.Kind {
	case core.LineKindAdded:
		marker = "+"
		markerStyle = addedMarkerStyle
		lineStyle = addedLineStyle
	case core.LineKindDeleted:
		marker = "-"
		markerStyle = deletedMarkerStyle
		lineStyle = deletedLineStyle
	}

	content := applySyntaxHighlighting(line.Content, line.SyntaxTokens, lineStyle)
	return fmt.Sprintf("%s %s %s %s", oldNum, newNum, markerStyle.Inherit(lineStyle).Render(marker), content)
}

func applySyntaxHighlighting(content string, tokens []core.SyntaxToken, baseStyle lipgloss.Style) string {
	if len(tokens) == 0 {
		return baseStyle.Render(content)
	}

	runes := []rune(content)
	var result strings.Builder
	lastEnd := 0
	background := baseStyle.GetBackground()

	for _, token := range tokens {
		start := min(max(token.Start, 0), len(runes))
		end := min(max(token.End, 0), len(runes))
		if start >= end {
			continue
		}
		if start > lastEnd {
			result.WriteString(baseStyle.Render(string(runes[lastEnd:start])))
		}
		tokenStyle := styleForSyntaxToken(token)
		if background != nil {
			tokenStyle = tokenStyle.Background(background)
		}
		result.WriteString(tokenStyle.Render(string(runes[start:end])))
		lastEnd = end
	}

	if lastEnd < len(runes) {
		result.WriteString(baseStyle.Render(string(runes[lastEnd:])))
	}

	return result.String()
}

func styleForSyntaxToken(token core.SyntaxToken) lipgloss.Style {
	if token.SourceType != "" {
		if tokenType, err := basechroma.TokenTypeString(token.SourceType); err == nil {
			entry := githubDarkStyle().Get(tokenType)
			style := lipgloss.NewStyle()
			if entry.Colour.IsSet() {
				style = style.Foreground(lipgloss.Color(entry.Colour.String()))
			}
			if entry.Bold == basechroma.Yes {
				style = style.Bold(true)
			}
			if entry.Italic == basechroma.Yes {
				style = style.Italic(true)
			}
			if entry.Underline == basechroma.Yes {
				style = style.Underline(true)
			}
			return style
		}
	}
	return styleForToken(token.Type)
}

func githubDarkStyle() *basechroma.Style {
	style := styles.Get("github-dark")
	if style == nil {
		return styles.Fallback
	}
	return style
}

func styleForToken(tokenType core.SemanticTokenType) lipgloss.Style {
	switch tokenType {
	case core.SemanticTokenKeyword:
		return keywordStyle
	case core.SemanticTokenFunction:
		return functionStyle
	case core.SemanticTokenTypeName:
		return typeStyle
	case core.SemanticTokenName:
		return nameStyle
	case core.SemanticTokenString:
		return stringStyle
	case core.SemanticTokenNumber:
		return numberStyle
	case core.SemanticTokenComment:
		return commentStyle
	case core.SemanticTokenOperator:
		return operatorStyle
	case core.SemanticTokenPunctuation:
		return punctuationStyle
	default:
		return lipgloss.NewStyle()
	}
}

func lineNumberWidth(file core.ReviewFile) int {
	width := 4
	for _, section := range file.Sections {
		for _, line := range section.Lines {
			if digits := len(fmt.Sprintf("%d", max(line.OldLineNumber, line.NewLineNumber))); digits > width {
				width = digits
			}
		}
	}
	return width
}

func formatLineNumber(number, width int) string {
	if number <= 0 {
		return strings.Repeat(" ", width)
	}
	return fmt.Sprintf("%*d", width, number)
}
