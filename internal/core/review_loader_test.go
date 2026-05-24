package core_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"tgdiff/internal/core"
	"tgdiff/internal/ports/mocks"
)

func TestReviewLoaderLoadBuildsReviewFilesAndAppliesSyntaxTokens(t *testing.T) {
	t.Parallel()

	resolver := mocks.NewMockBaseBranchResolver(t)
	diffLoader := mocks.NewMockGitDiffLoader(t)
	tokenizer := mocks.NewMockSyntaxTokenizer(t)
	resolver.EXPECT().ResolveBaseBranch("/repo").Return("main", nil)
	diffLoader.EXPECT().LoadBranchDiff("/repo", "main").Return("diff --git a/demo.go b/demo.go\nindex 1111111..2222222 100644\n--- a/demo.go\n+++ b/demo.go\n@@ -1 +1,2 @@\n package main\n+func main() {}\n", nil)
	tokenizer.EXPECT().Tokenize("demo.go", []string{"package main", "func main() {}"}).Return([][]core.SyntaxToken{
		{{Start: 0, End: 7, Type: core.SemanticTokenKeyword}},
		{{Start: 0, End: 4, Type: core.SemanticTokenKeyword}, {Start: 5, End: 9, Type: core.SemanticTokenFunction}},
	}, nil)
	loader := core.NewReviewLoader(resolver, diffLoader, tokenizer)

	files, err := loader.Load("/repo", 1)
	require.NoError(t, err)

	require.Len(t, files, 1)
	assert.Equal(t, "demo.go", files[0].Path)
	require.NotEmpty(t, files[0].Sections)

	var addedLine *core.ReviewLine
	for i := range files[0].Sections {
		for j := range files[0].Sections[i].Lines {
			line := &files[0].Sections[i].Lines[j]
			if line.Kind == core.LineKindAdded {
				addedLine = line
				break
			}
		}
	}
	require.NotNil(t, addedLine)
	assert.NotEmpty(t, addedLine.SyntaxTokens)
}

func TestReviewLoaderLoadReturnsResolveBaseBranchError(t *testing.T) {
	t.Parallel()

	resolver := mocks.NewMockBaseBranchResolver(t)
	diffLoader := mocks.NewMockGitDiffLoader(t)
	resolver.EXPECT().ResolveBaseBranch("/repo").Return("", errors.New("no base"))
	diffLoader.AssertNotCalled(t, "LoadBranchDiff", mock.Anything, mock.Anything)

	loader := core.NewReviewLoader(resolver, diffLoader, nil)
	files, err := loader.Load("/repo", 1)

	require.Error(t, err)
	assert.Nil(t, files)
	assert.ErrorContains(t, err, "resolve base branch")
}

func TestReviewLoaderLoadRestoresCollapsedGapsFromCurrentFile(t *testing.T) {
	t.Parallel()

	resolver := mocks.NewMockBaseBranchResolver(t)
	diffLoader := mocks.NewMockGitDiffLoader(t)
	reader := mocks.NewMockFileContentReader(t)
	resolver.EXPECT().ResolveBaseBranch("/repo").Return("main", nil)
	diffLoader.EXPECT().LoadBranchDiff("/repo", "main").Return("diff --git a/demo.go b/demo.go\nindex 1111111..2222222 100644\n--- a/demo.go\n+++ b/demo.go\n@@ -1,3 +1,3 @@\n line 1\n-line 2\n+line 2 changed\n line 3\n@@ -8,3 +8,3 @@\n line 8\n-line 9\n+line 9 changed\n line 10\n", nil)
	reader.EXPECT().ReadFileLines("/repo", "demo.go").Return([]string{"line 1", "line 2 changed", "line 3", "line 4", "line 5", "line 6", "line 7", "line 8", "line 9 changed", "line 10"}, nil)
	loader := core.NewReviewLoader(resolver, diffLoader, nil, reader)

	files, err := loader.Load("/repo", 1)
	require.NoError(t, err)
	require.Len(t, files, 1)
	require.Len(t, files[0].Sections, 3)
	assert.Equal(t, core.SectionKindContext, files[0].Sections[1].Kind)
	assert.Equal(t, 4, files[0].Sections[1].HiddenLineCount())
}

