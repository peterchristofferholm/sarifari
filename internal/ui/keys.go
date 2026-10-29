package ui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Move, Enter, Back, Group, Sev, Filter, Edit, Step, Help, Quit key.Binding
}

func newKeys() keyMap {
	return keyMap{
		Move:   key.NewBinding(key.WithKeys("j", "k", "up", "down"), key.WithHelp("j/k", "move")),
		Enter:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
		Back:   key.NewBinding(key.WithKeys("esc", "backspace", "-"), key.WithHelp("esc/-", "back")),
		Group:  key.NewBinding(key.WithKeys("tab", "shift+tab"), key.WithHelp("tab", "regroup")),
		Sev:    key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "severity")),
		Filter: key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Edit:   key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "open in editor")),
		Step:   key.NewBinding(key.WithKeys("n", "p"), key.WithHelp("n/p", "next/prev")),
		Help:   key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Quit:   key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

// screenHelp adapts per-screen bindings to help.KeyMap.
type screenHelp []key.Binding

func (h screenHelp) ShortHelp() []key.Binding  { return h }
func (h screenHelp) FullHelp() [][]key.Binding { return [][]key.Binding{h} }

func (m *Model) helpFor() screenHelp {
	k := m.keys
	switch m.screen {
	case screenFindings:
		return screenHelp{k.Move, k.Enter, k.Back, k.Filter, k.Edit, k.Help, k.Quit}
	case screenDetail:
		return screenHelp{k.Move, k.Step, k.Back, k.Edit, k.Help, k.Quit}
	}
	return screenHelp{k.Move, k.Enter, k.Group, k.Sev, k.Filter, k.Help, k.Quit}
}
