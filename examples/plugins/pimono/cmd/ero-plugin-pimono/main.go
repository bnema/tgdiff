package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"ero/pkg/plugin"
)

const providerID = "pimono"

type pimonoProvider struct {
	getenv func(string) string
	stderr io.Writer
}

func main() {
	provider := pimonoProvider{getenv: os.Getenv, stderr: os.Stderr}
	if err := plugin.ServeReviewProvider(context.Background(), provider, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func (p pimonoProvider) Initialize(_ context.Context, req plugin.InitializeRequest) (plugin.InitializeResult, error) {
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
			Label: "Pimono",
			Name:  "ero-plugin-pimono",
			Capabilities: plugin.ReviewProviderCapabilities{
				LoadRemoteComments: false,
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

func (p pimonoProvider) DetectContext(_ context.Context, _ plugin.DetectContextRequest) (plugin.DetectContextResult, error) {
	return plugin.DetectContextResult{Result: plugin.DetectionResult{Applicable: true, Reason: "Pimono can receive human review feedback"}}, nil
}

func (p pimonoProvider) LoadRemoteThreads(_ context.Context, _ plugin.LoadRemoteThreadsRequest) (plugin.LoadRemoteThreadsResult, error) {
	return plugin.LoadRemoteThreadsResult{}, plugin.NewError(plugin.ErrorUnsupportedCapability, "Pimono does not load remote review comments")
}

func (p pimonoProvider) PublishReview(_ context.Context, req plugin.PublishReviewParams) (plugin.PublishReviewResultData, error) {
	if p.getenv == nil {
		p.getenv = os.Getenv
	}
	if p.stderr == nil {
		p.stderr = os.Stderr
	}
	if p.getenv("PIMONO_DRY_RUN") != "1" {
		return plugin.PublishReviewResultData{}, plugin.NewError(plugin.ErrorAuthRequired, "set PIMONO_DRY_RUN=1 for local dry-run output until Pimono session wiring exists")
	}
	if err := json.NewEncoder(p.stderr).Encode(req.Payload); err != nil {
		return plugin.PublishReviewResultData{}, err
	}
	return plugin.PublishReviewResultData{Result: plugin.ReviewPublishResult{
		ProviderID:       providerID,
		ExternalReviewID: "dry-run-" + req.Payload.Context.Session.LocalReviewID,
		PublishedRefs:    []plugin.PublishedReviewCommentRef{},
	}}, nil
}
