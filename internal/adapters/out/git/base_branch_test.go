package git

import (
	"testing"

	ggit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepositoryLoaderResolveBaseBranch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func(t *testing.T, repo *ggit.Repository)
		expected string
	}{
		{
			name: "from origin HEAD",
			setup: func(t *testing.T, repo *ggit.Repository) {
				t.Helper()
				setHashRef(t, repo, "refs/remotes/origin/main", "1111111111111111111111111111111111111111")
				setSymbolicRef(t, repo, "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
			},
			expected: "main",
		},
		{
			name: "falls back when origin HEAD target is missing",
			setup: func(t *testing.T, repo *ggit.Repository) {
				t.Helper()
				setSymbolicRef(t, repo, "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
				setHashRef(t, repo, "refs/remotes/origin/master", "2222222222222222222222222222222222222222")
			},
			expected: "master",
		},
		{
			name: "origin HEAD beats fallbacks",
			setup: func(t *testing.T, repo *ggit.Repository) {
				t.Helper()
				setHashRef(t, repo, "refs/remotes/origin/master", "3333333333333333333333333333333333333333")
				setHashRef(t, repo, plumbing.NewBranchReferenceName("main").String(), "4444444444444444444444444444444444444444")
				setHashRef(t, repo, "refs/remotes/origin/main", "5555555555555555555555555555555555555555")
				setSymbolicRef(t, repo, "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
			},
			expected: "main",
		},
		{
			name: "remote fallback beats local fallback",
			setup: func(t *testing.T, repo *ggit.Repository) {
				t.Helper()
				setHashRef(t, repo, "refs/remotes/origin/master", "6666666666666666666666666666666666666666")
				setHashRef(t, repo, plumbing.NewBranchReferenceName("main").String(), "7777777777777777777777777777777777777777")
			},
			expected: "master",
		},
		{
			name: "local main beats local master",
			setup: func(t *testing.T, repo *ggit.Repository) {
				t.Helper()
				setHashRef(t, repo, plumbing.NewBranchReferenceName("main").String(), "8888888888888888888888888888888888888888")
				setHashRef(t, repo, plumbing.NewBranchReferenceName("master").String(), "9999999999999999999999999999999999999999")
			},
			expected: "main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			repo, err := ggit.PlainInit(dir, false)
			require.NoError(t, err)
			tt.setup(t, repo)

			loader := NewRepositoryLoader()
			branch, err := loader.ResolveBaseBranch(dir)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, branch)
		})
	}
}

func setHashRef(t *testing.T, repo *ggit.Repository, name, hash string) {
	t.Helper()
	require.NoError(t, repo.Storer.SetReference(plumbing.NewHashReference(
		plumbing.ReferenceName(name),
		plumbing.NewHash(hash),
	)))
}

func setSymbolicRef(t *testing.T, repo *ggit.Repository, name, target string) {
	t.Helper()
	require.NoError(t, repo.Storer.SetReference(plumbing.NewSymbolicReference(
		plumbing.ReferenceName(name),
		plumbing.ReferenceName(target),
	)))
}
