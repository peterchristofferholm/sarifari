package model

import (
	"cmp"
	"slices"
	"strings"
)

type GroupBy int

const (
	ByRule GroupBy = iota
	BySeverity
	ByFile
	ByDirectory // first path segment; files in the root go under "."
)

var groupByNames = [...]string{"rule", "severity", "file", "dir"}

func (g GroupBy) String() string { return groupByNames[g] }

// ParseGroupBy is the inverse of String.
func ParseGroupBy(s string) (GroupBy, bool) {
	i := slices.Index(groupByNames[:], s)
	return GroupBy(i), i >= 0
}

// Cycle steps d groupings forward (or back, if negative), wrapping around.
func (g GroupBy) Cycle(d int) GroupBy {
	n := len(groupByNames)
	return GroupBy(((int(g)+d)%n + n) % n)
}

type Group struct {
	Key      string
	Label    string
	Severity string // highest severity among members (error > warning > note)
	Findings []*Finding
}

var sevRank = map[string]int{"error": 3, "warning": 2, "note": 1}

// GroupFindings applies the severity filter (empty = all) and groups the
// findings. Groups sort by size descending then key; findings by path, line.
func GroupFindings(findings []*Finding, by GroupBy, sevFilter string) []Group {
	idx := map[string]int{}
	var out []Group
	for _, f := range findings {
		if sevFilter != "" && f.Severity != sevFilter {
			continue
		}
		key, label := groupKey(f, by)
		i, ok := idx[key]
		if !ok {
			i = len(out)
			idx[key] = i
			out = append(out, Group{Key: key, Label: label, Severity: f.Severity})
		}
		g := &out[i]
		g.Findings = append(g.Findings, f)
		if sevRank[f.Severity] > sevRank[g.Severity] {
			g.Severity = f.Severity
		}
	}
	for _, g := range out {
		slices.SortStableFunc(g.Findings, func(a, b *Finding) int {
			return cmp.Or(cmp.Compare(a.Path, b.Path), cmp.Compare(a.StartLine, b.StartLine))
		})
	}
	slices.SortStableFunc(out, func(a, b Group) int {
		return cmp.Or(cmp.Compare(len(b.Findings), len(a.Findings)), cmp.Compare(a.Key, b.Key))
	})
	return out
}

func groupKey(f *Finding, by GroupBy) (key, label string) {
	switch by {
	case BySeverity:
		return f.Severity, f.Severity
	case ByFile:
		return f.Path, f.Path
	case ByDirectory:
		if !f.HasLocation() {
			return NoLocation, NoLocation
		}
		dir, _, ok := strings.Cut(strings.TrimPrefix(f.Path, "/"), "/")
		if !ok || dir == "" {
			dir = "."
		}
		return dir, dir
	default:
		return f.RuleID, f.RuleName
	}
}
