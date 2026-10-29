package editor

import (
	"slices"
	"testing"
)

func TestRemoteEscape(t *testing.T) {
	for in, want := range map[string]string{
		"/a/b.go":     "/a/b.go",
		"/a b/c.go":   `/a\ b/c.go`,
		"/a#b%c|d.go": `/a\#b\%c\|d.go`,
		"/a<b>.go":    `/a\<lt>b>.go`,
	} {
		if got := remoteEscape(in); got != want {
			t.Errorf("%q -> %q, want %q", in, got, want)
		}
	}
}

func TestJumpFallbackCommand(t *testing.T) {
	t.Setenv("NVIM", "")
	t.Setenv("EDITOR", "nvim -R")
	cmd, quit, err := Jump("/x/y.go", 7)
	if err != nil || quit {
		t.Fatal(err, quit)
	}
	want := []string{"nvim", "-R", "+7", "/x/y.go"}
	if !slices.Equal(cmd.Args, want) {
		t.Errorf("args %v, want %v", cmd.Args, want)
	}
}
