package plugin_test

import (
	"testing"

	"ero/pkg/plugin"
)

func TestManifestValid(t *testing.T) {
	t.Parallel()

	m := plugin.Manifest{
		Name:            "github",
		Version:         "0.1.0",
		Protocol:        plugin.ProtocolVersion,
		ManifestVersion: "1",
		Runtime:         plugin.RuntimeConfig{Command: "./bin/ero-plugin-github"},
		Contributions: []plugin.Contribution{
			{Type: plugin.ContributionReviewProvider, ID: "github", Label: "GitHub"},
		},
	}

	if err := m.Validate(); err != nil {
		t.Fatalf("expected valid manifest, got: %v", err)
	}
}

func TestManifestUnsupportedManifestVersion(t *testing.T) {
	t.Parallel()

	m := plugin.Manifest{
		Name:            "github",
		Version:         "0.1.0",
		Protocol:        plugin.ProtocolVersion,
		ManifestVersion: "2",
		Runtime:         plugin.RuntimeConfig{Command: "./bin/plugin"},
		Contributions: []plugin.Contribution{
			{Type: plugin.ContributionReviewProvider, ID: "github", Label: "GitHub"},
		},
	}

	err := m.Validate()
	if err == nil {
		t.Fatal("expected error for unsupported manifest_version")
	}
}

func TestManifestUnsupportedProtocol(t *testing.T) {
	t.Parallel()

	m := plugin.Manifest{
		Name:            "github",
		Version:         "0.1.0",
		Protocol:        "ero.plugin.v99",
		ManifestVersion: "1",
		Runtime:         plugin.RuntimeConfig{Command: "./bin/plugin"},
		Contributions: []plugin.Contribution{
			{Type: plugin.ContributionReviewProvider, ID: "github", Label: "GitHub"},
		},
	}

	err := m.Validate()
	if err == nil {
		t.Fatal("expected error for unsupported protocol")
	}
}

func TestManifestMissingName(t *testing.T) {
	t.Parallel()

	m := plugin.Manifest{
		Version:         "0.1.0",
		Protocol:        plugin.ProtocolVersion,
		ManifestVersion: "1",
		Runtime:         plugin.RuntimeConfig{Command: "./bin/plugin"},
	}

	err := m.Validate()
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestManifestMissingVersion(t *testing.T) {
	t.Parallel()

	m := plugin.Manifest{
		Name:            "github",
		Protocol:        plugin.ProtocolVersion,
		ManifestVersion: "1",
		Runtime:         plugin.RuntimeConfig{Command: "./bin/plugin"},
		Contributions: []plugin.Contribution{
			{Type: plugin.ContributionReviewProvider, ID: "github", Label: "GitHub"},
		},
	}

	err := m.Validate()
	if err == nil {
		t.Fatal("expected error for missing version")
	}
}

func TestManifestMissingRuntimeCommand(t *testing.T) {
	t.Parallel()

	m := plugin.Manifest{
		Name:            "github",
		Version:         "0.1.0",
		Protocol:        plugin.ProtocolVersion,
		ManifestVersion: "1",
		Runtime:         plugin.RuntimeConfig{Command: ""},
	}

	err := m.Validate()
	if err == nil {
		t.Fatal("expected error for missing runtime command")
	}
}

func TestManifestRequiresContribution(t *testing.T) {
	t.Parallel()

	m := plugin.Manifest{
		Name:            "github",
		Version:         "0.1.0",
		Protocol:        plugin.ProtocolVersion,
		ManifestVersion: "1",
		Runtime:         plugin.RuntimeConfig{Command: "./bin/plugin"},
	}

	err := m.Validate()
	if err == nil {
		t.Fatal("expected error for missing contributions")
	}
}

func TestManifestMissingContributionType(t *testing.T) {
	t.Parallel()

	m := plugin.Manifest{
		Name:            "github",
		Version:         "0.1.0",
		Protocol:        plugin.ProtocolVersion,
		ManifestVersion: "1",
		Runtime:         plugin.RuntimeConfig{Command: "./bin/plugin"},
		Contributions: []plugin.Contribution{
			{Type: "", ID: "github", Label: "GitHub"},
		},
	}

	err := m.Validate()
	if err == nil {
		t.Fatal("expected error for missing contribution type")
	}
}

func TestManifestMissingContributionID(t *testing.T) {
	t.Parallel()

	m := plugin.Manifest{
		Name:            "github",
		Version:         "0.1.0",
		Protocol:        plugin.ProtocolVersion,
		ManifestVersion: "1",
		Runtime:         plugin.RuntimeConfig{Command: "./bin/plugin"},
		Contributions: []plugin.Contribution{
			{Type: plugin.ContributionReviewProvider, ID: "", Label: "GitHub"},
		},
	}

	err := m.Validate()
	if err == nil {
		t.Fatal("expected error for missing contribution id")
	}
}

func TestManifestMultipleContributions(t *testing.T) {
	t.Parallel()

	m := plugin.Manifest{
		Name:            "multi",
		Version:         "0.1.0",
		Protocol:        plugin.ProtocolVersion,
		ManifestVersion: "1",
		Runtime:         plugin.RuntimeConfig{Command: "./bin/plugin"},
		Contributions: []plugin.Contribution{
			{Type: plugin.ContributionReviewProvider, ID: "github", Label: "GitHub"},
			{Type: plugin.ContributionReviewProvider, ID: "gitlab", Label: "GitLab"},
		},
	}

	if err := m.Validate(); err != nil {
		t.Fatalf("expected valid manifest with multiple contributions, got: %v", err)
	}
}
