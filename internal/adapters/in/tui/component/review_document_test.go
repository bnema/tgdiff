package component

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ero/internal/core"
)

func TestReviewDocumentRenderWithAnchors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		files             []core.ReviewFile
		selectedFile      int
		selectedContext   int
		wantContent       []string
		wantFileAnchor    int
		wantLineAnchor    ReviewLineAnchor
		wantLineAnchorSet bool
	}{
		{
			name:            "records file and visible line rows",
			files:           []core.ReviewFile{reviewFile("demo.go", "package main")},
			selectedFile:    0,
			selectedContext: -1,
			wantContent:     []string{"demo.go", "package main"},
			wantFileAnchor:  0,
			wantLineAnchor: ReviewLineAnchor{
				FileIndex:    0,
				SectionIndex: 0,
				LineIndex:    0,
			},
			wantLineAnchorSet: true,
		},
		{
			name:              "empty review renders a message without anchors",
			files:             nil,
			selectedFile:      -1,
			selectedContext:   -1,
			wantContent:       []string{"Review", "No files to review"},
			wantLineAnchorSet: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rendered := NewReviewDocument(80, "enter").RenderWithAnchors(tt.files, tt.selectedFile, tt.selectedContext)
			plainContent := stripANSI(rendered.Content)
			for _, expected := range tt.wantContent {
				assert.Contains(t, plainContent, expected)
			}
			assert.Equal(t, strings.Join(rendered.Lines, "\n"), rendered.Content)

			if len(tt.files) > 0 {
				assert.Equal(t, tt.wantFileAnchor, rendered.Anchors.FileRows[0])
			}
			if tt.wantLineAnchorSet {
				_, ok := rendered.Anchors.LineRows[tt.wantLineAnchor]
				require.True(t, ok)
			} else {
				assert.Empty(t, rendered.Anchors.LineRows)
			}
		})
	}
}

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}

func reviewFile(path, content string) core.ReviewFile {
	return core.ReviewFile{
		Path: path,
		Sections: []core.ReviewSection{{
			ID:    path + "-changed",
			Kind:  core.SectionKindChanged,
			Lines: []core.ReviewLine{{NewLineNumber: 1, Content: content, Kind: core.LineKindAdded}},
		}},
	}
}
