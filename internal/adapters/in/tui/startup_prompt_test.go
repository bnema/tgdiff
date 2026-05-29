package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ero/internal/core"
)

func TestStartupPromptModelShortcutKeysSelectModes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  string
		want core.DiffMode
	}{
		{name: "staged shortcut", key: "s", want: core.DiffModeStaged},
		{name: "working shortcut", key: "u", want: core.DiffModeWorking},
		{name: "local shortcut", key: "a", want: core.DiffModeLocal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			updated, cmd := newStartupPromptModel().Update(keyPress(tt.key))
			model := requireStartupPromptModel(t, updated)

			assert.Equal(t, tt.want, model.selectedMode())
			require.NotNil(t, cmd)
		})
	}
}

func TestStartupPromptModelNavigation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		keys []tea.KeyPressMsg
		want core.DiffMode
	}{
		{name: "defaults to staged", want: core.DiffModeStaged},
		{name: "down selects working", keys: []tea.KeyPressMsg{{Code: tea.KeyDown}}, want: core.DiffModeWorking},
		{name: "j selects working", keys: []tea.KeyPressMsg{keyPress("j")}, want: core.DiffModeWorking},
		{name: "up clamps at staged", keys: []tea.KeyPressMsg{{Code: tea.KeyUp}}, want: core.DiffModeStaged},
		{name: "k clamps at staged", keys: []tea.KeyPressMsg{keyPress("k")}, want: core.DiffModeStaged},
		{name: "down clamps at local", keys: []tea.KeyPressMsg{{Code: tea.KeyDown}, {Code: tea.KeyDown}, {Code: tea.KeyDown}}, want: core.DiffModeLocal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			model := tea.Model(newStartupPromptModel())
			for _, key := range tt.keys {
				updated, _ := model.Update(key)
				model = updated
			}
			prompt := requireStartupPromptModel(t, model)
			assert.Equal(t, tt.want, prompt.selectedMode())
		})
	}
}

func TestStartupPromptModelCancelKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  tea.KeyPressMsg
	}{
		{name: "q", key: keyPress("q")},
		{name: "escape", key: tea.KeyPressMsg{Code: tea.KeyEsc}},
		{name: "ctrl c", key: tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			updated, cmd := newStartupPromptModel().Update(tt.key)
			model := requireStartupPromptModel(t, updated)

			assert.True(t, model.cancelled)
			require.NotNil(t, cmd)
		})
	}
}

func TestStartupPromptModelViewShowsOptions(t *testing.T) {
	t.Parallel()

	view := stripANSI(newStartupPromptModel().View().Content)

	assert.Contains(t, view, "Mixed local changes detected")
	assert.Contains(t, view, "Choose the diff scope to review")
	assert.Contains(t, view, "▸ Staged changes (s)")
	assert.Contains(t, view, "Unstaged/untracked changes (u)")
	assert.Contains(t, view, "All local changes (a)")
	assert.Contains(t, view, "↑/↓ move • "+enterKeyLabel()+" select • q quit")
}

func requireStartupPromptModel(t *testing.T, model tea.Model) startupPromptModel {
	t.Helper()
	prompt, ok := model.(startupPromptModel)
	require.True(t, ok)
	return prompt
}
