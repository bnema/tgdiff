package ports

import (
	"context"

	"ero/internal/core"
)

// PluginRegistry discovers installed plugins and their contributions.
type PluginRegistry interface {
	InstalledPlugins(ctx context.Context) ([]PluginDescriptor, error)
}

// PluginDescriptor holds static metadata about an installed plugin.
type PluginDescriptor struct {
	Name          string
	Version       string
	Source        string
	Path          string
	Contributions []PluginContribution
}

// PluginContribution describes a single capability that a plugin provides.
type PluginContribution struct {
	Type  string
	ID    string
	Label string
}

// ReviewProviderClient is the interface the app/TUI uses to interact with a
// single review provider instance. It is implemented by plugin adapters.
type ReviewProviderClient interface {
	Initialize(ctx context.Context) (core.ReviewProviderInfo, error)
	DetectContext(ctx context.Context, review core.ReviewContext) (core.DetectionResult, error)
	LoadRemoteThreads(ctx context.Context, review core.ReviewContext) ([]core.RemoteReviewThread, error)
	PublishReview(ctx context.Context, request core.PublishReviewRequest) (core.PublishReviewResult, error)
	Close() error
}
