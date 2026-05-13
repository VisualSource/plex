package tokenizer

import "unicode"

func isWhitespace(char rune) bool {
	return char == '\n' || char == '\t' || char == ' '
}

func isIdentStartCodePoint(char rune) bool {
	return unicode.IsLetter(char) || int(char) >= 0x0080 || char == '_'
}

func isIdentCodePoint(char rune) bool {
	return isIdentStartCodePoint(char) || unicode.IsDigit(char) || char == '-'
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
