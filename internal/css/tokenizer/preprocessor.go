package css_tokenizer

import (
	"bufio"
	"io"
)

// https://www.w3.org/TR/css-syntax-3/#input-preprocessing
//
// preprocessor wraps an io.Reader to perform CSS input preprocessing on the
// byte stream before it reaches a rune decoder:
//   - U+000D CARRIAGE RETURN, U+000C FORM FEED, and U+000D U+000A pairs
//     are replaced with a single U+000A LINE FEED.
//   - U+0000 NULL and surrogate code points (U+D800–U+DFFF) are replaced
//     with U+FFFD REPLACEMENT CHARACTER.
//
// `\r`, `\f`, and NULL are ASCII, which in UTF-8 always represent themselves
// and never appear as continuation bytes, so byte-level substitution is safe.
// Surrogates are detected by their (invalid) UTF-8 encoding `0xED 0xA0–0xBF
// 0x80–0xBF`; the leading `0xED` byte of a valid non-surrogate 3-byte sequence
// (U+D000–U+D7FF) is followed by `0x80–0x9F` so the ranges don't collide.
type preprocessor struct {
	input   *bufio.Reader
	pending []byte // unflushed bytes from a prior U+FFFD substitution
}

var replacementBytes = [...]byte{0xEF, 0xBF, 0xBD} // U+FFFD in UTF-8

func newPreprocessor(r io.Reader) *preprocessor {
	return &preprocessor{input: bufio.NewReader(r)}
}

func (p *preprocessor) Read(buf []byte) (int, error) {
	if len(buf) == 0 {
		return 0, nil
	}

	n := 0
	if len(p.pending) > 0 {
		c := copy(buf, p.pending)
		p.pending = p.pending[c:]
		n += c
		if n == len(buf) {
			return n, nil
		}
	}

	for n < len(buf) {
		b, err := p.input.ReadByte()
		if err != nil {
			if n > 0 {
				return n, nil
			}
			return 0, err
		}

		switch b {
		case 0x00:
			c := copy(buf[n:], replacementBytes[:])
			n += c
			if c < len(replacementBytes) {
				p.pending = append(p.pending, replacementBytes[c:]...)
				return n, nil
			}
		case 0xED:
			next, _ := p.input.Peek(2)
			if len(next) == 2 && next[0] >= 0xA0 && next[0] <= 0xBF && next[1] >= 0x80 && next[1] <= 0xBF {
				_, _ = p.input.Discard(2)
				c := copy(buf[n:], replacementBytes[:])
				n += c
				if c < len(replacementBytes) {
					p.pending = append(p.pending, replacementBytes[c:]...)
					return n, nil
				}
			} else {
				buf[n] = b
				n++
			}
		case '\r':
			next, err := p.input.Peek(1)
			if err == nil && len(next) == 1 && next[0] == '\n' {
				_, _ = p.input.Discard(1)
			}
			buf[n] = '\n'
			n++
		case '\f':
			buf[n] = '\n'
			n++
		default:
			buf[n] = b
			n++
		}
	}

	return n, nil
}
