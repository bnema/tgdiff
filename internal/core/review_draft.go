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
	ID           string               `json:"id"`
	FilePath     string               `json:"file"`
	Range        ReviewLineRange      `json:"range"`
	Body         string               `json:"body"`
	State        ReviewCommentState   `json:"state,omitempty"`
	ProviderRefs []ProviderCommentRef `json:"provider_refs,omitempty"`
}

// ProviderCommentRef maps a local comment to its remote published identifier.
type ProviderCommentRef struct {
	ProviderID  string `json:"provider_id"`
	ExternalID  string `json:"external_id"`
	ExternalURL string `json:"external_url,omitempty"`
}

// ReviewDraftSnapshot is an immutable copy of a ReviewDraft at a point in time.
type ReviewDraftSnapshot struct {
	ID             string          `json:"id"`
	Comments       []ReviewComment `json:"comments"`
	Decision       ReviewDecision  `json:"decision,omitempty"`
	Summary        string          `json:"summary,omitempty"`
	IdempotencyKey string          `json:"idempotency_key"`
}

type ReviewDraft struct {
	comments       []ReviewComment
	nextID         int
	decision       ReviewDecision
	summary        string
	id             string
	idempotencyKey string
}

func NewReviewDraft() *ReviewDraft {
	return &ReviewDraft{nextID: 1}
}

func ensureDraftIDs(d *ReviewDraft) {
	if d.id == "" {
		d.id = "draft-" + randomHex(8)
	}
	if d.idempotencyKey == "" {
		d.idempotencyKey = "publish-" + randomHex(16)
	}
}

// SetDecision records the review verdict on this draft.
func (d *ReviewDraft) SetDecision(decision ReviewDecision) {
	d.decision = decision
}

// Decision returns the current review verdict.
func (d *ReviewDraft) Decision() ReviewDecision {
	return d.decision
}

// SetSummary records a human-readable review summary. Leading/trailing
// whitespace is trimmed.
func (d *ReviewDraft) SetSummary(summary string) {
	d.summary = strings.TrimSpace(summary)
}

// Summary returns the human-readable review summary.
func (d *ReviewDraft) Summary() string {
	return d.summary
}

// IdempotencyKey returns a stable key for this draft, generating one if needed.
func (d *ReviewDraft) IdempotencyKey() string {
	ensureDraftIDs(d)
	return d.idempotencyKey
}

// Snapshot captures an immutable copy of the draft state.
func (d *ReviewDraft) Snapshot() ReviewDraftSnapshot {
	ensureDraftIDs(d)
	return ReviewDraftSnapshot{
		ID:             d.id,
		Comments:       d.Comments(),
		Decision:       d.decision,
		Summary:        d.summary,
		IdempotencyKey: d.idempotencyKey,
	}
}

func (d *ReviewDraft) AddComment(input ReviewCommentInput) (ReviewComment, error) {
	if err := validateReviewCommentInput(input); err != nil {
		return ReviewComment{}, err
	}
	comment := ReviewComment{
		ID:       fmt.Sprintf("comment-%d", d.nextID),
		FilePath: strings.TrimSpace(input.FilePath),
		Range:    input.Range,
		Body:     strings.TrimSpace(input.Body),
		State:    ReviewCommentStateLocal,
	}
	d.nextID++
	d.comments = append(d.comments, comment)
	return comment, nil
}

func (d *ReviewDraft) Comments() []ReviewComment {
	comments := append([]ReviewComment(nil), d.comments...)
	for i := range comments {
		comments[i].ProviderRefs = append([]ProviderCommentRef(nil), comments[i].ProviderRefs...)
	}
	return comments
}

func (d *ReviewDraft) ApplyPublishedRefs(providerID string, refs []PublishedReviewCommentRef) {
	for _, ref := range refs {
		for i := range d.comments {
			if d.comments[i].ID != ref.LocalCommentID {
				continue
			}
			d.comments[i].State = ReviewCommentStatePublished
			updated := false
			for j := range d.comments[i].ProviderRefs {
				if d.comments[i].ProviderRefs[j].ProviderID == providerID {
					d.comments[i].ProviderRefs[j].ExternalID = ref.ExternalID
					d.comments[i].ProviderRefs[j].ExternalURL = ref.ExternalURL
					updated = true
					break
				}
			}
			if !updated {
				d.comments[i].ProviderRefs = append(d.comments[i].ProviderRefs, ProviderCommentRef{ProviderID: providerID, ExternalID: ref.ExternalID, ExternalURL: ref.ExternalURL})
			}
		}
	}
}

func (d *ReviewDraft) Clear() {
	d.comments = nil
	d.nextID = 1
	d.decision = ""
	d.summary = ""
	d.id = ""
	d.idempotencyKey = ""
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
