package tui

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"

	"ero/internal/core"
	"ero/internal/ports"
)

func (m Model) closeReviewProvidersCmd() tea.Cmd {
	providers := append([]ports.ReviewProviderClient(nil), m.reviewProviders...)
	return func() tea.Msg {
		for _, provider := range providers {
			_ = provider.Close()
		}
		return nil
	}
}

func (m Model) loadReviewProvidersCmd() tea.Cmd {
	providers := make([]ports.ReviewProviderClient, len(m.reviewProviders))
	copy(providers, m.reviewProviders)
	reviewContext := m.reviewContext
	return func() tea.Msg {
		var infos []core.ReviewProviderInfo
		var threads []core.RemoteReviewThread
		clients := map[ports.ReviewProviderClient]core.ReviewProviderInfo{}
		var errs []string
		for _, provider := range providers {
			info, err := provider.Initialize(context.Background())
			if err != nil {
				errs = append(errs, fmt.Sprintf("provider init failed: %v", err))
				continue
			}
			detection, err := provider.DetectContext(context.Background(), reviewContext)
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s detect failed: %v", providerDisplayLabel(info), err))
				continue
			}
			if !detection.Applicable {
				if detection.Reason != "" {
					errs = append(errs, fmt.Sprintf("%s unavailable: %s", providerDisplayLabel(info), detection.Reason))
				}
				continue
			}
			infos = append(infos, info)
			clients[provider] = info
			if info.Capabilities.LoadRemoteComments {
				loaded, err := provider.LoadRemoteThreads(context.Background(), reviewContext)
				if err != nil {
					errs = append(errs, fmt.Sprintf("%s remote comments failed: %v", providerDisplayLabel(info), err))
					continue
				}
				for _, thread := range loaded {
					if thread.ProviderID == "" {
						thread.ProviderID = info.ID
					}
					threads = append(threads, thread)
				}
			}
		}
		return reviewProvidersLoadedMsg{infos: infos, threads: threads, clients: clients, errs: errs}
	}
}
