package source

import (
	"bytes"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// Highlight colors code for a terminal, choosing the lexer from filename.
// On any failure it returns the code unchanged.
func Highlight(filename, code, style string) string {
	lexer := lexers.Match(filename)
	if lexer == nil {
		lexer = lexers.Analyse(code)
	}
	if lexer == nil {
		return code
	}
	lexer = chroma.Coalesce(lexer)
	st := styles.Get(style)
	if st == nil {
		st = styles.Fallback
	}
	it, err := lexer.Tokenise(nil, code)
	if err != nil {
		return code
	}
	var buf bytes.Buffer
	if err := formatters.Get("terminal256").Format(&buf, st, it); err != nil {
		return code
	}
	return buf.String()
}
