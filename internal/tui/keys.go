package tui

import "charm.land/bubbles/v2/key"

type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	First    key.Binding
	Last     key.Binding
	PageDown key.Binding
	PageUp   key.Binding
	Focus    key.Binding
	Send     key.Binding
	Yank     key.Binding
	Edit     key.Binding
	Headers  key.Binding
	Help     key.Binding
	Quit     key.Binding
	Confirm  key.Binding
	Cancel   key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("h", "prev pane"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("l", "next pane"),
		),
		First: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("gg", "top"),
		),
		Last: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "bottom"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("ctrl+f"),
			key.WithHelp("^f", "page down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("ctrl+b"),
			key.WithHelp("^b", "page up"),
		),
		Focus: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "pane"),
		),
		Send: key.NewBinding(
			key.WithKeys("enter", "r"),
			key.WithHelp("enter", "send"),
		),
		Yank: key.NewBinding(
			key.WithKeys("y", "Y"),
			key.WithHelp("y", "copy body"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit var"),
		),
		Headers: key.NewBinding(
			key.WithKeys("H"),
			key.WithHelp("H", "headers"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "save"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Send, k.Yank, k.Focus, k.Edit, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Focus},
		{k.Send, k.Yank, k.Edit, k.Headers},
		{k.Left, k.Right, k.First, k.Last},
		{k.PageDown, k.PageUp},
		{k.Help, k.Quit},
	}
}
