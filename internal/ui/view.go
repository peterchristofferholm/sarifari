package ui

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/peterchristofferholm/sarifari/internal/model"
	"github.com/peterchristofferholm/sarifari/internal/source"
)

type groupItem model.Group

func (g groupItem) FilterValue() string {
	return g.Label + " " + g.Key + " " + g.Description + " " + model.Group(g).Subject()
}

type findingItem struct {
	f      *model.Finding
	byFile bool
}

func (i findingItem) FilterValue() string {
	return i.f.Title() + " " + i.f.Path + " " + i.f.Subject + " " + i.f.Message
}

// rowBar is the left selection marker, GitHub's blue accent on the active row.
func rowBar(sel bool) string {
	if sel {
		return lipgloss.NewStyle().Foreground(cAccent).Render("▌")
	}
	return " "
}

func writeRow(w io.Writer, width int, sel bool, title, right, meta string) {
	bar := rowBar(sel)
	cw := max(1, width-3)
	div := lipgloss.NewStyle().Foreground(cBorder).Render(strings.Repeat("─", max(1, width)))
	fmt.Fprintf(w, "%s %s\n%s %s\n%s", bar, joinLR(title, right, cw), bar, ansi.Truncate(meta, cw, "…"), div)
}

type summaryDelegate struct{}

func (summaryDelegate) Height() int                             { return 3 }
func (summaryDelegate) Spacing() int                            { return 0 }
func (summaryDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (summaryDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	g := item.(groupItem)
	sel := index == m.Index()
	label := boldStyle.Render(g.Label)
	if sel {
		label = accentBold.Render(g.Label)
	}
	meta := join(sevLabel(g.Severity), model.Group(g).Subject(), g.Description)
	if g.Description == "" && g.Key != g.Label {
		meta = join(meta, g.Key)
	}
	writeRow(w, m.Width(), sel, sevIcon(g.Severity)+" "+label, mutedStyle.Render(fmt.Sprint(len(g.Findings))), mutedStyle.Render("  "+meta))
}

type findingDelegate struct{}

func (findingDelegate) Height() int                             { return 3 }
func (findingDelegate) Spacing() int                            { return 0 }
func (findingDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (findingDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	it := item.(findingItem)
	f := it.f
	sel := index == m.Index()
	title := boldStyle.Render(f.Title())
	if sel {
		title = accentBold.Render(f.Title())
	}
	where := fmt.Sprintf("%s:%d", f.Path, f.StartLine)
	switch {
	case !f.HasLocation():
		where = f.Path
	case it.byFile:
		where = fmt.Sprintf("line %d", f.StartLine)
	}
	meta := join(where, f.Subject, f.RuleID)
	writeRow(w, m.Width(), sel, sevIcon(f.Severity)+" "+title, badge(f.Severity), mutedStyle.Render("  "+meta))
}

func (m Model) View() string {
	if m.width == 0 {
		return "loading…"
	}
	return m.headerView() + "\n" + m.bodyView() + "\n" + m.footerView()
}

func (m Model) headerView() string {
	g := m.group
	switch m.screen {
	case screenFindings:
		return boldStyle.Render(g.Label) + mutedStyle.Render(fmt.Sprintf(" · %d findings", len(g.Findings))) +
			"\n" + mutedStyle.Render("‹ esc   grouped by "+m.groupBy.String())
	case screenDetail:
		pos := fmt.Sprintf("%d/%d", m.fin.Index()+1, len(m.fin.VisibleItems()))
		return mutedStyle.Render("‹ "+g.Label+"  "+pos) + "\n" + mutedStyle.Render(strings.Repeat("─", m.width))
	}
	tool := m.tool
	if tool == "" {
		tool = "sarifari"
	}
	return boldStyle.Render(tool) + mutedStyle.Render(fmt.Sprintf(" · %d findings", m.total())) + "\n" + m.tabsView()
}

// tabsView renders the severity tabs with counts plus the active grouping.
func (m Model) tabsView() string {
	counts := map[string]int{"all": len(m.findings)}
	for _, f := range m.findings {
		counts[f.Severity]++
	}
	var tabs []string
	for i, s := range SeverityFilters {
		name := "All"
		if s != "all" {
			name = sevLabel(s)
		}
		cnt := mutedStyle.Render(fmt.Sprint(counts[s]))
		if i == m.sevIdx {
			tabs = append(tabs, accentBold.Underline(true).Render(name)+" "+cnt)
		} else {
			tabs = append(tabs, mutedStyle.Render(name)+" "+cnt)
		}
	}
	return joinLR(strings.Join(tabs, "   "), mutedStyle.Render("group: "+m.groupBy.String()), m.width)
}

func (m Model) bodyView() string {
	switch m.screen {
	case screenDetail:
		return m.vp.View()
	case screenFindings:
		return boxStyle.Width(m.width - 2).Render(m.fin.View())
	}
	if len(m.groups) == 0 {
		return boxStyle.Width(m.width - 2).Height(max(1, m.sum.Height())).Render(mutedStyle.Render("  No findings"))
	}
	return boxStyle.Width(m.width - 2).Render(m.sum.View())
}

func (m Model) footerView() string {
	status := statusStyle.Render(m.status)
	if m.status == "" && m.screen == screenSummary && m.groupBy == model.ByRule {
		if it, ok := m.sum.SelectedItem().(groupItem); ok {
			status = mutedStyle.Render(it.Key)
		}
	}
	return status + "\n" + m.help.View(m.helpFor())
}

// refreshDetail re-renders Detail content in place (resize, source arrival).
func (m *Model) refreshDetail() {
	if m.screen != screenDetail || m.detail == nil {
		return
	}
	y := m.vp.YOffset
	m.renderDetail()
	m.vp.SetYOffset(y)
}

func (m *Model) renderDetail() {
	f := m.detail
	w := max(30, m.width)
	var b strings.Builder

	title := lipgloss.NewStyle().Bold(true).Foreground(cFg).Width(max(10, w-12)).Render(f.Title())
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, title, " ", badge(f.Severity)) + "\n")
	b.WriteString(mutedStyle.Render(join(location(f), f.Subject, f.Tool)) + "\n\n")

	b.WriteString(m.codeBox(f, w) + "\n\n")

	b.WriteString(boldStyle.Render("Rule") + "\n" + mutedStyle.Render(strings.Repeat("─", w)) + "\n")
	b.WriteString(mutedStyle.Render("Tool ") + f.Tool + mutedStyle.Render("   Rule ID ") + f.RuleID + "\n")
	if help := m.ruleHelp(f, w); help != "" {
		b.WriteString(help + "\n")
	}
	if f.Rule != nil && f.Rule.HelpURI != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(cAccent).Underline(true).Render(f.Rule.HelpURI) + "\n")
	}
	m.vp.SetContent(b.String())
}

// join joins the non-empty parts with a middle dot.
func join(parts ...string) string {
	return strings.Join(slices.DeleteFunc(parts, func(s string) bool { return s == "" }), " · ")
}

func location(f *model.Finding) string {
	if !f.HasLocation() {
		return f.Path
	}
	return fmt.Sprintf("%s:%d:%d", f.Path, f.StartLine, f.StartCol)
}

// codeBox draws the bordered source block: file header, numbered lines with the
// match tinted, and the alert message as an annotation under the match.
func (m *Model) codeBox(f *model.Finding, w int) string {
	inner := w - 2
	sep := lipgloss.NewStyle().Foreground(cBorder).Render(strings.Repeat("─", inner))
	var rows []string
	rows = append(rows, boldStyle.Render(ansi.Truncate(location(f), inner, "…")), sep)

	annotate := func(closing bool) {
		fg, _ := sevColors(f.Severity)
		bar := lipgloss.NewStyle().Foreground(fg).Render("▌ ")
		label := lipgloss.NewStyle().Foreground(fg).Bold(true).Render(sevLabel(f.Severity))
		msg := lipgloss.NewStyle().Width(max(10, inner-2)).Render(label + " " + f.Message)
		rows = append(rows, sep)
		for _, l := range strings.Split(msg, "\n") {
			rows = append(rows, bar+l)
		}
		if closing {
			rows = append(rows, lipgloss.NewStyle().Foreground(cBorder).Render(strings.Repeat("─", inner)))
		}
	}
	// fallback shows the SARIF snippet (or nothing) plus the annotation.
	fallback := func(note string) string {
		rows = append(rows, mutedStyle.Render(note))
		if s := strings.TrimRight(f.Snippet, "\n"); s != "" {
			for _, l := range strings.Split(s, "\n") {
				rows = append(rows, ansi.Truncate(strings.ReplaceAll(l, "\t", "    "), inner, "…"))
			}
		}
		annotate(false)
		return frame(rows, inner)
	}

	if !f.HasLocation() {
		return fallback("no location")
	}
	src, ok := m.sources[f.Path]
	if !ok {
		rows = append(rows, mutedStyle.Render("loading…"))
		return frame(rows, inner)
	}
	if src.err != nil {
		return fallback("source unavailable: " + src.err.Error())
	}
	lines := src.lines
	end := max(f.EndLine, f.StartLine)
	if f.StartLine < 1 || f.StartLine > len(lines) {
		return fallback("file changed: line out of range")
	}
	from, to := max(1, f.StartLine-5), min(len(lines), end+5)

	hkey := fmt.Sprintf("%s:%d:%d", f.Path, from, to)
	hl, ok := m.hl[hkey]
	if !ok {
		hl = source.Highlight(f.Path, strings.Join(lines[from-1:to], "\n"), m.opts.Style)
		m.hl[hkey] = hl
	}
	out := strings.Split(hl, "\n")
	gw := len(fmt.Sprint(to))
	for i := from; i <= to; i++ {
		code := ""
		if i-from < len(out) {
			code = out[i-from]
		}
		gutter := mutedStyle.Render(fmt.Sprintf("%*d ", gw, i))
		line := ansi.Truncate(gutter+code, inner, "…")
		if i >= f.StartLine && i <= end {
			line = tint(line, inner, cMarkBg)
		}
		rows = append(rows, line)
		if i == end {
			annotate(i < to)
		}
	}
	return frame(rows, inner)
}

func frame(rows []string, inner int) string {
	return boxStyle.Width(inner).Render(strings.Join(rows, "\n"))
}

func (m *Model) ruleHelp(f *model.Finding, w int) string {
	r := f.Rule
	if r == nil {
		return ""
	}
	if md := strings.TrimSpace(r.Help.Markdown); md != "" {
		style := "dark"
		if !m.dark {
			style = "light"
		}
		if tr, err := glamour.NewTermRenderer(glamour.WithStandardStyle(style), glamour.WithWordWrap(w)); err == nil {
			if out, err := tr.Render(md); err == nil {
				return strings.Trim(out, "\n")
			}
		}
	}
	for _, t := range []string{r.Help.Text, r.FullDescription.Text, r.ShortDescription.Text} {
		if t = strings.TrimSpace(t); t != "" {
			return lipgloss.NewStyle().Width(w).Render(t)
		}
	}
	return ""
}
