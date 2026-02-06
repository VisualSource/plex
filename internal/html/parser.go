package html

import (
	"errors"
	"io"
	"slices"

	"github.com/VisualSource/plex/internal/dom"
	"github.com/VisualSource/plex/internal/html_tokenizer"
	"github.com/VisualSource/plex/internal/utils"
)

type InsertionMode int

const (
	mode_Initial InsertionMode = iota
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
	mode_InFrameset
	mode_AfterFrameset
	mode_AfterAfterBody
	mode_AfterAfterframeset
)

// https://html.spec.whatwg.org/#parse-state
type Parser struct {
	//https://html.spec.whatwg.org/#the-insertion-mode
	insertionMode InsertionMode
	// https://html.spec.whatwg.org/#the-stack-of-open-elements
	openElements []dom.Node
	//https://html.spec.whatwg.org/#the-list-of-active-formatting-elements
	activeFormattingElements []dom.Node

	//https://html.spec.whatwg.org/#the-element-pointers
	head dom.Node
	form dom.Node

	//https://html.spec.whatwg.org/#other-parsing-state-flags
	scripting       bool
	fameset         bool
	fosterParenting bool

	originalInsertionMode InsertionMode
	html_tokenizer        *html_tokenizer.Tokenizer
	document              *dom.Document
	// https://html.spec.whatwg.org/#frameset-ok-flag
	framesetOk string
}

func NewParser(reader io.RuneReader) *Parser {
	return &Parser{
		insertionMode:  mode_Initial,
		html_tokenizer: html_tokenizer.NewTokenizer(reader),
		framesetOk:     "ok",
	}
}

// https://html.spec.whatwg.org/#creating-and-inserting-nodes
func (m *Parser) getInsertionLocation(overrideTarget *dom.Node) dom.HTMLElement {
	var target dom.Node
	if overrideTarget != nil {
		target = *overrideTarget
	} else {
		// current node = last element on open elements stack
		target = m.openElements[len(m.openElements)-1]
	}

	var adjustedInsertionLocation dom.HTMLElement

	if m.fosterParenting && slices.Contains([]string{"table", "tbody", "tfoot", "thread", "tr"}, target.GetNodeName()) {
		// lastTemplate == last template in stack of open elements
		// lastTable == last table in open elements

		// if lastTemplate || lastTable && template < table
		//    adjustedInsertionLocation = lastTemplate.contents
		//    abort

		// if !lastTable
		//    adjustedInsertionLocation = openElements[len(openElements)-1]
		//    abort

		// if lastTable.parent
		//     adjustedInsertionLocation = lastTable.parent
		//     abort

		// if previous

	} else {
		adjustedInsertionLocation = target.(dom.HTMLElement)
	}

	return adjustedInsertionLocation
}

func (m *Parser) Parse() error {

outer:
	for {
		err := m.html_tokenizer.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		for {
			token := m.html_tokenizer.ConsumeToken()
			if token == nil {
				break
			}

			if _, ok := token.(html_tokenizer.TokenEOF); ok {
				break outer
			}

		}
	}

	return nil
}

// https://html.spec.whatwg.org/#reconstruct-the-active-formatting-elements
func (m *Parser) reconstructActiveFormattingEls() {}

func (m *Parser) hasElementInScope(target string, elementTypes ...string) bool {

	idx := 1
	for {
		node := m.openEls[len(m.openEls)-idx]

		if node.GetNodeName() == target {
			return true
		}

		if slices.Contains(elementTypes, node.GetNodeName()) {
			return false
		}

		idx++
	}
}

func (m *Parser) hasParticularElementInScope(element string) bool {
	return m.hasElementInScope(element, "applet", "caption", "html", "table", "td", "th", "marquee", "object", "select", "template", "mi", "mo", "mn", "ms", "mtext", "annotation-xml", "foreignObject", "desc", "title")
}

// https://html.spec.whatwg.org/#insert-a-character
func (m *Parser) insertCharacter(token *html_tokenizer.TokenCharacter) {

}

func (m *Parser) insertComment(token *html_tokenizer.CommentToken, target dom.Node) (*dom.CommentNode, error) {
	comment := dom.NewCommentNode(token.Data)

	if target != nil {

		if tag, ok := target.(*dom.HTMLElement); ok {
			tag.Append(comment)
		}
		return comment, nil
	}

	node, ok := m.openEls[len(m.openEls)-1].(*dom.HTMLElement)
	if ok {
		node.Append(comment)
	}

	return comment, nil
}

func (m *Parser) insertElement(node dom.Node, target dom.Node) {

	n, ok := m.openEls[len(m.openEls)-1].(*dom.HTMLElement)
	if ok {
		n.Append(node)
	}
	m.openEls = append(m.openEls, node)
}

func (m *Parser) InsertHtmlElement(token html_tokenizer.Token, namespace string, onlyAddToElementStack bool) (*dom.HTMLElement, error) {

	tag, ok := token.(*html_tokenizer.TagToken)
	if !ok {
		return nil, errors.New("invalid token type")
	}

	el := dom.NewHTMLElement(tag.Name)

	node, ok := m.openEls[len(m.openEls)-1].(*dom.HTMLElement)
	if ok {
		node.Append(el)
	}

	m.openEls = append(m.openEls, el)

	return el, nil
}

// https://html.spec.whatwg.org/#the-initial-insertion-mode
func (m *Parser) mode_Initial(token html_tokenizer.Token) error {

	if tag, ok := token.(*html_tokenizer.TokenCharacter); ok && slices.Contains([]rune{'\t', '\n', '\f', '\r', ' '}, tag.Data) {
		return nil
	}

	if tag, ok := token.(*html_tokenizer.CommentToken); ok {
		m.document.Append(dom.NewCommentNode(tag.Data))
		return nil
	}

	if tag, ok := token.(*html_tokenizer.DoctypeToken); ok {

		doctype := dom.NewDocumentType(
			utils.Alter(tag.Name, ""),
			utils.Alter(tag.SystemIdentifer, ""),
			utils.Alter(tag.PublicIdentifier, ""),
		)

		m.document.SetDocType(doctype)
		m.insertionMode = mode_BeforeHtml
		return nil
	}

	m.html_tokenizer.ReconsumeToken(token)
	m.insertionMode = mode_BeforeHtml
	return nil
}

// https://html.spec.whatwg.org/#the-before-html-insertion-mode
func (m *Parser) mode_BeforeHtml(token html_tokenizer.Token) error {

	if _, ok := token.(*html_tokenizer.DoctypeToken); ok {
		return nil
	}

	if tag, ok := token.(*html_tokenizer.CommentToken); ok {
		m.document.Append(dom.NewCommentNode(tag.Data))
		return nil
	}

	if tag, ok := token.(*html_tokenizer.TokenCharacter); ok && slices.Contains([]rune{'\t', '\n', '\f', '\r', ' '}, tag.Data) {
		return nil
	}

	if tag, ok := token.(*html_tokenizer.TagToken); ok {
		if tag.IsType(html_tokenizer.Token_StartTag) && tag.Name == "html" {
			el := dom.NewHTMLElement("html")

			m.openEls = append(m.openEls, el)
			m.document.Append(el)

			m.insertionMode = mode_BeforeHead
			return nil
		}
		if tag.IsType(html_tokenizer.Token_EndTag) && !slices.Contains([]string{"head", "body", "html", "br"}, tag.Name) {
			return nil
		}
	}

	el := dom.NewHTMLElement("html")

	m.openEls = append(m.openEls, el)
	m.document.Append(el)

	m.html_tokenizer.ReconsumeToken(token)
	m.insertionMode = mode_BeforeHead

	return nil
}

// https://html.spec.whatwg.org/#the-before-head-insertion-mode
func (m *Parser) mode_BeforeHead(token html_tokenizer.Token) error {

	if tag, ok := token.(*html_tokenizer.TokenCharacter); ok && slices.Contains([]rune{'\t', '\n', '\f', '\f', '\r', ' '}, tag.Data) {
		return nil
	}

	if tag, ok := token.(*html_tokenizer.CommentToken); ok {
		m.insertComment(tag, nil)
		return nil
	}

	if token.IsType(html_tokenizer.Token_DOCTYPE) {
		return nil
	}

	if tag, ok := token.(*html_tokenizer.TagToken); ok {
		if tag.IsType(html_tokenizer.Token_StartTag) && tag.Name == "html" {
			return nil
		}
		if tag.IsType(html_tokenizer.Token_StartTag) && tag.Name == "head" {
			el := dom.NewHTMLElement("head")
			m.document.Append(el)
			m.document.SetHead(el)

			m.insertionMode = mode_InHead
			return nil
		}
		if tag.IsType(html_tokenizer.Token_EndTag) && !slices.Contains([]string{"head", "body", "html", "br"}, tag.Name) {
			return nil
		}
	}

	el := dom.NewHTMLElement("head")
	m.document.Append(el)
	m.document.SetHead(el)

	m.insertionMode = mode_InHead
	m.html_tokenizer.ReconsumeToken(token)
	return nil
}

// https://html.spec.whatwg.org/#parsing-main-inhead
func (m *Parser) mode_InHead(token html_tokenizer.Token) error {

	if tag, ok := token.(*html_tokenizer.TokenCharacter); ok && slices.Contains([]rune{'\t', '\n', '\f', '\r', ' '}, tag.Data) {
		return nil
	}

	if tag, ok := token.(*html_tokenizer.CommentToken); ok {
		node, ok := m.openEls[len(m.openEls)-1].(*dom.HTMLElement)
		if ok {
			node.Append(dom.NewCommentNode(tag.Data))
		}
		return nil
	}

	if token.IsType(html_tokenizer.Token_DOCTYPE) {
		return nil
	}

	if tag, ok := token.(*html_tokenizer.TagToken); ok {
		if tag.IsType(html_tokenizer.Token_StartTag) {
			if tag.Name == "html" {
				return m.mode_InBody(token)
			}
			if slices.Contains([]string{"head", "basefont", "link", "bgsound"}, tag.Name) {
				_, err := m.InsertHtmlElement(token, "html", false)
				m.openEls = slices.Delete(m.openEls, 0, 1)
				return err
			}

			if tag.Name == "meta" {
				_, err := m.InsertHtmlElement(token, "html", false)
				m.openEls = slices.Delete(m.openEls, 0, 1)
				// TODO: do stuff with charset and http-equiv
				//       since we dont have a speculative html parser
				return err
			}

			if tag.Name == "title" {
				return nil
			}

			if (tag.Name == "noscript" && !m.scripting) || tag.Name == "noframes" || tag.Name == "style" {
				_, err := m.InsertHtmlElement(token, "html", false)

				m.html_tokenizer.SetState(html_tokenizer.State_RawText)

				m.originalInsertionMode = m.insertionMode
				m.insertionMode = mode_Text
				return err
			}

			if tag.Name == "noscript" && m.scripting {
				_, err := m.InsertHtmlElement(token, "html", false)
				m.insertionMode = mode_InHeadNoScript
				return err
			}

			if tag.Name == "script" {
				_, err := m.InsertHtmlElement(token, "html", false)

				m.html_tokenizer.SetState(html_tokenizer.State_ScriptData)
				m.originalInsertionMode = m.insertionMode
				m.insertionMode = mode_Text

				return err
			}

			if tag.Name == "template" {
				return nil
			}
		}

		if tag.IsType(html_tokenizer.Token_EndTag) {
			if tag.Name == "head" {
				m.openEls = slices.Delete(m.openEls, len(m.openEls)-1, len(m.openEls)-1)

				m.insertionMode = mode_AfterHead
				return nil
			}

			if tag.Name == "template" {
				return nil
			}

			if slices.Contains([]string{}, tag.Name) {
				return nil
			}
		}

	}

	if (token.IsType(html_tokenizer.Token_StartTag) && token.(*html_tokenizer.TagToken).Name == "head") || token.IsType(html_tokenizer.Token_EndTag) {
		return nil
	}

	m.openEls = slices.Delete(m.openEls, len(m.openEls)-1, len(m.openEls)-1)

	m.insertionMode = mode_AfterHead
	m.html_tokenizer.ReconsumeToken(token)
	return nil
}

// https://html.spec.whatwg.org/#parsing-main-inheadnoscript
func (m *Parser) mode_InHeaderNoScript(token html_tokenizer.Token) error {

	if token.IsType(html_tokenizer.Token_DOCTYPE) {
		return nil
	}

	if tag, ok := token.(*html_tokenizer.TagToken); ok {
		if tag.IsStartTag() && tag.Name == "html" {
			return m.mode_InBody(token)
		}
		if tag.IsEndTag() && tag.Name == "noscript" {
			return m.mode_InBody(token)
		}

		if (tag.IsStartTag() && slices.Contains([]string{"head", "noscript"}, tag.Name)) || tag.IsEndTag() {
			return nil
		}
	}

	if token.IsType(html_tokenizer.Token_Comment) ||
		isAnyRune(token, '\t', '\n', '\f', '\r', ' ') ||
		isAnyStartTag(token, "basefont", "bgsound", "link", "meta", "noframes", "style") {

		return m.mode_InHead(token)
	}

	m.openEls = m.openEls[:len(m.openEls)-2]
	m.html_tokenizer.ReconsumeToken(token)
	m.insertionMode = mode_InHead
	return nil
}

// https://html.spec.whatwg.org/#the-after-head-insertion-mode
func (m *Parser) mode_AfterHead(token html_tokenizer.Token) error {

	if isAnyRune(token, '\t', '\n', '\f', '\r', ' ') {
		m.insertCharacter(token.(*html_tokenizer.TokenCharacter))
		return nil
	}

	if tag, ok := token.(*html_tokenizer.CommentToken); ok {
		m.insertComment(tag, nil)
		return nil
	}

	if token.IsType(html_tokenizer.Token_DOCTYPE) {
		return nil
	}

	if tag, ok := token.(*html_tokenizer.TagToken); ok {
		if tag.IsStartTag() && tag.Name == "html" {
			return m.mode_InBody(token)
		}

		if tag.IsStartTag() && tag.Name == "body" {
			m.InsertHtmlElement(token, "html", true)
			m.framesetOk = "not ok"
			m.insertionMode = mode_InBody
			return nil
		}

		if tag.IsStartTag() && tag.Name == "frameset" {
			m.InsertHtmlElement(token, "html", true)
			m.insertionMode = mode_InFrameset
			return nil
		}

		if tag.IsStartTag() && isAnyTag(tag, "base", "basefont", "bgsound", "link", "meta", "noframes", "script", "style", "template", "title") {

			m.openEls = append(m.openEls, m.document.GetHead())

			r := m.mode_InHead(token)

			//TODO: remove head element from open stack

			return r
		}

		if tag.IsEndTag() && tag.Name == "template" {
			return m.mode_InHead(token)
		}

		if (tag.IsStartTag() && tag.Name == "head") || tag.IsEndTag() {
			return nil
		}
	}

	el := dom.NewHTMLElement("body")
	m.insertElement(el, nil)

	m.insertionMode = mode_InBody
	m.html_tokenizer.ReconsumeToken(token)
	return nil
}

// https://html.spec.whatwg.org/#parsing-main-inbody
func (m *Parser) mode_InBody(token html_tokenizer.Token) error {

	if tag, ok := token.(*html_tokenizer.TokenCharacter); ok {
		if tag.Data == '\u0000' {
			return nil
		}

		if !isAny(tag.Data, '\t', '\n', '\f', '\r', ' ') {
			m.framesetOk = "not ok"
		}

		m.reconstructActiveFormattingEls()
		m.insertCharacter(tag)
		return nil
	}

	if tag, ok := token.(*html_tokenizer.CommentToken); ok {
		_, err := m.insertComment(tag, nil)
		return err
	}

	if token.IsType(html_tokenizer.Token_DOCTYPE) {
		return nil
	}

	if tag, ok := token.(*html_tokenizer.TagToken); ok {

		if tag.IsType(html_tokenizer.Token_StartTag) {

			// TODO: reconstruct

			m.InsertHtmlElement(token, "html", false)

			return nil
		}

		if tag.IsType(html_tokenizer.Token_EndTag) {

			return nil
		}
	}

	if token.IsType(html_tokenizer.Token_EOF) {
		return nil
	}

	return nil
}

func (m *Parser) Mode_Text(token html_tokenizer.Token) error {

	if _, ok := token.(*html_tokenizer.TokenCharacter); ok {
		return nil
	}

	if token.IsType(html_tokenizer.Token_DOCTYPE) {
		return nil
	}

	if tag, ok := token.(*html_tokenizer.TagToken); ok {
		if tag.IsType(html_tokenizer.Token_EndTag) && tag.Name == "script" {
			return nil
		}

		if tag.IsType(html_tokenizer.Token_EndTag) {

		}
	}

	return nil
}

func (m *Parser) Mode_InTable(token html_tokenizer.Token) error {
	panic("not implemented")
}

func (m *Parser) Mode_InTableText(token html_tokenizer.Token) error {
	panic("not implemented")
}

func (m *Parser) Mode_InCaption(token html_tokenizer.Token) error {
	panic("not implemented")
}

func (m *Parser) Mode_InColumnGroup(token html_tokenizer.Token) error {
	panic("not implemented")
}

func (m *Parser) Mode_InTableBody(token html_tokenizer.Token) error {
	panic("not implemented")
}

func (m *Parser) Mode_InRow(token html_tokenizer.Token) error {
	panic("not implemented")
}

func (m *Parser) Mode_InCell(token html_tokenizer.Token) error {
	panic("not implemented")
}

func (m *Parser) Mode_InTemplate(token html_tokenizer.Token) error {
	panic("not implemented")
}

func (m *Parser) Mode_AfterBody(token html_tokenizer.Token) error {
	panic("not implemented")
}

func (m *Parser) Mode_InFrameset(token html_tokenizer.Token) error {
	panic("not implemented")
}

func (m *Parser) Mode_AfterFrameset(token html_tokenizer.Token) error {
	panic("not implemented")
}

func (m *Parser) Mode_AfterAfterBody(token html_tokenizer.Token) error {
	panic("not implemented")
}

func (m *Parser) Mode_AfterAfterFrameset(token html_tokenizer.Token) error {
	panic("not implemented")
}
