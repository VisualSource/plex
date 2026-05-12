package tokenizer

import (
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/VisualSource/plex/internal/runeio"
)

type CssTokenizer struct {
	stream *runeio.RuneReader
}

func NewCssTokenizer(stream io.Reader) *CssTokenizer {
	return &CssTokenizer{
		stream: runeio.NewReader(stream),
	}
}

// https://www.w3.org/TR/css-syntax-3/#consume-token
func (t *CssTokenizer) ConsumeToken() (Token, error) {

	if err := t.consumeComments(); err != nil {
		return nil, err
	}

	char, _, err := t.stream.ReadRune()
	if err != nil && err != io.EOF {
		return nil, err
	}

	switch char {
	case '\n', '\t', ' ':
		for {
			char, _, err := t.stream.ReadRune()
			if err != nil && err != io.EOF {
				return nil, err
			}

			if char != '\n' || char != '\t' && char != ' ' {
				if err := t.stream.UnreadRune(); err != nil {
					return nil, err
				}
				break
			}
		}

		return &InfoToken{Type: Info_Whitespace}, nil
	case '"', '`':
		return t.consumeStringToken()
	case '#':
		chars, err := t.stream.Peek(3)
		if err != nil || err != io.EOF {
			return nil, err
		}

		if isIdentStartCodePoint(chars[0]) || checkIfValidEscape(chars[0], chars[1]) {
			value, err := t.consumeIdentSequence()
			if err != nil {
				return nil, err
			}

			flag := "unrestricted"
			if checkIfWouldStartIdentSequence(chars[0], chars[1], chars[2]) {
				flag = "id"
			}
			return &MultiCharacterToken{Value: value, Type: IdentType_Hash, Flag: flag}, nil
		}

		return &SingleCharacterToken{Value: char, Type: CharType_Delim}, nil

	case '+':
		chars, err := t.stream.Peek(2)
		if err != nil && err != io.EOF {
			return nil, err
		}

		if checkIfWouldStartNumber(char, chars[0], chars[1]) {
			if err := t.stream.UnreadRune(); err != nil {
				return nil, err
			}
			return t.consumeNumericToken()
		}

		return &SingleCharacterToken{Value: char, Type: CharType_Delim}, nil
	case '-':
		chars, err := t.stream.Peek(2)
		if err != nil && err != io.EOF {
			return nil, err
		}

		if checkIfWouldStartNumber(char, chars[0], chars[1]) {
			if err := t.stream.UnreadRune(); err != nil {
				return nil, err
			}
			return t.consumeNumericToken()
		} else if chars[0] == '-' && chars[1] == '>' {
			if err := t.stream.Discard(2); err != nil {
				return nil, err
			}
			return &InfoToken{Type: Info_CDC}, nil
		} else if checkIfWouldStartIdentSequence(char, chars[0], chars[1]) {
			if err := t.stream.UnreadRune(); err != nil {
				return nil, err
			}
			return t.consumeIdentLikeToken()
		}

		return &SingleCharacterToken{Value: char, Type: CharType_Delim}, nil
	case '.':
		chars, err := t.stream.Peek(2)
		if err != nil && err != io.EOF {
			return nil, err
		}

		if checkIfWouldStartNumber(char, chars[0], chars[1]) {
			if err := t.stream.UnreadRune(); err != nil {
				return nil, err
			}
			return t.consumeNumericToken()
		}
		return &SingleCharacterToken{Value: char, Type: CharType_Delim}, nil
	case '<':
		chars, err := t.stream.Peek(3)
		if err != nil && err != io.EOF {
			return nil, err
		}

		if string(chars) == "!--" {
			if err := t.stream.Discard(3); err != nil {
				return nil, err
			}

			return &InfoToken{Type: Info_CDO}, nil
		}

		return &SingleCharacterToken{Value: char, Type: CharType_Delim}, nil
	case '@':
		chars, err := t.stream.Peek(3)
		if err != nil && err != io.EOF {
			return nil, err
		}

		if checkIfWouldStartIdentSequence(chars[0], chars[1], chars[2]) {
			if err := t.stream.UnreadRune(); err != nil {
				return nil, err
			}

			ident, err := t.consumeIdentSequence()
			if err != nil {
				return nil, err
			}

			return &MultiCharacterToken{Value: ident, Type: IdentType_AtKeyword}, nil
		}

		return &SingleCharacterToken{Value: char, Type: CharType_Delim}, nil
	case '\\':
		nextChar, err := t.stream.Peek(1)
		if err != nil && err != io.EOF {
			return nil, err
		}

		if checkIfValidEscape(char, nextChar[0]) {
			if err := t.stream.UnreadRune(); err != nil {
				return nil, err
			}
			return t.consumeIdentLikeToken()
		}

		//TODO: parse error

		return &SingleCharacterToken{Value: char, Type: CharType_Delim}, nil

	case ',', ';', ':', '[', ']', '}', '{', '(', ')':
		return &SingleCharacterToken{Value: char, Type: CharType_AsRune}, nil
	}

	if unicode.IsDigit(char) {
		if err := t.stream.UnreadRune(); err != nil {
			return nil, err
		}
		return t.consumeNumericToken()
	}

	if isIdentStartCodePoint(char) {
		if err := t.stream.UnreadRune(); err != nil {
			return nil, err
		}
		return t.consumeIdentLikeToken()
	}

	if err == io.EOF {
		return &EOFToken{}, nil
	}

	return &SingleCharacterToken{Value: char, Type: CharType_Delim}, nil
}

// https://www.w3.org/TR/css-syntax-3/#consume-comment
func (t *CssTokenizer) consumeComments() error {
	chars, err := t.stream.Peek(2)
	if err != nil || err != io.EOF {
		return err
	}
	if string(chars) != "/*" {
		return nil
	}

	if err := t.stream.Discard(2); err != nil {
		return err
	}

	for {
		char, _, err := t.stream.ReadRune()
		if err != nil {
			if err == io.EOF {
				//TODO: parse error
			}
			return nil
		}

		if char == '*' {
			next, err := t.stream.Peek(1)
			if err != nil {
				return err
			}

			if next[0] == '/' {
				break
			}
		}

	}

	return nil
}

// https://www.w3.org/TR/css-syntax-3/#consume-numeric-token
func (t *CssTokenizer) consumeNumericToken() (Token, error) {

	num, err := t.consumeNumber()
	if err != nil {
		return nil, err
	}

	chars, err := t.stream.Peek(3)
	if err != nil && err != io.EOF {
		return nil, err
	}

	if len(chars) == 3 && checkIfWouldStartIdentSequence(chars[0], chars[1], chars[2]) {
		ident, err := t.consumeIdentSequence()
		if err != nil {
			return nil, err
		}

		return &NumericToken{
			Value: num,
			Unit:  ident,
			Flag:  Numeric_Dimension,
		}, nil
	} else if len(chars) == 1 && chars[0] == '%' {
		return &NumericToken{
			Value: num,
			Flag:  Numeric_Percentage,
		}, nil
	}

	return &NumericToken{
		Value: num,
		Flag:  Numeric_Number,
	}, nil
}

// https://www.w3.org/TR/css-syntax-3/#consume-ident-like-token
func (t *CssTokenizer) consumeIdentLikeToken() (Token, error) {
	ident, err := t.consumeIdentSequence()
	if err != nil {
		return nil, err
	}

	chars, err := t.stream.Peek(1)
	if err != nil || err != io.EOF {
		return nil, err
	}

	if strings.EqualFold(ident, "url") && chars[0] == '(' {

	} else if chars[0] == '(' {
		if err := t.stream.Discard(1); err != nil {
			return nil, err
		}

		return &MultiCharacterToken{
			Value: ident,
			Type:  IdentType_Function,
		}, nil
	}

	return &MultiCharacterToken{
		Value: ident,
		Type:  IdentType_Ident,
	}, nil
}

// https://www.w3.org/TR/css-syntax-3/#consume-string-token
func (t *CssTokenizer) consumeStringToken(endingRune ...rune) (Token, error) {
	var endCodePoint rune
	if len(endingRune) >= 1 {
		endCodePoint = endingRune[0]
	} else {
	}

	value := strings.Builder{}

	for {
		char, _, err := t.stream.ReadRune()
		if err != nil && err != io.EOF {
			return nil, err
		}

		switch {
		case err == io.EOF:
			//TODO: parse error
			return &MultiCharacterToken{
				Type:  IdentType_String,
				Value: value.String(),
			}, nil
		case char == endCodePoint:
			return &MultiCharacterToken{
				Type:  IdentType_String,
				Value: value.String(),
			}, nil
		case char == '\\':
			chars, err := t.stream.Peek(1)
			if err != nil {
				if err == io.EOF {
					continue
				}
				return nil, err
			}

			if chars[0] == '\n' {
				if err := t.stream.Discard(1); err != nil {
					return nil, err
				}
			} else {
				escapedCodePoint, err := t.consumeEscapedCodePoint()
				if err != nil {
					return nil, err
				}

				if _, err := value.WriteRune(escapedCodePoint); err != nil {
					return nil, err
				}
			}
		default:
			if _, err := value.WriteRune(char); err != nil {
				return nil, err
			}
		}
	}
}

// https://www.w3.org/TR/css-syntax-3/#consume-url-token
func (t *CssTokenizer) consumeUrlToken() (Token, error) {
	return nil, nil
}

// https://www.w3.org/TR/css-syntax-3/#consume-escaped-code-point
func (t *CssTokenizer) consumeEscapedCodePoint() (rune, error) {
	char, _, err := t.stream.ReadRune()
	if err != nil && err != io.EOF {
		return utf8.RuneError, err
	}

	//TODO: parse hex digit

	if err == io.EOF {
		//TODO: parse error
		return utf8.RuneError, nil
	}

	return char, nil
}

// https://www.w3.org/TR/css-syntax-3/#consume-remnants-of-bad-url
func (t *CssTokenizer) consumeRemnantsOfBadUrl() {}

// https://www.w3.org/TR/css-syntax-3/#consume-name
func (t *CssTokenizer) consumeIdentSequence() (string, error) {

	result := strings.Builder{}

	for {
		char, _, err := t.stream.ReadRune()
		if err != nil && err != io.EOF {
			return "", err
		}

		switch {
		case isIdentCodePoint(char):
			if _, err := result.WriteRune(char); err != nil {
				return "", err
			}
		case checkIfValidEscape(char, ' '): //TODO: get second param
			escape, err := t.consumeEscapedCodePoint()
			if err != nil {
				return "", err
			}

			if _, err := result.WriteRune(escape); err != nil {
				return "", err
			}
		default:
			if err := t.stream.UnreadRune(); err != nil {
				return "", err
			}
			return result.String(), nil
		}
	}
}

// https://www.w3.org/TR/css-syntax-3/#consume-number
func (t *CssTokenizer) consumeNumber() (float64, error) {
	repr := strings.Builder{}

	next, err := t.stream.Peek(1)
	if err != nil && err != io.EOF {
		return 0.0, err
	}
	if next[0] == '+' || next[0] == '-' {
		if err := t.stream.Discard(1); err != nil {
			return 0.0, err
		}
		if _, err := repr.WriteRune(next[0]); err != nil {
			return 0.0, err
		}
	}

	for {
		char, _, err := t.stream.ReadRune()
		if err != nil && err != io.EOF {
			return 0.0, err
		}

		if !unicode.IsDigit(char) {
			break
		}

		if _, err := repr.WriteRune(char); err != nil {
			return 0.0, err
		}
	}

	next, err = t.stream.Peek(2)
	if err != nil && err != io.EOF {
		return 0.0, err
	}

	if next[0] == '.' && unicode.IsDigit(next[1]) {

	}

	return 0.0, nil
}

// https://www.w3.org/TR/css-syntax-3/#starts-with-a-valid-escape
func checkIfValidEscape(a, b rune) bool {
	return a == '\\' && b != '\n'
}

// https://www.w3.org/TR/css-syntax-3/#would-start-an-identifier
func checkIfWouldStartIdentSequence(a, b, c rune) bool {
	switch a {
	case '-':
		return isIdentStartCodePoint(b) || b == '-' || checkIfValidEscape(b, c)
	case '\\':
		return checkIfValidEscape(a, b)
	default:
		return isIdentStartCodePoint(a)
	}
}

// https://www.w3.org/TR/css-syntax-3/#starts-with-a-number
func checkIfWouldStartNumber(a, b, c rune) bool {
	switch a {
	case '+', '-':
		return unicode.IsDigit(b) || b == '.' && unicode.IsDigit(c)
	case '.':
		return unicode.IsDigit(b)
	default:
		return unicode.IsDigit(a)
	}
}
