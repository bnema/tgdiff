package git

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	ggit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepositoryLoaderLoadBranchDiffUsesMergeBaseAgainstRemoteBaseBranch(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	repo, err := ggit.PlainInit(dir, false)
	require.NoError(t, err)

	baseHash := writeAndCommitFile(t, repo, dir, "demo.txt", "base\n", "base commit")
	require.NoError(t, repo.Storer.SetReference(plumbing.NewHashReference(
		plumbing.ReferenceName("refs/remotes/origin/main"),
		baseHash,
	)))

	worktree, err := repo.Worktree()
	require.NoError(t, err)
	err = worktree.Checkout(&ggit.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName("feature"),
		Create: true,
	})
	require.NoError(t, err)
	featureHash := writeAndCommitFile(t, repo, dir, "demo.txt", "base\nfeature\n", "feature commit")

	err = worktree.Checkout(&ggit.CheckoutOptions{Branch: plumbing.NewBranchReferenceName("master")})
	require.NoError(t, err)
	mainHash := writeAndCommitFile(t, repo, dir, "demo.txt", "base\nmain only\n", "main commit")
	require.NoError(t, repo.Storer.SetReference(plumbing.NewHashReference(
		plumbing.ReferenceName("refs/remotes/origin/main"),
		mainHash,
	)))

	require.NoError(t, repo.Storer.SetReference(plumbing.NewHashReference(
		plumbing.NewBranchReferenceName("main"),
		featureHash,
	)))

	err = worktree.Checkout(&ggit.CheckoutOptions{Branch: plumbing.NewBranchReferenceName("feature")})
	require.NoError(t, err)

	loader := NewRepositoryLoader()
	diff, err := loader.LoadBranchDiff(dir, "main")
	require.NoError(t, err)
	assert.Contains(t, diff, "diff --git")
	assert.Contains(t, diff, "+feature")
	assert.NotContains(t, diff, "main only")
}

func TestRepositoryLoaderLoadBranchDiffIncludesUntrackedFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	repo, err := ggit.PlainInit(dir, false)
	require.NoError(t, err)
	writeAndCommitFile(t, repo, dir, "tracked.txt", "base\n", "base commit")

	err = os.WriteFile(filepath.Join(dir, "new.txt"), []byte("first\nsecond\n"), 0o644)
	require.NoError(t, err)

	loader := NewRepositoryLoader()
	diff, err := loader.LoadBranchDiff(dir, "master")
	require.NoError(t, err)
	assert.Contains(t, diff, "diff --git a/new.txt b/new.txt")
	assert.Contains(t, diff, "--- /dev/null")
	assert.Contains(t, diff, "+++ b/new.txt")
	assert.Contains(t, diff, "+first")
	assert.Contains(t, diff, "+second")
}

func TestRepositoryLoaderLoadBranchDiffErrorsWhenBaseBranchMissing(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	repo, err := ggit.PlainInit(dir, false)
	require.NoError(t, err)
	writeAndCommitFile(t, repo, dir, "demo.txt", "base\n", "base commit")

	loader := NewRepositoryLoader()
	_, err = loader.LoadBranchDiff(dir, "missing")
	require.Error(t, err)
}

func writeAndCommitFile(t *testing.T, repo *ggit.Repository, dir, name, content, message string) plumbing.Hash {
	t.Helper()

	err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644)
	require.NoError(t, err)

	worktree, err := repo.Worktree()
	require.NoError(t, err)
	_, err = worktree.Add(name)
	require.NoError(t, err)

	hash, err := worktree.Commit(message, &ggit.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)
	return hash
}
