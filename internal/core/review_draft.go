package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type ReviewLineRef struct {
	OldLineNumber int      `json:"old"`
	NewLineNumber int      `json:"new"`
	Kind          LineKind `json:"kind"`
}

func (r ReviewLineRef) valid() bool {
	return r.OldLineNumber > 0 || r.NewLineNumber > 0
}

type ReviewLineRange struct {
	Start ReviewLineRef `json:"start"`
	End   ReviewLineRef `json:"end"`
}

func (r ReviewLineRange) valid() bool {
	return r.Start.valid() && r.End.valid()
}

type ReviewCommentInput struct {
	FilePath string
	Range    ReviewLineRange
	Body     string
}

type ReviewComment struct {
	ID       string          `json:"id"`
	FilePath string          `json:"file"`
	Range    ReviewLineRange `json:"range"`
	Body     string          `json:"body"`
}

type ReviewDraft struct {
	comments []ReviewComment
	nextID   int
}

func NewReviewDraft() *ReviewDraft {
	return &ReviewDraft{nextID: 1}
}

func (d *ReviewDraft) AddComment(input ReviewCommentInput) (ReviewComment, error) {
	if err := validateReviewCommentInput(input); err != nil {
		return ReviewComment{}, err
	}
	if d.nextID <= 0 {
		d.nextID = 1
	}
	comment := ReviewComment{
		ID:       fmt.Sprintf("comment-%d", d.nextID),
		FilePath: strings.TrimSpace(input.FilePath),
		Range:    input.Range,
		Body:     strings.TrimSpace(input.Body),
	}
	d.nextID++
	d.comments = append(d.comments, comment)
	return comment, nil
}

func (d *ReviewDraft) Comments() []ReviewComment {
	return append([]ReviewComment(nil), d.comments...)
}

func (d *ReviewDraft) Clear() {
	d.comments = nil
	d.nextID = 1
}

func (d *ReviewDraft) ExportJSON() ([]byte, error) {
	comments := d.Comments()
	if comments == nil {
		comments = []ReviewComment{}
	}
	return json.MarshalIndent(struct {
		Comments []ReviewComment `json:"comments"`
	}{Comments: comments}, "", "  ")
}

func validateReviewCommentInput(input ReviewCommentInput) error {
	if strings.TrimSpace(input.FilePath) == "" {
		return errors.New("review comment file path is required")
	}
	if strings.TrimSpace(input.Body) == "" {
		return errors.New("review comment body is required")
	}
	if !input.Range.valid() {
		return errors.New("review comment range is required")
	}
	return nil
}
