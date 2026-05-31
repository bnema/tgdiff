package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReviewContextNormalizeDefaults(t *testing.T) {
	t.Parallel()

	ctx := ReviewContext{
		Repository: RepositoryMetadata{RepoPath: ".", WorktreeRoot: "/repo"},
		Target:     ReviewTargetMetadata{Mode: DiffModeBranch, BaseRef: "main", HeadRef: "feature"},
		Session:    ReviewSessionMetadata{EroVersion: "dev", ProtocolVersion: "ero.plugin.v1"},
	}

	normalized := ctx.Normalize()

	assert.NotEmpty(t, normalized.Session.LocalReviewID, "expected generated local review id")
	assert.NotEmpty(t, normalized.Session.IdempotencyKey, "expected generated idempotency key")
	assert.False(t, normalized.Session.CreatedAt.IsZero(), "expected created at timestamp")
}

func TestReviewContextNormalizePreservesExplicitIDs(t *testing.T) {
	t.Parallel()

	ctx := ReviewContext{
		Repository: RepositoryMetadata{RepoPath: ".", WorktreeRoot: "/repo"},
		Target:     ReviewTargetMetadata{Mode: DiffModeBranch, BaseRef: "main", HeadRef: "feature"},
		Session: ReviewSessionMetadata{
			EroVersion:      "dev",
			ProtocolVersion: "ero.plugin.v1",
			LocalReviewID:   "explicit-review-id",
			IdempotencyKey:  "explicit-key",
		},
	}

	normalized := ctx.Normalize()

	assert.Equal(t, "explicit-review-id", normalized.Session.LocalReviewID)
	assert.Equal(t, "explicit-key", normalized.Session.IdempotencyKey)
}

func TestReviewLineAnchorFromLineAdded(t *testing.T) {
	t.Parallel()

	anchor := NewReviewLineAnchor("internal/core/review.go", ReviewLine{
		OldLineNumber: 0,
		NewLineNumber: 12,
		Kind:          LineKindAdded,
	})

	assert.Equal(t, "internal/core/review.go", anchor.FilePath)
	assert.Equal(t, 0, anchor.OldLineNumber)
	assert.Equal(t, 12, anchor.NewLineNumber)
	assert.Equal(t, ReviewLineSideNew, anchor.Side)
}

func TestReviewLineAnchorFromLineDeleted(t *testing.T) {
	t.Parallel()

	anchor := NewReviewLineAnchor("internal/core/review.go", ReviewLine{
		OldLineNumber: 10,
		NewLineNumber: 0,
		Kind:          LineKindDeleted,
	})

	assert.Equal(t, "internal/core/review.go", anchor.FilePath)
	assert.Equal(t, 10, anchor.OldLineNumber)
	assert.Equal(t, 0, anchor.NewLineNumber)
	assert.Equal(t, ReviewLineSideOld, anchor.Side)
}

func TestReviewLineAnchorFromLineUnchanged(t *testing.T) {
	t.Parallel()

	anchor := NewReviewLineAnchor("internal/core/review.go", ReviewLine{
		OldLineNumber: 5,
		NewLineNumber: 5,
		Kind:          LineKindUnchanged,
	})

	// Unchanged lines anchor on the new side.
	assert.Equal(t, ReviewLineSideNew, anchor.Side)
	assert.Equal(t, 5, anchor.OldLineNumber)
	assert.Equal(t, 5, anchor.NewLineNumber)
}
