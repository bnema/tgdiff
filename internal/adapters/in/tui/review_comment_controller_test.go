package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ero/internal/core"
)

func TestReviewCommentRangeFromRowsUsesSelectionEndpoints(t *testing.T) {
	t.Parallel()

	rows := []ReviewRow{
		{Kind: ReviewRowKindLine, FilePath: "demo.go", Line: core.ReviewLine{NewLineNumber: 10, Kind: core.LineKindAdded}},
		{Kind: ReviewRowKindLine, FilePath: "demo.go", Line: core.ReviewLine{NewLineNumber: 11, Kind: core.LineKindAdded}},
		{Kind: ReviewRowKindLine, FilePath: "demo.go", Line: core.ReviewLine{OldLineNumber: 12, Kind: core.LineKindDeleted}},
	}

	filePath, lineRange, ok := reviewCommentRangeFromRows(rows)

	require.True(t, ok)
	assert.Equal(t, "demo.go", filePath)
	assert.Equal(t, core.ReviewLineRef{NewLineNumber: 10, Kind: core.LineKindAdded}, lineRange.Start)
	assert.Equal(t, core.ReviewLineRef{OldLineNumber: 12, Kind: core.LineKindDeleted}, lineRange.End)
}

func TestReviewCommentRangeFromRowsRejectsEmptyAndCrossFileSelections(t *testing.T) {
	t.Parallel()

	_, _, ok := reviewCommentRangeFromRows(nil)
	assert.False(t, ok)

	_, _, ok = reviewCommentRangeFromRows([]ReviewRow{
		{Kind: ReviewRowKindLine, FilePath: "a.go", Line: core.ReviewLine{NewLineNumber: 1, Kind: core.LineKindAdded}},
		{Kind: ReviewRowKindLine, FilePath: "b.go", Line: core.ReviewLine{NewLineNumber: 2, Kind: core.LineKindAdded}},
	})
	assert.False(t, ok)
}
