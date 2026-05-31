package pluginadapter

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/bnema/zerowrap"
	"github.com/google/shlex"

	"ero/internal/ports"
	pluginsdk "ero/pkg/plugin"
)

// ReviewProviderLoader builds review provider clients from installed plugin manifests.
type ReviewProviderLoader struct {
	registry ports.PluginRegistry
	timeout  time.Duration
}

// NewReviewProviderLoader creates a loader backed by an installed plugin registry.
func NewReviewProviderLoader(registry ports.PluginRegistry) *ReviewProviderLoader {
	return &ReviewProviderLoader{registry: registry, timeout: DefaultPluginTimeout}
}

// LoadReviewProviders implements ports.ReviewProviderLoader.
func (l *ReviewProviderLoader) LoadReviewProviders(ctx context.Context) ([]ports.ReviewProviderClient, error) {
	descriptors, err := l.registry.InstalledPlugins(ctx)
	if err != nil {
		return nil, err
	}
	log := zerowrap.FromCtx(ctx)
	providers := make([]ports.ReviewProviderClient, 0)
	for _, descriptor := range descriptors {
		manifest, err := LoadManifest(descriptor.Path)
		if err != nil {
			log.Warn().Err(err).Str("plugin_path", descriptor.Path).Msg("load plugin manifest failed")
			continue
		}
		command, args := splitRuntimeCommand(manifest.Runtime.Command)
		if command == "" {
			log.Warn().Str("plugin_path", descriptor.Path).Msg("plugin runtime command is empty")
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
			client, err := NewClientForContribution(command, args, descriptor.Path, contribution.ID, l.timeout)
			if err != nil {
				log.Warn().Err(err).Str("plugin_path", descriptor.Path).Str("contribution_id", contribution.ID).Msg("create plugin review provider client failed")
				continue
			}
			providers = append(providers, client)
		}
	}
	return providers, nil
}

func splitRuntimeCommand(command string) (string, []string) {
	fields, err := shlex.Split(command)
	if err != nil {
		fields = strings.Fields(command)
	}
	if len(fields) == 0 {
		return "", nil
	}
	return fields[0], fields[1:]
}
