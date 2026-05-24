package tui

import tea "charm.land/bubbletea/v2"

type Runner struct{}

func NewRunner() Runner {
	return Runner{}
}

func (r Runner) Run(model tea.Model) error {
	_, err := tea.NewProgram(model).Run()
	return err
}
