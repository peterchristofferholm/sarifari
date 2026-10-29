package source

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolve(t *testing.T) {
	root, other := t.TempDir(), t.TempDir()
	os.MkdirAll(filepath.Join(other, "a"), 0o755)
	want := filepath.Join(other, "a", "b.go")
	os.WriteFile(want, []byte("x"), 0o644)

	// found in the second root
	if got, err := Resolve([]string{root, other}, "a/b.go"); err != nil || got != want {
		t.Errorf("got %q, %v", got, err)
	}
	// absolute path is used directly
	if got, err := Resolve([]string{root}, want); err != nil || got != want {
		t.Errorf("abs: got %q, %v", got, err)
	}
	if _, err := Resolve([]string{root}, "a/missing.go"); err == nil {
		t.Error("missing file should error")
	}
	for _, bad := range []string{"../x.go", "a/../../x.go"} {
		if _, err := Resolve([]string{root}, bad); err == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
}

func TestReadLines(t *testing.T) {
	p := filepath.Join(t.TempDir(), "f.txt")
	os.WriteFile(p, []byte("a\r\n\tb\n"), 0o644)
	got, err := ReadLines(p)
	if err != nil || len(got) != 2 || got[1] != "    b" {
		t.Errorf("%q %v", got, err)
	}
	if _, err := ReadLines(p + ".missing"); err == nil {
		t.Error("want error")
	}
}
