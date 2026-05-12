package tokenizer

import "unicode"

func isIdentStartCodePoint(char rune) bool {
	return unicode.IsLetter(char) || int(char) >= 0x0080 || char == '_'
}

func isIdentCodePoint(char rune) bool {
	return isIdentStartCodePoint(char) || unicode.IsDigit(char) || char == '-'
}
