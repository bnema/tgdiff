package pluginadapter

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"ero/internal/ports"
	"ero/internal/ports/mocks"
)

func TestReviewProviderLoaderStartsOneClientPerReviewProviderContribution(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "ero-plugin.toml"), []byte(`name = "multi"
version = "0.1.0"
manifest_version = "1"
protocol = "ero.plugin.v1"

[runtime]
command = "cat"

[[contributions]]
type = "review_provider"
id = "github"
label = "GitHub"

[[contributions]]
type = "review_provider"
id = "gitlab"
label = "GitLab"
`), 0o644)
	require.NoError(t, err)

	registry := mocks.NewMockPluginRegistry(t)
	registry.EXPECT().InstalledPlugins(context.Background()).Return([]ports.PluginDescriptor{{
		Name: "multi",
		Path: dir,
		Contributions: []ports.PluginContribution{
			{Type: "review_provider", ID: "github", Label: "GitHub"},
			{Type: "review_provider", ID: "gitlab", Label: "GitLab"},
			{Type: "other", ID: "ignored", Label: "Ignored"},
		},
	}}, nil)

	providers, err := NewReviewProviderLoader(registry).LoadReviewProviders(context.Background())
	require.NoError(t, err)
	require.Len(t, providers, 2)
	for _, provider := range providers {
		require.NoError(t, provider.Close())
	}
}
