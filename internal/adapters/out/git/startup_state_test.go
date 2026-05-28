package git

import (
	"os"
	"path/filepath"
	"testing"

	ggit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tgdiff/internal/core"
)

func TestRepositoryLoaderReadStartupStateDetectsLocalChanges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(t *testing.T, repo *ggit.Repository, dir string)
		want  core.StartupState
	}{
		{
			name: "staged unstaged and untracked changes",
			setup: func(t *testing.T, repo *ggit.Repository, dir string) {
				writeAndCommitFile(t, repo, dir, "staged.txt", "base\n", "base commit")
				writeAndCommitFile(t, repo, dir, "working.txt", "base\n", "working base")

				worktree, err := repo.Worktree()
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filepath.Join(dir, "staged.txt"), []byte("base\nstaged\n"), 0o644))
				_, err = worktree.Add("staged.txt")
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filepath.Join(dir, "working.txt"), []byte("base\nworking\n"), 0o644))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new\n"), 0o644))
			},
			want: core.StartupState{HasStagedChanges: true, HasUnstagedChanges: true, HasUntrackedFiles: true, HasDefaultBranch: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			repo, err := ggit.PlainInit(dir, false)
			require.NoError(t, err)
			tt.setup(t, repo, dir)

			state, err := NewRepositoryLoader().ReadStartupState(dir)
			require.NoError(t, err)
			assert.Equal(t, tt.want.HasStagedChanges, state.HasStagedChanges)
			assert.Equal(t, tt.want.HasUnstagedChanges, state.HasUnstagedChanges)
			assert.Equal(t, tt.want.HasUntrackedFiles, state.HasUntrackedFiles)
			assert.Equal(t, tt.want.HasDefaultBranch, state.HasDefaultBranch)
		})
	}
}

func TestRepositoryLoaderReadStartupStateDetectsUpstreamAheadBehind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(t *testing.T, repo *ggit.Repository, dir string)
		want  core.StartupState
	}{
		{
			name: "ahead of upstream",
			setup: func(t *testing.T, repo *ggit.Repository, dir string) {
				baseHash := writeAndCommitFile(t, repo, dir, "demo.txt", "base\n", "base commit")
				setUpstream(t, repo, dir, "master", "origin", "main", baseHash)
				writeAndCommitFile(t, repo, dir, "demo.txt", "base\nlocal\n", "local commit")
			},
			want: core.StartupState{HasUpstream: true, Ahead: 1, HasDefaultBranch: true},
		},
		{
			name: "behind upstream",
			setup: func(t *testing.T, repo *ggit.Repository, dir string) {
				baseHash := writeAndCommitFile(t, repo, dir, "demo.txt", "base\n", "base commit")
				remoteHash := writeAndCommitFile(t, repo, dir, "demo.txt", "base\nremote\n", "remote commit")
				worktree, err := repo.Worktree()
				require.NoError(t, err)
				require.NoError(t, worktree.Reset(&ggit.ResetOptions{Commit: baseHash, Mode: ggit.HardReset}))
				setUpstream(t, repo, dir, "master", "origin", "main", remoteHash)
			},
			want: core.StartupState{HasUpstream: true, Behind: 1, HasDefaultBranch: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			repo, err := ggit.PlainInit(dir, false)
			require.NoError(t, err)
			tt.setup(t, repo, dir)

			state, err := NewRepositoryLoader().ReadStartupState(dir)
			require.NoError(t, err)
			assert.Equal(t, tt.want.HasUpstream, state.HasUpstream)
			assert.Equal(t, tt.want.Ahead, state.Ahead)
			assert.Equal(t, tt.want.Behind, state.Behind)
			assert.Equal(t, tt.want.HasDefaultBranch, state.HasDefaultBranch)
		})
	}
}

func setUpstream(t *testing.T, repo *ggit.Repository, dir, branch, remote, mergeBranch string, remoteHash plumbing.Hash) {
	t.Helper()
	referenceName := plumbing.ReferenceName("refs/remotes/" + remote + "/" + mergeBranch)
	require.NoError(t, repo.Storer.SetReference(plumbing.NewHashReference(referenceName, remoteHash)))
	config := "\n[remote \"" + remote + "\"]\n\turl = https://example.invalid/repo.git\n\tfetch = +refs/heads/*:refs/remotes/" + remote + "/*\n[branch \"" + branch + "\"]\n\tremote = " + remote + "\n\tmerge = refs/heads/" + mergeBranch + "\n"
	file, err := os.OpenFile(filepath.Join(dir, ".git", "config"), os.O_APPEND|os.O_WRONLY, 0)
	require.NoError(t, err)
	_, err = file.WriteString(config)
	require.NoError(t, err)
	require.NoError(t, file.Close())
}
