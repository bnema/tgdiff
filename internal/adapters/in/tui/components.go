package tui

import (
	"ero/internal/adapters/in/tui/component"
	"ero/internal/core"
)

type ReviewDocument struct {
	width    int
	document component.ReviewDocument
}

type ReviewDocumentRender = component.ReviewDocumentRender
type ReviewRowKind = component.ReviewRowKind

const (
	ReviewRowKindBlank    = component.ReviewRowKindBlank
	ReviewRowKindFile     = component.ReviewRowKindFile
	ReviewRowKindRule     = component.ReviewRowKindRule
	ReviewRowKindLine     = component.ReviewRowKindLine
	ReviewRowKindExpander = component.ReviewRowKindExpander
	ReviewRowKindMessage  = component.ReviewRowKindMessage
)

type ReviewRow = component.ReviewRow
type ReviewAnchors = component.ReviewAnchors
type ReviewLineAnchor = component.ReviewLineAnchor

func NewReviewDocument(width int) ReviewDocument {
	return ReviewDocument{width: width, document: component.NewReviewDocument(width, enterKeyLabel())}
}

func (c ReviewDocument) Render(files []core.ReviewFile, selectedContext int) string {
	return c.document.Render(files, selectedContext)
}

func (c ReviewDocument) RenderWithAnchors(files []core.ReviewFile, selectedFile, selectedContext int) ReviewDocumentRender {
	return c.document.RenderWithAnchors(files, selectedFile, selectedContext)
}

type StatusModel = component.StatusModel
type StatusBar = component.StatusBar
type KeyHint = component.KeyHint

func NewStatusBar(width int) StatusBar {
	return component.NewStatusBar(width)
}

func renderKeyHints(hints []KeyHint) string {
	return component.RenderKeyHints(hints)
}
