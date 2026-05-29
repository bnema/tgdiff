package presenter

import "ero/internal/core"

type ReviewRowKind string

const (
	ReviewRowKindBlank    ReviewRowKind = "blank"
	ReviewRowKindFile     ReviewRowKind = "file"
	ReviewRowKindRule     ReviewRowKind = "rule"
	ReviewRowKindLine     ReviewRowKind = "line"
	ReviewRowKindExpander ReviewRowKind = "expander"
	ReviewRowKindMessage  ReviewRowKind = "message"
)

type ReviewRow struct {
	Kind         ReviewRowKind
	FileIndex    int
	SectionIndex int
	LineIndex    int
	FilePath     string
	Line         core.ReviewLine
	Text         string
	Selectable   bool
}

type ReviewAnchors struct {
	FileRows map[int]int
	LineRows map[ReviewLineAnchor]int
}

type ReviewLineAnchor struct {
	FileIndex    int
	SectionIndex int
	LineIndex    int
}
