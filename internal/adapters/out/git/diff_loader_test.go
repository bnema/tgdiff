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

func TestRepositoryLoaderLoadBranchDiff(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		baseBranch string
		setup      func(t *testing.T, repo *ggit.Repository, dir string)
		assertion  func(t *testing.T, diff string, err error)
	}{
		{
			name:       "uses merge base against remote base branch",
			baseBranch: "main",
			setup: func(t *testing.T, repo *ggit.Repository, dir string) {
				t.Helper()
				baseHash := writeAndCommitFile(t, repo, dir, "demo.txt", "base\n", "base commit")
				require.NoError(t, repo.Storer.SetReference(plumbing.NewHashReference(
					plumbing.ReferenceName("refs/remotes/origin/main"),
					baseHash,
				)))

				worktree, err := repo.Worktree()
				require.NoError(t, err)
				err = worktree.Checkout(&ggit.CheckoutOptions{Branch: plumbing.NewBranchReferenceName("feature"), Create: true})
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
			},
			assertion: func(t *testing.T, diff string, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Contains(t, diff, "diff --git")
				assert.Contains(t, diff, "+feature")
				assert.NotContains(t, diff, "main only")
			},
		},
		{
			name:       "includes untracked files",
			baseBranch: "master",
			setup: func(t *testing.T, repo *ggit.Repository, dir string) {
				t.Helper()
				writeAndCommitFile(t, repo, dir, "tracked.txt", "base\n", "base commit")
				err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("first\nsecond\n"), 0o644)
				require.NoError(t, err)
			},
			assertion: func(t *testing.T, diff string, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Contains(t, diff, "diff --git a/new.txt b/new.txt")
				assert.Contains(t, diff, "--- /dev/null")
				assert.Contains(t, diff, "+++ b/new.txt")
				assert.Contains(t, diff, "+first")
				assert.Contains(t, diff, "+second")
			},
		},
		{
			name:       "errors when base branch is missing",
			baseBranch: "missing",
			setup: func(t *testing.T, repo *ggit.Repository, dir string) {
				t.Helper()
				writeAndCommitFile(t, repo, dir, "demo.txt", "base\n", "base commit")
			},
			assertion: func(t *testing.T, diff string, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Empty(t, diff)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			repo, err := ggit.PlainInit(dir, false)
			require.NoError(t, err)
			tt.setup(t, repo, dir)

			loader := NewRepositoryLoader()
			diff, err := loader.LoadBranchDiff(dir, tt.baseBranch)
			tt.assertion(t, diff, err)
		})
	}
}

func TestRepositoryLoaderLoadsGitDiffModes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func(t *testing.T, repo *ggit.Repository, dir string) (load func(*RepositoryLoader, string) (string, error))
		contains []string
		excludes []string
	}{
		{
			name: "working tree diff",
			setup: func(t *testing.T, repo *ggit.Repository, dir string) func(*RepositoryLoader, string) (string, error) {
				t.Helper()
				writeAndCommitFile(t, repo, dir, "demo.txt", "base\n", "base commit")
				require.NoError(t, os.WriteFile(filepath.Join(dir, "demo.txt"), []byte("base\nworking\n"), 0o644))
				return func(loader *RepositoryLoader, dir string) (string, error) { return loader.LoadWorkingTreeDiff(dir) }
			},
			contains: []string{"diff --git a/demo.txt b/demo.txt", "+working"},
		},
		{
			name: "staged diff",
			setup: func(t *testing.T, repo *ggit.Repository, dir string) func(*RepositoryLoader, string) (string, error) {
				t.Helper()
				writeAndCommitFile(t, repo, dir, "demo.txt", "base\n", "base commit")
				require.NoError(t, os.WriteFile(filepath.Join(dir, "demo.txt"), []byte("base\nstaged\n"), 0o644))
				worktree, err := repo.Worktree()
				require.NoError(t, err)
				_, err = worktree.Add("demo.txt")
				require.NoError(t, err)
				return func(loader *RepositoryLoader, dir string) (string, error) { return loader.LoadStagedDiff(dir) }
			},
			contains: []string{"diff --git a/demo.txt b/demo.txt", "+staged"},
		},
		{
			name: "local diff includes staged and unstaged tracked changes",
			setup: func(t *testing.T, repo *ggit.Repository, dir string) func(*RepositoryLoader, string) (string, error) {
				t.Helper()
				writeAndCommitFile(t, repo, dir, "staged.txt", "base\n", "base commit")
				writeAndCommitFile(t, repo, dir, "working.txt", "base\n", "working base")
				require.NoError(t, os.WriteFile(filepath.Join(dir, "staged.txt"), []byte("base\nstaged\n"), 0o644))
				worktree, err := repo.Worktree()
				require.NoError(t, err)
				_, err = worktree.Add("staged.txt")
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filepath.Join(dir, "working.txt"), []byte("base\nworking\n"), 0o644))
				return func(loader *RepositoryLoader, dir string) (string, error) { return loader.LoadLocalDiff(dir) }
			},
			contains: []string{"diff --git a/staged.txt b/staged.txt", "+staged", "diff --git a/working.txt b/working.txt", "+working"},
		},
		{
			name: "upstream diff",
			setup: func(t *testing.T, repo *ggit.Repository, dir string) func(*RepositoryLoader, string) (string, error) {
				t.Helper()
				baseHash := writeAndCommitFile(t, repo, dir, "demo.txt", "base\n", "base commit")
				require.NoError(t, repo.Storer.SetReference(plumbing.NewHashReference(plumbing.ReferenceName("refs/remotes/origin/main"), baseHash)))
				writeAndCommitFile(t, repo, dir, "demo.txt", "base\nfeature\n", "feature commit")
				return func(loader *RepositoryLoader, dir string) (string, error) {
					return loader.LoadUpstreamDiff(dir, "origin/main")
				}
			},
			contains: []string{"diff --git a/demo.txt b/demo.txt", "+feature"},
		},
		{
			name: "commit diff includes root commit contents",
			setup: func(t *testing.T, repo *ggit.Repository, dir string) func(*RepositoryLoader, string) (string, error) {
				t.Helper()
				hash := writeAndCommitFile(t, repo, dir, "demo.txt", "root\n", "root commit")
				return func(loader *RepositoryLoader, dir string) (string, error) {
					return loader.LoadCommitDiff(dir, hash.String())
				}
			},
			contains: []string{"diff --git a/demo.txt b/demo.txt", "+root"},
		},
		{
			name: "range diff",
			setup: func(t *testing.T, repo *ggit.Repository, dir string) func(*RepositoryLoader, string) (string, error) {
				t.Helper()
				baseHash := writeAndCommitFile(t, repo, dir, "demo.txt", "base\n", "base commit")
				headHash := writeAndCommitFile(t, repo, dir, "demo.txt", "base\nhead\n", "head commit")
				return func(loader *RepositoryLoader, dir string) (string, error) {
					return loader.LoadRangeDiff(dir, baseHash.String(), headHash.String())
				}
			},
			contains: []string{"diff --git a/demo.txt b/demo.txt", "+head"},
		},
		{
			name: "diff output disables color",
			setup: func(t *testing.T, repo *ggit.Repository, dir string) func(*RepositoryLoader, string) (string, error) {
				t.Helper()
				writeAndCommitFile(t, repo, dir, "demo.txt", "base\n", "base commit")
				config, err := os.OpenFile(filepath.Join(dir, ".git", "config"), os.O_APPEND|os.O_WRONLY, 0)
				require.NoError(t, err)
				_, err = config.WriteString("\n[color]\n\tui = always\n\tdiff = always\n")
				require.NoError(t, err)
				require.NoError(t, config.Close())
				require.NoError(t, os.WriteFile(filepath.Join(dir, "demo.txt"), []byte("base\nplain\n"), 0o644))
				return func(loader *RepositoryLoader, dir string) (string, error) { return loader.LoadWorkingTreeDiff(dir) }
			},
			contains: []string{"+plain"},
			excludes: []string{"\x1b["},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			repo, err := ggit.PlainInit(dir, false)
			require.NoError(t, err)
			load := tt.setup(t, repo, dir)

			diff, err := load(NewRepositoryLoader(), dir)
			require.NoError(t, err)
			for _, expected := range tt.contains {
				assert.Contains(t, diff, expected)
			}
			for _, unexpected := range tt.excludes {
				assert.NotContains(t, diff, unexpected)
			}
		})
	}
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
