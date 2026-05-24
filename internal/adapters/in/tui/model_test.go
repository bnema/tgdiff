package tui

import (
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tgdiff/internal/core"
)

func TestModelViewRendersSequentialReviewDocumentWithoutFileExplorer(t *testing.T) {
	t.Parallel()

	model := NewModel([]core.ReviewFile{
		reviewFile("zeta.go", "package zeta"),
		reviewFile("alpha.go", "package alpha"),
	})

	view := model.View().Content

	assert.Contains(t, view, "tgdiff")
	assert.NotContains(t, view, "Files")
	assert.NotContains(t, view, "▸")
	assert.Less(t, strings.Index(view, "alpha.go"), strings.Index(view, "zeta.go"))
	assert.Contains(t, view, "package alpha")
	assert.Contains(t, view, "package zeta")
}

func TestModelScrollsSequentialReviewDocumentWithJKAndArrows(t *testing.T) {
	t.Parallel()

	model := NewModel([]core.ReviewFile{reviewFileWithLines("demo.go", 80)})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	model = updated.(Model)

	initial := model.reviewViewport.YOffset()
	updated, _ = model.Update(tea.KeyPressMsg{Text: "j", Code: 'j'})
	model = updated.(Model)
	assert.Greater(t, model.reviewViewport.YOffset(), initial)

	afterJ := model.reviewViewport.YOffset()
	updated, _ = model.Update(tea.KeyPressMsg{Text: "k", Code: 'k'})
	model = updated.(Model)
	assert.Less(t, model.reviewViewport.YOffset(), afterJ)

	updated, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	model = updated.(Model)
	assert.Greater(t, model.reviewViewport.YOffset(), initial)
}

func TestModelViewRendersPolishedStatusBar(t *testing.T) {
	t.Parallel()

	model := NewModel([]core.ReviewFile{reviewFile("demo.go", "package main")})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 90, Height: 12})
	model = updated.(Model)

	view := stripANSI(model.View().Content)

	assert.Contains(t, view, "tgdiff")
	assert.Contains(t, view, "1 file")
	assert.Contains(t, view, "j/k")
	assert.Contains(t, view, "expand")
	assert.Contains(t, view, "quit")
	for line := range strings.SplitSeq(view, "\n") {
		assert.LessOrEqual(t, len([]rune(line)), 90)
	}
}

func TestFormatReviewLineAppliesSyntaxHighlightingAndLineNumbers(t *testing.T) {
	t.Parallel()

	line := core.ReviewLine{
		OldLineNumber: 4,
		NewLineNumber: 5,
		Content:       "func main() {}",
		Kind:          core.LineKindAdded,
		SyntaxTokens: []core.SyntaxToken{
			{Start: 0, End: 4, Type: core.SemanticTokenKeyword},
			{Start: 5, End: 9, Type: core.SemanticTokenFunction},
		},
	}

	rendered := formatReviewLine(line, 4)

	assert.Contains(t, rendered, "   4")
	assert.Contains(t, rendered, "   5")
	assert.Contains(t, rendered, "+")
	assert.Contains(t, rendered, "func")
	assert.Contains(t, rendered, "main")
	assert.Contains(t, rendered, "\x1b[")
}

func TestModelUpdateKeepsExpanderFocusVisibleAfterSectionExpands(t *testing.T) {
	t.Parallel()

	model := NewModel([]core.ReviewFile{{
		Path: "demo.go",
		Sections: []core.ReviewSection{
			{
				ID:   "context-1",
				Kind: core.SectionKindContext,
				Lines: []core.ReviewLine{
					{NewLineNumber: 1, Content: "line 1", Kind: core.LineKindUnchanged},
					{NewLineNumber: 2, Content: "line 2", Kind: core.LineKindUnchanged},
				},
			},
			{
				ID:   "changed-1",
				Kind: core.SectionKindChanged,
				Lines: []core.ReviewLine{{NewLineNumber: 3, Content: "changed", Kind: core.LineKindAdded}},
			},
			{
				ID:   "context-2",
				Kind: core.SectionKindContext,
				Lines: []core.ReviewLine{
					{NewLineNumber: 4, Content: "line 4", Kind: core.LineKindUnchanged},
					{NewLineNumber: 5, Content: "line 5", Kind: core.LineKindUnchanged},
				},
			},
		},
	}})

	updated, _ := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	model = updated.(Model)

	section := model.selectedContextSection()
	require.NotNil(t, section)
	assert.Equal(t, "context-2", section.ID)
}

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}

func reviewFile(path, content string) core.ReviewFile {
	return core.ReviewFile{
		Path: path,
		Sections: []core.ReviewSection{{
			ID:   path + "-changed",
			Kind: core.SectionKindChanged,
			Lines: []core.ReviewLine{{NewLineNumber: 1, Content: content, Kind: core.LineKindAdded}},
		}},
	}
}

func reviewFileWithLines(path string, count int) core.ReviewFile {
	lines := make([]core.ReviewLine, 0, count)
	for i := 1; i <= count; i++ {
		lines = append(lines, core.ReviewLine{NewLineNumber: i, Content: "line", Kind: core.LineKindAdded})
	}
	return core.ReviewFile{
		Path: path,
		Sections: []core.ReviewSection{{ID: "changed", Kind: core.SectionKindChanged, Lines: lines}},
	}
}
