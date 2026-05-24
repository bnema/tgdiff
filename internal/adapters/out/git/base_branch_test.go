package git

import (
	"testing"

	ggit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepositoryLoaderResolveBaseBranchFromOriginHEAD(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	repo, err := ggit.PlainInit(dir, false)
	require.NoError(t, err)
	require.NoError(t, repo.Storer.SetReference(plumbing.NewHashReference(
		plumbing.ReferenceName("refs/remotes/origin/main"),
		plumbing.NewHash("1111111111111111111111111111111111111111"),
	)))
	require.NoError(t, repo.Storer.SetReference(plumbing.NewSymbolicReference(
		plumbing.ReferenceName("refs/remotes/origin/HEAD"),
		plumbing.ReferenceName("refs/remotes/origin/main"),
	)))

	loader := NewRepositoryLoader()
	branch, err := loader.ResolveBaseBranch(dir)
	require.NoError(t, err)
	assert.Equal(t, "main", branch)
}

func TestRepositoryLoaderResolveBaseBranchFallsBackWhenOriginHEADTargetIsMissing(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	repo, err := ggit.PlainInit(dir, false)
	require.NoError(t, err)
	require.NoError(t, repo.Storer.SetReference(plumbing.NewSymbolicReference(
		plumbing.ReferenceName("refs/remotes/origin/HEAD"),
		plumbing.ReferenceName("refs/remotes/origin/main"),
	)))
	require.NoError(t, repo.Storer.SetReference(plumbing.NewHashReference(
		plumbing.ReferenceName("refs/remotes/origin/master"),
		plumbing.NewHash("2222222222222222222222222222222222222222"),
	)))

	loader := NewRepositoryLoader()
	branch, err := loader.ResolveBaseBranch(dir)
	require.NoError(t, err)
	assert.Equal(t, "master", branch)
}

func TestRepositoryLoaderResolveBaseBranchRespectsCandidatePrecedence(t *testing.T) {
	t.Parallel()

	t.Run("origin HEAD beats fallbacks", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		repo, err := ggit.PlainInit(dir, false)
		require.NoError(t, err)
		require.NoError(t, repo.Storer.SetReference(plumbing.NewHashReference(
			plumbing.ReferenceName("refs/remotes/origin/master"),
			plumbing.NewHash("3333333333333333333333333333333333333333"),
		)))
		require.NoError(t, repo.Storer.SetReference(plumbing.NewHashReference(
			plumbing.NewBranchReferenceName("main"),
			plumbing.NewHash("4444444444444444444444444444444444444444"),
		)))
		require.NoError(t, repo.Storer.SetReference(plumbing.NewHashReference(
			plumbing.ReferenceName("refs/remotes/origin/main"),
			plumbing.NewHash("5555555555555555555555555555555555555555"),
		)))
		require.NoError(t, repo.Storer.SetReference(plumbing.NewSymbolicReference(
			plumbing.ReferenceName("refs/remotes/origin/HEAD"),
			plumbing.ReferenceName("refs/remotes/origin/main"),
		)))

		loader := NewRepositoryLoader()
		branch, err := loader.ResolveBaseBranch(dir)
		require.NoError(t, err)
		assert.Equal(t, "main", branch)
	})

	t.Run("remote fallback beats local fallback", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		repo, err := ggit.PlainInit(dir, false)
		require.NoError(t, err)
		require.NoError(t, repo.Storer.SetReference(plumbing.NewHashReference(
			plumbing.ReferenceName("refs/remotes/origin/master"),
			plumbing.NewHash("6666666666666666666666666666666666666666"),
		)))
		require.NoError(t, repo.Storer.SetReference(plumbing.NewHashReference(
			plumbing.NewBranchReferenceName("main"),
			plumbing.NewHash("7777777777777777777777777777777777777777"),
		)))

		loader := NewRepositoryLoader()
		branch, err := loader.ResolveBaseBranch(dir)
		require.NoError(t, err)
		assert.Equal(t, "master", branch)
	})

	t.Run("local main beats local master", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		repo, err := ggit.PlainInit(dir, false)
		require.NoError(t, err)
		require.NoError(t, repo.Storer.SetReference(plumbing.NewHashReference(
			plumbing.NewBranchReferenceName("main"),
			plumbing.NewHash("8888888888888888888888888888888888888888"),
		)))
		require.NoError(t, repo.Storer.SetReference(plumbing.NewHashReference(
			plumbing.NewBranchReferenceName("master"),
			plumbing.NewHash("9999999999999999999999999999999999999999"),
		)))

		loader := NewRepositoryLoader()
		branch, err := loader.ResolveBaseBranch(dir)
		require.NoError(t, err)
		assert.Equal(t, "main", branch)
	})
}
