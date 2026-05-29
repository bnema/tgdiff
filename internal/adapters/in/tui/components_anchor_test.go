package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ero/internal/core"
)

func TestReviewDocumentRenderWithAnchorsRecordsFileAndVisibleLineRows(t *testing.T) {
	t.Parallel()

	files := []core.ReviewFile{
		{
			Path: "a.go",
			Sections: []core.ReviewSection{
				{ID: "changed", Kind: core.SectionKindChanged, Lines: []core.ReviewLine{{NewLineNumber: 1, Content: "package a", Kind: core.LineKindAdded}}},
				{ID: "context", Kind: core.SectionKindContext, Lines: []core.ReviewLine{
					{OldLineNumber: 2, NewLineNumber: 2, Content: "hidden", Kind: core.LineKindUnchanged},
					{OldLineNumber: 3, NewLineNumber: 3, Content: "visible", Kind: core.LineKindUnchanged},
				}, ExpandedBelow: 1},
			},
		},
		{
			Path:     "b.go",
			Sections: []core.ReviewSection{{ID: "changed", Kind: core.SectionKindChanged, Lines: []core.ReviewLine{{NewLineNumber: 4, Content: "package b", Kind: core.LineKindAdded}}}},
		},
	}

	rendered := NewReviewDocument(80).RenderWithAnchors(files, -1, -1)

	assert.Contains(t, rendered.Content, "a.go")
	assert.Equal(t, 0, rendered.Anchors.FileRows[0])
	assert.Equal(t, 6, rendered.Anchors.FileRows[1])
	assert.Equal(t, 2, rendered.Anchors.LineRows[ReviewLineAnchor{FileIndex: 0, SectionIndex: 0, LineIndex: 0}])
	assert.Equal(t, 4, rendered.Anchors.LineRows[ReviewLineAnchor{FileIndex: 0, SectionIndex: 1, LineIndex: 1}])
	_, hiddenAnchored := rendered.Anchors.LineRows[ReviewLineAnchor{FileIndex: 0, SectionIndex: 1, LineIndex: 0}]
	assert.False(t, hiddenAnchored)
}

func TestModelSyncReviewViewportStoresRenderAnchors(t *testing.T) {
	t.Parallel()

	model := NewModel([]core.ReviewFile{reviewFile("demo.go", "package main")})

	require.Equal(t, 0, model.reviewAnchors.FileRows[0])
	assert.Equal(t, 2, model.reviewAnchors.LineRows[ReviewLineAnchor{FileIndex: 0, SectionIndex: 0, LineIndex: 0}])
}
