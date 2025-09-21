package html

import (
	"io"
	"slices"
)

const (
	Mode_Initial = iota
	Mode_BeforeHtml
	Mode_BeforeHead
	Mode_InHead
	Mode_InHeadNoScript
	Mode_AfterHeader
	Mode_InBody
	Mode_Text
	Mode_InTable
	Mode_InTableText
	Mode_InCaption
	Mode_InColumnGroup
	Mode_InTableBody
	Mode_InRow
	Mode_InCell
	Mode_InTemplete
	Mode_AfterBody
	Mode_Frameset
	Mode_AfterFrameset
	Mode_AfterAfterBody
	Mode_AfterAfterframeset
)

type HTMLParser struct {
	openEls             []any
	insertionMode       int
	activeFormattingEls []any
	scripting           bool
	fameset             bool
	tokenizer           Tokenizer
}

func (m *HTMLParser) Parse() error {

outer:
	for {
		err := m.tokenizer.Parse()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		for {
			token := m.tokenizer.ConsumeToken()
			if token == nil {
				break
			}

			if token.token == Token_EOF {
				break outer
			}

		}
	}

	return nil
}

// https://html.spec.whatwg.org/#the-initial-insertion-mode
func (m *HTMLParser) Mode_Initial(token *Token) error {

	if token.token == Token_Character && slices.Contains([]string{
		"\t",
		"\n",
		"\f",
		"\r",
		" ",
	}, token.data) {
		return nil
	}

	if token.token == Token_Comment {
		// insert
		return nil
	}

	if token.token == Token_DOCTYPE {
		// append DocumentType

		m.insertionMode = Mode_BeforeHtml
		return nil
	}

	m.insertionMode = Mode_BeforeHtml
	return nil
}

func NewHTMLParser(reader io.RuneReader) *HTMLParser {
	return &HTMLParser{}
}
