package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"ero/internal/core"
)

type StartupPrompt struct{}

func NewStartupPrompt() StartupPrompt {
	return StartupPrompt{}
}

func (p StartupPrompt) PromptLocalChangeMode() (core.DiffMode, error) {
	model := newStartupPromptModel()
	result, err := tea.NewProgram(model).Run()
	if err != nil {
		return "", err
	}
	prompt, ok := result.(startupPromptModel)
	if !ok {
		return "", fmt.Errorf("unexpected startup prompt result %T", result)
	}
	if prompt.cancelled {
		return "", fmt.Errorf("startup mode selection cancelled")
	}
	return prompt.selectedMode(), nil
}

type startupPromptOption struct {
	mode        core.DiffMode
	key         string
	label       string
	description string
}

type startupPromptModel struct {
	options   []startupPromptOption
	selected  int
	cancelled bool
}

func newStartupPromptModel() startupPromptModel {
	return startupPromptModel{
		options: []startupPromptOption{
			{mode: core.DiffModeStaged, key: "s", label: "Staged changes", description: "What will be included in your next commit"},
			{mode: core.DiffModeWorking, key: "u", label: "Unstaged/untracked changes", description: "Worktree changes not yet staged, including new files"},
			{mode: core.DiffModeLocal, key: "a", label: "All local changes", description: "Staged + worktree/untracked changes together"},
		},
	}
}

func (m startupPromptModel) Init() tea.Cmd { return nil }

func (m startupPromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	keyString := key.String()
	if selected, ok := m.shortcutIndex(keyString); ok {
		m.selected = selected
		return m, tea.Quit
	}

	switch keyString {
	case "q", "esc", "ctrl+c":
		m.cancelled = true
		return m, tea.Quit
	case "up", "k":
		m.selected = max(m.selected-1, 0)
	case "down", "j":
		m.selected = min(m.selected+1, len(m.options)-1)
	case "enter":
		return m, tea.Quit
	}
	return m, nil
}

func (m startupPromptModel) shortcutIndex(key string) (int, bool) {
	for i, option := range m.options {
		if option.key == key {
			return i, true
		}
	}
	return 0, false
}

func (m startupPromptModel) View() tea.View {
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")).Render("Mixed local changes detected")
	subtitle := mutedStyle.Render("Choose the diff scope to review")
	lines := []string{title, subtitle, ""}
	for i, option := range m.options {
		cursor := "  "
		labelStyle := lipgloss.NewStyle().Bold(false)
		if i == m.selected {
			cursor = "▸ "
			labelStyle = labelStyle.Bold(true).Foreground(lipgloss.Color("86"))
		}
		shortcut := mutedStyle.Render("(" + option.key + ")")
		lines = append(lines, cursor+labelStyle.Render(option.label)+" "+shortcut)
		lines = append(lines, "    "+mutedStyle.Render(option.description))
	}
	lines = append(lines, "", mutedStyle.Render("↑/↓ move • enter select • q quit"))
	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m startupPromptModel) selectedMode() core.DiffMode {
	if m.selected < 0 || m.selected >= len(m.options) {
		return core.DiffModeStaged
	}
	return m.options[m.selected].mode
}
