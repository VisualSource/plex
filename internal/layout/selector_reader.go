package layout

import "github.com/VisualSource/plex/internal/css/tokenizer"

// tokenReader is a forward cursor over a list of component values. The list
// produced by ParseListOfComponentValues has no trailing EOF token, so the end
// of input is simply pos >= len(toks).
type tokenReader struct {
	toks []tokenizer.Token
	pos  int
}

func newTokenReader(toks []tokenizer.Token) *tokenReader {
	return &tokenReader{toks: toks}
}

func (r *tokenReader) atEnd() bool { return r.pos >= len(r.toks) }

// peek returns the current token without consuming it, or nil at end of input.
func (r *tokenReader) peek() tokenizer.Token {
	if r.atEnd() {
		return nil
	}
	return r.toks[r.pos]
}

// peekN looks ahead n tokens without consuming (peek == peekN(0)); nil if out of range.
func (r *tokenReader) peekN(n int) tokenizer.Token {
	i := r.pos + n
	if i < 0 || i >= len(r.toks) {
		return nil
	}
	return r.toks[i]
}

// next consumes and returns the current token, or nil at end of input.
func (r *tokenReader) next() tokenizer.Token {
	t := r.peek()
	if t != nil {
		r.pos++
	}
	return t
}

func (r *tokenReader) skipWhitespace() {
	for !r.atEnd() && r.toks[r.pos].IsToken() == tokenizer.TokenId_Whitespace {
		r.pos++
	}
}

// peekIsWhitespace reports whether the current token is whitespace, without consuming it.
func (r *tokenReader) peekIsWhitespace() bool {
	t := r.peek()
	return t != nil && t.IsToken() == tokenizer.TokenId_Whitespace
}

// mark/reset support backtracking (e.g. distinguishing a namespace '|' from the
// '||' column combinator).
func (r *tokenReader) mark() int   { return r.pos }
func (r *tokenReader) reset(m int) { r.pos = m }
