package tokenizer

import (
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/VisualSource/plex/internal/runeio"
)

type CssTokenizer struct {
	stream *runeio.RuneReader
}

func NewCssTokenizer(stream io.Reader) *CssTokenizer {
	return &CssTokenizer{
		stream: runeio.NewReader(newPreprocessor(stream)),
	}
}

// https://www.w3.org/TR/css-syntax-3/#consume-token
func (t *CssTokenizer) ConsumeToken() (Token, error) {
	defer func() {
		t.stream.Forget()
	}()

	if err := t.consumeComments(); err != nil {
		return nil, err
	}

	char, _, err := t.stream.ReadRune()
	if err != nil && err != io.EOF {
		return nil, err
	}

	if err == io.EOF {
		return &EOFToken{}, nil
	}

	switch char {
	case '\n', '\t', ' ':
		if err := t.consumeWhitespace(); err != nil {
			return nil, err
		}

		return NewDataToken(TokenId_Whitespace), nil
	case '"', '\'':
		return t.consumeStringToken(char)
	case '#':
		chars, err := t.stream.Peek(3)
		if err != nil && err != io.EOF {
			return nil, err
		}
		chars = padRunes(chars, 3)

		if isIdentCodePoint(chars[0]) || checkIfValidEscape(chars[0], chars[1]) {
			value, err := t.consumeIdentSequence()
			if err != nil {
				return nil, err
			}

			token := NewMultiCharacterToken(TokenId_Hash, value)
			token.Flag = "unrestricted"
			if checkIfWouldStartIdentSequence(chars[0], chars[1], chars[2]) {
				token.Flag = "id"
			}
			return token, nil
		}

		return NewSingleCharacterToken(TokenId_Delim, char), nil

	case '+':
		chars, err := t.stream.Peek(2)
		if err != nil && err != io.EOF {
			return nil, err
		}
		chars = padRunes(chars, 2)

		if checkIfWouldStartNumber(char, chars[0], chars[1]) {
			if err := t.stream.UnreadRune(); err != nil {
				return nil, err
			}
			return t.consumeNumericToken()
		}

		return NewSingleCharacterToken(TokenId_Delim, char), nil
	case '-':
		chars, err := t.stream.Peek(2)
		if err != nil && err != io.EOF {
			return nil, err
		}
		chars = padRunes(chars, 2)

		if checkIfWouldStartNumber(char, chars[0], chars[1]) {
			if err := t.stream.UnreadRune(); err != nil {
				return nil, err
			}
			return t.consumeNumericToken()
		} else if chars[0] == '-' && chars[1] == '>' {
			if err := t.stream.Discard(2); err != nil {
				return nil, err
			}
			return NewDataToken(TokenId_CDC), nil
		} else if checkIfWouldStartIdentSequence(char, chars[0], chars[1]) {
			if err := t.stream.UnreadRune(); err != nil {
				return nil, err
			}
			return t.consumeIdentLikeToken()
		}

		return NewSingleCharacterToken(TokenId_Delim, char), nil
	case '.':
		chars, err := t.stream.Peek(2)
		if err != nil && err != io.EOF {
			return nil, err
		}
		chars = padRunes(chars, 2)

		if checkIfWouldStartNumber(char, chars[0], chars[1]) {
			if err := t.stream.UnreadRune(); err != nil {
				return nil, err
			}
			return t.consumeNumericToken()
		}
		return NewSingleCharacterToken(TokenId_Delim, char), nil
	case '<':
		chars, err := t.stream.Peek(3)
		if err != nil && err != io.EOF {
			return nil, err
		}

		if string(chars) == "!--" {
			if err := t.stream.Discard(3); err != nil {
				return nil, err
			}

			return NewDataToken(TokenId_CDO), nil
		}

		return NewSingleCharacterToken(TokenId_Delim, char), nil
	case '@':
		chars, err := t.stream.Peek(3)
		if err != nil && err != io.EOF {
			return nil, err
		}
		chars = padRunes(chars, 3)

		if checkIfWouldStartIdentSequence(chars[0], chars[1], chars[2]) {
			ident, err := t.consumeIdentSequence()
			if err != nil {
				return nil, err
			}

			return NewMultiCharacterToken(TokenId_AtKeyword, ident), nil
		}

		return NewSingleCharacterToken(TokenId_Delim, char), nil
	case '\\':
		nextChar, err := t.stream.Peek(1)
		if err != nil && err != io.EOF {
			return nil, err
		}
		nextChar = padRunes(nextChar, 1)

		if checkIfValidEscape(char, nextChar[0]) {
			if err := t.stream.UnreadRune(); err != nil {
				return nil, err
			}
			return t.consumeIdentLikeToken()
		}

		//TODO: parse error

		return NewSingleCharacterToken(TokenId_Delim, char), nil

	case ',':
		return NewDataToken(TokenId_Comma), nil
	case ';':
		return NewDataToken(TokenId_Semicolon), nil
	case ':':
		return NewDataToken(TokenId_Colon), nil
	case '[':
		return NewDataToken(TokenId_BracketSquareOpen), nil
	case ']':
		return NewDataToken(TokenId_BracketSquareClose), nil
	case '{':
		return NewDataToken(TokenId_BracketCurlyOpen), nil
	case '}':
		return NewDataToken(TokenId_BracketCurlyClose), nil
	case '(':
		return NewDataToken(TokenId_BracketParamOpen), nil
	case ')':
		return NewDataToken(TokenId_BracketParamClose), nil
	}

	if isDigit(char) {
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

	return NewSingleCharacterToken(TokenId_Delim, char), nil
}

func (t *CssTokenizer) consumeWhitespace() error {
	for {
		char, _, err := t.stream.ReadRune()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if isWhitespace(char) {
			continue
		}

		if err := t.stream.UnreadRune(); err != nil {
			return err
		}
		break
	}

	return nil
}

// https://www.w3.org/TR/css-syntax-3/#consume-comment
func (t *CssTokenizer) consumeComments() error {
	for {
		chars, err := t.stream.Peek(2)
		if err != nil && err != io.EOF {
			return err
		}
		if len(chars) != 2 || string(chars) != "/*" {
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
					return nil
				}
				return err
			}

			if char == '*' {
				next, err := t.stream.Peek(1)
				if err != nil {
					return err
				}

				if len(next) == 1 && next[0] == '/' {
					if err := t.stream.Discard(1); err != nil {
						return err
					}
					break
				}
			}

		}
	}
}

// https://www.w3.org/TR/css-syntax-3/#consume-numeric-token
func (t *CssTokenizer) consumeNumericToken() (Token, error) {

	num, numType, err := t.consumeNumber()
	if err != nil {
		return nil, err
	}

	chars, err := t.stream.Peek(3)
	if err != nil && err != io.EOF {
		return nil, err
	}

	var a, b, c rune
	if len(chars) > 0 {
		a = chars[0]
	}
	if len(chars) > 1 {
		b = chars[1]
	}
	if len(chars) > 2 {
		c = chars[2]
	}

	if checkIfWouldStartIdentSequence(a, b, c) {
		ident, err := t.consumeIdentSequence()
		if err != nil {
			return nil, err
		}

		token := NewNumericToken(TokenId_Dimension, num)
		token.Flag = numType
		token.Unit = ident

		return token, nil
	} else if a == '%' {
		if err := t.stream.Discard(1); err != nil {
			return nil, err
		}
		return NewNumericToken(TokenId_Percentage, num), nil
	}

	token := NewNumericToken(TokenId_Number, num)
	token.Flag = numType

	return token, nil
}

func isQuote(r rune) bool { return r == '"' || r == '\'' }

// https://www.w3.org/TR/css-syntax-3/#consume-ident-like-token
func (t *CssTokenizer) consumeIdentLikeToken() (Token, error) {
	ident, err := t.consumeIdentSequence()
	if err != nil {
		return nil, err
	}

	chars, err := t.stream.Peek(1)
	if err != nil && err != io.EOF {
		return nil, err
	}

	var next rune
	if len(chars) > 0 {
		next = chars[0]
	}

	if strings.EqualFold(ident, "url") && next == '(' {
		if err := t.stream.Discard(1); err != nil {
			return nil, err
		}
		for {
			ws, err := t.stream.Peek(2)
			if err != nil && err != io.EOF {
				return nil, err
			}
			if len(ws) < 2 || !isWhitespace(ws[0]) || !isWhitespace(ws[1]) {
				break
			}
			if err := t.stream.Discard(1); err != nil {
				return nil, err
			}
		}
		la, err := t.stream.Peek(2)
		if err != nil && err != io.EOF {
			return nil, err
		}

		switch {
		case len(la) >= 1 && isQuote(la[0]):
			return NewMultiCharacterToken(TokenId_Function, ident), nil
		case len(la) >= 2 && isWhitespace(la[0]) && isQuote(la[1]):
			return NewMultiCharacterToken(TokenId_Function, ident), nil
		default:
			return t.consumeUrlToken()
		}
	}

	if next == '(' {
		if err := t.stream.Discard(1); err != nil {
			return nil, err
		}

		return NewMultiCharacterToken(TokenId_Function, ident), nil
	}

	return NewMultiCharacterToken(TokenId_Ident, ident), nil
}

// https://www.w3.org/TR/css-syntax-3/#consume-string-token
func (t *CssTokenizer) consumeStringToken(endingRune rune) (Token, error) {
	value := strings.Builder{}

	for {
		char, _, err := t.stream.ReadRune()
		if err != nil && err != io.EOF {
			return nil, err
		}

		switch {
		case err == io.EOF:
			//TODO: parse error
			return NewMultiCharacterToken(TokenId_String, value.String()), nil
		case char == endingRune:
			return NewMultiCharacterToken(TokenId_String, value.String()), nil
		case char == '\n':
			if err := t.stream.UnreadRune(); err != nil {
				return nil, err
			}

			return NewDataToken(TokenId_BadString), nil

		case char == '\\':
			chars, err := t.stream.Peek(1)
			if err != nil && err != io.EOF {
				return nil, err
			}

			// Next input code point is EOF: do nothing (per spec).
			if len(chars) == 0 {
				continue
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
	if err := t.consumeWhitespace(); err != nil {
		return nil, err
	}

	value := strings.Builder{}

	for {
		char, _, err := t.stream.ReadRune()
		if err != nil {
			if err == io.EOF {
				//TODO: parse error
				return NewMultiCharacterToken(TokenId_Url, value.String()), nil
			}
			return nil, err
		}

		switch char {
		case ')':
			return NewMultiCharacterToken(TokenId_Url, value.String()), nil
		case '\n', '\t', ' ':
			if err := t.consumeWhitespace(); err != nil {
				return nil, err
			}
			chars, err := t.stream.Peek(1)
			if err != nil && err != io.EOF {
				return nil, err
			}

			if len(chars) == 0 || chars[0] == ')' {
				if len(chars) == 0 {
					//TODO: parse error
				} else {
					if err := t.stream.Discard(1); err != nil {
						return nil, err
					}
				}

				return NewMultiCharacterToken(TokenId_Url, value.String()), nil
			}

			if err := t.consumeRemnantsOfBadUrl(); err != nil {
				return nil, err
			}
			return NewDataToken(TokenId_BadUrl), nil
		case '"', '\'', '(':
			//TODO: parse error
			if err := t.consumeRemnantsOfBadUrl(); err != nil {
				return nil, err
			}
			return NewDataToken(TokenId_BadUrl), nil
		case '\\':
			chars, err := t.stream.Peek(1)
			if err != nil && err != io.EOF {
				return nil, err
			}

			if len(chars) > 0 && checkIfValidEscape(char, chars[0]) {
				escaped, err := t.consumeEscapedCodePoint()
				if err != nil {
					return nil, err
				}

				value.WriteRune(escaped)
			} else {
				//TODO: parse error
				if err := t.consumeRemnantsOfBadUrl(); err != nil {
					return nil, err
				}
				return NewDataToken(TokenId_BadUrl), nil
			}

		default:
			if isNonPrintableCodePoint(char) {
				//TODO: parse error
				if err := t.consumeRemnantsOfBadUrl(); err != nil {
					return nil, err
				}
				return NewDataToken(TokenId_BadUrl), nil
			}

			if _, err := value.WriteRune(char); err != nil {
				return nil, err
			}
		}

	}
}

// https://www.w3.org/TR/css-syntax-3/#consume-escaped-code-point
func (t *CssTokenizer) consumeEscapedCodePoint() (rune, error) {
	char, _, err := t.stream.ReadRune()
	if err != nil && err != io.EOF {
		return utf8.RuneError, err
	}

	if isHexDigit(char) {
		hex := []rune{char}
		for len(hex) < 6 {
			peeked, err := t.stream.Peek(1)
			if err != nil && err != io.EOF {
				return utf8.RuneError, err
			}
			if len(peeked) == 0 || !isHexDigit(peeked[0]) {
				break
			}
			if err := t.stream.Discard(1); err != nil {
				return utf8.RuneError, err
			}
			hex = append(hex, peeked[0])
		}

		// consume one trailing whitespace if present
		peeked, err := t.stream.Peek(1)
		if err != nil && err != io.EOF {
			return utf8.RuneError, err
		}
		if len(peeked) > 0 && isWhitespace(peeked[0]) {
			if err := t.stream.Discard(1); err != nil {
				return utf8.RuneError, err
			}
		}

		n, err := strconv.ParseUint(string(hex), 16, 32)
		if err != nil {
			return utf8.RuneError, err
		}
		if n == 0 || (n >= 0xD800 && n <= 0xDFFF) || n > 0x10FFFF {
			return utf8.RuneError, nil
		}
		return rune(n), nil
	}

	if err == io.EOF {
		//TODO: parse error
		return utf8.RuneError, nil
	}

	return char, nil
}

// https://www.w3.org/TR/css-syntax-3/#consume-name
func (t *CssTokenizer) consumeIdentSequence() (string, error) {

	result := strings.Builder{}

	for {
		char, _, err := t.stream.ReadRune()
		if err != nil {
			if err == io.EOF {
				return result.String(), nil
			}
			return "", err
		}

		if isIdentCodePoint(char) {
			if _, err := result.WriteRune(char); err != nil {
				return "", err
			}
			continue
		}

		next, err := t.stream.Peek(1)
		if err != nil {
			return "", err
		}
		next = padRunes(next, 1)

		if checkIfValidEscape(char, next[0]) {
			escape, err := t.consumeEscapedCodePoint()
			if err != nil {
				return "", err
			}

			if _, err := result.WriteRune(escape); err != nil {
				return "", err
			}

			continue
		}

		if err := t.stream.UnreadRune(); err != nil {
			return "", err
		}
		return result.String(), nil
	}
}

// https://www.w3.org/TR/css-syntax-3/#consume-number
func (t *CssTokenizer) consumeNumber() (float64, string, error) {
	repr := strings.Builder{}
	numType := "integer"

	next, err := t.stream.Peek(1)
	if err != nil && err != io.EOF {
		return 0.0, "", err
	}
	if len(next) > 0 && (next[0] == '+' || next[0] == '-') {
		if err := t.stream.Discard(1); err != nil {
			return 0.0, numType, err
		}
		if _, err := repr.WriteRune(next[0]); err != nil {
			return 0.0, numType, err
		}
	}

	if err := t.consumeDigits(&repr); err != nil {
		return 0, "", err
	}

	next, err = t.stream.Peek(2)
	if err != nil && err != io.EOF {
		return 0.0, "", err
	}

	if len(next) == 2 && next[0] == '.' && isDigit(next[1]) {
		if err := t.stream.Discard(2); err != nil {
			return 0.0, "", err
		}
		repr.WriteRune(next[0])
		repr.WriteRune(next[1])
		numType = "number"
		if err := t.consumeDigits(&repr); err != nil {
			return 0, "", err
		}
	}

	next, err = t.stream.Peek(3)
	if err != nil && err != io.EOF {
		return 0.0, "", err
	}

	if len(next) >= 2 && (next[0] == 'e' || next[0] == 'E') {
		switch {
		case len(next) >= 3 && (next[1] == '-' || next[1] == '+') && isDigit(next[2]):
			if err := t.stream.Discard(3); err != nil {
				return 0.0, "", err
			}
			repr.WriteString(string(next))
			numType = "number"
			if err := t.consumeDigits(&repr); err != nil {
				return 0, "", err
			}
		case isDigit(next[1]):
			if err := t.stream.Discard(2); err != nil {
				return 0.0, "", err
			}
			repr.WriteString(string(next[:2]))
			numType = "number"
			if err := t.consumeDigits(&repr); err != nil {
				return 0, "", err
			}
		}
	}

	value, err := strconv.ParseFloat(repr.String(), 64)
	if err != nil {
		return 0.0, "", err
	}

	return value, numType, nil
}

func (t *CssTokenizer) consumeDigits(builder *strings.Builder) error {
	for {
		char, _, err := t.stream.ReadRune()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if !isDigit(char) {
			return t.stream.UnreadRune()
		}

		builder.WriteRune(char)
	}
}

// https://www.w3.org/TR/css-syntax-3/#consume-remnants-of-bad-url
func (t *CssTokenizer) consumeRemnantsOfBadUrl() error {

	for {
		char, _, err := t.stream.ReadRune()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if char == ')' {
			return nil
		}

		next, err := t.stream.Peek(1)
		if err != nil && err != io.EOF {
			return err
		}
		if len(next) == 1 && checkIfValidEscape(char, next[0]) {
			if _, err := t.consumeEscapedCodePoint(); err != nil {
				return err
			}
		}
	}
}
