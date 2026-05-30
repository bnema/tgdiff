package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReviewDraftSupportsDecisionSummaryAndIdempotency(t *testing.T) {
	t.Parallel()

	draft := NewReviewDraft()
	draft.SetDecision(ReviewDecisionRequestChanges)
	draft.SetSummary("Please address these comments.")

	assert.Equal(t, ReviewDecisionRequestChanges, draft.Decision())
	assert.Equal(t, "Please address these comments.", draft.Summary())
	assert.NotEmpty(t, draft.IdempotencyKey(), "expected idempotency key")
}

func TestReviewDraftSnapshotCarriesFullState(t *testing.T) {
	t.Parallel()

	draft := NewReviewDraft()
	_, err := draft.AddComment(ReviewCommentInput{
		FilePath: "demo.go",
		Range:    validReviewLineRange(),
		Body:     "looks good",
	})
	require.NoError(t, err)

	draft.SetDecision(ReviewDecisionApprove)
	draft.SetSummary("LGTM")

	snap := draft.Snapshot()

	assert.Equal(t, ReviewDecisionApprove, snap.Decision)
	assert.Equal(t, "LGTM", snap.Summary)
	assert.NotEmpty(t, snap.IdempotencyKey)
	assert.Len(t, snap.Comments, 1)
	assert.Equal(t, "looks good", snap.Comments[0].Body)
}

func TestProviderCapabilitiesDecisionSupport(t *testing.T) {
	t.Parallel()

	caps := ReviewProviderCapabilities{
		PublishReview: true,
		Decisions:     []ReviewDecision{ReviewDecisionComment},
	}

	assert.True(t, caps.SupportsDecision(ReviewDecisionComment))
	assert.False(t, caps.SupportsDecision(ReviewDecisionApprove))
	assert.True(t, caps.SupportsDecision(""), "empty decision should be supported")
}

func TestProviderCapabilitiesNilDecisions(t *testing.T) {
	t.Parallel()

	caps := ReviewProviderCapabilities{PublishReview: true}

	assert.True(t, caps.SupportsDecision(""), "empty decision should be supported")
	assert.False(t, caps.SupportsDecision(ReviewDecisionComment))
}

func TestReviewCommentHasStateLocalOnAdd(t *testing.T) {
	t.Parallel()

	draft := NewReviewDraft()
	comment, err := draft.AddComment(ReviewCommentInput{
		FilePath: "demo.go",
		Range:    validReviewLineRange(),
		Body:     "body",
	})
	require.NoError(t, err)

	assert.Equal(t, ReviewCommentStateLocal, comment.State)
}
