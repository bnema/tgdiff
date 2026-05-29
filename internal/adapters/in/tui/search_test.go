package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ero/internal/core"
)

func TestFuzzyFileResultsMatchPathCharactersInOrder(t *testing.T) {
	t.Parallel()

	files := []core.ReviewFile{
		reviewFile("internal/adapters/in/tui/model.go", "package tui"),
		reviewFile("internal/core/review_loader.go", "package core"),
		reviewFile("README.md", "# Ero"),
	}

	results := fuzzyFileResults(files, "tuim")

	require.NotEmpty(t, results)
	assert.Equal(t, "internal/adapters/in/tui/model.go", results[0].Path)
	assert.Equal(t, 0, results[0].FileIndex)
	assert.NotContains(t, resultPaths(results), "README.md")
}

func TestFuzzyFileResultsReturnAllFilesSortedByPathForEmptyQuery(t *testing.T) {
	t.Parallel()

	files := []core.ReviewFile{
		reviewFile("b.go", "package b"),
		reviewFile("a.go", "package a"),
	}

	results := fuzzyFileResults(files, "")

	require.Len(t, results, 2)
	assert.Equal(t, "a.go", results[0].Path)
	assert.Equal(t, "b.go", results[1].Path)
}

func TestGrepResultsSearchAllReviewLinesInDocumentOrder(t *testing.T) {
	t.Parallel()

	files := []core.ReviewFile{
		{
			Path: "a.go",
			Sections: []core.ReviewSection{
				{ID: "changed", Kind: core.SectionKindChanged, Lines: []core.ReviewLine{{NewLineNumber: 10, Content: "func Target() {}", Kind: core.LineKindAdded}}},
				{ID: "hidden", Kind: core.SectionKindContext, Lines: []core.ReviewLine{{OldLineNumber: 11, NewLineNumber: 11, Content: "target in hidden context", Kind: core.LineKindUnchanged}}},
			},
		},
		{
			Path:     "b.go",
			Sections: []core.ReviewSection{{ID: "changed", Kind: core.SectionKindChanged, Lines: []core.ReviewLine{{NewLineNumber: 3, Content: "no match", Kind: core.LineKindAdded}}}},
		},
	}

	results := grepResults(files, "target")

	require.Len(t, results, 2)
	assert.Equal(t, SearchResult{FileIndex: 0, SectionIndex: 0, LineIndex: 0, Path: "a.go", LineNumber: 10, Preview: "func Target() {}"}, results[0])
	assert.Equal(t, SearchResult{FileIndex: 0, SectionIndex: 1, LineIndex: 0, Path: "a.go", LineNumber: 11, Preview: "target in hidden context"}, results[1])
}

func TestGrepResultsIgnoreEmptyQuery(t *testing.T) {
	t.Parallel()

	assert.Empty(t, grepResults([]core.ReviewFile{reviewFile("demo.go", "anything")}, ""))
}

func resultPaths(results []SearchResult) []string {
	paths := make([]string, 0, len(results))
	for _, result := range results {
		paths = append(paths, result.Path)
	}
	return paths
}
