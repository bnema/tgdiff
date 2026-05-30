package core

// ReviewWorkspace aggregates mutable in-memory state for a review session.
type ReviewWorkspace struct {
	Context        ReviewContext
	Draft          *ReviewDraft
	RemoteThreads  []RemoteReviewThread
	PublishResults map[string]PublishReviewResult
	PublishErrors  map[string]string
}

// NewReviewWorkspace creates an empty workspace for the normalized review context.
func NewReviewWorkspace(ctx ReviewContext) *ReviewWorkspace {
	return &ReviewWorkspace{
		Context:        ctx,
		Draft:          NewReviewDraft(),
		PublishResults: make(map[string]PublishReviewResult),
		PublishErrors:  make(map[string]string),
	}
}

// AddRemoteThreads appends imported threads, stamping the provider ID where absent.
func (w *ReviewWorkspace) AddRemoteThreads(providerID string, threads []RemoteReviewThread) {
	if w == nil || len(threads) == 0 {
		return
	}
	for _, thread := range threads {
		if thread.ProviderID == "" {
			thread.ProviderID = providerID
		}
		w.RemoteThreads = append(w.RemoteThreads, thread)
	}
}

// RecordPublishResult stores a successful provider publish result and clears any prior error.
func (w *ReviewWorkspace) RecordPublishResult(result PublishReviewResult) {
	if w == nil || result.ProviderID == "" {
		return
	}
	if w.PublishResults == nil {
		w.PublishResults = make(map[string]PublishReviewResult)
	}
	w.PublishResults[result.ProviderID] = result
	if w.PublishErrors != nil {
		delete(w.PublishErrors, result.ProviderID)
	}
}

// RecordPublishError stores a failed provider publish state without disturbing other results.
func (w *ReviewWorkspace) RecordPublishError(providerID string, err error) {
	if w == nil || providerID == "" || err == nil {
		return
	}
	if w.PublishErrors == nil {
		w.PublishErrors = make(map[string]string)
	}
	w.PublishErrors[providerID] = err.Error()
}
