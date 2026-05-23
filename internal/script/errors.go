package script

import "errors"

var (
	ErrUnexpectedEOF             = errors.New("unexpected eof")
	ErrMissingStartQuote         = errors.New("missing start quote in string")
	ErrUnexpectedNewlineInString = errors.New("unexpected newline in string")
	ErrUnexpectedCharacter       = errors.New("unexpected character")
)
