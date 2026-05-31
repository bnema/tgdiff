package main

import (
	"context"
	"testing"

	"ero/pkg/plugin"
)

func TestDetectContextRequiresGitHubRemote(t *testing.T) {
	provider := githubProvider{}
	result, err := provider.DetectContext(context.Background(), plugin.DetectContextRequest{Context: plugin.ReviewContext{Repository: plugin.RepositoryMetadata{Remotes: []plugin.GitRemote{{Name: "origin", URL: "git@github.com:owner/repo.git"}}}}})
	if err != nil {
		t.Fatalf("DetectContext returned error: %v", err)
	}
	if !result.Result.Applicable {
		t.Fatalf("expected GitHub remote to be applicable: %#v", result)
	}
}

func TestPublishReviewRequiresAssociatedPullRequest(t *testing.T) {
	provider := githubProvider{execGH: func(context.Context, ...string) (string, string, error) {
		return "", "no pull requests found", assertAnError{}
	}}
	_, err := provider.PublishReview(context.Background(), plugin.PublishReviewParams{})
	if plugin.AsError(err) == nil || plugin.AsError(err).Code != plugin.ErrorNotApplicable {
		t.Fatalf("expected not_applicable, got %v", err)
	}
}

func TestPublishReviewReportsUnimplementedAfterPRDetected(t *testing.T) {
	provider := githubProvider{execGH: func(context.Context, ...string) (string, string, error) {
		return `{"number": 12, "url": "https://github.com/owner/repo/pull/12"}`, "", nil
	}}
	_, err := provider.PublishReview(context.Background(), plugin.PublishReviewParams{})
	if plugin.AsError(err) == nil || plugin.AsError(err).Code != plugin.ErrorUnsupportedCapability {
		t.Fatalf("expected unsupported_capability, got %v", err)
	}
}

type assertAnError struct{}

func (assertAnError) Error() string { return "assert error" }
