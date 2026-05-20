package parser

import "errors"

var ErrSyntax = errors.New("syntax error")
var ErrNoValue = errors.New("no valid value")
var ErrInvalidRule = errors.New("invalid rule")
