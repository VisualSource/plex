package cssom

import css_tokenizer "github.com/VisualSource/plex/internal/css/tokenizer"

// nonWS returns tokens with whitespace entries removed.
func nonWS(tokens []css_tokenizer.Token) []css_tokenizer.Token {
	out := make([]css_tokenizer.Token, 0, len(tokens))
	for _, t := range tokens {
		if t.IsToken() != css_tokenizer.TokenId_Whitespace {
			out = append(out, t)
		}
	}
	return out
}

// tokenScanner is a simple cursor over a token slice used when parsing calc expressions.
type tokenScanner struct {
	tokens []css_tokenizer.Token
	pos    int
}

func (s *tokenScanner) done() bool { return s.pos >= len(s.tokens) }
func (s *tokenScanner) peek() css_tokenizer.Token {
	if s.done() {
		return nil
	}
	return s.tokens[s.pos]
}
func (s *tokenScanner) next() css_tokenizer.Token {
	t := s.peek()
	if t != nil {
		s.pos++
	}
	return t
}
func (s *tokenScanner) skipWS() {
	for !s.done() && s.tokens[s.pos].IsToken() == css_tokenizer.TokenId_Whitespace {
		s.pos++
	}
}
