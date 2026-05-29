package tui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"ero/internal/core"
)

var (
	inlineCommentStyle      = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, false, false, true).BorderForeground(lipgloss.Color("62")).PaddingLeft(1)
	inlineCommentIconStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("81")).Bold(true)
	inlineCommentIDStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Bold(true)
	inlineCommentBodyStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("248"))
	inlineCommentMutedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

type ReviewAnnotations struct {
	Comments []core.ReviewComment
	Editor   *InlineCommentEditor
}

type InlineCommentEditor struct {
	FilePath string
	Range    core.ReviewLineRange
	Editor   CommentEditor
}

func (c ReviewDocument) RenderWithAnnotations(files []core.ReviewFile, selectedFile, selectedContext int, annotations ReviewAnnotations) ReviewDocumentRender {
	rendered := c.RenderWithAnchors(files, selectedFile, selectedContext)
	if len(annotations.Comments) == 0 && annotations.Editor == nil {
		return rendered
	}

	lineIndents := reviewLineContentIndents(files)
	lines := make([]string, 0, len(rendered.Lines)+len(annotations.Comments)*2+4)
	rows := make([]ReviewRow, 0, len(rendered.Rows)+len(annotations.Comments)*2+4)
	for index, row := range rendered.Rows {
		lines = append(lines, rendered.Lines[index])
		rows = append(rows, row)
		indent := inlineAnnotationIndent(row, lineIndents)
		for _, comment := range annotations.Comments {
			if commentBelongsAfterRow(comment, row) {
				for _, commentLine := range renderInlineComment(comment, indent) {
					lines = append(lines, commentLine)
					rows = append(rows, ReviewRow{Kind: ReviewRowKindMessage, FileIndex: row.FileIndex, SectionIndex: row.SectionIndex, FilePath: row.FilePath, Text: commentLine})
				}
			}
		}
		if annotations.Editor != nil && editorBelongsAfterRow(*annotations.Editor, row) {
			availableWidth := max(c.width-len([]rune(indent)), 1)
			for editorLine := range strings.SplitSeq(annotations.Editor.Editor.ViewWithWidth(availableWidth), "\n") {
				indentedLine := indent + editorLine
				lines = append(lines, indentedLine)
				rows = append(rows, ReviewRow{Kind: ReviewRowKindMessage, FileIndex: row.FileIndex, SectionIndex: row.SectionIndex, FilePath: row.FilePath, Text: indentedLine})
			}
		}
	}
	rendered.Content = strings.Join(lines, "\n")
	rendered.Lines = lines
	rendered.Rows = rows
	rendered.Anchors = reviewAnchorsFromRows(rows)
	return rendered
}

func reviewAnchorsFromRows(rows []ReviewRow) ReviewAnchors {
	anchors := ReviewAnchors{FileRows: map[int]int{}, LineRows: map[ReviewLineAnchor]int{}}
	for rowIndex, row := range rows {
		switch row.Kind {
		case ReviewRowKindFile:
			if _, exists := anchors.FileRows[row.FileIndex]; !exists {
				anchors.FileRows[row.FileIndex] = rowIndex
			}
		case ReviewRowKindLine:
			anchors.LineRows[ReviewLineAnchor{FileIndex: row.FileIndex, SectionIndex: row.SectionIndex, LineIndex: row.LineIndex}] = rowIndex
		}
	}
	return anchors
}

func commentBelongsAfterRow(comment core.ReviewComment, row ReviewRow) bool {
	if row.Kind != ReviewRowKindLine || row.FilePath != comment.FilePath {
		return false
	}
	return reviewLineMatchesRef(row.Line, comment.Range.End)
}

func editorBelongsAfterRow(editor InlineCommentEditor, row ReviewRow) bool {
	if row.Kind != ReviewRowKindLine || row.FilePath != editor.FilePath {
		return false
	}
	return reviewLineMatchesRef(row.Line, editor.Range.End)
}

func reviewLineMatchesRef(line core.ReviewLine, ref core.ReviewLineRef) bool {
	if ref.NewLineNumber > 0 && line.NewLineNumber == ref.NewLineNumber {
		return true
	}
	if ref.OldLineNumber > 0 && line.OldLineNumber == ref.OldLineNumber {
		return true
	}
	return false
}

func reviewLineContentIndents(files []core.ReviewFile) map[int]string {
	indents := make(map[int]string, len(files))
	for fileIndex, file := range files {
		indents[fileIndex] = strings.Repeat(" ", lineNumberWidth(file)*2+4)
	}
	return indents
}

func inlineAnnotationIndent(row ReviewRow, lineIndents map[int]string) string {
	if row.Kind != ReviewRowKindLine {
		return ""
	}
	return lineIndents[row.FileIndex]
}

func renderInlineComment(comment core.ReviewComment, indent string) []string {
	bodyLines := strings.Split(comment.Body, "\n")
	lines := make([]string, 0, len(bodyLines)+1)
	header := inlineCommentIconStyle.Render(nerdIconComment) + " " + inlineCommentIDStyle.Render(displayReviewCommentID(comment.ID))
	lines = append(lines, indent+inlineCommentStyle.Render(header))
	for _, bodyLine := range bodyLines {
		lines = append(lines, indent+inlineCommentStyle.Render(inlineCommentBodyStyle.Render(bodyLine)))
	}
	return lines
}

func displayReviewCommentID(id string) string {
	if number, ok := strings.CutPrefix(id, "comment-"); ok && number != "" {
		return "#" + number
	}
	return inlineCommentMutedStyle.Render(id)
}
