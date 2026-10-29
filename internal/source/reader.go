package source

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Resolve finds the file for a SARIF path. Absolute paths are used as given.
// Relative paths must stay inside the root and are tried against each root in
// order; the first existing file wins.
func Resolve(roots []string, p string) (string, error) {
	native := filepath.FromSlash(p)
	if filepath.IsAbs(native) {
		full := filepath.Clean(native)
		if _, err := os.Stat(full); err != nil {
			return "", fmt.Errorf("%s: not found", full)
		}
		return full, nil
	}
	if !filepath.IsLocal(native) {
		return "", fmt.Errorf("%s escapes the root", p)
	}
	var tried []string
	for _, root := range roots {
		absRoot, err := filepath.Abs(root)
		if err != nil {
			continue
		}
		full := filepath.Join(absRoot, native)
		if _, err := os.Stat(full); err == nil {
			return full, nil
		}
		tried = append(tried, absRoot)
	}
	return "", fmt.Errorf("%s not found under %s (use --root)", p, strings.Join(tried, ", "))
}

// ReadLines returns the file split into lines with tabs expanded to 4 spaces.
func ReadLines(abs string) ([]string, error) {
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\t", "    ")
	return strings.Split(strings.TrimSuffix(text, "\n"), "\n"), nil
}
