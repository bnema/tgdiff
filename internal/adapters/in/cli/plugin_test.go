package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pluginadapter "ero/internal/adapters/out/plugin"
)

// fakePluginManager implements PluginManager for CLI tests.
type fakePluginManager struct {
	plugins       []pluginadapter.InstalledPlugin
	installResult pluginadapter.InstallResult
	installErr    error
	removeResult  pluginadapter.RemoveResult
	removeErr     error
	updateResults []pluginadapter.UpdateResult
	updateErr     error
}

func (f *fakePluginManager) Install(_ context.Context, _ string) (pluginadapter.InstallResult, error) {
	return f.installResult, f.installErr
}

func (f *fakePluginManager) List(_ context.Context) ([]pluginadapter.InstalledPlugin, error) {
	return f.plugins, nil
}

func (f *fakePluginManager) Update(_ context.Context, source string) ([]pluginadapter.UpdateResult, error) {
	return f.updateResults, f.updateErr
}

func (f *fakePluginManager) Remove(_ context.Context, nameOrSource string) (pluginadapter.RemoveResult, error) {
	return f.removeResult, f.removeErr
}

func TestPluginCommandRegisteredUnderParent(t *testing.T) {
	t.Parallel()

	fake := &fakePluginManager{}
	cmd := NewPluginCommand(fake, nil)
	require.NotNil(t, cmd)

	assert.Equal(t, "plugin", cmd.Use)

	// List subcommand.
	listCmd, _, err := cmd.Find([]string{"list"})
	require.NoError(t, err)
	assert.Equal(t, "list", listCmd.Use)
	assert.Equal(t, "List installed plugins", listCmd.Short)
}

func TestPluginListEmpty(t *testing.T) {
	t.Parallel()

	fake := &fakePluginManager{}
	cmd := NewPluginCommand(fake, nil)

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"list"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, out.String(), "No plugins installed")
}

func TestPluginListHumanOutput(t *testing.T) {
	t.Parallel()

	fake := &fakePluginManager{
		plugins: []pluginadapter.InstalledPlugin{
			{
				Name:          "github",
				Version:       "0.1.0",
				Source:        "git:github.com/ero-plugins/github@v0.1.0",
				Path:          "/data/plugins/github",
				Contributions: []string{"review_provider:github"},
			},
			{
				Name:          "pimono",
				Version:       "0.2.0",
				Source:        "/home/user/dev/ero-plugin-pimono",
				Path:          "/home/user/dev/ero-plugin-pimono",
				Contributions: []string{"review_provider:pimono"},
			},
		},
	}

	cmd := NewPluginCommand(fake, nil)

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"list"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "github v0.1.0")
	assert.Contains(t, output, "review_provider:github")
	assert.Contains(t, output, "git:github.com/ero-plugins/github@v0.1.0")
	assert.Contains(t, output, "pimono v0.2.0")
}

func TestPluginListJSONOutput(t *testing.T) {
	t.Parallel()

	fake := &fakePluginManager{
		plugins: []pluginadapter.InstalledPlugin{
			{Name: "github", Version: "0.1.0", Source: "git:github.com/ero-plugins/github@v0.1.0", Contributions: []string{"review_provider:github"}},
		},
	}

	cmd := NewPluginCommand(fake, nil)

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"list", "--json"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result []pluginadapter.InstalledPlugin
	err = json.Unmarshal(out.Bytes(), &result)
	require.NoError(t, err)

	require.Len(t, result, 1)
	assert.Equal(t, "github", result[0].Name)
	assert.Equal(t, "0.1.0", result[0].Version)
}

func TestPluginInstallHumanOutput(t *testing.T) {
	t.Parallel()

	fake := &fakePluginManager{
		installResult: pluginadapter.InstallResult{
			Name:    "github",
			Version: "0.1.0",
			Source:  "git:github.com/ero-plugins/github@v0.1.0",
			Path:    "/data/plugins/github",
		},
	}

	cmd := NewPluginCommand(fake, nil)

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"install", "git:github.com/ero-plugins/github@v0.1.0"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "Installed plugin github v0.1.0")
	assert.Contains(t, output, "git:github.com/ero-plugins/github@v0.1.0")
}

func TestPluginInstallJSONOutput(t *testing.T) {
	t.Parallel()

	fake := &fakePluginManager{
		installResult: pluginadapter.InstallResult{
			Name: "github", Version: "0.1.0", Source: "git:github.com/ero-plugins/github@v0.1.0", Path: "/data/plugins/github",
		},
	}

	cmd := NewPluginCommand(fake, nil)

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"install", "--json", "git:github.com/ero-plugins/github@v0.1.0"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result pluginadapter.InstallResult
	err = json.Unmarshal(out.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "github", result.Name)
	assert.Equal(t, "0.1.0", result.Version)
}

func TestPluginInstallMissingArgs(t *testing.T) {
	t.Parallel()

	fake := &fakePluginManager{}
	cmd := NewPluginCommand(fake, nil)

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"install"})

	err := cmd.Execute()
	require.Error(t, err)
}

func TestPluginUpdateHumanOutput(t *testing.T) {
	t.Parallel()

	fake := &fakePluginManager{
		updateResults: []pluginadapter.UpdateResult{
			{Name: "github", PreviousRef: "abc1234def", UpdatedRef: "xyz5678abc"},
			{Name: "pimono", Message: "pinned to v0.1.0, skipping update"},
		},
	}

	cmd := NewPluginCommand(fake, nil)

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"update"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "github: abc1234 → xyz5678")
	assert.Contains(t, output, "pimono: pinned")
}

func TestPluginUpdateJSONOutput(t *testing.T) {
	t.Parallel()

	fake := &fakePluginManager{
		updateResults: []pluginadapter.UpdateResult{
			{Name: "github", PreviousRef: "abc1234def", UpdatedRef: "xyz5678abc"},
		},
	}

	cmd := NewPluginCommand(fake, nil)

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"update", "--json"})

	err := cmd.Execute()
	require.NoError(t, err)

	var results []pluginadapter.UpdateResult
	err = json.Unmarshal(out.Bytes(), &results)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "github", results[0].Name)
}

func TestPluginRemoveHumanOutput(t *testing.T) {
	t.Parallel()

	fake := &fakePluginManager{
		removeResult: pluginadapter.RemoveResult{
			Name:        "github",
			Source:      "git:github.com/ero-plugins/github@v0.1.0",
			RemovedRepo: true,
		},
	}

	cmd := NewPluginCommand(fake, nil)

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"remove", "github"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, out.String(), "Removed plugin github")
}

func TestPluginRemoveJSONOutput(t *testing.T) {
	t.Parallel()

	fake := &fakePluginManager{
		removeResult: pluginadapter.RemoveResult{
			Name: "github", Source: "git:github.com/ero-plugins/github@v0.1.0", RemovedRepo: false,
		},
	}

	cmd := NewPluginCommand(fake, nil)

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"remove", "--json", "github"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result pluginadapter.RemoveResult
	err = json.Unmarshal(out.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "github", result.Name)
	assert.False(t, result.RemovedRepo)
}

func TestPluginUpdateFiltered(t *testing.T) {
	t.Parallel()

	fake := &fakePluginManager{
		updateResults: []pluginadapter.UpdateResult{
			{Name: "github", PreviousRef: "abc", UpdatedRef: "def"},
		},
	}

	cmd := NewPluginCommand(fake, nil)

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"update", "git:github.com/ero-plugins/github"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, out.String(), "github")
}

func TestPluginCommandWiresContext(t *testing.T) {
	t.Parallel()

	fake := &fakePluginManager{
		plugins: []pluginadapter.InstalledPlugin{
			{Name: "test", Version: "0.1.0"},
		},
	}

	cmd := NewPluginCommand(fake, nil)
	require.NotNil(t, cmd)

	// Verify subcommand count.
	assert.Len(t, cmd.Commands(), 4)
	names := make([]string, len(cmd.Commands()))
	for i, c := range cmd.Commands() {
		names[i] = c.Use
	}

	// All expected subcommands should be present.
	for _, name := range []string{"list", "install", "update", "remove"} {
		found := false
		for _, n := range names {
			if strings.HasPrefix(n, name) {
				found = true
				break
			}
		}
		assert.True(t, found, "expected subcommand %q", name)
	}
}
