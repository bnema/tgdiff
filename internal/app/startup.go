package app

import (
	"errors"
	"fmt"

	"ero/internal/core"
	"ero/internal/ports"
)

func resolveStartupRequest(request core.ReviewRequest, reader ports.StartupStateReader[core.StartupState], prompt startupPrompt, isInteractive func() bool) (core.ReviewRequest, error) {
	if reader == nil {
		return request, nil
	}

	state, err := reader.ReadStartupState(request.RepoPath)
	if err != nil {
		return request, fmt.Errorf("read startup state: %w", err)
	}

	decision := core.ResolveStartupDecision(state)
	switch decision.Kind {
	case core.StartupDecisionUseMode:
		request.DiffMode = decision.DiffMode
		return request, nil
	case core.StartupDecisionPromptLocalChanges:
		if isInteractive == nil || !isInteractive() {
			return request, fmt.Errorf("mixed staged and worktree/untracked changes detected; choose explicitly: ero staged, ero working, or ero local")
		}
		if prompt == nil {
			return request, fmt.Errorf("startup prompt is nil")
		}
		mode, err := prompt.PromptLocalChangeMode()
		if err != nil {
			return request, err
		}
		request.DiffMode = mode
		return request, nil
	case core.StartupDecisionNoReviewableChanges:
		return request, errors.New(decision.Message)
	default:
		return request, fmt.Errorf("unsupported startup decision %q", decision.Kind)
	}
}
