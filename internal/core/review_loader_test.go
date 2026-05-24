package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReviewLoaderLoadBuildsReviewFilesAndAppliesSyntaxTokens(t *testing.T) {
	t.Parallel()

	resolver := &stubBaseBranchResolver{branch: "main"}
	diffLoader := &stubGitDiffLoader{diff: "diff --git a/demo.go b/demo.go\nindex 1111111..2222222 100644\n--- a/demo.go\n+++ b/demo.go\n@@ -1 +1,2 @@\n package main\n+func main() {}\n"}
	tokenizer := &stubSyntaxTokenizer{}
	loader := NewReviewLoader(resolver, diffLoader, tokenizer)

	files, err := loader.Load("/repo", 1)
	require.NoError(t, err)

	assert.Equal(t, "/repo", resolver.repoPath)
	assert.Equal(t, "/repo", diffLoader.repoPath)
	assert.Equal(t, "main", diffLoader.baseBranch)
	require.Len(t, files, 1)
	assert.Equal(t, "demo.go", files[0].Path)
	assert.Equal(t, "demo.go", tokenizer.filename)
	assert.Equal(t, []string{"package main", "func main() {}"}, tokenizer.lines)
	require.NotEmpty(t, files[0].Sections)

	var addedLine *ReviewLine
	for i := range files[0].Sections {
		for j := range files[0].Sections[i].Lines {
			line := &files[0].Sections[i].Lines[j]
			if line.Kind == LineKindAdded {
				addedLine = line
				break
			}
		}
	}
	require.NotNil(t, addedLine)
	assert.NotEmpty(t, addedLine.SyntaxTokens)
}

type stubBaseBranchResolver struct {
	repoPath string
	branch   string
}

func (s *stubBaseBranchResolver) ResolveBaseBranch(repoPath string) (string, error) {
	s.repoPath = repoPath
	return s.branch, nil
}

type stubGitDiffLoader struct {
	repoPath    string
	baseBranch  string
	diff        string
}

func (s *stubGitDiffLoader) LoadBranchDiff(repoPath, baseBranch string) (string, error) {
	s.repoPath = repoPath
	s.baseBranch = baseBranch
	return s.diff, nil
}

func TestReviewLoaderLoadRestoresCollapsedGapsFromCurrentFile(t *testing.T) {
	t.Parallel()

	resolver := &stubBaseBranchResolver{branch: "main"}
	diffLoader := &stubGitDiffLoader{diff: "diff --git a/demo.go b/demo.go\nindex 1111111..2222222 100644\n--- a/demo.go\n+++ b/demo.go\n@@ -1,3 +1,3 @@\n line 1\n-line 2\n+line 2 changed\n line 3\n@@ -8,3 +8,3 @@\n line 8\n-line 9\n+line 9 changed\n line 10\n"}
	reader := &stubFileContentReader{linesByPath: map[string][]string{
		"demo.go": {"line 1", "line 2 changed", "line 3", "line 4", "line 5", "line 6", "line 7", "line 8", "line 9 changed", "line 10"},
	}}
	loader := NewReviewLoader(resolver, diffLoader, nil, reader)

	files, err := loader.Load("/repo", 1)
	require.NoError(t, err)
	require.Len(t, files, 1)
	require.Len(t, files[0].Sections, 3)
	assert.Equal(t, SectionKindContext, files[0].Sections[1].Kind)
	assert.Equal(t, 4, files[0].Sections[1].HiddenLineCount())
}

type stubSyntaxTokenizer struct {
	filename string
	lines    []string
}

func (s *stubSyntaxTokenizer) Tokenize(filename string, lines []string) ([][]SyntaxToken, error) {
	s.filename = filename
	s.lines = append([]string(nil), lines...)
	return [][]SyntaxToken{
		{{Start: 0, End: 7, Type: SemanticTokenKeyword}},
		{{Start: 0, End: 4, Type: SemanticTokenKeyword}, {Start: 5, End: 9, Type: SemanticTokenFunction}},
	}, nil
}

func (s *stubSyntaxTokenizer) Language(filename string) string {
	return "go"
}

type stubFileContentReader struct {
	linesByPath map[string][]string
}

func (s *stubFileContentReader) ReadFileLines(repoPath, path string) ([]string, error) {
	return append([]string(nil), s.linesByPath[path]...), nil
}
