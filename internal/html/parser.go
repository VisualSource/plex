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

func getValue(value *string, defaultValue string) string {
	if value == nil {
		return defaultValue
	}
	return *value
}

type HTMLParser struct {
	openEls             []Node
	insertionMode       int
	activeFormattingEls []any
	scripting           bool
	fameset             bool
	tokenizer           Tokenizer
	document            *Document
}

func (m *HTMLParser) Parse() error {

outer:
	for {
		err := m.tokenizer.Next()
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
		m.document.Append(NewCommentNode(token.data))
		return nil
	}

	if token.token == Token_DOCTYPE {
		m.document.doctype = DocumentType{
			publicId: getValue(token.doctypeData.publicIdentifier, ""),
			systemId: getValue(token.doctypeData.systemIdentifier, ""),
			name:     getValue(token.doctypeData.name, ""),
		}

		m.insertionMode = Mode_BeforeHtml
		return nil
	}

	m.tokenizer.tokens = append(m.tokenizer.tokens, token)
	m.insertionMode = Mode_BeforeHtml
	return nil
}

func (m *HTMLParser) Mode_BeforeHtml(token *Token) error {
	if token.token == Token_DOCTYPE {
		return nil
	}

	if token.token == Token_Comment {
		m.document.Append(NewCommentNode(token.data))
		return nil
	}

	if token.token == Token_Character && slices.Contains([]string{
		"\t",
		"\n",
		"\f",
		"\r",
		" ",
	}, token.data) {
		return nil
	}

	if token.token == Token_StartTag && token.tagData.name == "html" {

		el := NewHTMLElement("html")

		m.openEls = append(m.openEls, el)
		m.document.Append(el)

		m.insertionMode = Mode_BeforeHead
		return nil
	}

	if token.token == Token_EndTag && !slices.Contains([]string{
		"head", "body", "html", "br",
	}, token.tagData.name) {
		return nil
	}

	m.tokenizer.tokens = append(m.tokenizer.tokens, token)
	m.insertionMode = Mode_BeforeHead

	return nil
}

func (m *HTMLParser) Mode_BeforeHead() error {
	return nil
}

func (m *HTMLParser) Mode_InHead() error {
	return nil
}

func (m *HTMLParser) Mode_InHeaderNoScript() error {
	return nil
}

func (m *HTMLParser) Mode_AfterHead() error {
	return nil
}

func (m *HTMLParser) Mode_InBody() error {
	return nil
}

func (m *HTMLParser) Mode_Text() error {
	return nil
}

func (m *HTMLParser) Mode_InTable() error {
	panic("not implemented")
}

func (m *HTMLParser) Mode_InTableText() error {
	panic("not implemented")
}

func (m *HTMLParser) Mode_InCaption() error {
	panic("not implemented")
}

func (m *HTMLParser) Mode_InColumnGroup() error {
	panic("not implemented")
}

func (m *HTMLParser) Mode_InTableBody() error {
	panic("not implemented")
}

func (m *HTMLParser) Mode_InRow() error {
	panic("not implemented")
}

func (m *HTMLParser) Mode_InCell() error {
	panic("not implemented")
}

func (m *HTMLParser) Mode_InTemplate() error {
	panic("not implemented")
}

func (m *HTMLParser) Mode_AfterBody() error {
	panic("not implemented")
}

func (m *HTMLParser) Mode_InFrameset() error {
	panic("not implemented")
}

func (m *HTMLParser) Mode_AfterFrameset() error {
	panic("not implemented")
}

func (m *HTMLParser) Mode_AfterAfterBody() error {
	panic("not implemented")
}

func (m *HTMLParser) Mode_AfterAfterFrameset() error {
	panic("not implemented")
}

func NewHTMLParser(reader io.RuneReader) *HTMLParser {
	return &HTMLParser{}
}
