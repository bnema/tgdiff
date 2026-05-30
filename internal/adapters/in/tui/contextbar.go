package tui

import (
	"ero/internal/adapters/in/tui/component"
	"ero/internal/core"
)

type ContextBarAction = component.ContextBarAction

const (
	ContextBarActionShowMore = component.ContextBarActionShowMore
	ContextBarActionShowAll  = component.ContextBarActionShowAll
)

type ContextBarPosition = component.ContextBarPosition

const (
	ContextBarBetweenChanges = component.ContextBarBetweenChanges
	ContextBarAtFileStart    = component.ContextBarAtFileStart
	ContextBarAtFileEnd      = component.ContextBarAtFileEnd
	ContextBarOnlySection    = component.ContextBarOnlySection
)

type ContextBarViewModel = component.ContextBarViewModel

func NewContextBarViewModel(file core.ReviewFile, sectionIndex int) ContextBarViewModel {
	return component.NewContextBarViewModel(file, sectionIndex)
}

type ContextBar struct {
	bar component.ContextBar
}

func NewContextBar(width int) ContextBar {
	return ContextBar{bar: component.NewContextBar(width, enterKeyLabel())}
}

func (c ContextBar) View(model ContextBarViewModel, selected bool) string {
	return c.bar.View(model, selected)
}
