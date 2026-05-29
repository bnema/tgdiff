package core_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"ero/internal/core"
	"ero/internal/ports/mocks"
)

func TestReviewLoaderLoadBuildsReviewFilesAndAppliesSyntaxTokens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "loads diff and applies tokenizer rows"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
		})
	}
}

func TestReviewLoaderLoadReviewDefaultsToBranchDiffMode(t *testing.T) {
	t.Parallel()

	resolver := mocks.NewMockBaseBranchResolver(t)
	diffLoader := mocks.NewMockGitDiffLoader(t)
	resolver.EXPECT().ResolveBaseBranch("/repo").Return("main", nil)
	diffLoader.EXPECT().LoadBranchDiff("/repo", "main").Return("diff --git a/demo.go b/demo.go\nindex 1111111..2222222 100644\n--- a/demo.go\n+++ b/demo.go\n@@ -1 +1,2 @@\n package main\n+func main() {}\n", nil)
	loader := core.NewReviewLoader(resolver, diffLoader, nil)

	files, err := loader.LoadReview(core.ReviewRequest{RepoPath: "/repo", ContextLines: 1})

	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, "demo.go", files[0].Path)
}

func sampleUnifiedDiff() string {
	return "diff --git a/demo.go b/demo.go\nindex 1111111..2222222 100644\n--- a/demo.go\n+++ b/demo.go\n@@ -1 +1,2 @@\n package main\n+func main() {}\n"
}

func TestReviewLoaderLoadReviewDispatchesDiffModes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request core.ReviewRequest
		expect  func(*mocks.MockBaseBranchResolver, *mocks.MockGitDiffLoader)
	}{
		{
			name:    "working tree",
			request: core.ReviewRequest{RepoPath: "/repo", ContextLines: 1, DiffMode: core.DiffModeWorking},
			expect: func(_ *mocks.MockBaseBranchResolver, diffLoader *mocks.MockGitDiffLoader) {
				diffLoader.EXPECT().LoadWorkingTreeDiff("/repo").Return(sampleUnifiedDiff(), nil)
			},
		},
		{
			name:    "staged",
			request: core.ReviewRequest{RepoPath: "/repo", ContextLines: 1, DiffMode: core.DiffModeStaged},
			expect: func(_ *mocks.MockBaseBranchResolver, diffLoader *mocks.MockGitDiffLoader) {
				diffLoader.EXPECT().LoadStagedDiff("/repo").Return(sampleUnifiedDiff(), nil)
			},
		},
		{
			name:    "local",
			request: core.ReviewRequest{RepoPath: "/repo", ContextLines: 1, DiffMode: core.DiffModeLocal},
			expect: func(_ *mocks.MockBaseBranchResolver, diffLoader *mocks.MockGitDiffLoader) {
				diffLoader.EXPECT().LoadLocalDiff("/repo").Return(sampleUnifiedDiff(), nil)
			},
		},
		{
			name:    "upstream default",
			request: core.ReviewRequest{RepoPath: "/repo", ContextLines: 1, DiffMode: core.DiffModeUpstream},
			expect: func(_ *mocks.MockBaseBranchResolver, diffLoader *mocks.MockGitDiffLoader) {
				diffLoader.EXPECT().LoadUpstreamDiff("/repo", "@{upstream}").Return(sampleUnifiedDiff(), nil)
			},
		},
		{
			name:    "upstream explicit",
			request: core.ReviewRequest{RepoPath: "/repo", ContextLines: 1, DiffMode: core.DiffModeUpstream, UpstreamRef: "origin/main"},
			expect: func(_ *mocks.MockBaseBranchResolver, diffLoader *mocks.MockGitDiffLoader) {
				diffLoader.EXPECT().LoadUpstreamDiff("/repo", "origin/main").Return(sampleUnifiedDiff(), nil)
			},
		},
		{
			name:    "commit default",
			request: core.ReviewRequest{RepoPath: "/repo", ContextLines: 1, DiffMode: core.DiffModeCommit},
			expect: func(_ *mocks.MockBaseBranchResolver, diffLoader *mocks.MockGitDiffLoader) {
				diffLoader.EXPECT().LoadCommitDiff("/repo", "HEAD").Return(sampleUnifiedDiff(), nil)
			},
		},
		{
			name:    "commit explicit revision",
			request: core.ReviewRequest{RepoPath: "/repo", ContextLines: 1, DiffMode: core.DiffModeCommit, Revision: "HEAD~1"},
			expect: func(_ *mocks.MockBaseBranchResolver, diffLoader *mocks.MockGitDiffLoader) {
				diffLoader.EXPECT().LoadCommitDiff("/repo", "HEAD~1").Return(sampleUnifiedDiff(), nil)
			},
		},
		{
			name:    "range default",
			request: core.ReviewRequest{RepoPath: "/repo", ContextLines: 1, DiffMode: core.DiffModeRange},
			expect: func(resolver *mocks.MockBaseBranchResolver, diffLoader *mocks.MockGitDiffLoader) {
				resolver.EXPECT().ResolveBaseBranch("/repo").Return("main", nil)
				diffLoader.EXPECT().LoadRangeDiff("/repo", "main", "HEAD").Return(sampleUnifiedDiff(), nil)
			},
		},
		{
			name:    "range explicit revisions",
			request: core.ReviewRequest{RepoPath: "/repo", ContextLines: 1, DiffMode: core.DiffModeRange, BaseRevision: "v1", HeadRevision: "v2"},
			expect: func(_ *mocks.MockBaseBranchResolver, diffLoader *mocks.MockGitDiffLoader) {
				diffLoader.EXPECT().LoadRangeDiff("/repo", "v1", "v2").Return(sampleUnifiedDiff(), nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resolver := mocks.NewMockBaseBranchResolver(t)
			diffLoader := mocks.NewMockGitDiffLoader(t)
			tt.expect(resolver, diffLoader)
			loader := core.NewReviewLoader(resolver, diffLoader, nil)

			files, err := loader.LoadReview(tt.request)

			require.NoError(t, err)
			require.Len(t, files, 1)
			assert.Equal(t, "demo.go", files[0].Path)
		})
	}
}

func TestReviewLoaderLoadReviewReturnsDiffModeErrors(t *testing.T) {
	t.Parallel()

	boom := errors.New("boom")
	tests := []struct {
		name          string
		request       core.ReviewRequest
		expect        func(*mocks.MockBaseBranchResolver, *mocks.MockGitDiffLoader)
		errorContains string
	}{
		{
			name:          "branch diff error is wrapped",
			request:       core.ReviewRequest{RepoPath: "/repo", ContextLines: 1, DiffMode: core.DiffModeBranch},
			errorContains: "load branch diff",
			expect: func(resolver *mocks.MockBaseBranchResolver, diffLoader *mocks.MockGitDiffLoader) {
				resolver.EXPECT().ResolveBaseBranch("/repo").Return("main", nil)
				diffLoader.EXPECT().LoadBranchDiff("/repo", "main").Return("", boom)
			},
		},
		{
			name:          "working tree diff error",
			request:       core.ReviewRequest{RepoPath: "/repo", ContextLines: 1, DiffMode: core.DiffModeWorking},
			errorContains: "boom",
			expect: func(_ *mocks.MockBaseBranchResolver, diffLoader *mocks.MockGitDiffLoader) {
				diffLoader.EXPECT().LoadWorkingTreeDiff("/repo").Return("", boom)
			},
		},
		{
			name:          "staged diff error",
			request:       core.ReviewRequest{RepoPath: "/repo", ContextLines: 1, DiffMode: core.DiffModeStaged},
			errorContains: "boom",
			expect: func(_ *mocks.MockBaseBranchResolver, diffLoader *mocks.MockGitDiffLoader) {
				diffLoader.EXPECT().LoadStagedDiff("/repo").Return("", boom)
			},
		},
		{
			name:          "local diff error",
			request:       core.ReviewRequest{RepoPath: "/repo", ContextLines: 1, DiffMode: core.DiffModeLocal},
			errorContains: "boom",
			expect: func(_ *mocks.MockBaseBranchResolver, diffLoader *mocks.MockGitDiffLoader) {
				diffLoader.EXPECT().LoadLocalDiff("/repo").Return("", boom)
			},
		},
		{
			name:          "upstream diff error",
			request:       core.ReviewRequest{RepoPath: "/repo", ContextLines: 1, DiffMode: core.DiffModeUpstream},
			errorContains: "boom",
			expect: func(_ *mocks.MockBaseBranchResolver, diffLoader *mocks.MockGitDiffLoader) {
				diffLoader.EXPECT().LoadUpstreamDiff("/repo", "@{upstream}").Return("", boom)
			},
		},
		{
			name:          "commit diff error",
			request:       core.ReviewRequest{RepoPath: "/repo", ContextLines: 1, DiffMode: core.DiffModeCommit},
			errorContains: "boom",
			expect: func(_ *mocks.MockBaseBranchResolver, diffLoader *mocks.MockGitDiffLoader) {
				diffLoader.EXPECT().LoadCommitDiff("/repo", "HEAD").Return("", boom)
			},
		},
		{
			name:          "range base resolution error",
			request:       core.ReviewRequest{RepoPath: "/repo", ContextLines: 1, DiffMode: core.DiffModeRange},
			errorContains: "resolve base branch",
			expect: func(resolver *mocks.MockBaseBranchResolver, diffLoader *mocks.MockGitDiffLoader) {
				resolver.EXPECT().ResolveBaseBranch("/repo").Return("", boom)
				diffLoader.AssertNotCalled(t, "LoadRangeDiff", mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			name:          "range diff error",
			request:       core.ReviewRequest{RepoPath: "/repo", ContextLines: 1, DiffMode: core.DiffModeRange, BaseRevision: "v1", HeadRevision: "v2"},
			errorContains: "boom",
			expect: func(_ *mocks.MockBaseBranchResolver, diffLoader *mocks.MockGitDiffLoader) {
				diffLoader.EXPECT().LoadRangeDiff("/repo", "v1", "v2").Return("", boom)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resolver := mocks.NewMockBaseBranchResolver(t)
			diffLoader := mocks.NewMockGitDiffLoader(t)
			tt.expect(resolver, diffLoader)
			loader := core.NewReviewLoader(resolver, diffLoader, nil)

			files, err := loader.LoadReview(tt.request)

			require.Error(t, err)
			assert.Nil(t, files)
			assert.ErrorContains(t, err, tt.errorContains)
		})
	}
}

func TestDiffModeIsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		mode core.DiffMode
		want bool
	}{
		{core.DiffModeBranch, true},
		{core.DiffModeWorking, true},
		{core.DiffModeStaged, true},
		{core.DiffModeLocal, true},
		{core.DiffModeUpstream, true},
		{core.DiffModeCommit, true},
		{core.DiffModeRange, true},
		{"", false},
		{core.DiffMode("bogus"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			assert.Equal(t, tt.want, tt.mode.IsValid())
		})
	}
}

func TestReviewLoaderLoadReviewRejectsInvalidDiffMode(t *testing.T) {
	t.Parallel()

	resolver := mocks.NewMockBaseBranchResolver(t)
	diffLoader := mocks.NewMockGitDiffLoader(t)
	loader := core.NewReviewLoader(resolver, diffLoader, nil)

	files, err := loader.LoadReview(core.ReviewRequest{RepoPath: "/repo", ContextLines: 1, DiffMode: core.DiffMode("bogus")})

	require.Error(t, err)
	assert.Nil(t, files)
	assert.ErrorContains(t, err, "invalid diff mode \"bogus\"")
	resolver.AssertNotCalled(t, "ResolveBaseBranch", mock.Anything)
	diffLoader.AssertNotCalled(t, "LoadBranchDiff", mock.Anything, mock.Anything)
}

func TestReviewLoaderLoadReturnsResolveBaseBranchError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		resolveErr error
	}{
		{name: "base branch resolver fails", resolveErr: errors.New("no base")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resolver := mocks.NewMockBaseBranchResolver(t)
			diffLoader := mocks.NewMockGitDiffLoader(t)
			resolver.EXPECT().ResolveBaseBranch("/repo").Return("", tt.resolveErr)
			diffLoader.AssertNotCalled(t, "LoadBranchDiff", mock.Anything, mock.Anything)

			loader := core.NewReviewLoader(resolver, diffLoader, nil)
			files, err := loader.Load("/repo", 1)

			require.Error(t, err)
			assert.Nil(t, files)
			assert.ErrorContains(t, err, "resolve base branch")
		})
	}
}

func TestReviewLoaderLoadRestoresCollapsedGapsFromCurrentFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		currentFileLines []string
		expectSections   int
		expectHidden     int
	}{
		{
			name:             "inter hunk gap from current file",
			currentFileLines: []string{"line 1", "line 2 changed", "line 3", "line 4", "line 5", "line 6", "line 7", "line 8", "line 9 changed", "line 10"},
			expectSections:   3,
			expectHidden:     4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resolver := mocks.NewMockBaseBranchResolver(t)
			diffLoader := mocks.NewMockGitDiffLoader(t)
			reader := mocks.NewMockFileContentReader(t)
			resolver.EXPECT().ResolveBaseBranch("/repo").Return("main", nil)
			diffLoader.EXPECT().LoadBranchDiff("/repo", "main").Return("diff --git a/demo.go b/demo.go\nindex 1111111..2222222 100644\n--- a/demo.go\n+++ b/demo.go\n@@ -1,3 +1,3 @@\n line 1\n-line 2\n+line 2 changed\n line 3\n@@ -8,3 +8,3 @@\n line 8\n-line 9\n+line 9 changed\n line 10\n", nil)
			reader.EXPECT().ReadFileLines("/repo", "demo.go").Return(tt.currentFileLines, nil)
			loader := core.NewReviewLoader(resolver, diffLoader, nil, reader)

			files, err := loader.Load("/repo", 1)
			require.NoError(t, err)
			require.Len(t, files, 1)
			require.Len(t, files[0].Sections, tt.expectSections)
			assert.Equal(t, core.SectionKindContext, files[0].Sections[1].Kind)
			assert.Equal(t, tt.expectHidden, files[0].Sections[1].HiddenLineCount())
		})
	}
}
