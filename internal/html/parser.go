package html

import (
	"io"
)

type HTMLParser struct {
	index         int
	reader        io.RuneReader
	insertionMode int
	tokens        []Token
}

func (m *HTMLParser) Parse() error {

	return nil
}

func NewHTMLParser(reader io.RuneReader) *HTMLParser {
	return &HTMLParser{
		index:         0,
		reader:        reader,
		insertionMode: 0,
	}
}
