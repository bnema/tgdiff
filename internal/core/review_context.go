package core

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// ReviewContext captures the full metadata about a review session — repository, target
// revision, diff stats, file metadata, and session-level identifiers.
type ReviewContext struct {
	Repository RepositoryMetadata    `json:"repository"`
	Target     ReviewTargetMetadata  `json:"target"`
	Diff       DiffMetadata          `json:"diff"`
	Files      []ReviewFileMetadata  `json:"files"`
	Session    ReviewSessionMetadata `json:"session"`
}

// RepositoryMetadata describes the repository and its remotes.
type RepositoryMetadata struct {
	RepoPath      string      `json:"repo_path"`
	WorktreeRoot  string      `json:"worktree_root"`
	Remotes       []GitRemote `json:"remotes"`
	DefaultBranch string      `json:"default_branch,omitempty"`
	CurrentBranch string      `json:"current_branch,omitempty"`
	HeadSHA       string      `json:"head_sha,omitempty"`
}

// GitRemote is a named remote URL.
type GitRemote struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// ReviewTargetMetadata describes the revision range under review.
type ReviewTargetMetadata struct {
	Mode         DiffMode `json:"mode"`
	BaseRef      string   `json:"base_ref,omitempty"`
	HeadRef      string   `json:"head_ref,omitempty"`
	BaseSHA      string   `json:"base_sha,omitempty"`
	HeadSHA      string   `json:"head_sha,omitempty"`
	MergeBaseSHA string   `json:"merge_base_sha,omitempty"`
}

// DiffMetadata holds aggregate diff statistics.
type DiffMetadata struct {
	FilesChanged int `json:"files_changed"`
	Additions    int `json:"additions"`
	Deletions    int `json:"deletions"`
}

// ReviewFileStatus is the change status of a file under review.
type ReviewFileStatus string

const (
	ReviewFileStatusModified ReviewFileStatus = "modified"
	ReviewFileStatusAdded    ReviewFileStatus = "added"
	ReviewFileStatusDeleted  ReviewFileStatus = "deleted"
	ReviewFileStatusRenamed  ReviewFileStatus = "renamed"
)

// ReviewFileMetadata captures per-file change details with hunk and line anchors.
type ReviewFileMetadata struct {
	Path        string             `json:"path"`
	OldPath     string             `json:"old_path,omitempty"`
	Status      ReviewFileStatus   `json:"status"`
	Language    string             `json:"language,omitempty"`
	Hunks       []ReviewHunkAnchor `json:"hunks"`
	LineAnchors []ReviewLineAnchor `json:"line_anchors"`
}

// ReviewHunkAnchor references a unified section (hunk) in a review file.
type ReviewHunkAnchor struct {
	SectionID    string `json:"section_id"`
	OldStartLine int    `json:"old_start_line,omitempty"`
	NewStartLine int    `json:"new_start_line,omitempty"`
}

// ReviewLineSide indicates which side of a diff a line resides on.
type ReviewLineSide string

const (
	ReviewLineSideOld ReviewLineSide = "old"
	ReviewLineSideNew ReviewLineSide = "new"
)

// ReviewLineAnchor is a stable reference to a line in a diff file. It picks the
// side (old or new) using the canonical rule: deletions anchor on the old side;
// all other line kinds anchor on the new side.
type ReviewLineAnchor struct {
	FilePath      string         `json:"file_path"`
	OldLineNumber int            `json:"old_line_number,omitempty"`
	NewLineNumber int            `json:"new_line_number,omitempty"`
	Side          ReviewLineSide `json:"side"`
}

// ReviewSessionMetadata holds review-session identifiers and timestamps.
type ReviewSessionMetadata struct {
	EroVersion      string    `json:"ero_version"`
	ProtocolVersion string    `json:"protocol_version"`
	LocalReviewID   string    `json:"local_review_id"`
	CreatedAt       time.Time `json:"created_at"`
	IdempotencyKey  string    `json:"idempotency_key"`
}

// Normalize returns a copy of c with auto-generated session identifiers
// for any fields that are still empty. Callers can pre-set IDs to make
// tests deterministic.
func (c ReviewContext) Normalize() ReviewContext {
	if c.Session.LocalReviewID == "" {
		c.Session.LocalReviewID = "review-" + randomHex(8)
	}
	if c.Session.IdempotencyKey == "" {
		c.Session.IdempotencyKey = "publish-" + randomHex(16)
	}
	if c.Session.CreatedAt.IsZero() {
		c.Session.CreatedAt = time.Now().UTC()
	}
	return c
}

// NewReviewLineAnchor creates a ReviewLineAnchor from a file path and a review
// line. Deletions anchor on the old side; all other line kinds anchor on the
// new side.
func NewReviewLineAnchor(path string, line ReviewLine) ReviewLineAnchor {
	anchor := ReviewLineAnchor{
		FilePath:      path,
		OldLineNumber: line.OldLineNumber,
		NewLineNumber: line.NewLineNumber,
	}
	if line.Kind == LineKindDeleted {
		anchor.Side = ReviewLineSideOld
	} else {
		anchor.Side = ReviewLineSideNew
	}
	return anchor
}

func randomHex(bytes int) string {
	buf := make([]byte, bytes)
	if _, err := rand.Read(buf); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format("20060102150405.000000000")))
	}
	return hex.EncodeToString(buf)
}
