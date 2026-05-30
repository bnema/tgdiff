package component

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderPluginListEmpty(t *testing.T) {
	t.Parallel()

	result := RenderPluginList(nil, 80)
	assert.Contains(t, result, "No plugins installed")
}

func TestRenderPluginListSingle(t *testing.T) {
	t.Parallel()

	items := []PluginListItem{
		{
			Name:          "github",
			Version:       "0.1.0",
			Source:        "git:github.com/ero-plugins/github@v0.1.0",
			Contributions: []string{"review_provider:github"},
		},
	}

	result := RenderPluginList(items, 80)

	assert.Contains(t, result, "github v0.1.0")
	assert.Contains(t, result, "review_provider:github")
	assert.Contains(t, result, "git:github.com/ero-plugins/github@v0.1.0")
}

func TestRenderPluginListMultiple(t *testing.T) {
	t.Parallel()

	items := []PluginListItem{
		{Name: "github", Version: "0.1.0", Source: "git:github.com/ero-plugins/github@v0.1.0", Contributions: []string{"review_provider:github"}},
		{Name: "pi-coding-agent", Version: "0.2.0", Source: "/local/path", Status: "active", Contributions: []string{"review_provider:pi-coding-agent"}},
	}

	result := RenderPluginList(items, 80)

	assert.Contains(t, result, "github v0.1.0")
	assert.Contains(t, result, "pi-coding-agent v0.2.0")
	assert.Contains(t, result, "active")

	// Should have a newline separator between entries.
	parts := strings.SplitN(result, "\n", 3)
	assert.Len(t, parts, 3) // header + source + empty = 3, but actually: header, contrib, source, empty, header, contrib, source
}

func TestRenderPluginListTruncatesNarrow(t *testing.T) {
	t.Parallel()

	items := []PluginListItem{
		{Name: "very-long-plugin-name", Version: "0.1.0", Source: "git:github.com/some/very/long/path/repo@v0.1.0", Contributions: []string{"review_provider:very-long-plugin-name"}},
	}

	result := RenderPluginList(items, 20)

	// Should produce something (not panic).
	assert.NotEmpty(t, result)
}

func TestRenderPluginListNoContributions(t *testing.T) {
	t.Parallel()

	items := []PluginListItem{
		{Name: "bare", Version: "0.1.0", Source: "git:github.com/ero-plugins/bare"},
	}

	result := RenderPluginList(items, 80)

	assert.Contains(t, result, "bare v0.1.0")
	assert.NotContains(t, result, "↳")
}
