package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"ero/pkg/plugin"
)

const providerID = "github"

type githubProvider struct {
	getenv func(string) string
}

func main() {
	provider := githubProvider{getenv: os.Getenv}
	if err := plugin.ServeReviewProvider(context.Background(), provider, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func (p githubProvider) Initialize(_ context.Context, req plugin.InitializeRequest) (plugin.InitializeResult, error) {
	if req.Protocol != plugin.ProtocolVersion {
		return plugin.InitializeResult{}, plugin.NewErrorf(plugin.ErrorInvalidRequest, "unsupported protocol %q", req.Protocol)
	}
	if req.ContributionID != "" && req.ContributionID != providerID {
		return plugin.InitializeResult{}, plugin.NewErrorf(plugin.ErrorInvalidRequest, "unsupported contribution %q", req.ContributionID)
	}
	return plugin.InitializeResult{
		Protocol: plugin.ProtocolVersion,
		Provider: plugin.ReviewProviderInfo{
			ID:    providerID,
			Label: "GitHub",
			Name:  "ero-plugin-github",
			Capabilities: plugin.ReviewProviderCapabilities{
				LoadRemoteComments: true,
				PublishReview:      true,
				Decisions: []plugin.ReviewDecision{
					plugin.ReviewDecisionComment,
					plugin.ReviewDecisionRequestChanges,
					plugin.ReviewDecisionApprove,
				},
				IdempotentPublish: true,
			},
		},
	}, nil
}

func (p githubProvider) DetectContext(_ context.Context, req plugin.DetectContextRequest) (plugin.DetectContextResult, error) {
	for _, remote := range req.Context.Repository.Remotes {
		if isGitHubRemote(remote.URL) {
			return plugin.DetectContextResult{Result: plugin.DetectionResult{Applicable: true, Reason: "GitHub remote detected"}}, nil
		}
	}
	return plugin.DetectContextResult{Result: plugin.DetectionResult{Applicable: false, Reason: "no GitHub remote detected"}}, nil
}

func (p githubProvider) LoadRemoteThreads(_ context.Context, _ plugin.LoadRemoteThreadsRequest) (plugin.LoadRemoteThreadsResult, error) {
	return plugin.LoadRemoteThreadsResult{}, plugin.NewError(plugin.ErrorAuthRequired, "GitHub remote comment loading requires a GitHub API implementation and credentials")
}

func (p githubProvider) PublishReview(_ context.Context, req plugin.PublishReviewParams) (plugin.PublishReviewResultData, error) {
	if p.getenv == nil {
		p.getenv = os.Getenv
	}
	if p.getenv("GITHUB_TOKEN") == "" && p.getenv("GH_TOKEN") == "" {
		return plugin.PublishReviewResultData{}, plugin.NewError(plugin.ErrorAuthRequired, "set GITHUB_TOKEN or GH_TOKEN to publish to GitHub")
	}
	return plugin.PublishReviewResultData{Result: plugin.ReviewPublishResult{
		ProviderID:       providerID,
		ExternalReviewID: "dry-run-" + req.Payload.Context.Session.LocalReviewID,
		PublishedRefs:    []plugin.PublishedReviewCommentRef{},
	}}, nil
}

func isGitHubRemote(url string) bool {
	url = strings.ToLower(url)
	return strings.Contains(url, "github.com:") || strings.Contains(url, "github.com/")
}
