package plugin

import (
	"fmt"
	"strings"
)

// ---------- manifest v1 ----------

// Manifest is the plugin descriptor read from ero-plugin.toml.
type Manifest struct {
	Name            string         `toml:"name" json:"name"`
	Version         string         `toml:"version" json:"version"`
	Description     string         `toml:"description" json:"description,omitempty"`
	Protocol        string         `toml:"protocol" json:"protocol"`
	ManifestVersion string         `toml:"manifest_version" json:"manifest_version"`
	Contributions   []Contribution `toml:"contributions" json:"contributions"`
	Runtime         RuntimeConfig  `toml:"runtime" json:"runtime"`
	Build           BuildConfig    `toml:"build" json:"build"`
}

// Contribution describes a single capability that a plugin provides.
type Contribution struct {
	Type  string `toml:"type" json:"type"`
	ID    string `toml:"id" json:"id"`
	Label string `toml:"label" json:"label"`
}

// RuntimeConfig specifies how to launch the plugin subprocess.
type RuntimeConfig struct {
	Command string `toml:"command" json:"command"`
}

// BuildConfig is an optional build step for local plugin development.
type BuildConfig struct {
	Command string `toml:"command" json:"command,omitempty"`
}

// Validate checks that the manifest contains the required fields and that
// version constraints are satisfied. It returns the first validation error
// or nil.
func (m Manifest) Validate() error {
	if m.ManifestVersion != "1" {
		return fmt.Errorf("unsupported manifest_version %q (expected \"1\")", m.ManifestVersion)
	}
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("manifest name is required")
	}
	if strings.TrimSpace(m.Version) == "" {
		return fmt.Errorf("manifest version is required")
	}
	if m.Protocol != ProtocolVersion {
		return fmt.Errorf("unsupported protocol %q (expected %q)", m.Protocol, ProtocolVersion)
	}
	if strings.TrimSpace(m.Runtime.Command) == "" {
		return fmt.Errorf("runtime command is required")
	}
	if len(m.Contributions) == 0 {
		return fmt.Errorf("at least one contribution is required")
	}
	for i, c := range m.Contributions {
		if strings.TrimSpace(c.Type) == "" {
			return fmt.Errorf("contributions[%d]: type is required", i)
		}
		if strings.TrimSpace(c.ID) == "" {
			return fmt.Errorf("contributions[%d]: id is required", i)
		}
	}
	return nil
}
