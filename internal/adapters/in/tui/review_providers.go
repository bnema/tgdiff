package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/bnema/zerowrap"

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
	ctx := m.ctx
	return func() tea.Msg {
		log := zerowrap.FromCtx(ctx)
		log.Info().Int("provider_count", len(providers)).Msg("loading review providers")
		var infos []core.ReviewProviderInfo
		var threads []core.RemoteReviewThread
		clients := map[ports.ReviewProviderClient]core.ReviewProviderInfo{}
		var errs []string
		for _, provider := range providers {
			info, err := provider.Initialize(ctx)
			if err != nil {
				log.Error().Err(err).Msg("review provider init failed")
				errs = append(errs, fmt.Sprintf("provider init failed: %v", err))
				continue
			}
			providerLog := log.WithField("provider_id", info.ID).WithField("provider_label", providerDisplayLabel(info))
			providerLog.Info().Msg("review provider initialized")
			detection, err := provider.DetectContext(ctx, reviewContext)
			if err != nil {
				providerLog.Error().Err(err).Msg("review provider detect failed")
				errs = append(errs, fmt.Sprintf("%s detect failed: %v", providerDisplayLabel(info), err))
				continue
			}
			providerLog.Info().Bool("applicable", detection.Applicable).Str("reason", detection.Reason).Msg("review provider detection completed")
			if !detection.Applicable {
				if detection.Reason != "" {
					errs = append(errs, fmt.Sprintf("%s unavailable: %s", providerDisplayLabel(info), detection.Reason))
				}
				continue
			}
			infos = append(infos, info)
			clients[provider] = info
			if info.Capabilities.LoadRemoteComments {
				loaded, err := provider.LoadRemoteThreads(ctx, reviewContext)
				if err != nil {
					providerLog.Error().Err(err).Msg("review provider remote comments failed")
					errs = append(errs, fmt.Sprintf("%s remote comments failed: %v", providerDisplayLabel(info), err))
					continue
				}
				providerLog.Info().Int("thread_count", len(loaded)).Msg("review provider remote comments loaded")
				for _, thread := range loaded {
					if thread.ProviderID == "" {
						thread.ProviderID = info.ID
					}
					threads = append(threads, thread)
				}
			}
		}
		log.Info().Int("available_provider_count", len(infos)).Int("remote_thread_count", len(threads)).Int("error_count", len(errs)).Msg("review providers loaded")
		return reviewProvidersLoadedMsg{infos: infos, threads: threads, clients: clients, errs: errs}
	}
}
