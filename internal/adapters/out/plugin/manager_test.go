package pluginadapter

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func skipIfNoGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}

// setupTestManager creates a Manager with temp config and data directories.
func setupTestManager(t *testing.T) *Manager {
	t.Helper()
	configDir := t.TempDir()
	dataDir := t.TempDir()
	return newManager(configDir, dataDir, "git")
}

// setupTestManagerGit creates a Manager, skipping if git is unavailable.
func setupTestManagerGit(t *testing.T) *Manager {
	t.Helper()
	skipIfNoGit(t)
	return setupTestManager(t)
}

// createLocalPlugin creates a temp dir, inits git, and writes a valid manifest.
func createLocalPlugin(t *testing.T, name, version string) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@test.test")
	runGit(t, dir, "config", "user.name", "Test")
	writeTestManifest(t, dir, name, version)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "init")
	return dir
}

func writeTestManifest(t *testing.T, dir, name, version string) {
	t.Helper()
	content := "name = \"" + name + "\"\n" +
		"version = \"" + version + "\"\n" +
		"manifest_version = \"1\"\n" +
		"protocol = \"ero.plugin.v1\"\n" +
		"[runtime]\n" +
		"command = \"./bin/" + name + "\"\n" +
		"[[contributions]]\n" +
		"type = \"review_provider\"\n" +
		"id = \"" + name + "\"\n" +
		"label = \"Test Plugin\"\n"
	err := os.WriteFile(filepath.Join(dir, "ero-plugin.toml"), []byte(content), 0o644)
	require.NoError(t, err)
}

func TestManagerInstallLocal(t *testing.T) {
	skipIfNoGit(t)

	mgr := setupTestManagerGit(t)
	dir := createLocalPlugin(t, "local-plugin", "1.0.0")

	ctx := context.Background()
	result, err := mgr.Install(ctx, dir)
	require.NoError(t, err)

	assert.Equal(t, "local-plugin", result.Name)
	assert.Equal(t, "1.0.0", result.Version)
	assert.Equal(t, dir, result.Path)

	// Verify in list.
	plugins, err := mgr.List(ctx)
	require.NoError(t, err)
	require.Len(t, plugins, 1)
	assert.Equal(t, "local-plugin", plugins[0].Name)
}

func TestManagerInstallGitFileURL(t *testing.T) {
	skipIfNoGit(t)

	mgr := setupTestManagerGit(t)
	remote := createLocalPlugin(t, "git-plugin", "1.0.0")

	ctx := context.Background()
	result, err := mgr.Install(ctx, "file://"+remote)
	require.NoError(t, err)

	assert.Equal(t, "git-plugin", result.Name)
	assert.Equal(t, "1.0.0", result.Version)
	assert.NotEqual(t, remote, result.Path)
	assert.DirExists(t, filepath.Join(mgr.dataDir, "plugins"))

	plugins, err := mgr.List(ctx)
	require.NoError(t, err)
	require.Len(t, plugins, 1)
	assert.Equal(t, "git-plugin", plugins[0].Name)
}

func TestManagerInstallGitPinnedRef(t *testing.T) {
	skipIfNoGit(t)

	mgr := setupTestManagerGit(t)
	remote := createLocalPlugin(t, "pinned-plugin", "1.0.0")
	runGit(t, remote, "tag", "v1.0.0")
	writeTestManifest(t, remote, "pinned-plugin", "2.0.0")
	runGit(t, remote, "add", ".")
	runGit(t, remote, "commit", "-m", "v2")

	ctx := context.Background()
	result, err := mgr.Install(ctx, "file://"+remote+"@v1.0.0")
	require.NoError(t, err)

	assert.Equal(t, "pinned-plugin", result.Name)
	assert.Equal(t, "1.0.0", result.Version)

	updates, err := mgr.Update(ctx, "")
	require.NoError(t, err)
	require.Len(t, updates, 1)
	assert.Contains(t, updates[0].Message, "pinned")
}

func TestManagerUpdateGitFileURL(t *testing.T) {
	skipIfNoGit(t)

	mgr := setupTestManagerGit(t)
	remote := createLocalPlugin(t, "update-git", "1.0.0")

	ctx := context.Background()
	_, err := mgr.Install(ctx, "file://"+remote)
	require.NoError(t, err)
	plugins, err := mgr.List(ctx)
	require.NoError(t, err)
	require.Len(t, plugins, 1)
	assert.Equal(t, "1.0.0", plugins[0].Version)

	writeTestManifest(t, remote, "update-git", "2.0.0")
	runGit(t, remote, "add", ".")
	runGit(t, remote, "commit", "-m", "v2")

	updates, err := mgr.Update(ctx, "")
	require.NoError(t, err)
	require.Len(t, updates, 1)
	assert.NotEmpty(t, updates[0].PreviousRef)
	assert.NotEmpty(t, updates[0].UpdatedRef)
	assert.NotEqual(t, updates[0].PreviousRef, updates[0].UpdatedRef)

	plugins, err = mgr.List(ctx)
	require.NoError(t, err)
	require.Len(t, plugins, 1)
	assert.Equal(t, "2.0.0", plugins[0].Version)
}

func TestManagerRemoveGitDeletesManagedClone(t *testing.T) {
	skipIfNoGit(t)

	mgr := setupTestManagerGit(t)
	remote := createLocalPlugin(t, "remove-git", "1.0.0")

	ctx := context.Background()
	installed, err := mgr.Install(ctx, "file://"+remote)
	require.NoError(t, err)
	assert.DirExists(t, installed.Path)

	removed, err := mgr.Remove(ctx, "remove-git")
	require.NoError(t, err)
	assert.True(t, removed.RemovedRepo)
	assert.NoDirExists(t, installed.Path)
	assert.DirExists(t, remote)
}

func TestManagerInstallDuplicateRejected(t *testing.T) {
	skipIfNoGit(t)

	mgr := setupTestManagerGit(t)
	dir := createLocalPlugin(t, "dup-plugin", "0.1.0")

	ctx := context.Background()
	_, err := mgr.Install(ctx, dir)
	require.NoError(t, err)

	_, err = mgr.Install(ctx, dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already in config")
}

func TestManagerRemove(t *testing.T) {
	skipIfNoGit(t)

	mgr := setupTestManagerGit(t)
	dir := createLocalPlugin(t, "rm-plugin", "1.0.0")

	ctx := context.Background()
	result, err := mgr.Install(ctx, dir)
	require.NoError(t, err)

	removeResult, err := mgr.Remove(ctx, result.Name)
	require.NoError(t, err)
	assert.Equal(t, result.Name, removeResult.Name)
	assert.False(t, removeResult.RemovedRepo) // local sources don't delete repos

	// Verify removed from config.
	plugins, err := mgr.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, plugins)

	// Verify local repo still exists.
	_, err = os.Stat(dir)
	assert.NoError(t, err, "local source repo should not be deleted")
}

func TestManagerRemoveBySource(t *testing.T) {
	skipIfNoGit(t)

	mgr := setupTestManagerGit(t)
	dir := createLocalPlugin(t, "rmby-source", "1.0.0")

	ctx := context.Background()
	_, err := mgr.Install(ctx, dir)
	require.NoError(t, err)

	// Remove by source string.
	_, err = mgr.Remove(ctx, dir)
	require.NoError(t, err)

	plugins, err := mgr.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, plugins)
}

func TestManagerRemoveNotFound(t *testing.T) {

	mgr := setupTestManager(t)

	ctx := context.Background()
	_, err := mgr.Remove(ctx, "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestManagerListEmptyConfig(t *testing.T) {
	t.Parallel()

	mgr := setupTestManager(t)

	ctx := context.Background()
	plugins, err := mgr.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, plugins)
}

func TestManagerListMultiple(t *testing.T) {
	skipIfNoGit(t)

	mgr := setupTestManagerGit(t)
	dir1 := createLocalPlugin(t, "plugin-one", "0.1.0")
	dir2 := createLocalPlugin(t, "plugin-two", "0.2.0")

	ctx := context.Background()
	_, err := mgr.Install(ctx, dir1)
	require.NoError(t, err)
	_, err = mgr.Install(ctx, dir2)
	require.NoError(t, err)

	plugins, err := mgr.List(ctx)
	require.NoError(t, err)
	assert.Len(t, plugins, 2)

	names := make(map[string]bool)
	for _, p := range plugins {
		names[p.Name] = true
	}
	assert.True(t, names["plugin-one"])
	assert.True(t, names["plugin-two"])
}

func TestManagerUpdateLocalSkipped(t *testing.T) {
	skipIfNoGit(t)

	mgr := setupTestManagerGit(t)
	dir := createLocalPlugin(t, "update-local", "0.1.0")

	ctx := context.Background()
	_, err := mgr.Install(ctx, dir)
	require.NoError(t, err)

	updates, err := mgr.Update(ctx, "")
	require.NoError(t, err)

	require.Len(t, updates, 1)
	assert.Contains(t, updates[0].Message, "local sources are not updated automatically")
}

func TestManagerUpdatePinnedConfig(t *testing.T) {

	mgr := setupTestManager(t)

	// Directly write a pinned config entry.
	cfg, err := mgr.loadConfig()
	require.NoError(t, err)
	cfg.Plugins = append(cfg.Plugins, pluginEntry{Source: "git:github.com/ero-plugins/test@v0.1.0"})
	err = mgr.saveConfig(cfg)
	require.NoError(t, err)

	ctx := context.Background()
	updates, err := mgr.Update(ctx, "")
	require.NoError(t, err)

	require.Len(t, updates, 1)
	assert.Contains(t, updates[0].Message, "pinned")
}

func TestManagerUpdateFiltered(t *testing.T) {
	skipIfNoGit(t)

	mgr := setupTestManagerGit(t)
	dir := createLocalPlugin(t, "filter-one", "0.1.0")
	_ = createLocalPlugin(t, "filter-two", "0.2.0")

	ctx := context.Background()
	_, err := mgr.Install(ctx, dir)
	require.NoError(t, err)
	// Install second plugin.
	dir2 := createLocalPlugin(t, "filter-two", "0.2.0")
	_, err = mgr.Install(ctx, dir2)
	require.NoError(t, err)

	// Update only filter-one.
	updates, err := mgr.Update(ctx, dir)
	require.NoError(t, err)
	assert.Len(t, updates, 1)
	assert.Equal(t, "filter-one", updates[0].Name)
}

func TestManifestLoaderValid(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestManifest(t, dir, "valid-plugin", "2.0.0")

	manifest, err := LoadManifest(dir)
	require.NoError(t, err)

	assert.Equal(t, "valid-plugin", manifest.Name)
	assert.Equal(t, "2.0.0", manifest.Version)
	assert.Equal(t, "ero.plugin.v1", manifest.Protocol)
	require.Len(t, manifest.Contributions, 1)
	assert.Equal(t, "review_provider", manifest.Contributions[0].Type)
	assert.Equal(t, "valid-plugin", manifest.Contributions[0].ID)
}

func TestManifestLoaderMissingFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	_, err := LoadManifest(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ero-plugin.toml")
}

func TestManifestLoaderInvalidTOML(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "ero-plugin.toml"), []byte("not valid toml {{{"), 0o644)
	require.NoError(t, err)

	_, err = LoadManifest(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ero-plugin.toml")
}

func TestManifestLoaderValidationError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	content := "name = \"bad\"\nversion = \"0.1.0\"\nmanifest_version = \"1\"\nprotocol = \"ero.plugin.v1\"\n[runtime]\ncommand = \"\"\n"
	err := os.WriteFile(filepath.Join(dir, "ero-plugin.toml"), []byte(content), 0o644)
	require.NoError(t, err)

	_, err = LoadManifest(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "command is required")
}
