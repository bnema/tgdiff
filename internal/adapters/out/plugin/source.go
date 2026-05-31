package pluginadapter

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// SourceType classifies how a plugin source was resolved.
type SourceType string

const (
	SourceTypeGit   SourceType = "git"
	SourceTypeLocal SourceType = "local"
)

// Source holds the parsed representation of a plugin origin.
type Source struct {
	Type      SourceType
	Raw       string
	Repo      string // host/path without scheme or ref
	Host      string // "github.com", "gitlab.com", etc.
	Path      string // owner/repo path component
	Ref       string // branch, tag, or commit
	Pinned    bool   // true when a specific ref is specified
	LocalPath string // set for SourceTypeLocal
}

// ParseSource detects whether raw is a git URL (HTTPS, SSH, or git:
// shorthand), or a local filesystem path, and returns a parsed Source.
func ParseSource(raw string) (Source, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Source{}, fmt.Errorf("empty plugin source")
	}

	// git: shorthand — "git:github.com/owner/repo@ref"
	if strings.HasPrefix(raw, "git:") {
		return parseGitShorthand(raw)
	}

	// Explicit HTTPS or SSH URLs.
	if strings.Contains(raw, "://") || strings.Contains(raw, "@") {
		return parseGitURL(raw)
	}

	// Local filesystem path.
	return ParseLocalSource(raw)
}

// ParseLocalSource validates that path points to a local plugin directory. The
// directory may be a Git repository root, a Git worktree, or a plugin
// subdirectory inside a larger repository, as long as it contains
// ero-plugin.toml.
func ParseLocalSource(path string) (Source, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return Source{}, fmt.Errorf("plugin source %q: %w", path, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return Source{}, fmt.Errorf("plugin source %q: %w", path, err)
	}
	if !info.IsDir() {
		return Source{}, fmt.Errorf("plugin source %q is not a directory", path)
	}
	if err := validateLocalPluginPath(abs); err != nil {
		return Source{}, fmt.Errorf("plugin source %q: %w", path, err)
	}

	base := filepath.Base(abs)
	return Source{
		Type:      SourceTypeLocal,
		Raw:       path,
		Repo:      abs,
		Host:      "local",
		Path:      base,
		LocalPath: abs,
	}, nil
}

func validateLocalPluginPath(abs string) error {
	if _, err := os.Stat(filepath.Join(abs, "ero-plugin.toml")); err == nil {
		return nil
	}

	gitPath := filepath.Join(abs, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return fmt.Errorf("missing ero-plugin.toml")
	}
	if info.IsDir() {
		return nil
	}
	data, err := os.ReadFile(gitPath)
	if err != nil || !strings.HasPrefix(strings.TrimSpace(string(data)), "gitdir:") {
		return fmt.Errorf(".git is not a directory or valid gitfile")
	}
	return nil
}

// parseGitShorthand handles "git:github.com/owner/repo@ref".
func parseGitShorthand(raw string) (Source, error) {
	remainder := strings.TrimPrefix(raw, "git:")
	if remainder == "" {
		return Source{}, fmt.Errorf("empty git: shorthand")
	}

	refPart := ""
	repoPart := remainder
	if idx := strings.LastIndex(remainder, "@"); idx != -1 {
		repoPart = remainder[:idx]
		refPart = remainder[idx+1:]
	}

	host, repoPath := splitHostPath(repoPart)
	pinned := refPart != ""

	return Source{
		Type:   SourceTypeGit,
		Raw:    raw,
		Repo:   repoPart,
		Host:   host,
		Path:   repoPath,
		Ref:    refPart,
		Pinned: pinned,
	}, nil
}

// parseGitURL handles HTTPS and SSH git URLs, e.g.,
// "https://github.com/owner/repo.git" or "git@github.com:owner/repo.git".
func parseGitURL(raw string) (Source, error) {
	var repo string
	var host string
	var repoPath string
	var ref string

	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err != nil {
			return Source{}, fmt.Errorf("invalid git URL %q: %w", raw, err)
		}
		host = u.Host
		repoPath = strings.TrimPrefix(u.Path, "/")
		if idx := strings.LastIndex(repoPath, "@"); idx != -1 {
			ref = repoPath[idx+1:]
			repoPath = repoPath[:idx]
		}
		repoPath = strings.TrimSuffix(repoPath, ".git")
		if host == "" {
			repo = repoPath
		} else {
			repo = host + "/" + repoPath
		}
	} else {
		// SSH: git@host:owner/repo.git or git@host:owner/repo.git@ref.
		parts := strings.SplitN(raw, ":", 2)
		if len(parts) != 2 {
			return Source{}, fmt.Errorf("invalid git URL %q: expected host:path format", raw)
		}
		host = strings.TrimPrefix(parts[0], "git@")
		repoPath = parts[1]
		if idx := strings.LastIndex(repoPath, "@"); idx != -1 {
			ref = repoPath[idx+1:]
			repoPath = repoPath[:idx]
		}
		repoPath = strings.TrimSuffix(repoPath, ".git")
		repo = host + "/" + repoPath
	}

	return Source{
		Type:   SourceTypeGit,
		Raw:    raw,
		Repo:   repo,
		Host:   host,
		Path:   repoPath,
		Ref:    ref,
		Pinned: ref != "",
	}, nil
}

// splitHostPath splits "github.com/owner/repo" into ("github.com", "owner/repo").
func splitHostPath(raw string) (host, path string) {
	parts := strings.SplitN(raw, "/", 2)
	if len(parts) < 2 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

// ---------- XDG helpers ----------

// ConfigDir returns the ero config directory, respecting XDG_CONFIG_HOME.
func ConfigDir() string {
	return xdgDir("XDG_CONFIG_HOME", ".config", "ero")
}

// DataDir returns the ero data directory, respecting XDG_DATA_HOME.
func DataDir() string {
	return xdgDir("XDG_DATA_HOME", ".local/share", "ero")
}

// CacheDir returns the ero cache directory, respecting XDG_CACHE_HOME.
func CacheDir() string {
	return xdgDir("XDG_CACHE_HOME", ".cache", "ero")
}

func xdgDir(envVar, defaultBase, appName string) string {
	if dir := os.Getenv(envVar); dir != "" {
		return filepath.Join(dir, appName)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", appName)
	}
	return filepath.Join(home, defaultBase, appName)
}
