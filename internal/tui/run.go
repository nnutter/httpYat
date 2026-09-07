package tui

import tea "charm.land/bubbletea/v2"

// Run starts the interactive client.
func Run(m Model) error {
	_, err := tea.NewProgram(m).Run()
	return err
}
