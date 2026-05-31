package main

import (
	"context"
	"slices"
	"strings"
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

func TestPublishReviewSubmitsGitHubReview(t *testing.T) {
	var calls [][]string
	provider := githubProvider{execGH: func(_ context.Context, args ...string) (string, string, error) {
		calls = append(calls, slices.Clone(args))
		if len(calls) == 1 {
			return `{"number": 12, "url": "https://github.com/owner/repo/pull/12"}`, "", nil
		}
		return `{"id": 99, "html_url": "https://github.com/owner/repo/pull/12#pullrequestreview-99"}`, "", nil
	}}
	result, err := provider.PublishReview(context.Background(), plugin.PublishReviewParams{Payload: plugin.ReviewPublishPayload{
		Context: plugin.ReviewContext{Repository: plugin.RepositoryMetadata{HeadSHA: "abc123"}},
		Draft: plugin.ReviewDraftSnapshot{Decision: plugin.ReviewDecisionRequestChanges, Summary: "Please adjust", Comments: []plugin.ReviewComment{{
			ID:       "comment-1",
			FilePath: "docs/plugins.md",
			Body:     "tighten this",
			Range: plugin.ReviewLineRange{
				Start: plugin.ReviewLineRef{NewLineNumber: 9},
				End:   plugin.ReviewLineRef{NewLineNumber: 15},
			},
		}}},
	}})
	if err != nil {
		t.Fatalf("PublishReview returned error: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("expected 2 gh calls, got %#v", calls)
	}
	publish := strings.Join(calls[1], "\x00")
	for _, want := range []string{
		"api", "-X", "POST", "repos/{owner}/{repo}/pulls/12/reviews",
		"commit_id=abc123", "event=REQUEST_CHANGES", "body=Please adjust",
		"comments[][path]=docs/plugins.md", "comments[][body]=tighten this",
		"comments[][start_line]=9", "comments[][line]=15", "comments[][side]=RIGHT",
	} {
		if !strings.Contains(publish, want) {
			t.Fatalf("publish args missing %q: %#v", want, calls[1])
		}
	}
	if result.Result.ExternalReviewID != "99" || result.Result.ExternalURL == "" || len(result.Result.PublishedRefs) != 1 || result.Result.PublishedRefs[0].LocalCommentID != "comment-1" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestGithubCommentArgsMapsDeletedLinesToLeftSide(t *testing.T) {
	args, err := githubCommentArgs(plugin.ReviewComment{ID: "c", FilePath: "old.go", Body: "remove", Range: plugin.ReviewLineRange{Start: plugin.ReviewLineRef{OldLineNumber: 4}, End: plugin.ReviewLineRef{OldLineNumber: 4}}})
	if err != nil {
		t.Fatalf("githubCommentArgs returned error: %v", err)
	}
	joined := strings.Join(args, "\x00")
	if !strings.Contains(joined, "comments[][line]=4") || !strings.Contains(joined, "comments[][side]=LEFT") {
		t.Fatalf("unexpected args: %#v", args)
	}
}

type assertAnError struct{}

func (assertAnError) Error() string { return "assert error" }
