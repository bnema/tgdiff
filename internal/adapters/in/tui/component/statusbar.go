package component

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"ero/internal/adapters/in/tui/theme"
)

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
		statusSegment{style: theme.StatusAppStyle, label: model.AppName},
		statusSegment{style: theme.StatusModeStyle, label: model.Mode},
		statusSegment{style: theme.StatusInfoStyle, label: fileCountLabel(model.FileCount)},
	)
	percent := renderStatusSegments(leftWidth-lipgloss.Width(prefix), statusSegment{style: theme.StatusInfoStyle, label: fmt.Sprintf("%3.0f%%", model.ScrollPercent*100)})

	middleLabel := model.CurrentFile
	if model.Message != "" {
		middleLabel = " " + model.Message
	}
	middleWidth := leftWidth - lipgloss.Width(prefix) - lipgloss.Width(percent)
	middle := ""
	if middleLabel != "" && middleWidth > 0 {
		middle = renderStatusSegments(middleWidth, statusSegment{style: theme.StatusInfoStyle, label: middleLabel})
	}
	left := prefix + middle + percent
	gap := max(width-lipgloss.Width(left)-lipgloss.Width(right), 0)
	bar := left + theme.StatusBaseStyle.Render(strings.Repeat(" ", gap)) + right
	return theme.StatusBaseStyle.Width(width).Render(bar)
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
	hint := KeyHint{Key: "?", Label: "help"}
	label := hint.Key + " " + hint.Label
	full := RenderKeyHints([]KeyHint{hint})
	if lipgloss.Width(full) <= width {
		return full
	}
	return theme.StatusInfoStyle.Render(TruncateRunes(label, max(width-theme.StatusInfoStyle.GetHorizontalPadding(), 0)))
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
		rendered.WriteString(segment.style.Render(TruncateRunes(segment.label, labelWidth)))
	}
	return rendered.String()
}

func RenderKeyHints(hints []KeyHint) string {
	parts := make([]string, 0, len(hints))
	for _, hint := range hints {
		parts = append(parts, theme.StatusKeyStyle.Render(hint.Key)+theme.StatusHintTextStyle.Render(" "+hint.Label))
	}
	return theme.StatusBaseStyle.Render(strings.Join(parts, theme.StatusHintTextStyle.Render("  ")))
}

func fileCountLabel(count int) string {
	if count == 1 {
		return "1 file"
	}
	return fmt.Sprintf("%d files", count)
}

func TruncateRunes(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if ansi.StringWidth(value) <= width {
		return value
	}
	return ansi.Truncate(value, width, "…")
}
