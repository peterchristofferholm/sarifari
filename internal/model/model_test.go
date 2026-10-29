package model

import (
	"testing"

	"github.com/peterchristofferholm/sarifari/internal/sarif"
)

func load(t *testing.T, name string) []*Finding {
	t.Helper()
	l, _, err := sarif.Load("../../testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return FromLog(l)
}

func TestRuleResolution(t *testing.T) {
	f := load(t, "ruleindex.sarif.json")
	tests := []struct {
		i       int
		id      string
		sev     string
		ruleNil bool
	}{
		{0, "r1", "error", false},   // index hit, id from rule, level from rule default
		{1, "r0", "warning", false}, // id lookup, fallback severity
		{2, "r1", "error", false},   // out-of-range index falls back to id
	}
	for _, tc := range tests {
		got := f[tc.i]
		if got.RuleID != tc.id || got.Severity != tc.sev || (got.Rule == nil) != tc.ruleNil {
			t.Errorf("finding %d = %+v", tc.i, got)
		}
	}
	n := load(t, "norules.sarif.json")[0]
	if n.Rule != nil || n.RuleID != "a.b.c" || n.RuleName != "c" {
		t.Errorf("norules: %+v", n)
	}
}

func TestSeverityFallback(t *testing.T) {
	f := load(t, "minimal.sarif.json")
	if f[2].Severity != "note" {
		t.Errorf("want rule default note, got %q", f[2].Severity)
	}
	if got := load(t, "nolocation.sarif.json")[0].Severity; got != "note" {
		t.Errorf("result level should win, got %q", got)
	}
}

func TestPathNormalization(t *testing.T) {
	for in, want := range map[string]string{
		"./src/a.go":  "src/a.go",
		"file://a.go": "a.go",
		`src\b\c.go`:  "src/b/c.go",
		"plain.go":    "plain.go",
	} {
		if got := normalizePath(in); got != want {
			t.Errorf("%q -> %q, want %q", in, got, want)
		}
	}
}

func TestNoLocation(t *testing.T) {
	f := load(t, "nolocation.sarif.json")[0]
	if f.HasLocation() || f.Path != NoLocation || f.StartLine != 0 {
		t.Errorf("%+v", f)
	}
}

func TestMultiRun(t *testing.T) {
	f := load(t, "multirun.sarif.json")
	if len(f) != 3 || f[0].Tool != "A" || f[2].Tool != "B" {
		t.Errorf("%+v", f)
	}
}

func TestGrouping(t *testing.T) {
	f := load(t, "minimal.sarif.json")
	tests := []struct {
		by   GroupBy
		keys []string
	}{
		{ByRule, []string{"go.lang.security.sqli", "go.lang.best.todo"}},
		{BySeverity, []string{"error", "note"}},
		{ByFile, []string{"src/db/a.go", "src/db/q.go", "main.go"}}, // all count 1: severity, then key
		{ByDirectory, []string{"src", "."}},
	}
	for _, tc := range tests {
		gs := GroupFindings(f, tc.by, "")
		var keys []string
		sum := 0
		for _, g := range gs {
			keys = append(keys, g.Key)
			sum += len(g.Findings)
		}
		if len(keys) != len(tc.keys) {
			t.Fatalf("%v: keys %v", tc.by, keys)
		}
		for i := range keys {
			if keys[i] != tc.keys[i] {
				t.Errorf("%v: keys %v, want %v", tc.by, keys, tc.keys)
				break
			}
		}
		if sum != len(f) {
			t.Errorf("%v: counts sum %d, want %d", tc.by, sum, len(f))
		}
	}
	// findings sorted by path within a group
	g := GroupFindings(f, ByRule, "")[0]
	if g.Findings[0].Path != "src/db/a.go" || g.Severity != "error" {
		t.Errorf("%+v", g)
	}
}

func TestSeverityFilterAffectsCounts(t *testing.T) {
	f := load(t, "minimal.sarif.json")
	gs := GroupFindings(f, ByRule, "note")
	if len(gs) != 1 || len(gs[0].Findings) != 1 {
		t.Errorf("%+v", gs)
	}
	if gs := GroupFindings(f, ByFile, "error"); len(gs) != 2 {
		t.Errorf("%+v", gs)
	}
}

func TestTitle(t *testing.T) {
	f := load(t, "minimal.sarif.json")
	if got := f[0].Title(); got != "sqli" { // short description wins
		t.Errorf("got %q", got)
	}
	if got := f[2].Title(); got != "Todo" {
		t.Errorf("got %q", got)
	}
	if got := (&Finding{RuleID: "a.b-c_d", RuleName: "b-c_d"}).Title(); got != "B c d" {
		t.Errorf("got %q", got)
	}
}

func TestGroupByCycleAndParse(t *testing.T) {
	if ByRule.Cycle(-1) != ByDirectory || ByDirectory.Cycle(1) != ByRule {
		t.Error("cycle should wrap")
	}
	if g, ok := ParseGroupBy("dir"); !ok || g != ByDirectory {
		t.Errorf("got %v %v", g, ok)
	}
	if _, ok := ParseGroupBy("nope"); ok {
		t.Error("want parse failure")
	}
}

func TestTrivy(t *testing.T) {
	f := load(t, "trivy.sarif.json")
	if f[0].Subject != "golang.org/x/sys@v0.38.0" {
		t.Errorf("subject %q", f[0].Subject)
	}
	if f[0].Description() != "" { // only repeats the rule ID
		t.Errorf("description %q", f[0].Description())
	}
	gs := GroupFindings(f, ByRule, "")
	if gs[0].Key != "CVE-2026-1" { // error before note on a count tie
		t.Fatalf("order: %v", gs[0].Key)
	}
	if gs[0].Description != "x/net: XSS via HTML parsing" || gs[0].Subject() != "golang.org/x/net@v0.38.0" {
		t.Errorf("%+v", gs[0])
	}
	if g := GroupFindings(f, ByFile, "")[0]; g.Subject() != "" {
		t.Errorf("mixed subjects should be empty, got %q", g.Subject())
	}
}
