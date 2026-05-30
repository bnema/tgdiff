package app

import (
	"context"
	"os/exec"
	"strings"

	pluginadapter "ero/internal/adapters/out/plugin"
	"ero/internal/ports"
	pluginsdk "ero/pkg/plugin"
)

func buildReviewProviders(registry ports.PluginRegistry) ([]ports.ReviewProviderClient, error) {
	descriptors, err := registry.InstalledPlugins(context.Background())
	if err != nil {
		return nil, err
	}
	providers := make([]ports.ReviewProviderClient, 0)
	for _, descriptor := range descriptors {
		manifest, err := pluginadapter.LoadManifest(descriptor.Path)
		if err != nil {
			continue
		}
		command, args := splitRuntimeCommand(manifest.Runtime.Command)
		if command == "" {
			continue
		}
		if !strings.Contains(command, "/") {
			if resolved, err := exec.LookPath(command); err == nil {
				command = resolved
			}
		}
		for _, contribution := range descriptor.Contributions {
			if contribution.Type != pluginsdk.ContributionReviewProvider {
				continue
			}
			client, err := pluginadapter.NewClientForContribution(command, args, descriptor.Path, contribution.ID, pluginadapter.DefaultPluginTimeout)
			if err != nil {
				continue
			}
			providers = append(providers, client)
		}
	}
	return providers, nil
}

func splitRuntimeCommand(command string) (string, []string) {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return "", nil
	}
	return fields[0], fields[1:]
}
