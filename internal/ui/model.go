package ui

import (
	"slices"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/peterchristofferholm/sarifari/internal/model"
)

type screen int

const (
	screenSummary screen = iota
	screenFindings
	screenDetail
)

// SeverityFilters are the values of the severity filter, in cycling order.
var SeverityFilters = []string{"all", "error", "warning", "note"}

const headerHeight = 2

type Options struct {
	Roots    []string // directories SARIF paths are resolved against, in order
	Style    string   // chroma style
	GroupBy  model.GroupBy
	Severity string // all|error|warning|note
	Notice   string // initial status line, e.g. a loader warning
}

type Model struct {
	opts     Options
	findings []*model.Finding
	tool     string
	groupBy  model.GroupBy
	sevIdx   int
	groups   []model.Group

	screen   screen
	sum, fin list.Model
	group    model.Group // open in Findings and Detail
	vp       viewport.Model
	detail   *model.Finding

	help          help.Model
	keys          keyMap
	width, height int
	status        string

	sources map[string]sourceFile // by finding path
	hl      map[string]string     // highlighted windows
	dark    bool
}

type sourceFile struct {
	lines []string
	err   error
}

func New(findings []*model.Finding, opts Options) Model {
	dark := lipgloss.HasDarkBackground()
	if opts.Style == "" {
		opts.Style = "github"
		if dark {
			opts.Style = "github-dark"
		}
	}
	m := Model{
		opts: opts, findings: findings, groupBy: opts.GroupBy,
		sevIdx: max(0, slices.Index(SeverityFilters, opts.Severity)),
		help:   help.New(), keys: newKeys(), status: opts.Notice,
		sources: map[string]sourceFile{}, hl: map[string]string{}, dark: dark,
	}
	if len(findings) > 0 {
		m.tool = findings[0].Tool
	}
	m.help.Styles.ShortKey = mutedStyle.Foreground(cFg)
	m.help.Styles.ShortDesc = mutedStyle
	m.help.Styles.ShortSeparator = mutedStyle
	m.help.Styles.Ellipsis = mutedStyle
	m.sum = newList(summaryDelegate{})
	m.fin = newList(findingDelegate{})
	m.vp = viewport.New(0, 0)
	m.setGroups()
	return m
}

func newList(d list.ItemDelegate) list.Model {
	l := list.New(nil, d, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.DisableQuitKeybindings()
	l.FilterInput.Prompt = " / "
	l.KeyMap.ShowFullHelp.SetEnabled(false)
	l.KeyMap.CloseFullHelp.SetEnabled(false)
	return l
}

func (m Model) Init() tea.Cmd { return nil }

func (m *Model) sevFilter() string {
	if m.sevIdx == 0 {
		return ""
	}
	return SeverityFilters[m.sevIdx]
}

// setGroups regroups under the active filter, keeping the cursor on the same
// group key where it still exists.
func (m *Model) setGroups() {
	prev := ""
	if it, ok := m.sum.SelectedItem().(groupItem); ok {
		prev = it.Key
	}
	m.groups = model.GroupFindings(m.findings, m.groupBy, m.sevFilter())
	items := make([]list.Item, len(m.groups))
	sel := 0
	for i, g := range m.groups {
		items[i] = groupItem(g)
		if g.Key == prev {
			sel = i
		}
	}
	m.sum.SetItems(items)
	m.sum.Select(sel)
}

func (m *Model) total() int {
	n := 0
	for _, g := range m.groups {
		n += len(g.Findings)
	}
	return n
}

func (m *Model) activeList() *list.Model {
	if m.screen == screenFindings {
		return &m.fin
	}
	return &m.sum
}

func (m *Model) layout() {
	if m.width == 0 {
		return
	}
	m.help.Width = m.width
	body := max(1, m.height-headerHeight-lipgloss.Height(m.footerView()))
	// lists live inside a 1-cell border
	m.sum.SetSize(max(1, m.width-2), max(1, body-2))
	m.fin.SetSize(max(1, m.width-2), max(1, body-2))
	m.vp.Width, m.vp.Height = m.width, body
}
