package ui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type keyMap struct {
	Up          key.Binding
	Down        key.Binding
	Expand      key.Binding
	Filter      key.Binding
	ClearFilter key.Binding
	Sort        key.Binding
	Pause       key.Binding
	Refresh     key.Binding
	Help        key.Binding
	Quit        key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("up/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("down/j", "move down"),
		),
		Expand: key.NewBinding(
			key.WithKeys("enter", " "),
			key.WithHelp("enter", "expand"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		ClearFilter: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "clear filter"),
		),
		Sort: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "sort"),
		),
		Pause: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "pause"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Expand, k.Filter, k.Sort, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Expand},
		{k.Filter, k.ClearFilter, k.Sort},
		{k.Pause, k.Refresh, k.Help, k.Quit},
	}
}

func keyMatches(msg tea.KeyMsg, binding key.Binding) bool {
	return key.Matches(msg, binding)
}
