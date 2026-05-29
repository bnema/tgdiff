package tui

import (
	"fmt"
	"strings"

	"ero/internal/core"
)

type ContextBarAction string

const (
	ContextBarActionShowMore ContextBarAction = "show_more"
	ContextBarActionShowAll  ContextBarAction = "show_all"
)

type ContextBarPosition int

const (
	ContextBarBetweenChanges ContextBarPosition = iota
	ContextBarAtFileStart
	ContextBarAtFileEnd
	ContextBarOnlySection
)

type ContextBarViewModel struct {
	HiddenLines int
	Position    ContextBarPosition
}

func NewContextBarViewModel(file core.ReviewFile, sectionIndex int) ContextBarViewModel {
	if sectionIndex < 0 || sectionIndex >= len(file.Sections) {
		return ContextBarViewModel{}
	}

	position := ContextBarBetweenChanges
	switch {
	case len(file.Sections) == 1:
		position = ContextBarOnlySection
	case sectionIndex == 0:
		position = ContextBarAtFileStart
	case sectionIndex == len(file.Sections)-1:
		position = ContextBarAtFileEnd
	}

	return ContextBarViewModel{
		HiddenLines: file.Sections[sectionIndex].HiddenLineCount(),
		Position:    position,
	}
}

type ContextBar struct {
	width int
}

func NewContextBar(width int) ContextBar {
	return ContextBar{width: width}
}

func (c ContextBar) View(model ContextBarViewModel, selected bool) string {
	label := contextBarLabel(model)
	style := mutedStyle
	if selected {
		style = selectedExpander
	}
	return style.Inline(true).MaxWidth(c.width).Render(label)
}

func contextBarLabel(model ContextBarViewModel) string {
	parts := []string{"⋯ " + hiddenLinesLabel(model.HiddenLines) + contextBarLocationLabel(model.Position)}
	parts = append(parts, contextBarActionLabels()...)
	return strings.Join(parts, " · ")
}

func contextBarLocationLabel(position ContextBarPosition) string {
	switch position {
	case ContextBarAtFileStart:
		return " from beginning of file"
	case ContextBarAtFileEnd:
		return " to end of file"
	case ContextBarOnlySection:
		return " in file"
	default:
		return " between changes"
	}
}

func contextBarActionLabels() []string {
	return []string{"[enter] show more", "[a] show all"}
}

func hiddenLinesLabel(hidden int) string {
	if hidden == 1 {
		return "1 hidden line"
	}
	return fmt.Sprintf("%d hidden lines", hidden)
}
