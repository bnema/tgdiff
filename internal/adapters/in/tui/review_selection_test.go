package tui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"ero/internal/core"
)

func TestReviewDocumentHighlightsSelectedContextOnlyInSelectedFile(t *testing.T) {
	t.Parallel()

	files := []core.ReviewFile{
		{
			Path: "a.go",
			Sections: []core.ReviewSection{
				{ID: "a-context", Kind: core.SectionKindContext, Lines: []core.ReviewLine{{NewLineNumber: 1, Content: "a hidden", Kind: core.LineKindUnchanged}}},
			},
		},
		{
			Path: "b.go",
			Sections: []core.ReviewSection{
				{ID: "b-context", Kind: core.SectionKindContext, Lines: []core.ReviewLine{{NewLineNumber: 1, Content: "b hidden", Kind: core.LineKindUnchanged}}},
			},
		},
	}

	rendered := NewReviewDocument(80).RenderWithAnchors(files, 1, 0).Content
	lines := strings.Split(rendered, "\n")

	assert.NotContains(t, lines[2], "\x1b[1;")
	assert.Contains(t, lines[6], "\x1b[1;")
}

func TestModelSelectedContextHighlightSurvivesGrepJumpThatExpandsEarlierContext(t *testing.T) {
	t.Parallel()

	model := NewModel([]core.ReviewFile{{
		Path: "demo.go",
		Sections: []core.ReviewSection{
			{ID: "first", Kind: core.SectionKindContext, Lines: []core.ReviewLine{{NewLineNumber: 1, Content: "needle", Kind: core.LineKindUnchanged}}},
			{ID: "changed", Kind: core.SectionKindChanged, Lines: []core.ReviewLine{{NewLineNumber: 2, Content: "changed", Kind: core.LineKindAdded}}},
			{ID: "second", Kind: core.SectionKindContext, Lines: []core.ReviewLine{{NewLineNumber: 3, Content: "later hidden", Kind: core.LineKindUnchanged}}},
		},
	}})

	model.jumpToLine(SearchResult{FileIndex: 0, SectionIndex: 0, LineIndex: 0})

	view := model.reviewViewport.View()
	assert.Contains(t, stripANSI(view), "1 hidden line")
	assert.Contains(t, view, "\x1b[1;")
}
