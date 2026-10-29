package ui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"sarifari/internal/editor"
	"sarifari/internal/model"
	"sarifari/internal/source"
)

type sourceLoadedMsg struct {
	path string
	sourceFile
}

type editorDoneMsg struct{ err error }

func loadSource(roots []string, p string) tea.Cmd {
	return func() tea.Msg {
		abs, err := source.Resolve(roots, p)
		if err != nil {
			return sourceLoadedMsg{p, sourceFile{err: err}}
		}
		lines, err := source.ReadLines(abs)
		return sourceLoadedMsg{p, sourceFile{lines, err}}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.layout()
		m.refreshDetail()
		return m, nil
	case sourceLoadedMsg:
		m.sources[msg.path] = msg.sourceFile
		m.refreshDetail()
		return m, nil
	case editorDoneMsg:
		if msg.err != nil {
			m.status = "editor: " + msg.err.Error()
		}
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m.passToActive(msg)
}

func (m Model) passToActive(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.screen {
	case screenDetail:
		m.vp, cmd = m.vp.Update(msg)
	case screenFindings:
		m.fin, cmd = m.fin.Update(msg)
	default:
		m.sum, cmd = m.sum.Update(msg)
	}
	return m, cmd
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := m.keys
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}
	// While typing a filter, every key belongs to the list.
	if m.screen != screenDetail && m.activeList().SettingFilter() {
		return m.passToActive(msg)
	}
	m.status = ""
	filterApplied := m.screen != screenDetail && m.activeList().FilterState() == list.FilterApplied

	switch {
	case key.Matches(msg, k.Quit):
		return m, tea.Quit
	case key.Matches(msg, k.Help):
		m.help.ShowAll = !m.help.ShowAll
		m.layout()
		return m, nil
	case key.Matches(msg, k.Back):
		if filterApplied && msg.String() == "esc" {
			return m.passToActive(msg) // esc clears the filter first
		}
		return m.back()
	}

	switch m.screen {
	case screenSummary:
		switch {
		case key.Matches(msg, k.Enter):
			if it, ok := m.sum.SelectedItem().(groupItem); ok {
				m.openGroup(model.Group(it))
			}
			return m, nil
		case key.Matches(msg, k.Group):
			m.groupBy = m.groupBy.Cycle(direction(msg, "shift+tab"))
			m.setGroups()
			return m, nil
		case key.Matches(msg, k.Sev):
			m.sevIdx = (m.sevIdx + 1) % len(SeverityFilters)
			m.setGroups()
			return m, nil
		}
	case screenFindings:
		switch {
		case key.Matches(msg, k.Enter):
			if it, ok := m.fin.SelectedItem().(findingItem); ok {
				return m, m.showDetail(it.f)
			}
			return m, nil
		case key.Matches(msg, k.Edit):
			if it, ok := m.fin.SelectedItem().(findingItem); ok {
				return m, m.jump(it.f)
			}
			return m, nil
		}
	case screenDetail:
		switch {
		case key.Matches(msg, k.Step):
			return m, m.step(direction(msg, "p"))
		case key.Matches(msg, k.Edit):
			return m, m.jump(m.detail)
		}
	}
	return m.passToActive(msg)
}

// direction is -1 when msg is the backwards key of a forward/backward pair.
func direction(msg tea.KeyMsg, backward string) int {
	if msg.String() == backward {
		return -1
	}
	return 1
}

func (m Model) back() (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenDetail:
		m.screen = screenFindings
	case screenFindings:
		m.screen = screenSummary
	default:
		return m, tea.Quit
	}
	m.layout()
	return m, nil
}

func (m *Model) openGroup(g model.Group) {
	m.group = g
	items := make([]list.Item, len(g.Findings))
	for i, f := range g.Findings {
		items[i] = findingItem{f: f, byFile: m.groupBy == model.ByFile}
	}
	m.fin.ResetFilter()
	m.fin.SetItems(items)
	m.fin.Select(0)
	m.screen = screenFindings
	m.layout()
}

func (m *Model) showDetail(f *model.Finding) tea.Cmd {
	m.detail = f
	m.screen = screenDetail
	m.layout()
	m.renderDetail()
	m.vp.GotoTop()
	if !f.HasLocation() {
		return nil
	}
	if _, ok := m.sources[f.Path]; ok {
		return nil
	}
	return loadSource(m.opts.Roots, f.Path)
}

// step moves to the next/previous visible finding while staying in Detail.
func (m *Model) step(d int) tea.Cmd {
	items := m.fin.VisibleItems()
	pos := m.fin.Index() + d
	if pos < 0 || pos >= len(items) {
		return nil
	}
	m.fin.Select(pos)
	return m.showDetail(items[pos].(findingItem).f)
}

func (m *Model) jump(f *model.Finding) tea.Cmd {
	if f == nil || !f.HasLocation() {
		m.status = "no location: cannot open in editor"
		return nil
	}
	abs, err := source.Resolve(m.opts.Roots, f.Path)
	if err != nil {
		m.status = err.Error()
		return nil
	}
	cmd, quit, err := editor.Jump(abs, f.StartLine)
	if err != nil {
		m.status = err.Error()
		return nil
	}
	if quit {
		return tea.Quit
	}
	return tea.ExecProcess(cmd, func(err error) tea.Msg { return editorDoneMsg{err} })
}
