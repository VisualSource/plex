package script

import (
	"io"
	"strings"
	"unicode"

	"github.com/VisualSource/plex/internal/runeio"
)

type Tokenizer struct {
	stream runeio.RuneReader
	tokens []Token
	row    int64
	col    int64
}

func NewTokenizer() *Tokenizer {
	return &Tokenizer{}
}

func (t *Tokenizer) Tokenize() ([]Token, error) {
	for {
		char, _, err := t.stream.ReadRune()
		if err != nil && err != io.EOF {
			return nil, err
		}
		if err == io.EOF {
			break
		}
		t.row++

		switch char {
		case '\n', '\r', '\t', ' ':
			switch char {
			case '\n':
				t.row = 0
				t.col++
			case '\r':
				isNext, err := t.isNext('\n')
				if err != nil {
					return nil, err
				}

				if isNext {
					if err := t.stream.Discard(1); err != nil {
						return nil, err
					}

					t.row = 0
					t.col++
				}
			}

			continue

		case '"':
			if err := t.stream.UnreadRune(); err != nil {
				return nil, err
			}
			t.row--

			t.consumeString()

		case '/':
			next, err := t.stream.Peek(1)
			if err != nil && err != io.EOF {
				return nil, err
			}

			if err != io.EOF && (next[0] == '/' || next[0] == '*') {
				if err := t.stream.UnreadRune(); err != nil {
					return nil, err
				}
				t.row--

				switch next[0] {
				case '/':
					if err := t.consumeSinglelineComment(); err != nil {
						return nil, err
					}
				case '*':
					if err := t.consumeMultilineComment(); err != nil {
						return nil, err
					}
				}
				continue
			}

			t.tokens = append(t.tokens, NewDataToken(delimMap[char], t.row, t.col))

		case '-', '+', '.':
			next, err := t.stream.Peek(1)
			if err != nil && err != io.EOF {
				return nil, err
			}

			switch {
			case err == io.EOF:
			case unicode.IsDigit(next[0]):
				if err := t.stream.UnreadRune(); err != nil {
					return nil, err
				}
				t.row--
				t.consumeNumber()
				continue
			case char != '.' && char == next[0]:
				if err := t.stream.Discard(1); err != nil {
					return nil, err
				}
				t.row++

				tt := TokenType_Incrment
				if char == '-' {
					tt = TokenType_Decrement
				}

				t.tokens = append(t.tokens, NewDataToken(tt, t.row, t.col))
				continue
			}

			t.tokens = append(t.tokens, NewDataToken(delimMap[char], t.row, t.col))
		case '=':
			next, err := t.stream.Peek(1)
			if err != nil && err != io.EOF {
				return nil, err
			}

			switch {
			case err == io.EOF:
			case next[0] == '>':
				if err := t.stream.Discard(1); err != nil {
					return nil, err
				}
				t.row++
				t.tokens = append(t.tokens, NewDataToken(TokenType_FatArrow, t.row, t.col))
				continue
			case next[0] == '+':

				if err := t.stream.Discard(1); err != nil {
					return nil, err
				}
				t.row++
				t.tokens = append(t.tokens, NewDataToken(TokenType_EqualEqual, t.row, t.col))
				continue
			}
			t.tokens = append(t.tokens, NewDataToken(delimMap[char], t.row, t.col))
		case '>':
			isNext, err := t.isNext('=')
			if err != nil {
				return nil, err
			}
			if isNext {
				if err := t.stream.Discard(1); err != nil {
					return nil, err
				}
				t.row++
				t.tokens = append(t.tokens, NewDataToken(TokenType_GreaterThenOrEqaul, t.row, t.col))
				continue
			}
			t.tokens = append(t.tokens, NewDataToken(delimMap[char], t.row, t.col))
		case '<':
			isNext, err := t.isNext('=')
			if err != nil {
				return nil, err
			}
			if isNext {
				if err := t.stream.Discard(1); err != nil {
					return nil, err
				}
				t.row++
				t.tokens = append(t.tokens, NewDataToken(TokenType_LessThenOrEqual, t.row, t.col))
				continue
			}
			t.tokens = append(t.tokens, NewDataToken(delimMap[char], t.row, t.col))
		case '(', ')', ';', '*', '%', '?', ':', ',', '[', ']', '{', '}':
			t.tokens = append(t.tokens, NewDataToken(delimMap[char], t.row, t.col))

		case '|':
			next, err := t.isNext('|')
			if err != nil {
				return nil, err
			}
			if next {
				if err := t.stream.Discard(1); err != nil {
					return nil, err
				}
				t.row++
				t.tokens = append(t.tokens, NewDataToken(TokenType_OR, t.row, t.col))
				continue
			}

			return nil, ErrUnexpectedCharacter
		case '&':
			next, err := t.isNext('&')
			if err != nil {
				return nil, err
			}
			if next {
				if err := t.stream.Discard(1); err != nil {
					return nil, err
				}
				t.row++
				t.tokens = append(t.tokens, NewDataToken(TOkenType_AND, t.row, t.col))
				continue
			}

			return nil, ErrUnexpectedCharacter
		case '!':
			next, err := t.isNext('=')
			if err != nil {
				return nil, err
			}
			if next {
				if err := t.stream.Discard(1); err != nil {
					return nil, err
				}
				t.row++
				t.tokens = append(t.tokens, NewDataToken(TokenType_NotEqual, t.row, t.col))
				continue
			}

			fallthrough
		default:
			if unicode.IsDigit(char) {
				if err := t.stream.UnreadRune(); err != nil {
					return nil, err
				}
				t.row--

				if err := t.consumeNumber(); err != nil {
					return nil, err
				}

				continue
			}

			if char == '_' || unicode.IsLetter(char) {
				if err := t.stream.UnreadRune(); err != nil {
					return nil, err
				}
				t.row--
				if err := t.consumeIdent(); err != nil {
					return nil, err
				}
				continue
			}

			return nil, ErrUnexpectedCharacter
		}
	}

	return t.tokens, nil
}

func (t *Tokenizer) isNext(char rune) (bool, error) {
	next, err := t.stream.Peek(1)
	if err != nil && err != io.EOF {
		return false, err
	}

	if err == io.EOF {
		return false, nil
	}

	return next[0] == char, nil
}

// call should have consumed // before calling
func (t *Tokenizer) consumeSinglelineComment() error {

loop:
	for {
		char, _, err := t.stream.ReadRune()
		if err != nil && err != io.EOF {
			return err
		}

		if err == io.EOF {
			break
		}
		t.row++

		switch char {
		case '\n':
			t.row = 0
			t.col++
			break loop
		case '\r':
			isNext, err := t.isNext('\n')
			if err != nil {
				return err
			}

			if isNext {
				t.row = 0
				t.col++
				if err := t.stream.Discard(1); err != nil {
					return err
				}
				break loop
			}

		}

	}

	return nil
}

// caller should have consume /* before calling
func (t *Tokenizer) consumeMultilineComment() error {

loop:
	for {
		char, _, err := t.stream.ReadRune()
		if err != nil && err != io.EOF {
			return err
		}

		if err == io.EOF {
			break
		}
		t.row++

		switch char {
		case '*':
			isNext, err := t.isNext('/')
			if err != nil {
				return err
			}

			if isNext {
				if err := t.stream.Discard(1); err != nil {
					return err
				}
				t.row++
				break loop
			}

		case '\n':
			t.row = 0
			t.col++
		case '\r':
			isNext, err := t.isNext('\n')
			if err != nil {
				return err
			}

			if isNext {
				if err := t.stream.Discard(1); err != nil {
					return err
				}
				t.row = 0
				t.col++
			}
		}

	}

	return nil
}

func (t *Tokenizer) consumeIdent() error {

	ident := strings.Builder{}

	for {
		char, _, err := t.stream.ReadRune()
		if err != nil && err != io.EOF {
			return err
		}
		if err == io.EOF {
			break
		}
		t.row++

		if char == '_' || unicode.IsLetter(char) || unicode.IsDigit(char) {
			ident.WriteRune(char)
			continue
		}

		if err := t.stream.UnreadRune(); err != nil {
			return err
		}
		t.row--

		break
	}

	value := ident.String()

	switch value {
	case "while", "if", "else", "struct", "fn", "mut":
		t.tokens = append(t.tokens, NewIdentToken(value, t.row, t.col))
	case "int", "int64", "int32", "int16", "int8", "uint", "u64", "u32", "u16", "u8", "bool":
		t.tokens = append(t.tokens, NewIdentToken(value, t.row, t.col))
	default:
		t.tokens = append(t.tokens, NewIdentToken(value, t.row, t.col))
	}

	return nil
}

func (t *Tokenizer) consumeNumber() error {

	return nil
}

func (t *Tokenizer) consumeString() error {
	s, _, err := t.stream.ReadRune()
	if err != nil && err != io.EOF {
		return err
	}

	if err == io.EOF {
		return ErrUnexpectedEOF
	}

	if s != '"' {
		return ErrMissingStartQuote
	}
	t.row++

	value := strings.Builder{}

lo:
	for {
		c, _, err := t.stream.ReadRune()
		if err != nil && err != io.EOF {
			return err
		}
		if err == io.EOF {
			return ErrUnexpectedEOF
		}
		t.row++

		switch c {
		case '"':
			break lo
		case '\r':
			isNext, err := t.isNext('\n')
			if err != nil {
				return err
			}
			if isNext {
				return ErrUnexpectedNewlineInString
			}

		case '\n':
			return ErrUnexpectedNewlineInString
		}

		value.WriteRune(c)
	}

	t.tokens = append(t.tokens, NewStringToken(value.String(), t.row, t.col))

	return nil
}
