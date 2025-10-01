package html

import (
	"io"
	"slices"

	dom "github.com/VisualSource/plex/internal/html/dom"
	tokenizer "github.com/VisualSource/plex/internal/html/tokenizer"
)

const (
	mode_Initial = iota
	mode_BeforeHtml
	mode_BeforeHead
	mode_InHead
	mode_InHeadNoScript
	mode_AfterHead
	mode_InBody
	mode_Text
	mode_InTable
	mode_InTableText
	mode_InCaption
	mode_InColumnGroup
	mode_InTableBody
	mode_InRow
	mode_InCell
	mode_InTemplete
	mode_AfterBody
	mode_Frameset
	mode_AfterFrameset
	mode_AfterAfterBody
	mode_AfterAfterframeset
)

func getValue(value *string, defaultValue string) string {
	if value == nil {
		return defaultValue
	}
	return *value
}

type HTMLParser struct {
	openEls               []dom.Node
	insertionMode         int
	activeFormattingEls   []any
	scripting             bool
	fameset               bool
	originalInsertionMode int
	tokenizer             tokenizer.Tokenizer
	document              *dom.Document
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

			if token.token == tokenizer.Token_EOF {
				break outer
			}

		}
	}

	return nil
}

func (m *HTMLParser) InsertHtmlElement(token *tokenizer.Token, namespace string, onlyAddToElementStack bool) *HTMLElement {

	el := dom.NewHTMLElement(token.tagData.name)

	node, ok := m.openEls[len(m.openEls)-1].(*dom.HTMLElement)
	if ok {
		node.Append(el)
	}

	m.openEls = append(m.openEls, el)

	return el
}

// https://html.spec.whatwg.org/#the-initial-insertion-mode
func (m *HTMLParser) Mode_Initial(token *tokenizer.Token) error {

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
		m.document.Append(dom.NewCommentNode(token.data))
		return nil
	}

	if token.token == tokenizer.Token_DOCTYPE {
		m.document.doctype = dom.NewDocumentType(
			getValue(token.doctypeData.name, ""),
			getValue(token.doctypeData.systemIdentifier, ""),
			getValue(token.doctypeData.publicIdentifier, ""),
		)

		m.insertionMode = mode_BeforeHtml
		return nil
	}

	m.tokenizer.ReconsumeToken(token)
	m.insertionMode = mode_BeforeHtml
	return nil
}

func (m *HTMLParser) Mode_BeforeHtml(token *tokenizer.Token) error {
	if token.token == tokenizer.Token_DOCTYPE {
		return nil
	}

	if token.token == tokenizer.Token_Comment {
		m.document.Append(dom.NewCommentNode(token.data))
		return nil
	}

	if token.token == tokenizer.Token_Character && slices.Contains([]string{
		"\t",
		"\n",
		"\f",
		"\r",
		" ",
	}, token.data) {
		return nil
	}

	if token.token == tokenizer.Token_StartTag && token.tagData.name == "html" {

		el := dom.NewHTMLElement("html")

		m.openEls = append(m.openEls, el)
		m.document.Append(el)

		m.insertionMode = mode_BeforeHead
		return nil
	}

	if token.token == tokenizer.Token_EndTag && !slices.Contains([]string{
		"head", "body", "html", "br",
	}, token.tagData.name) {
		return nil
	}

	el := dom.NewHTMLElement("html")

	m.openEls = append(m.openEls, el)
	m.document.Append(el)

	m.tokenizer.ReconsumeToken(token)
	m.insertionMode = mode_BeforeHead

	return nil
}

func (m *HTMLParser) Mode_BeforeHead(token *tokenizer.Token) error {

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
		node, ok := m.openEls[len(m.openEls)-1].(*dom.HTMLElement)
		if ok {
			node.Append(dom.NewCommentNode(token.data))
		}
		return nil
	}

	if token.token == tokenizer.Token_DOCTYPE {
		return nil
	}

	if token.token == tokenizer.Token_StartTag && token.tagData.name == "html" {
		return nil
	}

	if token.token == tokenizer.Token_StartTag && token.tagData.name == "head" {
		el := dom.NewHTMLElement("head")
		m.document.Append(el)
		m.document.head = el

		m.insertionMode = mode_InHead
		return nil
	}

	if token.token == tokenizer.Token_EndTag && !slices.Contains([]string{"head", "body", "html", "br"}, token.tagData.name) {
		return nil
	}

	el := dom.NewHTMLElement("head")
	m.document.Append(el)
	m.document.head = el

	m.insertionMode = mode_InHead
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
		node, ok := m.openEls[len(m.openEls)-1].(*dom.HTMLElement)
		if ok {
			node.Append(dom.NewCommentNode(token.data))
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
		m.insertionMode = mode_InHeadNoScript
		return nil
	}

	if token.token == tokenizer.Token_StartTag && token.tagData.name == "script" {

		m.InsertHtmlElement(token, "html", false)

		m.tokenizer.SetState(tokenizer.State_ScriptData)
		m.originalInsertionMode = m.insertionMode
		m.insertionMode = mode_Text
	}

	if token.token == tokenizer.Token_EndTag && token.tagData.name == "head" {
		m.openEls = slices.Delete(m.openEls, len(m.openEls)-1, len(m.openEls)-1)

		m.insertionMode = mode_AfterHead
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

	m.insertionMode = mode_AfterHead
	m.tokenizer.ReconsumeToken(token)
	return nil
}

func (m *HTMLParser) Mode_InHeaderNoScript(token *tokenizer.Token) error {
	return nil
}

func (m *HTMLParser) Mode_AfterHead(token *tokenizer.Token) error {
	return nil
}

func (m *HTMLParser) Mode_InBody(token *tokenizer.Token) error {
	return nil
}

func (m *HTMLParser) Mode_Text(token *tokenizer.Token) error {
	return nil
}

func (m *HTMLParser) Mode_InTable(token *tokenizer.Token) error {
	panic("not implemented")
}

func (m *HTMLParser) Mode_InTableText(token *tokenizer.Token) error {
	panic("not implemented")
}

func (m *HTMLParser) Mode_InCaption(token *tokenizer.Token) error {
	panic("not implemented")
}

func (m *HTMLParser) Mode_InColumnGroup(token *tokenizer.Token) error {
	panic("not implemented")
}

func (m *HTMLParser) Mode_InTableBody(token *tokenizer.Token) error {
	panic("not implemented")
}

func (m *HTMLParser) Mode_InRow(token *tokenizer.Token) error {
	panic("not implemented")
}

func (m *HTMLParser) Mode_InCell(token *tokenizer.Token) error {
	panic("not implemented")
}

func (m *HTMLParser) Mode_InTemplate(token *tokenizer.Token) error {
	panic("not implemented")
}

func (m *HTMLParser) Mode_AfterBody(token *tokenizer.Token) error {
	panic("not implemented")
}

func (m *HTMLParser) Mode_InFrameset(token *tokenizer.Token) error {
	panic("not implemented")
}

func (m *HTMLParser) Mode_AfterFrameset(token *tokenizer.Token) error {
	panic("not implemented")
}

func (m *HTMLParser) Mode_AfterAfterBody(token *tokenizer.Token) error {
	panic("not implemented")
}

func (m *HTMLParser) Mode_AfterAfterFrameset(token *tokenizer.Token) error {
	panic("not implemented")
}

func NewHTMLParser(reader io.RuneReader) *HTMLParser {
	return &HTMLParser{}
}
