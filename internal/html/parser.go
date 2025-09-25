package html

import (
	"io"
	"slices"

	tokenizer "github.com/VisualSource/plex/internal/html/tokenizer"
)

const (
	Mode_Initial = iota
	Mode_BeforeHtml
	Mode_BeforeHead
	Mode_InHead
	Mode_InHeadNoScript
	Mode_AfterHead
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
	openEls               []Node
	insertionMode         int
	activeFormattingEls   []any
	scripting             bool
	fameset               bool
	originalInsertionMode int
	tokenizer             tokenizer.Tokenizer
	document              *Document
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

func (m *HTMLParser) InsertHtmlElement(token *tokenizer.Token, namespace string, onlyAddToElementStack bool) *HTMLElement {

	el := NewHTMLElement(token.tagData.name)

	node, ok := m.openEls[len(m.openEls)-1].(*HTMLElement)
	if ok {
		node.Append(el)
	}

	m.openEls = append(m.openEls, el)

	return el
}

// https://html.spec.whatwg.org/#the-initial-insertion-mode
func (m *HTMLParser) Mode_Initial(token *tokenizer.Token) error {

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

	m.tokenizer.ReconsumeToken(token)
	m.insertionMode = Mode_BeforeHtml
	return nil
}

func (m *HTMLParser) Mode_BeforeHtml(token *tokenizer.Token) error {
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

	el := NewHTMLElement("html")

	m.openEls = append(m.openEls, el)
	m.document.Append(el)

	m.tokenizer.ReconsumeToken(token)
	m.insertionMode = Mode_BeforeHead

	return nil
}

func (m *HTMLParser) Mode_BeforeHead(token *tokenizer.Token) error {

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
		node, ok := m.openEls[len(m.openEls)-1].(*HTMLElement)
		if ok {
			node.Append(NewCommentNode(token.data))
		}
		return nil
	}

	if token.token == Token_DOCTYPE {
		return nil
	}

	if token.token == Token_StartTag && token.tagData.name == "html" {
		return nil
	}

	if token.token == Token_StartTag && token.tagData.name == "head" {
		el := NewHTMLElement("head")
		m.document.Append(el)
		m.document.head = el

		m.insertionMode = Mode_InHead
		return nil
	}

	if token.token == Token_EndTag && !slices.Contains([]string{"head", "body", "html", "br"}, token.tagData.name) {
		return nil
	}

	el := NewHTMLElement("head")
	m.document.Append(el)
	m.document.head = el

	m.insertionMode = Mode_InHead
	m.tokenizer.ReconsumeToken(token)
	return nil
}

// https://html.spec.whatwg.org/#parsing-main-inhead
func (m *HTMLParser) Mode_InHead(token *tokenizer.Token) error {
	if token.token == tokenizer.Token_Character && slices.Contains([]string{
		"\t",
		"\n",
		"\f",
		"\r",
		" ",
	}, token.data) {
		return nil
	}

	if token.token == tokenizer.Token_Comment {
		node, ok := m.openEls[len(m.openEls)-1].(*HTMLElement)
		if ok {
			node.Append(NewCommentNode(token.data))
		}
		return nil
	}

	if token.token == tokenizer.Token_DOCTYPE {
		return nil
	}

	if token.token == tokenizer.Token_StartTag && token.tagData.name == "html" {
		return m.Mode_InBody()
	}

	if token.token == tokenizer.Token_StartTag && slices.Contains([]string{"base", "basefont", "link", "bgsound"}, token.tagData.name) {
		m.InsertHtmlElement(token, "html", false)
		m.openEls = slices.Delete(m.openEls, 0, 1)
		return nil
	}

	if token.token == tokenizer.Token_StartTag && token.tagData.name == "meta" {
		m.InsertHtmlElement(token, "html", false)
		m.openEls = slices.Delete(m.openEls, 0, 1)

		// TODO: do stuff with charset and http-equiv
		//       since we dont have a speculative html parser

		return nil
	}

	if token.token == tokenizer.Token_StartTag && token.tagData.name == "title" {
		return nil
	}

	if token.token == tokenizer.Token_StartTag && ((token.tagData.name == "noscript" && !m.scripting) || token.tagData.name == "noframes" || token.tagData.name == "style") {
		m.InsertHtmlElement(token, "html", false)

		m.tokenizer.SetState(tokenizer.State_RawText)

		m.originalInsertionMode = m.insertionMode
		m.insertionMode = Mode_Text
		return nil
	}

	if token.token == tokenizer.Token_StartTag && token.tagData.name == "noscript" && m.scripting {
		m.InsertHtmlElement(token, "html", false)
		m.insertionMode = Mode_InHeadNoScript
		return nil
	}

	if token.token == tokenizer.Token_StartTag && token.tagData.name == "script" {

		m.InsertHtmlElement(token, "html", false)

		m.tokenizer.SetState(tokenizer.State_ScriptData)
		m.originalInsertionMode = m.insertionMode
		m.insertionMode = Mode_Text
	}

	if token.token == tokenizer.Token_EndTag && token.tagData.name == "head" {
		m.openEls = slices.Delete(m.openEls, len(m.openEls)-1, len(m.openEls)-1)

		m.insertionMode = Mode_AfterHead
		return nil
	}

	if token.token == tokenizer.Token_StartTag && token.tagData.name == "template" {
		return nil
	}

	if token.token == tokenizer.Token_EndTag && token.tagData.name == "template" {
		return nil
	}

	if token.token == tokenizer.Token_EndTag && slices.Contains([]string{}, token.tagData.name) {
	}

	if (token.token == tokenizer.Token_StartTag && token.tagData.name == "head") || token.token == tokenizer.Token_EndTag {
		return nil
	}

	m.openEls = slices.Delete(m.openEls, len(m.openEls)-1, len(m.openEls)-1)

	m.insertionMode = Mode_AfterHead
	m.tokenizer.ReconsumeToken(token)
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
