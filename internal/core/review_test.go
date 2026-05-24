package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildReviewFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		lines         []ReviewLine
		contextWindow int
		expectKinds   []ReviewSectionKind
		expectHidden  []int
		expectVisible map[int][]string
	}{
		{
			name: "collapses large unchanged gaps",
			lines: []ReviewLine{
				{NewLineNumber: 1, Content: "line 1", Kind: LineKindUnchanged},
				{NewLineNumber: 2, Content: "line 2", Kind: LineKindUnchanged},
				{NewLineNumber: 3, Content: "line 3", Kind: LineKindUnchanged},
				{NewLineNumber: 4, Content: "line 4", Kind: LineKindAdded},
				{NewLineNumber: 5, Content: "line 5", Kind: LineKindUnchanged},
				{NewLineNumber: 6, Content: "line 6", Kind: LineKindUnchanged},
				{NewLineNumber: 7, Content: "line 7", Kind: LineKindUnchanged},
				{NewLineNumber: 8, Content: "line 8", Kind: LineKindUnchanged},
				{NewLineNumber: 9, Content: "line 9", Kind: LineKindDeleted},
				{NewLineNumber: 10, Content: "line 10", Kind: LineKindAdded},
				{NewLineNumber: 11, Content: "line 11", Kind: LineKindUnchanged},
				{NewLineNumber: 12, Content: "line 12", Kind: LineKindUnchanged},
				{NewLineNumber: 13, Content: "line 13", Kind: LineKindUnchanged},
			},
			contextWindow: 1,
			expectKinds:   []ReviewSectionKind{SectionKindContext, SectionKindChanged, SectionKindContext, SectionKindChanged, SectionKindContext},
			expectHidden:  []int{2, 0, 2, 0, 2},
			expectVisible: map[int][]string{
				1: {"line 3", "line 4", "line 5"},
				3: {"line 8", "line 9", "line 10", "line 11"},
			},
		},
		{
			name: "merges adjacent changed ranges",
			lines: []ReviewLine{
				{NewLineNumber: 1, Content: "line 1", Kind: LineKindUnchanged},
				{NewLineNumber: 2, Content: "line 2", Kind: LineKindUnchanged},
				{NewLineNumber: 3, Content: "line 3", Kind: LineKindAdded},
				{NewLineNumber: 4, Content: "line 4", Kind: LineKindUnchanged},
				{NewLineNumber: 5, Content: "line 5", Kind: LineKindDeleted},
				{NewLineNumber: 6, Content: "line 6", Kind: LineKindUnchanged},
				{NewLineNumber: 7, Content: "line 7", Kind: LineKindUnchanged},
			},
			contextWindow: 1,
			expectKinds:   []ReviewSectionKind{SectionKindContext, SectionKindChanged, SectionKindContext},
			expectHidden:  []int{1, 0, 1},
			expectVisible: map[int][]string{1: {"line 2", "line 3", "line 4", "line 5", "line 6"}},
		},
		{
			name: "treats negative context window as zero",
			lines: []ReviewLine{
				{NewLineNumber: 1, Content: "line 1", Kind: LineKindUnchanged},
				{NewLineNumber: 2, Content: "line 2", Kind: LineKindAdded},
				{NewLineNumber: 3, Content: "line 3", Kind: LineKindUnchanged},
			},
			contextWindow: -3,
			expectKinds:   []ReviewSectionKind{SectionKindContext, SectionKindChanged, SectionKindContext},
			expectHidden:  []int{1, 0, 1},
			expectVisible: map[int][]string{1: {"line 2"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reviewFile := BuildReviewFile("demo.go", tt.lines, tt.contextWindow)
			require.Len(t, reviewFile.Sections, len(tt.expectKinds))

			for i, expectedKind := range tt.expectKinds {
				assert.Equal(t, expectedKind, reviewFile.Sections[i].Kind)
				assert.Equal(t, tt.expectHidden[i], reviewFile.Sections[i].HiddenLineCount())
			}
			for sectionIndex, expectedContents := range tt.expectVisible {
				visible := reviewFile.Sections[sectionIndex].VisibleLines()
				require.Len(t, visible, len(expectedContents))
				assert.Equal(t, expectedContents, reviewLineContents(visible))
			}
		})
	}
}

func TestReviewSectionExpansion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		operations     []func(*ReviewSection)
		expectContents []string
		expectHidden   int
	}{
		{
			name: "expands above below then all",
			operations: []func(*ReviewSection){
				func(section *ReviewSection) {
					assert.Empty(t, section.VisibleLines())
					assert.Equal(t, 4, section.HiddenLineCount())
				},
				func(section *ReviewSection) {
					section.ExpandAbove(1)
					require.Len(t, section.VisibleLines(), 1)
					assert.Equal(t, "a", section.VisibleLines()[0].Content)
					assert.Equal(t, 3, section.HiddenLineCount())
				},
				func(section *ReviewSection) {
					section.ExpandBelow(2)
					require.Len(t, section.VisibleLines(), 3)
					assert.Equal(t, []string{"a", "c", "d"}, reviewLineContents(section.VisibleLines()))
					assert.Equal(t, 1, section.HiddenLineCount())
				},
				func(section *ReviewSection) { section.ExpandAll() },
			},
			expectContents: []string{"a", "b", "c", "d"},
			expectHidden:   0,
		},
		{
			name: "does not duplicate lines when over expanded",
			operations: []func(*ReviewSection){
				func(section *ReviewSection) { section.ExpandAbove(3) },
				func(section *ReviewSection) { section.ExpandBelow(3) },
			},
			expectContents: []string{"a", "b", "c", "d"},
			expectHidden:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			section := ReviewSection{
				ID:   "ctx-1",
				Kind: SectionKindContext,
				Lines: []ReviewLine{
					{NewLineNumber: 20, Content: "a", Kind: LineKindUnchanged},
					{NewLineNumber: 21, Content: "b", Kind: LineKindUnchanged},
					{NewLineNumber: 22, Content: "c", Kind: LineKindUnchanged},
					{NewLineNumber: 23, Content: "d", Kind: LineKindUnchanged},
				},
			}

			for _, operation := range tt.operations {
				operation(&section)
			}

			visible := section.VisibleLines()
			require.Len(t, visible, len(tt.expectContents))
			assert.Equal(t, tt.expectContents, reviewLineContents(visible))
			assert.Equal(t, tt.expectHidden, section.HiddenLineCount())
		})
	}
}

func reviewLineContents(lines []ReviewLine) []string {
	contents := make([]string, 0, len(lines))
	for _, line := range lines {
		contents = append(contents, line.Content)
	}
	return contents
}
