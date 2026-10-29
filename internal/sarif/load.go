package sarif

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// Load reads and parses a SARIF file. The returned warning is non-empty when
// the file loaded but looks unusual (e.g. an unexpected version).
func Load(path string) (log *Log, warning string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("read %s: %w", path, err)
	}
	return parse(data)
}

func parse(data []byte) (*Log, string, error) {
	var l Log
	if err := json.Unmarshal(data, &l); err != nil {
		return nil, "", fmt.Errorf("invalid JSON: %w", err)
	}
	if len(l.Runs) == 0 {
		return nil, "", errors.New("SARIF file contains no runs")
	}
	var warning string
	if l.Version != "2.1.0" {
		warning = fmt.Sprintf("unexpected SARIF version %q (expected 2.1.0)", l.Version)
	}
	return &l, warning, nil
}
