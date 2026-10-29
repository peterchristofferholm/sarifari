package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Palette modelled on GitHub's Primer colors (light / dark).
var (
	cFg     = lipgloss.AdaptiveColor{Light: "#1f2328", Dark: "#e6edf3"}
	cMuted  = lipgloss.AdaptiveColor{Light: "#59636e", Dark: "#8b949e"}
	cBorder = lipgloss.AdaptiveColor{Light: "#d1d9e0", Dark: "#30363d"}
	cAccent = lipgloss.AdaptiveColor{Light: "#0969da", Dark: "#58a6ff"}
	cMarkBg = lipgloss.AdaptiveColor{Light: "#fff8c5", Dark: "#3a2d07"}

	cRed    = lipgloss.AdaptiveColor{Light: "#d1242f", Dark: "#f85149"}
	cYellow = lipgloss.AdaptiveColor{Light: "#9a6700", Dark: "#d29922"}
	cBlue   = lipgloss.AdaptiveColor{Light: "#0969da", Dark: "#58a6ff"}

	cRedBg    = lipgloss.AdaptiveColor{Light: "#ffebe9", Dark: "#3d1214"}
	cYellowBg = lipgloss.AdaptiveColor{Light: "#fff8c5", Dark: "#3a2d07"}
	cBlueBg   = lipgloss.AdaptiveColor{Light: "#ddf4ff", Dark: "#0c2d6b"}
)

var (
	boldStyle   = lipgloss.NewStyle().Bold(true).Foreground(cFg)
	mutedStyle  = lipgloss.NewStyle().Foreground(cMuted)
	accentBold  = lipgloss.NewStyle().Foreground(cAccent).Bold(true)
	statusStyle = lipgloss.NewStyle().Foreground(cYellow)
	boxStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(cBorder)
)

func sevLabel(sev string) string {
	switch sev {
	case "error":
		return "Error"
	case "warning":
		return "Warning"
	case "note":
		return "Note"
	}
	return "None"
}

func sevColors(sev string) (fg, bg lipgloss.AdaptiveColor) {
	switch sev {
	case "error":
		return cRed, cRedBg
	case "warning":
		return cYellow, cYellowBg
	case "note":
		return cBlue, cBlueBg
	}
	return cMuted, cBorder
}

func sevIcon(sev string) string {
	fg, _ := sevColors(sev)
	return lipgloss.NewStyle().Foreground(fg).Render("●")
}

// badge is the GitHub-style severity label.
func badge(sev string) string {
	fg, bg := sevColors(sev)
	return lipgloss.NewStyle().Foreground(fg).Background(bg).Bold(true).Padding(0, 1).Render(sevLabel(sev))
}

// joinLR lays out left and right within w columns, truncating left first.
func joinLR(left, right string, w int) string {
	rw := lipgloss.Width(right)
	if rw > 0 {
		rw++ // gap
	}
	left = ansi.Truncate(left, max(1, w-rw), "…")
	pad := w - lipgloss.Width(left) - lipgloss.Width(right)
	return left + strings.Repeat(" ", max(0, pad)) + right
}

// tint paints a full-width background behind an already-highlighted line.
func tint(line string, w int, bg lipgloss.TerminalColor) string {
	probe := lipgloss.NewStyle().Background(bg).Render("X")
	i := strings.Index(probe, "X")
	if i <= 0 { // no color support
		return line
	}
	pre := probe[:i]
	if pad := w - ansi.StringWidth(line); pad > 0 {
		line += strings.Repeat(" ", pad)
	}
	return pre + strings.ReplaceAll(line, "\x1b[0m", "\x1b[0m"+pre) + "\x1b[0m"
}
