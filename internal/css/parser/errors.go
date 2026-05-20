package parser

import "errors"

var (
	ErrSyntax      = errors.New("syntax error")
	ErrInvalidRule = errors.New("invalid rule")
)
