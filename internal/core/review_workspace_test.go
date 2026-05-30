package core

import (
	"errors"
	"testing"
)

func TestReviewWorkspaceAddsRemoteThreads(t *testing.T) {
	workspace := NewReviewWorkspace(ReviewContext{})
	workspace.AddRemoteThreads("github", []RemoteReviewThread{{ExternalID: "t1"}})

	if len(workspace.RemoteThreads) != 1 {
		t.Fatalf("expected remote thread")
	}
	if workspace.RemoteThreads[0].ProviderID != "github" {
		t.Fatalf("expected provider id to be set, got %#v", workspace.RemoteThreads[0])
	}
}

func TestReviewWorkspaceRecordsSuccessfulPublishRefs(t *testing.T) {
	workspace := NewReviewWorkspace(ReviewContext{})
	workspace.RecordPublishError("github", errors.New("old error"))
	workspace.RecordPublishResult(PublishReviewResult{
		ProviderID:    "github",
		PublishedRefs: []PublishedReviewCommentRef{{LocalCommentID: "comment-1", ExternalID: "remote-1"}},
	})

	result, ok := workspace.PublishResults["github"]
	if !ok || len(result.PublishedRefs) != 1 || result.PublishedRefs[0].ExternalID != "remote-1" {
		t.Fatalf("unexpected result: %#v", workspace.PublishResults)
	}
	if _, ok := workspace.PublishErrors["github"]; ok {
		t.Fatalf("expected successful publish to clear previous error")
	}
}

func TestReviewWorkspaceRetainsFailedProviderState(t *testing.T) {
	workspace := NewReviewWorkspace(ReviewContext{})
	workspace.RecordPublishResult(PublishReviewResult{ProviderID: "github"})
	workspace.RecordPublishError("pimono", errors.New("auth required"))

	if workspace.PublishErrors["pimono"] != "auth required" {
		t.Fatalf("unexpected errors: %#v", workspace.PublishErrors)
	}
	if _, ok := workspace.PublishResults["github"]; !ok {
		t.Fatalf("expected existing successful provider state to remain")
	}
}
