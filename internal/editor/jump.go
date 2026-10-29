package editor

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Jump opens abs at line. Inside a Neovim terminal ($NVIM set) it drives the
// parent instance and reports quit=true; otherwise it returns a command for
// the caller to run with tea.ExecProcess.
func Jump(abs string, line int) (cmd *exec.Cmd, quit bool, err error) {
	if line < 1 {
		line = 1
	}
	if server := os.Getenv("NVIM"); server != "" {
		keys := fmt.Sprintf(`<C-\><C-n><cmd>close<CR><cmd>edit +%d %s<CR>`, line, remoteEscape(abs))
		out, err := exec.Command("nvim", "--server", server, "--remote-send", keys).CombinedOutput()
		if err != nil {
			return nil, false, fmt.Errorf("nvim remote-send: %w: %s", err, strings.TrimSpace(string(out)))
		}
		return nil, true, nil
	}
	ed := os.Getenv("EDITOR")
	if ed == "" {
		ed = "vi"
	}
	parts := strings.Fields(ed)
	args := append(parts[1:], fmt.Sprintf("+%d", line), abs)
	return exec.Command(parts[0], args...), false, nil
}

// remoteEscape mirrors Vim's fnameescape(), then protects '<' from being read
// as key notation by --remote-send.
func remoteEscape(p string) string {
	var b strings.Builder
	for _, r := range p {
		if strings.ContainsRune(" \t\n*?[{`$\\%#'\"|!<", r) {
			b.WriteByte('\\')
		}
		if r == '<' {
			b.WriteString("<lt>")
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
