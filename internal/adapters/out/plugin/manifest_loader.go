package pluginadapter

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"

	pluginsdk "ero/pkg/plugin"
)

const manifestFileName = "ero-plugin.toml"

// LoadManifest reads and validates the ero-plugin.toml from pluginRoot.
// Errors are prefixed with the manifest path for clarity.
func LoadManifest(pluginRoot string) (pluginsdk.Manifest, error) {
	manifestPath := filepath.Join(pluginRoot, manifestFileName)
	manifestPath = filepath.Clean(manifestPath)

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return pluginsdk.Manifest{}, fmt.Errorf("%s: %w", manifestPath, err)
	}

	var manifest pluginsdk.Manifest
	if err := toml.Unmarshal(data, &manifest); err != nil {
		return pluginsdk.Manifest{}, fmt.Errorf("%s: %w", manifestPath, err)
	}

	if err := manifest.Validate(); err != nil {
		return pluginsdk.Manifest{}, fmt.Errorf("%s: %w", manifestPath, err)
	}

	return manifest, nil
}
