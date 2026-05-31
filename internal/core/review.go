package core

import "fmt"

type LineKind string

const (
	LineKindUnchanged LineKind = "unchanged"
	LineKindAdded     LineKind = "added"
	LineKindDeleted   LineKind = "deleted"
)

type ReviewLine struct {
	OldLineNumber int
	NewLineNumber int
	Content       string
	Kind          LineKind
	SyntaxTokens  []SyntaxToken
}

func (l ReviewLine) IsChanged() bool {
	return l.Kind == LineKindAdded || l.Kind == LineKindDeleted
}

type ReviewSectionKind string

const (
	SectionKindChanged ReviewSectionKind = "changed"
	SectionKindContext ReviewSectionKind = "context"
)

type ReviewSection struct {
	ID            string
	Kind          ReviewSectionKind
	Lines         []ReviewLine
	ExpandedAbove int
	ExpandedBelow int
}

func (s *ReviewSection) ExpandAbove(count int) {
	if s.Kind != SectionKindContext || count <= 0 {
		return
	}

	s.ExpandedAbove += count
	if s.ExpandedAbove > len(s.Lines) {
		s.ExpandedAbove = len(s.Lines)
	}
}

func (s *ReviewSection) ExpandBelow(count int) {
	if s.Kind != SectionKindContext || count <= 0 {
		return
	}

	s.ExpandedBelow += count
	if s.ExpandedBelow > len(s.Lines) {
		s.ExpandedBelow = len(s.Lines)
	}
}

func (s *ReviewSection) ExpandAll() {
	if s.Kind != SectionKindContext {
		return
	}

	s.ExpandedAbove = len(s.Lines)
	s.ExpandedBelow = 0
}

func (s ReviewSection) VisibleLines() []ReviewLine {
	if s.Kind == SectionKindChanged {
		return append([]ReviewLine(nil), s.Lines...)
	}

	visibleCount := s.visibleCount()
	if visibleCount == 0 {
		return nil
	}
	if visibleCount >= len(s.Lines) {
		return append([]ReviewLine(nil), s.Lines...)
	}

	result := make([]ReviewLine, 0, visibleCount)
	if s.ExpandedAbove > 0 {
		result = append(result, s.Lines[:min(s.ExpandedAbove, len(s.Lines))]...)
	}
	if s.ExpandedBelow > 0 {
		start := len(s.Lines) - min(s.ExpandedBelow, len(s.Lines)-min(s.ExpandedAbove, len(s.Lines)))
		result = append(result, s.Lines[start:]...)
	}
	return result
}

func (s ReviewSection) HiddenLineCount() int {
	if s.Kind != SectionKindContext {
		return 0
	}

	hidden := len(s.Lines) - s.visibleCount()
	if hidden < 0 {
		return 0
	}
	return hidden
}

func (s ReviewSection) visibleCount() int {
	visible := s.ExpandedAbove + s.ExpandedBelow
	if visible > len(s.Lines) {
		return len(s.Lines)
	}
	return visible
}

type ReviewFile struct {
	Path     string
	OldPath  string
	Status   ReviewFileStatus
	Sections []ReviewSection
}

func BuildReviewFile(path string, lines []ReviewLine, contextWindow int) ReviewFile {
	return BuildReviewFileWithMetadata(path, "", ReviewFileStatusModified, lines, contextWindow)
}

func BuildReviewFileWithMetadata(path, oldPath string, status ReviewFileStatus, lines []ReviewLine, contextWindow int) ReviewFile {
	if status == "" {
		status = ReviewFileStatusModified
	}
	reviewFile := ReviewFile{Path: path, OldPath: oldPath, Status: status}
	if len(lines) == 0 {
		return reviewFile
	}
	if contextWindow < 0 {
		contextWindow = 0
	}

	visibleRanges := changedRanges(lines, contextWindow)
	if len(visibleRanges) == 0 {
		reviewFile.Sections = []ReviewSection{newContextSection(1, lines)}
		return reviewFile
	}

	sectionIndex := 1
	cursor := 0
	for _, r := range visibleRanges {
		if cursor < r.start {
			reviewFile.Sections = append(reviewFile.Sections, newContextSection(sectionIndex, lines[cursor:r.start]))
			sectionIndex++
		}

		reviewFile.Sections = append(reviewFile.Sections, ReviewSection{
			ID:    fmt.Sprintf("section-%d", sectionIndex),
			Kind:  SectionKindChanged,
			Lines: append([]ReviewLine(nil), lines[r.start:r.end+1]...),
		})
		sectionIndex++
		cursor = r.end + 1
	}

	if cursor < len(lines) {
		reviewFile.Sections = append(reviewFile.Sections, newContextSection(sectionIndex, lines[cursor:]))
	}

	return reviewFile
}

type lineRange struct {
	start int
	end   int
}

func changedRanges(lines []ReviewLine, contextWindow int) []lineRange {
	var ranges []lineRange

	for i := 0; i < len(lines); i++ {
		if !lines[i].IsChanged() {
			continue
		}

		start := i
		for i+1 < len(lines) && lines[i+1].IsChanged() {
			i++
		}
		end := i

		rangeStart := max(0, start-contextWindow)
		rangeEnd := min(len(lines)-1, end+contextWindow)

		if len(ranges) > 0 && rangeStart <= ranges[len(ranges)-1].end+1 {
			if rangeEnd > ranges[len(ranges)-1].end {
				ranges[len(ranges)-1].end = rangeEnd
			}
			continue
		}

		ranges = append(ranges, lineRange{start: rangeStart, end: rangeEnd})
	}

	return ranges
}

func newContextSection(index int, lines []ReviewLine) ReviewSection {
	return ReviewSection{
		ID:    fmt.Sprintf("section-%d", index),
		Kind:  SectionKindContext,
		Lines: append([]ReviewLine(nil), lines...),
	}
}
