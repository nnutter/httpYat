package tui

import "charm.land/lipgloss/v2"

type styles struct {
	header     lipgloss.Style
	title      lipgloss.Style
	muted      lipgloss.Style
	border     lipgloss.Style
	active     lipgloss.Style
	cursor     lipgloss.Style
	statusOK   lipgloss.Style
	statusFail lipgloss.Style
	method     map[string]lipgloss.Style
}

func newStyles() styles {
	accent := lipgloss.Color("#7dd3fc")
	muted := lipgloss.Color("#64748b")
	ok := lipgloss.Color("#4ade80")
	fail := lipgloss.Color("#fb7185")
	return styles{
		header: lipgloss.NewStyle().Bold(true).Foreground(accent),
		title:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#e2e8f0")),
		muted:  lipgloss.NewStyle().Foreground(muted),
		border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(muted).
			Padding(0, 1),
		active: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1),
		cursor:     lipgloss.NewStyle().Foreground(accent).Bold(true),
		statusOK:   lipgloss.NewStyle().Foreground(ok).Bold(true),
		statusFail: lipgloss.NewStyle().Foreground(fail).Bold(true),
		method: map[string]lipgloss.Style{
			"GET":    lipgloss.NewStyle().Foreground(ok).Bold(true),
			"POST":   lipgloss.NewStyle().Foreground(lipgloss.Color("#facc15")).Bold(true),
			"PUT":    lipgloss.NewStyle().Foreground(accent).Bold(true),
			"PATCH":  lipgloss.NewStyle().Foreground(lipgloss.Color("#c084fc")).Bold(true),
			"DELETE": lipgloss.NewStyle().Foreground(fail).Bold(true),
		},
	}
}

func (s styles) methodStyle(method string) lipgloss.Style {
	if st, ok := s.method[method]; ok {
		return st
	}
	return lipgloss.NewStyle().Bold(true)
}
