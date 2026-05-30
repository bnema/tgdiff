package tui

import (
	"context"
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"

	"ero/internal/core"
	"ero/internal/ports"
)

func TestUppercasePOpensPublishOverlayWithProvider(t *testing.T) {
	provider := &fakeReviewProvider{info: core.ReviewProviderInfo{ID: "pi-coding-agent", Label: "pi-coding-agent", Capabilities: core.ReviewProviderCapabilities{PublishReview: true}}}
	m := NewModelWithReviewProviders([]core.ReviewFile{reviewFile("demo.go", "package main")}, nil, nil, core.ReviewRequest{}, nil, core.ReviewContext{}, nil)
	m.reviewProviders = []ports.ReviewProviderClient{fakeProviderAsPort{provider}}
	m.providerInfos = []core.ReviewProviderInfo{provider.info}
	m.providerInfoByClient = map[ports.ReviewProviderClient]core.ReviewProviderInfo{m.reviewProviders[0]: provider.info}

	updated, cmd := m.Update(keyPress("P"))
	m = updated.(Model)

	require.Nil(t, cmd)
	require.True(t, m.publish.active)
	require.True(t, m.publish.selected["pi-coding-agent"])
	require.Contains(t, stripANSI(m.View().Content), "Publish review")
}

func TestPublishReviewSuccess(t *testing.T) {
	provider := &fakeReviewProvider{info: core.ReviewProviderInfo{ID: "github", Label: "GitHub", Capabilities: core.ReviewProviderCapabilities{PublishReview: true, Decisions: []core.ReviewDecision{core.ReviewDecisionComment}}}}
	m := NewModelWithReviewProviders([]core.ReviewFile{reviewFile("demo.go", "package main")}, nil, nil, core.ReviewRequest{}, nil, core.ReviewContext{}, nil)
	m.reviewProviders = []ports.ReviewProviderClient{fakeProviderAsPort{provider}}
	m.providerInfos = []core.ReviewProviderInfo{provider.info}
	m.reviewDraft.SetDecision(core.ReviewDecisionComment)
	m, _ = m.openPublishReview()
	updated, cmd := m.publishSelectedProviders()
	m = updated
	require.NotNil(t, cmd)
	msg := cmd().(publishReviewCompletedMsg)
	msg.results = []core.PublishReviewResult{{ProviderID: "github", PublishedRefs: []core.PublishedReviewCommentRef{{LocalCommentID: "comment-1", ExternalID: "remote-1"}}}}
	m.reviewDraft.ApplyPublishedRefs("github", nil)
	_, err := m.reviewDraft.AddComment(core.ReviewCommentInput{FilePath: "demo.go", Range: publishTestRange(), Body: "body"})
	require.NoError(t, err)
	model, _ := m.Update(msg)
	m = model.(Model)
	require.False(t, m.publish.active)
	require.Len(t, provider.requests, 1)
	require.Equal(t, core.ReviewDecisionComment, provider.requests[0].Draft.Decision)
	comments := m.reviewDraft.Comments()
	require.Len(t, comments, 1)
	require.Len(t, comments[0].ProviderRefs, 1)
	require.Equal(t, "remote-1", comments[0].ProviderRefs[0].ExternalID)
}

func TestPublishReviewFailedProvider(t *testing.T) {
	provider := &fakeReviewProvider{info: core.ReviewProviderInfo{ID: "github", Capabilities: core.ReviewProviderCapabilities{PublishReview: true}}, publishErr: errors.New("auth required")}
	m := NewModelWithReviewProviders([]core.ReviewFile{reviewFile("demo.go", "package main")}, nil, nil, core.ReviewRequest{}, nil, core.ReviewContext{}, nil)
	m.reviewProviders = []ports.ReviewProviderClient{fakeProviderAsPort{provider}}
	m.providerInfos = []core.ReviewProviderInfo{provider.info}
	m, _ = m.openPublishReview()
	updated, cmd := m.publishSelectedProviders()
	m = updated
	model, _ := m.Update(cmd())
	m = model.(Model)
	require.True(t, m.publish.active)
	require.Contains(t, m.publish.message, "auth required")
}

func TestStatusBarShowsProviderPublishHint(t *testing.T) {
	provider := &fakeReviewProvider{info: core.ReviewProviderInfo{ID: "pi-coding-agent", Label: "pi-coding-agent", Capabilities: core.ReviewProviderCapabilities{PublishReview: true}}}
	m := NewModelWithReviewProviders([]core.ReviewFile{reviewFile("demo.go", "package main")}, nil, nil, core.ReviewRequest{}, nil, core.ReviewContext{}, nil)
	m.providerInfos = []core.ReviewProviderInfo{provider.info}

	view := stripANSI(m.View().Content)
	require.Contains(t, view, "1 provider")
	require.Contains(t, view, "P publish")
}

func TestProviderUnavailableReasonAppearsInStatusBar(t *testing.T) {
	provider := &fakeReviewProvider{
		info:      core.ReviewProviderInfo{ID: "pi-coding-agent", Label: "pi-coding-agent", Capabilities: core.ReviewProviderCapabilities{PublishReview: true}},
		detection: &core.DetectionResult{Applicable: false, Reason: "no active bridge session"},
	}
	m := NewModelWithReviewProviders([]core.ReviewFile{reviewFile("demo.go", "package main")}, nil, nil, core.ReviewRequest{}, nil, core.ReviewContext{}, []ports.ReviewProviderClient{fakeProviderAsPort{provider}})

	cmd := m.Init()
	require.NotNil(t, cmd)
	updated, expire := m.Update(cmd())
	m = updated.(Model)

	require.NotNil(t, expire)
	require.Empty(t, m.providerInfos)
	require.Contains(t, stripANSI(m.View().Content), "pi-coding-agent unavailable: no active bridge session")
}

func publishTestRange() core.ReviewLineRange {
	return core.ReviewLineRange{Start: core.ReviewLineRef{NewLineNumber: 1, Kind: core.LineKindAdded}, End: core.ReviewLineRef{NewLineNumber: 1, Kind: core.LineKindAdded}}
}

func TestPublishReviewUsesClientMatchedByProviderID(t *testing.T) {
	firstClient := &fakeReviewProvider{info: core.ReviewProviderInfo{ID: "first", Capabilities: core.ReviewProviderCapabilities{PublishReview: true}}}
	selectedClient := &fakeReviewProvider{info: core.ReviewProviderInfo{ID: "selected", Capabilities: core.ReviewProviderCapabilities{PublishReview: true}}}
	m := NewModelWithReviewProviders([]core.ReviewFile{reviewFile("demo.go", "package main")}, nil, nil, core.ReviewRequest{}, nil, core.ReviewContext{}, nil)
	m.reviewProviders = []ports.ReviewProviderClient{fakeProviderAsPort{firstClient}, fakeProviderAsPort{selectedClient}}
	m.providerInfos = []core.ReviewProviderInfo{selectedClient.info}
	m.providerInfoByClient = map[ports.ReviewProviderClient]core.ReviewProviderInfo{m.reviewProviders[0]: firstClient.info, m.reviewProviders[1]: selectedClient.info}
	m, _ = m.openPublishReview()
	updated, cmd := m.publishSelectedProviders()
	m = updated
	require.NotNil(t, cmd)
	_ = cmd().(publishReviewCompletedMsg)
	require.Empty(t, firstClient.requests)
	require.Len(t, selectedClient.requests, 1)
}

func TestPublishReviewUnsupportedDecisionWarning(t *testing.T) {
	provider := &fakeReviewProvider{info: core.ReviewProviderInfo{ID: "pi-coding-agent", Capabilities: core.ReviewProviderCapabilities{PublishReview: true, Decisions: []core.ReviewDecision{core.ReviewDecisionComment}}}}
	m := NewModelWithReviewProviders([]core.ReviewFile{reviewFile("demo.go", "package main")}, nil, nil, core.ReviewRequest{}, nil, core.ReviewContext{}, nil)
	m.reviewProviders = []ports.ReviewProviderClient{fakeProviderAsPort{provider}}
	m.providerInfos = []core.ReviewProviderInfo{provider.info}
	m.reviewDraft.SetDecision(core.ReviewDecisionApprove)
	m, _ = m.openPublishReview()
	updated, cmd := m.publishSelectedProviders()
	m = updated
	require.Nil(t, cmd)
	require.Contains(t, m.publish.message, "Decision unsupported")
	require.Empty(t, provider.requests)

	updated, cmd = m.publishSelectedProviders()
	m = updated
	require.NotNil(t, cmd)
	_ = cmd().(publishReviewCompletedMsg)
	require.Len(t, provider.requests, 1)
	require.Empty(t, provider.requests[0].Draft.Decision)
}

type fakeReviewProvider struct {
	info       core.ReviewProviderInfo
	detection  *core.DetectionResult
	threads    []core.RemoteReviewThread
	publishErr error
	requests   []core.PublishReviewRequest
}

type fakeProviderAsPort struct{ *fakeReviewProvider }

func (f fakeProviderAsPort) Initialize(context.Context) (core.ReviewProviderInfo, error) {
	return f.info, nil
}
func (f fakeProviderAsPort) DetectContext(context.Context, core.ReviewContext) (core.DetectionResult, error) {
	if f.detection != nil {
		return *f.detection, nil
	}
	return core.DetectionResult{Applicable: true}, nil
}
func (f fakeProviderAsPort) LoadRemoteThreads(context.Context, core.ReviewContext) ([]core.RemoteReviewThread, error) {
	return f.threads, nil
}
func (f fakeProviderAsPort) PublishReview(_ context.Context, request core.PublishReviewRequest) (core.PublishReviewResult, error) {
	f.requests = append(f.requests, request)
	if f.publishErr != nil {
		return core.PublishReviewResult{}, f.publishErr
	}
	return core.PublishReviewResult{ProviderID: request.ProviderID}, nil
}
func (f fakeProviderAsPort) Close() error { return nil }

var _ tea.Cmd
