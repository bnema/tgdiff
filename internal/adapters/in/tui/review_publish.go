package tui

import (
	"context"
	"fmt"
	"maps"
	"strings"

	tea "charm.land/bubbletea/v2"

	"ero/internal/core"
)

type publishState struct {
	active                 bool
	selected               map[string]bool
	publishing             bool
	message                string
	unsupportedDecision    map[string]core.ReviewDecision
	confirmWithoutDecision map[string]bool
}

type publishReviewCompletedMsg struct {
	results []core.PublishReviewResult
	errors  map[string]error
}

func (m Model) openPublishReview() (Model, tea.Cmd) {
	if len(m.providerInfos) == 0 {
		m.setCopyFeedback("No review providers available")
		return m, nil
	}
	selected := make(map[string]bool, len(m.providerInfos))
	for i, info := range m.providerInfos {
		if i == 0 && info.Capabilities.PublishReview {
			selected[info.ID] = true
		}
	}
	m.publish = publishState{
		active:                 true,
		selected:               selected,
		unsupportedDecision:    map[string]core.ReviewDecision{},
		confirmWithoutDecision: map[string]bool{},
		message:                "Select provider and press enter to publish",
	}
	return m, nil
}

func (m Model) updatePublishReview(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.publish = publishState{}
		return m, nil
	case "enter":
		return m.publishSelectedProviders()
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		idx := int(msg.String()[0] - '1')
		if idx >= 0 && idx < len(m.providerInfos) {
			id := m.providerInfos[idx].ID
			if m.publish.selected == nil {
				m.publish.selected = map[string]bool{}
			}
			m.publish.selected[id] = !m.publish.selected[id]
		}
	}
	return m, nil
}

func (m Model) publishSelectedProviders() (Model, tea.Cmd) {
	if m.publish.publishing {
		return m, nil
	}
	if m.reviewDraft == nil {
		m.reviewDraft = core.NewReviewDraft()
	}
	selectedInfos := m.selectedPublishProviderInfos()
	if len(selectedInfos) == 0 {
		m.publish.message = "Select at least one provider"
		return m, nil
	}
	decision := m.reviewDraft.Decision()
	m.publish.unsupportedDecision = map[string]core.ReviewDecision{}
	for _, info := range selectedInfos {
		if !info.Capabilities.PublishReview {
			m.publish.message = fmt.Sprintf("%s does not support publishing", providerDisplayLabel(info))
			return m, nil
		}
		if !info.Capabilities.SupportsDecision(decision) && !m.publish.confirmWithoutDecision[info.ID] {
			m.publish.unsupportedDecision[info.ID] = decision
		}
	}
	if len(m.publish.unsupportedDecision) > 0 {
		m.publish.message = "Decision unsupported by selected provider; press enter again to publish without decision"
		if m.publish.confirmWithoutDecision == nil {
			m.publish.confirmWithoutDecision = map[string]bool{}
		}
		for id := range m.publish.unsupportedDecision {
			m.publish.confirmWithoutDecision[id] = true
		}
		return m, nil
	}
	providers := append([]core.ReviewProviderInfo(nil), selectedInfos...)
	clients := append([]providerClientWithInfo(nil), m.providerClientsFor(providers)...)
	ctx := m.reviewContext
	draft := m.reviewDraft.Snapshot()
	confirmWithoutDecision := copyBoolMap(m.publish.confirmWithoutDecision)
	m.publish.publishing = true
	m.publish.message = "Publishing review…"
	return m, func() tea.Msg {
		results := make([]core.PublishReviewResult, 0, len(clients))
		errs := map[string]error{}
		for _, item := range clients {
			payload := draft
			if confirmWithoutDecision[item.info.ID] {
				payload.Decision = ""
			}
			result, err := item.client.PublishReview(context.Background(), core.PublishReviewRequest{ProviderID: item.info.ID, Context: ctx, Draft: payload})
			if err != nil {
				errs[item.info.ID] = err
				continue
			}
			results = append(results, result)
		}
		return publishReviewCompletedMsg{results: results, errors: errs}
	}
}

func (m Model) handlePublishReviewCompleted(msg publishReviewCompletedMsg) (Model, tea.Cmd) {
	m.publish.publishing = false
	if len(msg.results) > 0 {
		m.applyPublishedRefs(msg.results)
	}
	ambiguous := ambiguousProviderIDs(msg.results)
	if len(msg.errors) > 0 || len(ambiguous) > 0 {
		parts := make([]string, 0, len(msg.errors)+len(ambiguous)+1)
		if len(msg.results) > 0 {
			parts = append(parts, fmt.Sprintf("published to %d %s", len(msg.results), pluralize("provider", len(msg.results))))
		}
		for id, err := range msg.errors {
			parts = append(parts, id+": "+err.Error())
		}
		if len(ambiguous) > 0 {
			parts = append(parts, "ambiguous: "+strings.Join(ambiguous, ", "))
		}
		m.publish.message = "Partial publish: " + strings.Join(parts, "; ")
		m.setCopyFeedback(m.publish.message)
		m.syncReviewViewport()
		return m, m.expireCopyFeedbackCmd()
	}
	m.publish = publishState{}
	m.setCopyFeedback(fmt.Sprintf("Published review to %d %s", len(msg.results), pluralize("provider", len(msg.results))))
	m.syncReviewViewport()
	return m, m.expireCopyFeedbackCmd()
}

func ambiguousProviderIDs(results []core.PublishReviewResult) []string {
	ids := make([]string, 0)
	for _, result := range results {
		if result.Ambiguous {
			ids = append(ids, result.ProviderID)
		}
	}
	return ids
}

func (m Model) selectedPublishProviderInfos() []core.ReviewProviderInfo {
	infos := make([]core.ReviewProviderInfo, 0)
	for _, info := range m.providerInfos {
		if m.publish.selected[info.ID] {
			infos = append(infos, info)
		}
	}
	return infos
}

type providerClientWithInfo struct {
	info   core.ReviewProviderInfo
	client reviewProviderClient
}

type reviewProviderClient interface {
	PublishReview(context.Context, core.PublishReviewRequest) (core.PublishReviewResult, error)
}

func (m Model) providerClientsFor(infos []core.ReviewProviderInfo) []providerClientWithInfo {
	selected := make(map[string]core.ReviewProviderInfo, len(infos))
	for _, info := range infos {
		selected[info.ID] = info
	}
	result := make([]providerClientWithInfo, 0, len(infos))
	for i, client := range m.reviewProviders {
		providerInfo, ok := m.providerInfoByClient[client]
		if !ok && i < len(m.providerInfos) {
			providerInfo = m.providerInfos[i]
			ok = true
		}
		if !ok {
			continue
		}
		if info, ok := selected[providerInfo.ID]; ok {
			result = append(result, providerClientWithInfo{info: info, client: client})
		}
	}
	return result
}

func (m *Model) applyPublishedRefs(results []core.PublishReviewResult) {
	if m.reviewDraft == nil {
		return
	}
	for _, result := range results {
		m.reviewDraft.ApplyPublishedRefs(result.ProviderID, result.PublishedRefs)
	}
}

func copyBoolMap(input map[string]bool) map[string]bool {
	out := make(map[string]bool, len(input))
	maps.Copy(out, input)
	return out
}

func (m Model) renderPublishOverlay(content string) string {
	var b strings.Builder
	b.WriteString(content)
	b.WriteString("\n\nPublish review\n")
	if m.publish.message != "" {
		b.WriteString(m.publish.message)
		b.WriteString("\n")
	}
	if m.publish.publishing {
		b.WriteString("Publishing…\n")
	}
	for i, info := range m.providerInfos {
		mark := " "
		if m.publish.selected[info.ID] {
			mark = "x"
		}
		fmt.Fprintf(&b, "%d. [%s] %s", i+1, mark, providerDisplayLabel(info))
		if decision, ok := m.publish.unsupportedDecision[info.ID]; ok {
			fmt.Fprintf(&b, " (does not support %s)", decision)
		}
		b.WriteString("\n")
	}
	b.WriteString("Enter publish · 1-9 toggle · Esc cancel")
	return b.String()
}

func providerDisplayLabel(info core.ReviewProviderInfo) string {
	if info.Label != "" {
		return info.Label
	}
	if info.Name != "" {
		return info.Name
	}
	return info.ID
}
