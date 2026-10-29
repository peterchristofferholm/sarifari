package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/peterchristofferholm/sarifari/internal/model"
	"github.com/peterchristofferholm/sarifari/internal/sarif"
	"github.com/peterchristofferholm/sarifari/internal/ui"
)

func usage(msg string) {
	if msg != "" {
		fmt.Fprintln(os.Stderr, "sarifari:", msg)
	}
	fmt.Fprintln(os.Stderr, "usage: sarifari [flags] <file.sarif>")
	flag.PrintDefaults()
	os.Exit(2)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "sarifari:", err)
	os.Exit(1)
}

func main() {
	root := flag.String("root", "", "root directory for resolving source paths (default: current directory, then the SARIF file's directory)")
	group := flag.String("group", "rule", "initial grouping: rule|severity|file|dir")
	sev := flag.String("severity", "all", "initial severity filter: all|error|warning|note")
	style := flag.String("style", "", "chroma style for source snippets (default: github or github-dark by terminal)")
	flag.Usage = func() { usage("") }
	flag.Parse()

	if flag.NArg() != 1 {
		usage("exactly one SARIF file is required")
	}
	path := flag.Arg(0)
	groupBy, ok := model.ParseGroupBy(*group)
	if !ok {
		usage(fmt.Sprintf("invalid --group %q", *group))
	}
	if !slices.Contains(ui.SeverityFilters, *sev) {
		usage(fmt.Sprintf("invalid --severity %q", *sev))
	}

	sarifLog, warning, err := sarif.Load(path)
	if err != nil {
		fatal(err)
	}
	roots := []string{*root}
	if *root == "" {
		roots = []string{".", filepath.Dir(path)}
	}
	m := ui.New(model.FromLog(sarifLog), ui.Options{Roots: roots, Style: *style, GroupBy: groupBy, Severity: *sev, Notice: warning})
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fatal(err)
	}
}
