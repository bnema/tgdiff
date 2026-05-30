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

func TestPublishReviewRequiresToken(t *testing.T) {
	provider := githubProvider{getenv: func(string) string { return "" }}
	_, err := provider.PublishReview(context.Background(), plugin.PublishReviewParams{})
	if plugin.AsError(err) == nil || plugin.AsError(err).Code != plugin.ErrorAuthRequired {
		t.Fatalf("expected auth_required, got %v", err)
	}
}
