package tui

import (
	"fmt"
	"strings"

	basechroma "github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/styles"
	"charm.land/lipgloss/v2"

	"tgdiff/internal/core"
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

func NewReviewDocument(width int) ReviewDocument {
	return ReviewDocument{width: width}
}

func (c ReviewDocument) Render(files []core.ReviewFile, selectedContext int) string {
	if len(files) == 0 {
		return strings.Join([]string{
			panelTitleStyle.Render("Review"),
			mutedStyle.Render("No files to review"),
		}, "\n")
	}

	lines := make([]string, 0)
	contextOrdinal := 0
	for fileIndex, file := range files {
		if fileIndex > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, renderFileHeader(file, c.width))
		lines = append(lines, fileRuleStyle.Render(strings.Repeat("─", max(c.width, 1))))

		numberWidth := lineNumberWidth(file)
		expander := NewContextExpander(c.width)
		for _, section := range file.Sections {
			switch section.Kind {
			case core.SectionKindChanged:
				for _, line := range section.VisibleLines() {
					lines = append(lines, formatReviewLine(line, numberWidth))
				}
			case core.SectionKindContext:
				above, below := splitContextSection(section)
				for _, line := range above {
					lines = append(lines, formatReviewLine(line, numberWidth))
				}
				if hidden := section.HiddenLineCount(); hidden > 0 {
					lines = append(lines, expander.Render(hidden, contextOrdinal == selectedContext))
				}
				for _, line := range below {
					lines = append(lines, formatReviewLine(line, numberWidth))
				}
				contextOrdinal++
			}
		}
	}

	return strings.Join(lines, "\n")
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

type ContextExpander struct {
	width int
}

func NewContextExpander(width int) ContextExpander {
	return ContextExpander{width: width}
}

func (c ContextExpander) Render(hidden int, selected bool) string {
	label := fmt.Sprintf("⋯ %s · [a] above · [b] below · [enter] all", hiddenLinesLabel(hidden))
	style := mutedStyle
	if selected {
		style = selectedExpander
	}
	return style.Width(c.width).Render(label)
}

type StatusModel struct {
	AppName       string
	Mode          string
	FileCount     int
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
	left := lipgloss.JoinHorizontal(lipgloss.Center,
		statusAppStyle.Render(model.AppName),
		statusModeStyle.Render(model.Mode),
		statusInfoStyle.Render(fileCountLabel(model.FileCount)),
		statusInfoStyle.Render(fmt.Sprintf("%3.0f%%", model.ScrollPercent*100)),
	)
	hints := []KeyHint{
		{Key: "↑↓/j/k", Label: "scroll"},
		{Key: "a/b", Label: "expand"},
		{Key: "q", Label: "quit"},
	}
	right := renderKeyHints(hints)
	gap := max(width-lipgloss.Width(left)-lipgloss.Width(right), 1)
	bar := left + statusBaseStyle.Render(strings.Repeat(" ", gap)) + right
	return statusBaseStyle.Width(width).Render(bar)
}

type KeyHint struct {
	Key   string
	Label string
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
	if token.ChromaType != "" {
		if tokenType, err := basechroma.TokenTypeString(token.ChromaType); err == nil {
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

func splitContextSection(section core.ReviewSection) (above []core.ReviewLine, below []core.ReviewLine) {
	if section.Kind != core.SectionKindContext {
		return nil, nil
	}
	if section.HiddenLineCount() == 0 {
		return section.VisibleLines(), nil
	}

	aboveCount := min(section.ExpandedAbove, len(section.Lines))
	if aboveCount > 0 {
		above = append(above, section.Lines[:aboveCount]...)
	}

	belowCount := min(section.ExpandedBelow, len(section.Lines)-aboveCount)
	if belowCount > 0 {
		below = append(below, section.Lines[len(section.Lines)-belowCount:]...)
	}

	return above, below
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

func hiddenLinesLabel(hidden int) string {
	if hidden == 1 {
		return "1 hidden line"
	}
	return fmt.Sprintf("%d hidden lines", hidden)
}
