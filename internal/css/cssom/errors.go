package cssom

import "errors"

var (
	ErrUnsupportedProperty = errors.New("unsupported property")
	ErrInvalidCssValue     = errors.New("invalid css value")
)
