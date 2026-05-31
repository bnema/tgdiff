package core

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReviewDraftAddsCommentsAndExportsStableJSON(t *testing.T) {
	t.Parallel()

	draft := NewReviewDraft()

	first, err := draft.AddComment(ReviewCommentInput{
		FilePath: "internal/demo.go",
		Range: ReviewLineRange{
			Start: ReviewLineRef{OldLineNumber: 0, NewLineNumber: 10, Kind: LineKindAdded},
			End:   ReviewLineRef{OldLineNumber: 0, NewLineNumber: 12, Kind: LineKindAdded},
		},
		Body: "first line\nsecond line",
	})
	require.NoError(t, err)
	second, err := draft.AddComment(ReviewCommentInput{
		FilePath: "internal/demo.go",
		Range: ReviewLineRange{
			Start: ReviewLineRef{OldLineNumber: 14, NewLineNumber: 0, Kind: LineKindDeleted},
			End:   ReviewLineRef{OldLineNumber: 14, NewLineNumber: 0, Kind: LineKindDeleted},
		},
		Body: "follow-up",
	})
	require.NoError(t, err)

	assert.Equal(t, "comment-1", first.ID)
	assert.Equal(t, "comment-2", second.ID)
	assert.Len(t, draft.Comments(), 2)

	exported, err := draft.ExportJSON()
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"comments": [
			{
				"id": "comment-1",
				"file": "internal/demo.go",
				"range": {
					"start": {"old": 0, "new": 10, "kind": "added"},
					"end": {"old": 0, "new": 12, "kind": "added"}
				},
				"body": "first line\nsecond line",
				"state": "local"
			},
			{
				"id": "comment-2",
				"file": "internal/demo.go",
				"range": {
					"start": {"old": 14, "new": 0, "kind": "deleted"},
					"end": {"old": 14, "new": 0, "kind": "deleted"}
				},
				"body": "follow-up",
				"state": "local"
			}
		]
	}`, string(exported))
}

func TestReviewDraftRejectsInvalidComments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input ReviewCommentInput
	}{
		{name: "missing file", input: ReviewCommentInput{Body: "body", Range: validReviewLineRange()}},
		{name: "missing body", input: ReviewCommentInput{FilePath: "demo.go", Range: validReviewLineRange()}},
		{name: "missing start line", input: ReviewCommentInput{FilePath: "demo.go", Body: "body", Range: ReviewLineRange{End: ReviewLineRef{NewLineNumber: 1, Kind: LineKindAdded}}}},
		{name: "missing end line", input: ReviewCommentInput{FilePath: "demo.go", Body: "body", Range: ReviewLineRange{Start: ReviewLineRef{NewLineNumber: 1, Kind: LineKindAdded}}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			draft := NewReviewDraft()
			_, err := draft.AddComment(tt.input)

			require.Error(t, err)
			assert.Empty(t, draft.Comments())
		})
	}
}

func TestReviewDraftCommentsReturnsDeepCopyAndClearResetsIDs(t *testing.T) {
	t.Parallel()

	draft := NewReviewDraft()
	_, err := draft.AddComment(ReviewCommentInput{FilePath: "demo.go", Range: validReviewLineRange(), Body: "body"})
	require.NoError(t, err)
	draft.comments[0].ProviderRefs = []ProviderCommentRef{{ProviderID: "github", ExternalID: "1"}}

	comments := draft.Comments()
	comments[0].Body = "mutated"
	comments[0].ProviderRefs[0].ExternalID = "mutated"
	stored := draft.Comments()[0]
	assert.Equal(t, "body", stored.Body)
	assert.Equal(t, "1", stored.ProviderRefs[0].ExternalID)

	draft.Clear()
	assert.Empty(t, draft.Comments())

	comment, err := draft.AddComment(ReviewCommentInput{FilePath: "demo.go", Range: validReviewLineRange(), Body: "new body"})
	require.NoError(t, err)
	assert.Equal(t, "comment-1", comment.ID)
}

func TestReviewDraftApplyPublishedRefs(t *testing.T) {
	t.Parallel()

	draft := NewReviewDraft()
	comment, err := draft.AddComment(ReviewCommentInput{FilePath: "demo.go", Range: validReviewLineRange(), Body: "body"})
	require.NoError(t, err)

	draft.ApplyPublishedRefs("github", []PublishedReviewCommentRef{{LocalCommentID: comment.ID, ExternalID: "remote-1", ExternalURL: "https://example.com"}})

	comments := draft.Comments()
	require.Len(t, comments, 1)
	assert.Equal(t, ReviewCommentStatePublished, comments[0].State)
	require.Len(t, comments[0].ProviderRefs, 1)
	assert.Equal(t, "github", comments[0].ProviderRefs[0].ProviderID)
	assert.Equal(t, "remote-1", comments[0].ProviderRefs[0].ExternalID)
}

func TestReviewDraftExportEmptyReview(t *testing.T) {
	t.Parallel()

	exported, err := NewReviewDraft().ExportJSON()
	require.NoError(t, err)

	assert.JSONEq(t, `{"comments": []}`, string(exported))
	var decoded map[string][]ReviewComment
	require.NoError(t, json.Unmarshal(exported, &decoded))
	assert.Empty(t, decoded["comments"])
}

func validReviewLineRange() ReviewLineRange {
	return ReviewLineRange{
		Start: ReviewLineRef{NewLineNumber: 1, Kind: LineKindAdded},
		End:   ReviewLineRef{NewLineNumber: 2, Kind: LineKindAdded},
	}
}
