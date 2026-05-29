package css_tokenizer

import "unicode"

func isWhitespace(char rune) bool {
	return char == '\n' || char == '\t' || char == ' '
}

// https://www.w3.org/TR/css-syntax-3/#ident-start-code-point
func isIdentStartCodePoint(char rune) bool {
	return unicode.IsLetter(char) || isNonASCIIIdentCodePoint(char) || char == '_'
}

// https://www.w3.org/TR/css-syntax-3/#non-ascii-ident-code-point
func isNonASCIIIdentCodePoint(char rune) bool {
	switch {
	case char == 0x00B7:
		return true
	case char >= 0x00C0 && char <= 0x00D6:
		return true
	case char >= 0x00D8 && char <= 0x00F6:
		return true
	case char >= 0x00F8 && char <= 0x037D:
		return true
	case char >= 0x037F && char <= 0x1FFF:
		return true
	case char == 0x200C, char == 0x200D, char == 0x203F, char == 0x2040:
		return true
	case char >= 0x2070 && char <= 0x218F:
		return true
	case char >= 0x2C00 && char <= 0x2FEF:
		return true
	case char >= 0x3001 && char <= 0xD7FF:
		return true
	case char >= 0xF900 && char <= 0xFDCF:
		return true
	case char >= 0xFDF0 && char <= 0xFFFD:
		return true
	case char >= 0x10000 && char <= 0xEFFFF:
		return true
	}
	return false
}

func isIdentCodePoint(char rune) bool {
	return isIdentStartCodePoint(char) || isDigit(char) || char == '-'
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

// https://drafts.csswg.org/css-syntax/#check-if-three-code-points-would-start-a-unicode-range
func checkIfWouldStartUnicodeRange(a, b, c rune) bool {
	return (a == 'u' || a == 'U') && b == '+' && (c == '?' || isHexDigit(c))
}

// https://www.w3.org/TR/css-syntax-3/#starts-with-a-number
func checkIfWouldStartNumber(a, b, c rune) bool {
	switch a {
	case '+', '-':
		return isDigit(b) || b == '.' && isDigit(c)
	case '.':
		return isDigit(b)
	default:
		return isDigit(a)
	}
}

// https://www.w3.org/TR/css-syntax-3/#non-printable-code-point
func isNonPrintableCodePoint(char rune) bool {
	a := int(char)
	return a >= 0x0000 && a <= 0x0008 || a == 0x000B || a >= 0x000E && a <= 0x001F || a == 0x007F
}

func isHexDigit(r rune) bool {
	return (r >= '0' && r <= '9') ||
		(r >= 'a' && r <= 'f') ||
		(r >= 'A' && r <= 'F')
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

// padRunes returns a slice of length n. If chars is shorter (e.g. a short
// Peek result near EOF), the tail is padded with rune(0), which the checkIf
// helpers all treat as a non-matching character.
func padRunes(chars []rune, n int) []rune {
	if len(chars) >= n {
		return chars
	}
	out := make([]rune, n)
	copy(out, chars)
	return out
}
