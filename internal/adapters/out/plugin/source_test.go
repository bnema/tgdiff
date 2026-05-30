package pluginadapter

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSourceGitShorthand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		raw    string
		source Source
	}{
		{
			name: "simple shorthand",
			raw:  "git:github.com/owner/repo",
			source: Source{
				Type: SourceTypeGit, Raw: "git:github.com/owner/repo",
				Repo: "github.com/owner/repo", Host: "github.com", Path: "owner/repo",
			},
		},
		{
			name: "shorthand with ref",
			raw:  "git:github.com/owner/repo@v0.1.0",
			source: Source{
				Type: SourceTypeGit, Raw: "git:github.com/owner/repo@v0.1.0",
				Repo: "github.com/owner/repo", Host: "github.com", Path: "owner/repo",
				Ref: "v0.1.0", Pinned: true,
			},
		},
		{
			name: "shorthand with branch ref",
			raw:  "git:github.com/owner/repo@main",
			source: Source{
				Type: SourceTypeGit, Raw: "git:github.com/owner/repo@main",
				Repo: "github.com/owner/repo", Host: "github.com", Path: "owner/repo",
				Ref: "main", Pinned: true,
			},
		},
		{
			name: "gitlab shorthand",
			raw:  "git:gitlab.com/group/proj@v2.0.0",
			source: Source{
				Type: SourceTypeGit, Raw: "git:gitlab.com/group/proj@v2.0.0",
				Repo: "gitlab.com/group/proj", Host: "gitlab.com", Path: "group/proj",
				Ref: "v2.0.0", Pinned: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			source, err := ParseSource(tt.raw)
			require.NoError(t, err)
			assert.Equal(t, tt.source, source)
		})
	}
}

func TestParseSourceGitURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		raw    string
		source Source
	}{
		{
			name: "https url",
			raw:  "https://github.com/owner/repo.git",
			source: Source{
				Type: SourceTypeGit, Raw: "https://github.com/owner/repo.git",
				Repo: "github.com/owner/repo", Host: "github.com", Path: "owner/repo",
			},
		},
		{
			name: "https url without .git",
			raw:  "https://github.com/owner/repo",
			source: Source{
				Type: SourceTypeGit, Raw: "https://github.com/owner/repo",
				Repo: "github.com/owner/repo", Host: "github.com", Path: "owner/repo",
			},
		},
		{
			name: "ssh url",
			raw:  "git@github.com:owner/repo.git",
			source: Source{
				Type: SourceTypeGit, Raw: "git@github.com:owner/repo.git",
				Repo: "github.com/owner/repo", Host: "github.com", Path: "owner/repo",
			},
		},
		{
			name: "https url with ref",
			raw:  "https://github.com/owner/repo.git@v1.2.3",
			source: Source{
				Type: SourceTypeGit, Raw: "https://github.com/owner/repo.git@v1.2.3",
				Repo: "github.com/owner/repo", Host: "github.com", Path: "owner/repo",
				Ref: "v1.2.3", Pinned: true,
			},
		},
		{
			name: "ssh url with ref",
			raw:  "git@github.com:owner/repo.git@main",
			source: Source{
				Type: SourceTypeGit, Raw: "git@github.com:owner/repo.git@main",
				Repo: "github.com/owner/repo", Host: "github.com", Path: "owner/repo",
				Ref: "main", Pinned: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			source, err := ParseSource(tt.raw)
			require.NoError(t, err)
			assert.Equal(t, tt.source, source)
		})
	}
}

func TestParseSourceLocal(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	t.Parallel()

	t.Run("valid local git repo", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		runGit(t, dir, "init")

		source, err := ParseSource(dir)
		require.NoError(t, err)
		assert.Equal(t, SourceTypeLocal, source.Type)
		assert.Equal(t, "local", source.Host)
		assert.Equal(t, dir, source.LocalPath)
	})

	t.Run("non-git directory fails", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		_, err := ParseSource(dir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not a git repository")
	})

	t.Run("worktree gitfile works", func(t *testing.T) {
		t.Parallel()

		baseDir := t.TempDir()
		runGit(t, baseDir, "init")
		runGit(t, baseDir, "commit", "--allow-empty", "-m", "init")

		wtDir := filepath.Join(t.TempDir(), "worktree")
		runGit(t, baseDir, "worktree", "add", wtDir)

		source, err := ParseSource(wtDir)
		require.NoError(t, err)
		assert.Equal(t, SourceTypeLocal, source.Type)
	})

	t.Run("empty string fails", func(t *testing.T) {
		t.Parallel()

		_, err := ParseSource("")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})
}

func TestXDGDirs(t *testing.T) {
	// Sub-tests use t.Setenv which cannot be parallel.

	t.Run("ConfigDir uses env", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "/custom/config")
		dir := ConfigDir()
		assert.Equal(t, "/custom/config/ero", dir)
	})

	t.Run("DataDir uses env", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", "/custom/data")
		dir := DataDir()
		assert.Equal(t, "/custom/data/ero", dir)
	})

	t.Run("CacheDir uses env", func(t *testing.T) {
		t.Setenv("XDG_CACHE_HOME", "/custom/cache")
		dir := CacheDir()
		assert.Equal(t, "/custom/cache/ero", dir)
	})

	t.Run("ConfigDir falls back to home", func(t *testing.T) {
		t.Setenv("HOME", "/home/test")
		dir := ConfigDir()
		assert.Contains(t, dir, ".config/ero")
	})
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, string(out))
}
