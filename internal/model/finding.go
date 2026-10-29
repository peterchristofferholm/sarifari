package model

import (
	"net/url"
	"strings"

	"sarifari/internal/sarif"
)

const NoLocation = "(no location)"

type Finding struct {
	RuleID    string
	RuleName  string // last dotted segment of RuleID, for compact display
	Severity  string // "error" | "warning" | "note" | "none"
	Message   string
	Path      string // forward slashes, as given in the SARIF minus file:// and ./
	StartLine int
	StartCol  int
	EndLine   int
	EndCol    int
	Snippet   string
	Tool      string
	Rule      *sarif.Rule // may be nil
}

// HasLocation reports whether the finding can be shown in source or jumped to.
func (f *Finding) HasLocation() bool { return f.Path != NoLocation }

// FromLog flattens every result of every run into Findings.
func FromLog(l *sarif.Log) []*Finding {
	var out []*Finding
	for ri := range l.Runs {
		run := &l.Runs[ri]
		byID := make(map[string]*sarif.Rule, len(run.Tool.Driver.Rules))
		for i := range run.Tool.Driver.Rules {
			r := &run.Tool.Driver.Rules[i]
			byID[r.ID] = r
		}
		for i := range run.Results {
			out = append(out, newFinding(&run.Results[i], run, byID))
		}
	}
	return out
}

func newFinding(res *sarif.Result, run *sarif.Run, byID map[string]*sarif.Rule) *Finding {
	rules := run.Tool.Driver.Rules
	var rule *sarif.Rule
	if res.RuleIndex != nil && *res.RuleIndex >= 0 && *res.RuleIndex < len(rules) {
		rule = &rules[*res.RuleIndex]
	} else if r, ok := byID[res.RuleID]; ok {
		rule = r
	}

	id := res.RuleID
	if id == "" && rule != nil {
		id = rule.ID
	}
	if id == "" {
		id = "(unknown)"
	}

	sev := res.Level
	if sev == "" && rule != nil {
		sev = rule.DefaultConfig.Level
	}
	if sev == "" {
		sev = "warning"
	}

	f := &Finding{
		RuleID:   id,
		RuleName: id[strings.LastIndex(id, ".")+1:],
		Severity: sev,
		Message:  res.Message.Text,
		Path:     NoLocation,
		Tool:     run.Tool.Driver.Name,
		Rule:     rule,
	}
	if f.RuleName == "" {
		f.RuleName = id
	}
	if len(res.Locations) > 0 {
		p := res.Locations[0].Physical
		if uri := normalizePath(p.Artifact.URI); uri != "" {
			f.Path = uri
			f.StartLine = p.Region.StartLine
			f.StartCol = p.Region.StartColumn
			f.EndLine = p.Region.EndLine
			f.EndCol = p.Region.EndColumn
			f.Snippet = p.Region.Snippet.Text
		}
	}
	return f
}

// normalizePath strips file:// and leading ./ and converts to forward slashes.
func normalizePath(uri string) string {
	uri = strings.TrimPrefix(uri, "file://")
	if dec, err := url.PathUnescape(uri); err == nil {
		uri = dec
	}
	uri = strings.ReplaceAll(uri, "\\", "/")
	for strings.HasPrefix(uri, "./") {
		uri = uri[2:]
	}
	return uri
}

// Title is a human-readable alert title: the rule's short description when it
// is concise, otherwise the rule name with separators turned into spaces.
func (f *Finding) Title() string {
	if f.Rule != nil {
		if t := strings.TrimSpace(f.Rule.ShortDescription.Text); t != "" && len(t) <= 100 && !strings.Contains(t, "\n") {
			return t
		}
	}
	t := strings.Join(strings.FieldsFunc(f.RuleName, func(r rune) bool { return r == '-' || r == '_' }), " ")
	if t == "" {
		return f.RuleID
	}
	return strings.ToUpper(t[:1]) + t[1:]
}
