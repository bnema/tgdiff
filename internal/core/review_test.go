package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildReviewFileCollapsesLargeUnchangedGaps(t *testing.T) {
	t.Parallel()

	lines := []ReviewLine{
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
	}

	reviewFile := BuildReviewFile("demo.go", lines, 1)
	require.Len(t, reviewFile.Sections, 5)

	assert.Equal(t, SectionKindContext, reviewFile.Sections[0].Kind)
	assert.Equal(t, 2, reviewFile.Sections[0].HiddenLineCount())

	assert.Equal(t, SectionKindChanged, reviewFile.Sections[1].Kind)
	require.Len(t, reviewFile.Sections[1].VisibleLines(), 3)
	assert.Equal(t, "line 3", reviewFile.Sections[1].VisibleLines()[0].Content)
	assert.Equal(t, "line 5", reviewFile.Sections[1].VisibleLines()[2].Content)

	assert.Equal(t, SectionKindContext, reviewFile.Sections[2].Kind)
	assert.Equal(t, 2, reviewFile.Sections[2].HiddenLineCount())

	assert.Equal(t, SectionKindChanged, reviewFile.Sections[3].Kind)
	require.Len(t, reviewFile.Sections[3].VisibleLines(), 4)
	assert.Equal(t, "line 8", reviewFile.Sections[3].VisibleLines()[0].Content)
	assert.Equal(t, "line 11", reviewFile.Sections[3].VisibleLines()[3].Content)

	assert.Equal(t, SectionKindContext, reviewFile.Sections[4].Kind)
	assert.Equal(t, 2, reviewFile.Sections[4].HiddenLineCount())
}

func TestBuildReviewFileMergesAdjacentChangedRanges(t *testing.T) {
	t.Parallel()

	lines := []ReviewLine{
		{NewLineNumber: 1, Content: "line 1", Kind: LineKindUnchanged},
		{NewLineNumber: 2, Content: "line 2", Kind: LineKindUnchanged},
		{NewLineNumber: 3, Content: "line 3", Kind: LineKindAdded},
		{NewLineNumber: 4, Content: "line 4", Kind: LineKindUnchanged},
		{NewLineNumber: 5, Content: "line 5", Kind: LineKindDeleted},
		{NewLineNumber: 6, Content: "line 6", Kind: LineKindUnchanged},
		{NewLineNumber: 7, Content: "line 7", Kind: LineKindUnchanged},
	}

	reviewFile := BuildReviewFile("demo.go", lines, 1)
	require.Len(t, reviewFile.Sections, 3)
	assert.Equal(t, SectionKindContext, reviewFile.Sections[0].Kind)
	assert.Equal(t, SectionKindChanged, reviewFile.Sections[1].Kind)
	assert.Equal(t, SectionKindContext, reviewFile.Sections[2].Kind)
	require.Len(t, reviewFile.Sections[1].VisibleLines(), 5)
	assert.Equal(t, "line 2", reviewFile.Sections[1].VisibleLines()[0].Content)
	assert.Equal(t, "line 6", reviewFile.Sections[1].VisibleLines()[4].Content)
}

func TestBuildReviewFileTreatsNegativeContextWindowAsZero(t *testing.T) {
	t.Parallel()

	lines := []ReviewLine{
		{NewLineNumber: 1, Content: "line 1", Kind: LineKindUnchanged},
		{NewLineNumber: 2, Content: "line 2", Kind: LineKindAdded},
		{NewLineNumber: 3, Content: "line 3", Kind: LineKindUnchanged},
	}

	reviewFile := BuildReviewFile("demo.go", lines, -3)
	require.Len(t, reviewFile.Sections, 3)
	assert.Equal(t, 1, reviewFile.Sections[0].HiddenLineCount())
	require.Len(t, reviewFile.Sections[1].VisibleLines(), 1)
	assert.Equal(t, "line 2", reviewFile.Sections[1].VisibleLines()[0].Content)
	assert.Equal(t, 1, reviewFile.Sections[2].HiddenLineCount())
}

func TestReviewSectionExpansion(t *testing.T) {
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

	assert.Empty(t, section.VisibleLines())
	assert.Equal(t, 4, section.HiddenLineCount())

	section.ExpandAbove(1)
	require.Len(t, section.VisibleLines(), 1)
	assert.Equal(t, "a", section.VisibleLines()[0].Content)
	assert.Equal(t, 3, section.HiddenLineCount())

	section.ExpandBelow(2)
	require.Len(t, section.VisibleLines(), 3)
	assert.Equal(t, []string{"a", "c", "d"}, []string{
		section.VisibleLines()[0].Content,
		section.VisibleLines()[1].Content,
		section.VisibleLines()[2].Content,
	})
	assert.Equal(t, 1, section.HiddenLineCount())

	section.ExpandAll()
	require.Len(t, section.VisibleLines(), 4)
	assert.Equal(t, 0, section.HiddenLineCount())
}

func TestReviewSectionExpansionDoesNotDuplicateLinesWhenOverExpanded(t *testing.T) {
	t.Parallel()

	section := ReviewSection{
		ID:   "ctx-2",
		Kind: SectionKindContext,
		Lines: []ReviewLine{
			{NewLineNumber: 30, Content: "a", Kind: LineKindUnchanged},
			{NewLineNumber: 31, Content: "b", Kind: LineKindUnchanged},
			{NewLineNumber: 32, Content: "c", Kind: LineKindUnchanged},
			{NewLineNumber: 33, Content: "d", Kind: LineKindUnchanged},
		},
	}

	section.ExpandAbove(3)
	section.ExpandBelow(3)

	visible := section.VisibleLines()
	require.Len(t, visible, 4)
	assert.Equal(t, []string{"a", "b", "c", "d"}, []string{
		visible[0].Content,
		visible[1].Content,
		visible[2].Content,
		visible[3].Content,
	})
	assert.Equal(t, 0, section.HiddenLineCount())
}
