package html_parser

import (
	"io"

	"github.com/VisualSource/plex/internal/dom"
	"github.com/VisualSource/plex/internal/html_tokenizer"
)

type HtmlParser struct {
	tokenizer     *html_tokenizer.Tokenizer
	insertionMode InsertionMode
	// https://html.spec.whatwg.org/#the-stack-of-open-elements
	openElementsStack []any
	// https://html.spec.whatwg.org/#the-list-of-active-formatting-elements
	activeFormattingElements []any
	// https://html.spec.whatwg.org/#other-parsing-state-flags
	scriptingMode ScriptingMode

	framesetOk bool
}

func NewHtmlParser(stream io.Reader) *HtmlParser {
	return &HtmlParser{
		insertionMode: mode_Initial,
		tokenizer:     html_tokenizer.NewTokenizer(stream),
	}
}

// https://html.spec.whatwg.org/#tree-construction
func (p *HtmlParser) Parse() (*dom.Document, error) {

parseLoop:
	for {
		err := p.tokenizer.Next()
		if err != io.EOF {
			return nil, err
		}
		for p.tokenizer.HasEmittedTokens() {
			token := p.tokenizer.ConsumeToken()

			switch token.(type) {
			case *html_tokenizer.TokenEOF:
				break parseLoop
			}
		}

	}

	return nil, nil
}
