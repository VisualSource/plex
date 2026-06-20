package script

import (
	"io"
	"strings"
	"unicode"

	"github.com/VisualSource/plex/internal/runeio"
)

type Tokenizer struct {
	stream *runeio.RuneReader
	tokens []Token
	row    int64
	col    int64
}

func NewTokenizer(stream io.Reader) *Tokenizer {
	return &Tokenizer{
		stream: runeio.NewReader(stream),
		tokens: make([]Token, 0),
		row:    0,
		col:    0,
	}
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
			start := NewPosition(t.row, t.col)
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

			t.tokens = append(t.tokens, NewDataToken(delimMap[char], start, NewPosition(t.row, t.col)))

		case '-', '+', '.':
			start := NewPosition(t.row, t.col)
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
				if err := t.consumeNumber(); err != nil {
					return nil, err
				}
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

				t.tokens = append(t.tokens, NewDataToken(tt, start, NewPosition(t.row, t.col)))
				continue
			}

			t.tokens = append(t.tokens, NewDataToken(delimMap[char], start, NewPosition(t.row, t.col)))
		case '=':
			start := NewPosition(t.row, t.col)
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
				t.tokens = append(t.tokens, NewDataToken(TokenType_FatArrow, start, NewPosition(t.row, t.col)))
				continue
			case next[0] == '=':

				if err := t.stream.Discard(1); err != nil {
					return nil, err
				}
				t.row++
				t.tokens = append(t.tokens, NewDataToken(TokenType_EqualEqual, start, NewPosition(t.row, t.col)))
				continue
			}
			t.tokens = append(t.tokens, NewDataToken(delimMap[char], start, NewPosition(t.row, t.col)))
		case '>':
			start := NewPosition(t.row, t.col)
			isNext, err := t.isNext('=')
			if err != nil {
				return nil, err
			}
			if isNext {

				if err := t.stream.Discard(1); err != nil {
					return nil, err
				}
				t.row++
				t.tokens = append(t.tokens, NewDataToken(TokenType_GreaterThenOrEqaul, start, NewPosition(t.row, t.col)))
				continue
			}
			t.tokens = append(t.tokens, NewDataToken(delimMap[char], start, NewPosition(t.row, t.col)))
		case '<':
			start := NewPosition(t.row, t.col)
			isNext, err := t.isNext('=')
			if err != nil {
				return nil, err
			}
			if isNext {
				if err := t.stream.Discard(1); err != nil {
					return nil, err
				}
				t.row++
				t.tokens = append(t.tokens, NewDataToken(TokenType_LessThenOrEqual, start, NewPosition(t.row, t.col)))
				continue
			}
			t.tokens = append(t.tokens, NewDataToken(delimMap[char], start, NewPosition(t.row, t.col)))
		case '*':
			start := NewPosition(t.row, t.col)
			isNext, err := t.isNext('*')
			if err != nil {
				return nil, err
			}
			if isNext {
				if err := t.stream.Discard(1); err != nil {
					return nil, err
				}
				t.row++
				t.tokens = append(t.tokens, NewDataToken(TokenType_Power, start, NewPosition(t.row, t.col)))
				continue
			}
			t.tokens = append(t.tokens, NewDataToken(delimMap[char], start, NewPosition(t.row, t.col)))
		case '(', ')', ';', '%', '?', ':', ',', '[', ']', '{', '}':
			t.tokens = append(t.tokens, NewDataToken(delimMap[char], NewPosition(t.row, t.col), NewPosition(t.row, t.col)))
		case '|':
			next, err := t.isNext('|')
			if err != nil {
				return nil, err
			}
			if next {
				start := NewPosition(t.col, t.row)
				if err := t.stream.Discard(1); err != nil {
					return nil, err
				}
				t.row++
				t.tokens = append(t.tokens, NewDataToken(TokenType_OR, start, NewPosition(t.row, t.col)))
				continue
			}

			return nil, ErrUnexpectedCharacter
		case '&':
			next, err := t.isNext('&')
			if err != nil {
				return nil, err
			}
			if next {
				start := NewPosition(t.col, t.row)
				if err := t.stream.Discard(1); err != nil {
					return nil, err
				}
				t.row++
				t.tokens = append(t.tokens, NewDataToken(TokenType_AND, start, NewPosition(t.row, t.col)))
				continue
			}

			return nil, ErrUnexpectedCharacter
		case '!':
			next, err := t.isNext('=')
			if err != nil {
				return nil, err
			}
			if next {
				start := NewPosition(t.col, t.row)
				if err := t.stream.Discard(1); err != nil {
					return nil, err
				}
				t.row++
				t.tokens = append(t.tokens, NewDataToken(TokenType_NotEqual, start, NewPosition(t.row, t.col)))
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
	start := NewPosition(t.col, t.row)
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
	end := NewPosition(t.col, t.row)

	switch value {
	case "false", "true", "import", "break", "from", "let", "return", "while", "impl", "if", "else", "struct", "fn", "mut":
		t.tokens = append(t.tokens, NewKeywordToken(value, start, end))
	case "null", "int", "int64", "int32", "int16", "int8", "uint", "u64", "u32", "u16", "u8", "bool":
		t.tokens = append(t.tokens, NewIdentToken(value, start, end))
	default:
		t.tokens = append(t.tokens, NewIdentToken(value, start, end))
	}

	return nil
}

func (t *Tokenizer) consumeDigits(rep *strings.Builder) error {
	for {
		char, _, err := t.stream.ReadRune()
		if err != nil && err != io.EOF {
			return err
		}
		if err == io.EOF {
			break
		}
		t.row++

		if unicode.IsDigit(char) {
			rep.WriteRune(char)
			continue
		}

		if err := t.stream.UnreadRune(); err != nil {
			return err
		}
		t.row--
		break
	}

	return nil
}

func (t *Tokenizer) consumeNumber() error {
	start := NewPosition(t.col, t.row)
	seenDot := false
	value := strings.Builder{}

	//#region +-. check
	next, err := t.stream.Peek(2)
	if err != nil && err != io.EOF {
		return err
	}
	if len(next) > 0 && (next[0] == '+' || next[0] == '-' || next[0] == '.') {
		if err := t.stream.Discard(1); err != nil {
			return err
		}
		t.row++
		value.WriteRune(next[0])

		if len(next) > 1 && next[0] != '.' && next[1] == '.' {
			seenDot = true
			if err := t.stream.Discard(1); err != nil {
				return err
			}
			t.row++
			value.WriteRune(next[1])
		}
	}
	//#endregion

	if err := t.consumeDigits(&value); err != nil {
		return err
	}

	//#region .<DIGIT> check !seenDot
	next, err = t.stream.Peek(2)
	if err != nil && err != io.EOF {
		return err
	}

	if len(next) >= 2 && next[0] == '.' {
		if seenDot {
			return ErrUnexpectedCharacter
		}

		if unicode.IsDigit(next[1]) {
			if err := t.stream.Discard(2); err != nil {
				return err
			}
			t.row += 2
			value.WriteRune(next[0])
			value.WriteRune(next[1])

			if err := t.consumeDigits(&value); err != nil {
				return err
			}
		}
	}

	//#region e notion check
	next, err = t.stream.Peek(3)
	if err != nil && err != io.EOF {
		return err
	}

	if len(next) >= 2 && (next[0] == 'e' || next[0] == 'E') {
		switch {
		case len(next) >= 3 && (next[1] == '-' || next[1] == '+') && unicode.IsDigit(next[2]):
			if err := t.stream.Discard(3); err != nil {
				return err
			}
			value.WriteString(string(next))

			if err := t.consumeDigits(&value); err != nil {
				return err
			}
		case unicode.IsDigit(next[1]):
			if err := t.stream.Discard(2); err != nil {
				return err
			}
			t.row++
			value.WriteString(string(next[:2]))

			if err := t.consumeDigits(&value); err != nil {
				return err
			}
		}
	}
	//#endregion

	end := NewPosition(t.col, t.row)
	t.tokens = append(t.tokens, NewNumberToken(value.String(), start, end))

	return nil
}

func (t *Tokenizer) consumeString() error {
	start := NewPosition(t.col, t.row)
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

	end := NewPosition(t.col, t.row)

	t.tokens = append(t.tokens, NewStringToken(value.String(), start, end))

	return nil
}
