package tui

import (
	"testing"

	"github.com/stretchr/testify/require"

	"ero/internal/core"
)

func TestRemoteReviewAnnotations(t *testing.T) {
	rendered := NewReviewDocument(80).RenderWithAnnotations([]core.ReviewFile{reviewFile("demo.go", "package main")}, -1, -1, ReviewAnnotations{
		RemoteThreads: []core.RemoteReviewThread{{
			ProviderID: "github",
			FilePath:   "demo.go",
			Range:      core.ReviewLineRange{Start: core.ReviewLineRef{NewLineNumber: 1, Kind: core.LineKindAdded}, End: core.ReviewLineRef{NewLineNumber: 1, Kind: core.LineKindAdded}},
			Comments:   []core.RemoteReviewComment{{Author: "octocat", Body: "remote note"}},
		}},
	})
	view := stripANSI(rendered.Content)
	require.Contains(t, view, "[github]")
	require.Contains(t, view, "remote read-only")
	require.Contains(t, view, "octocat: remote note")
}

func TestRemoteReviewAnnotationsUnmapped(t *testing.T) {
	rendered := NewReviewDocument(80).RenderWithAnnotations([]core.ReviewFile{reviewFile("demo.go", "package main")}, -1, -1, ReviewAnnotations{
		RemoteThreads: []core.RemoteReviewThread{{ProviderID: "github", Unmapped: true, Comments: []core.RemoteReviewComment{{Body: "orphaned"}}}},
	})
	view := stripANSI(rendered.Content)
	require.Contains(t, view, "[github] unmapped: orphaned")
}
