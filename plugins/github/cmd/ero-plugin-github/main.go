package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	gh "github.com/cli/go-gh/v2"

	"ero/pkg/plugin"
)

const providerID = "github"

type githubProvider struct {
	getenv func(string) string
	execGH func(context.Context, ...string) (string, string, error)
}

type ghPR struct {
	Number int    `json:"number"`
	URL    string `json:"url"`
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

func (p githubProvider) PublishReview(ctx context.Context, req plugin.PublishReviewParams) (plugin.PublishReviewResultData, error) {
	pr, err := p.currentPullRequest(ctx)
	if err != nil {
		return plugin.PublishReviewResultData{}, err
	}
	return plugin.PublishReviewResultData{}, plugin.NewErrorf(plugin.ErrorUnsupportedCapability, "GitHub PR #%d detected, but publishing review comments is not implemented yet", pr.Number)
}

func (p githubProvider) currentPullRequest(ctx context.Context) (ghPR, error) {
	if p.execGH == nil {
		p.execGH = execGH
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	stdout, stderr, err := p.execGH(ctx, "pr", "view", "--json", "number,url")
	if err != nil {
		message := strings.TrimSpace(stderr)
		if message == "" {
			message = "no pull request associated with the current branch"
		}
		return ghPR{}, plugin.NewErrorf(plugin.ErrorNotApplicable, "no pull request found for current branch: %s", message)
	}
	var pr ghPR
	if err := json.Unmarshal([]byte(stdout), &pr); err != nil || pr.Number == 0 {
		return ghPR{}, plugin.NewError(plugin.ErrorNotApplicable, "no pull request found for current branch")
	}
	return pr, nil
}

func execGH(ctx context.Context, args ...string) (string, string, error) {
	stdout, stderr, err := gh.ExecContext(ctx, args...)
	return stdout.String(), stderr.String(), err
}

func isGitHubRemote(url string) bool {
	url = strings.ToLower(url)
	return strings.Contains(url, "github.com:") || strings.Contains(url, "github.com/")
}
