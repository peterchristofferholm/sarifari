package ui

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/peterchristofferholm/sarifari/internal/model"
	"github.com/peterchristofferholm/sarifari/internal/sarif"
)

func fixture(t *testing.T, name string) ([]*model.Finding, string) {
	t.Helper()
	l, _, err := sarif.Load("../../testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "src/db"), 0o755)
	var src strings.Builder
	for i := 1; i <= 20; i++ {
		src.WriteString("// line\n")
	}
	os.WriteFile(filepath.Join(root, "src/db/q.go"), []byte(src.String()), 0o644)
	return model.FromLog(l), root
}

func press(m tea.Model, keys ...string) tea.Model {
	for _, k := range keys {
		var msg tea.KeyMsg
		switch k {
		case "enter":
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		case "esc":
			msg = tea.KeyMsg{Type: tea.KeyEsc}
		case "tab":
			msg = tea.KeyMsg{Type: tea.KeyTab}
		default:
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		}
		var cmd tea.Cmd
		m, cmd = m.Update(msg)
		// run source-load commands synchronously, ignore quit
		if cmd != nil {
			if out := cmd(); out != nil {
				if _, ok := out.(sourceLoadedMsg); ok {
					m, _ = m.Update(out)
				}
			}
		}
	}
	return m
}

func TestNavigationFlow(t *testing.T) {
	f, root := fixture(t, "minimal.sarif")
	var m tea.Model = New(f, Options{Roots: []string{root}, Style: "github-dark"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	v := m.View()
	if !strings.Contains(v, "3 findings") || !strings.Contains(v, "sqli") {
		t.Fatalf("summary:\n%s", v)
	}

	m = press(m, "enter") // into sqli group (2 findings)
	if v = m.View(); !strings.Contains(v, "2 findings") || !strings.Contains(v, "src/db/a.go:3") {
		t.Fatalf("findings:\n%s", v)
	}

	m = press(m, "j", "enter") // second finding: q.go, file exists
	v = m.View()
	if !strings.Contains(v, "bad sql") || !strings.Contains(v, "// line") || !strings.Contains(v, "2/2") {
		t.Fatalf("detail:\n%s", v)
	}

	m = press(m, "p") // previous
	if v = m.View(); !strings.Contains(v, "1/2") {
		t.Fatalf("after p:\n%s", v)
	}
	m = press(m, "esc", "-")
	if v = m.View(); !strings.Contains(v, "3 findings") {
		t.Fatalf("back on summary:\n%s", v)
	}

	// cursor position restored after drilling in and out
	m = press(m, "j", "enter", "esc")
	if it := m.(Model).sum.SelectedItem().(groupItem); it.Key != "go.lang.best.todo" {
		t.Errorf("cursor not restored: %s", it.Key)
	}

	m = press(m, "tab") // regroup by severity
	if v = m.View(); !strings.Contains(v, "group: severity") {
		t.Fatalf("regroup:\n%s", v)
	}
	m = press(m, "s") // severity filter: error
	if v = m.View(); !strings.Contains(v, "2 findings") {
		t.Fatalf("filter:\n%s", v)
	}
}

func TestEmptyAndMissingSource(t *testing.T) {
	var m tea.Model = New(nil, Options{Roots: []string{t.TempDir()}})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	if !strings.Contains(m.View(), "No findings") {
		t.Error("empty state")
	}

	f, _ := fixture(t, "minimal.sarif")
	m = New(f, Options{Roots: []string{t.TempDir()}}) // root has no files
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	m = press(m, "enter", "enter")
	if v := m.View(); !strings.Contains(v, "SARIF snippet") && !strings.Contains(v, "source unavailable") {
		t.Errorf("want fallback notice:\n%s", v)
	}

	nl, _ := fixture(t, "nolocation.sarif")
	m = New(nl, Options{Roots: []string{t.TempDir()}})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	m = press(m, "enter", "enter", "e")
	if !strings.Contains(m.View(), "no location") {
		t.Error("jump should be disabled with a status message")
	}
}

func TestTeatestSmoke(t *testing.T) {
	f, root := fixture(t, "minimal.sarif")
	tm := teatest.NewTestModel(t, New(f, Options{Roots: []string{root}, Style: "github-dark"}), teatest.WithInitialTermSize(100, 30))
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	out, err := io.ReadAll(tm.FinalOutput(t, teatest.WithFinalTimeout(5*time.Second)))
	if err != nil || !bytes.Contains(out, []byte("severity")) {
		t.Errorf("err=%v out=%q", err, out)
	}
}

func TestWrappedAnnotationHasNoBlankRows(t *testing.T) {
	f := &model.Finding{Path: "a.go", StartLine: 1, EndLine: 1, Severity: "error",
		Message: strings.Repeat("word ", 40)}
	m := New([]*model.Finding{f}, Options{Style: "github"})
	m.sources["a.go"] = sourceFile{lines: []string{"x := 1"}}
	var bars []int
	for i, l := range strings.Split(m.codeBox(f, 40), "\n") {
		if strings.Contains(l, "▌") {
			bars = append(bars, i)
		}
	}
	if len(bars) < 2 || bars[len(bars)-1]-bars[0] != len(bars)-1 {
		t.Errorf("annotation rows not contiguous: %v", bars)
	}
}

func TestDashGoesBackExceptWhileFiltering(t *testing.T) {
	f, root := fixture(t, "minimal.sarif")
	var m tea.Model = New(f, Options{Roots: []string{root}, Style: "github"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = press(m, "enter", "/", "-")
	if got := m.(Model).screen; got != screenFindings {
		t.Fatalf("'-' while filtering should be typed, screen = %v", got)
	}
	m = press(m, "esc") // cancel the filter
	if got := m.(Model).screen; got != screenFindings {
		t.Fatalf("esc should only cancel the filter, screen = %v", got)
	}
	m = press(m, "-")
	if got := m.(Model).screen; got != screenSummary {
		t.Errorf("'-' should go back, screen = %v", got)
	}
}
