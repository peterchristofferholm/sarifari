package sarif

import "testing"

func TestParseErrors(t *testing.T) {
	if _, _, err := parse([]byte("{nope")); err == nil {
		t.Error("want JSON error")
	}
	if _, _, err := parse([]byte(`{"version":"2.1.0","runs":[]}`)); err == nil {
		t.Error("want no-runs error")
	}
	if _, _, err := Load("does-not-exist.sarif"); err == nil {
		t.Error("want read error")
	}
}

func TestParseVersionWarning(t *testing.T) {
	_, w, err := parse([]byte(`{"version":"2.0.0","runs":[{}]}`))
	if err != nil || w == "" {
		t.Errorf("warning=%q err=%v", w, err)
	}
	_, w, _ = parse([]byte(`{"version":"2.1.0","runs":[{}]}`))
	if w != "" {
		t.Errorf("unexpected warning %q", w)
	}
}
