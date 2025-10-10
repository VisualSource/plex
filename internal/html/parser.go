package html

import (
	"errors"
	"io"
	"slices"

	dom "github.com/VisualSource/plex/internal/html/dom"
	tokenizer "github.com/VisualSource/plex/internal/html/tokenizer"
	"github.com/VisualSource/plex/internal/utils"
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
	mode_InFrameset
	mode_AfterFrameset
	mode_AfterAfterBody
	mode_AfterAfterframeset
)

type HTMLParser struct {
	openEls               []dom.Node
	insertionMode         int
	activeFormattingEls   []any
	scripting             bool
	fameset               bool
	originalInsertionMode int
	tokenizer             tokenizer.Tokenizer
	document              *dom.Document
	// https://html.spec.whatwg.org/#frameset-ok-flag
	framesetOk string
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

			if token.IsType(tokenizer.Token_EOF) {
				break outer
			}

		}
	}

	return nil
}

// https://html.spec.whatwg.org/#reconstruct-the-active-formatting-elements
func (m *HTMLParser) reconstructActiveFormattingEls() {}

// https://html.spec.whatwg.org/#insert-a-character
func (m *HTMLParser) insertCharacter(token *tokenizer.TokenCharacter) {

}

func (m *HTMLParser) insertComment(token *tokenizer.CommentToken, target dom.Node) (*dom.CommentNode, error) {
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

func (m *HTMLParser) insertElement(node dom.Node, target dom.Node) {

	n, ok := m.openEls[len(m.openEls)-1].(*dom.HTMLElement)
	if ok {
		n.Append(node)
	}
	m.openEls = append(m.openEls, node)
}

func (m *HTMLParser) InsertHtmlElement(token tokenizer.Token, namespace string, onlyAddToElementStack bool) (*dom.HTMLElement, error) {

	tag, ok := token.(*tokenizer.TagToken)
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
func (m *HTMLParser) mode_Initial(token tokenizer.Token) error {

	if tag, ok := token.(*tokenizer.TokenCharacter); ok && slices.Contains([]rune{'\t', '\n', '\f', '\r', ' '}, tag.Data) {
		return nil
	}

	if tag, ok := token.(*tokenizer.CommentToken); ok {
		m.document.Append(dom.NewCommentNode(tag.Data))
		return nil
	}

	if tag, ok := token.(*tokenizer.DoctypeToken); ok {

		doctype := dom.NewDocumentType(
			utils.Alter(tag.Name, ""),
			utils.Alter(tag.SystemIdentifer, ""),
			utils.Alter(tag.PublicIdentifier, ""),
		)

		m.document.SetDocType(doctype)
		m.insertionMode = mode_BeforeHtml
		return nil
	}

	m.tokenizer.ReconsumeToken(token)
	m.insertionMode = mode_BeforeHtml
	return nil
}

// https://html.spec.whatwg.org/#the-before-html-insertion-mode
func (m *HTMLParser) mode_BeforeHtml(token tokenizer.Token) error {

	if _, ok := token.(*tokenizer.DoctypeToken); ok {
		return nil
	}

	if tag, ok := token.(*tokenizer.CommentToken); ok {
		m.document.Append(dom.NewCommentNode(tag.Data))
		return nil
	}

	if tag, ok := token.(*tokenizer.TokenCharacter); ok && slices.Contains([]rune{'\t', '\n', '\f', '\r', ' '}, tag.Data) {
		return nil
	}

	if tag, ok := token.(*tokenizer.TagToken); ok {
		if tag.IsType(tokenizer.Token_StartTag) && tag.Name == "html" {
			el := dom.NewHTMLElement("html")

			m.openEls = append(m.openEls, el)
			m.document.Append(el)

			m.insertionMode = mode_BeforeHead
			return nil
		}
		if tag.IsType(tokenizer.Token_EndTag) && !slices.Contains([]string{"head", "body", "html", "br"}, tag.Name) {
			return nil
		}
	}

	el := dom.NewHTMLElement("html")

	m.openEls = append(m.openEls, el)
	m.document.Append(el)

	m.tokenizer.ReconsumeToken(token)
	m.insertionMode = mode_BeforeHead

	return nil
}

// https://html.spec.whatwg.org/#the-before-head-insertion-mode
func (m *HTMLParser) mode_BeforeHead(token tokenizer.Token) error {

	if tag, ok := token.(*tokenizer.TokenCharacter); ok && slices.Contains([]rune{'\t', '\n', '\f', '\f', '\r', ' '}, tag.Data) {
		return nil
	}

	if tag, ok := token.(*tokenizer.CommentToken); ok {
		m.insertComment(tag, nil)
		return nil
	}

	if token.IsType(tokenizer.Token_DOCTYPE) {
		return nil
	}

	if tag, ok := token.(*tokenizer.TagToken); ok {
		if tag.IsType(tokenizer.Token_StartTag) && tag.Name == "html" {
			return nil
		}
		if tag.IsType(tokenizer.Token_StartTag) && tag.Name == "head" {
			el := dom.NewHTMLElement("head")
			m.document.Append(el)
			m.document.SetHead(el)

			m.insertionMode = mode_InHead
			return nil
		}
		if tag.IsType(tokenizer.Token_EndTag) && !slices.Contains([]string{"head", "body", "html", "br"}, tag.Name) {
			return nil
		}
	}

	el := dom.NewHTMLElement("head")
	m.document.Append(el)
	m.document.SetHead(el)

	m.insertionMode = mode_InHead
	m.tokenizer.ReconsumeToken(token)
	return nil
}

// https://html.spec.whatwg.org/#parsing-main-inhead
func (m *HTMLParser) mode_InHead(token tokenizer.Token) error {

	if tag, ok := token.(*tokenizer.TokenCharacter); ok && slices.Contains([]rune{'\t', '\n', '\f', '\r', ' '}, tag.Data) {
		return nil
	}

	if tag, ok := token.(*tokenizer.CommentToken); ok {
		node, ok := m.openEls[len(m.openEls)-1].(*dom.HTMLElement)
		if ok {
			node.Append(dom.NewCommentNode(tag.Data))
		}
		return nil
	}

	if token.IsType(tokenizer.Token_DOCTYPE) {
		return nil
	}

	if tag, ok := token.(*tokenizer.TagToken); ok {
		if tag.IsType(tokenizer.Token_StartTag) {
			if tag.Name == "html" {
				return m.Mode_InBody(token)
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

				m.tokenizer.SetState(tokenizer.State_RawText)

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

				m.tokenizer.SetState(tokenizer.State_ScriptData)
				m.originalInsertionMode = m.insertionMode
				m.insertionMode = mode_Text

				return err
			}

			if tag.Name == "template" {
				return nil
			}
		}

		if tag.IsType(tokenizer.Token_EndTag) {
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

	if (token.IsType(tokenizer.Token_StartTag) && token.(*tokenizer.TagToken).Name == "head") || token.IsType(tokenizer.Token_EndTag) {
		return nil
	}

	m.openEls = slices.Delete(m.openEls, len(m.openEls)-1, len(m.openEls)-1)

	m.insertionMode = mode_AfterHead
	m.tokenizer.ReconsumeToken(token)
	return nil
}

// https://html.spec.whatwg.org/#parsing-main-inheadnoscript
func (m *HTMLParser) mode_InHeaderNoScript(token tokenizer.Token) error {

	if token.IsType(tokenizer.Token_DOCTYPE) {
		return nil
	}

	if tag, ok := token.(*tokenizer.TagToken); ok {
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

	if token.IsType(tokenizer.Token_Comment) ||
		isAnyRune(token, '\t', '\n', '\f', '\r', ' ') ||
		isAnyStartTag(token, "basefont", "bgsound", "link", "meta", "noframes", "style") {

		return m.mode_InHead(token)
	}

	m.openEls = m.openEls[:len(m.openEls)-2]
	m.tokenizer.ReconsumeToken(token)
	m.insertionMode = mode_InHead
	return nil
}

// https://html.spec.whatwg.org/#the-after-head-insertion-mode
func (m *HTMLParser) mode_AfterHead(token tokenizer.Token) error {

	if isAnyRune(token, '\t', '\n', '\f', '\r', ' ') {
		m.insertCharacter(token.(*tokenizer.TokenCharacter))
		return nil
	}

	if tag, ok := token.(*tokenizer.CommentToken); ok {
		m.insertComment(tag, nil)
		return nil
	}

	if token.IsType(tokenizer.Token_DOCTYPE) {
		return nil
	}

	if tag, ok := token.(*tokenizer.TagToken); ok {
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
	m.tokenizer.ReconsumeToken(token)
	return nil
}

// https://html.spec.whatwg.org/#parsing-main-inbody
func (m *HTMLParser) mode_InBody(token tokenizer.Token) error {

	if tag, ok := token.(*tokenizer.TokenCharacter); ok {
		if tag.Data == '\u0000' {
			return nil
		}

		if !slices.Contains([]rune{'\t', '\n', '\f', '\r', ' '}, tag.Data) {
			m.framesetOk = "not ok"
		}

		m.reconstructActiveFormattingEls()
		m.insertCharacter(tag)
		return nil
	}

	if tag, ok := token.(*tokenizer.TagToken); ok {

		if tag.IsType(tokenizer.Token_StartTag) && tag.Name == "html" {
			return nil
		}

		if (tag.IsType(tokenizer.Token_StartTag) && slices.Contains([]string{}, tag.Name)) || (tag.IsType(tokenizer.Token_EndTag) && tag.Name == "template") {
			return nil
		}

		if tag.IsType(tokenizer.Token_StartTag) && tag.Name == "body" {
			return nil
		}

		if tag.IsType(tokenizer.Token_StartTag) && tag.Name == "frameset" {
			return nil
		}

		if tag.IsType(tokenizer.Token_EndTag) && tag.Name == "body" {
			return nil
		}

		if tag.IsType(tokenizer.Token_EndTag) && tag.Name == "html" {
			return nil
		}

		if tag.IsType(tokenizer.Token_StartTag) && slices.Contains([]string{
			"address", "article", "aside", "blockquote", "center", "details", "dialog", "dir", "div", "dl",
			"fieldset", "figcaption", "figure", "footer", "header", "hgroup", "main", "menu", "nav", "ol",
			"p", "search", "section", "summary", "ul",
		}, tag.Name) {
			return nil
		}

		if tag.IsType(tokenizer.Token_StartTag) && slices.Contains([]string{"h1", "h2", "h3", "h4", "h5", "h6"}, tag.Name) {
			return nil
		}

		if tag.IsType(tokenizer.Token_StartTag) && slices.Contains([]string{"pre", "listing"}, tag.Name) {
			return nil
		}

		if tag.IsType(tokenizer.Token_StartTag) && tag.Name == "form" {
			return nil
		}

		if tag.IsType(tokenizer.Token_StartTag) && tag.Name == "li" {
			return nil
		}

		if tag.IsType(tokenizer.Token_StartTag) && slices.Contains([]string{"dd", "dt"}, tag.Name) {
			return nil
		}

		if tag.IsType(tokenizer.Token_StartTag) && tag.Name == "plaintext" {
			return nil
		}

		if tag.IsType(tokenizer.Token_StartTag) && tag.Name == "button" {
			return nil
		}

		if tag.IsType(tokenizer.Token_EndTag) && slices.Contains([]string{
			"address", "article", "aside", "blockquote", "button", "center", "details", "dialog", "dir", "div", "dl",
			"fieldset", "figcaption", "figure", "footer", "header", "hgroup", "listing", "main", "menu", "nav", "ol",
			"pre", "search", "section", "select", "summary", "ul",
		}, tag.Name) {
			return nil
		}

		if tag.IsType(tokenizer.Token_EndTag) && tag.Name == "form" {
			return nil
		}

		if tag.IsType(tokenizer.Token_EndTag) && tag.Name == "p" {
			return nil
		}

		if tag.IsType(tokenizer.Token_EndTag) && tag.Name == "li" {
			return nil
		}

		if tag.IsType(tokenizer.Token_EndTag) && slices.Contains([]string{"dd", "dt"}, tag.Name) {
			return nil
		}

		if tag.IsType(tokenizer.Token_EndTag) && slices.Contains([]string{"h1", "h2", "h3", "h4", "h5", "h6"}, tag.Name) {
			return nil
		}

		if tag.IsType(tokenizer.Token_EndTag) && tag.Name == "sarcasm" {
			return nil
		}

		if tag.IsType(tokenizer.Token_StartTag) && tag.Name == "a" {
			return nil
		}

		if tag.IsType(tokenizer.Token_StartTag) && slices.Contains([]string{
			"b", "big", "code", "em", "font", "i", "s", "small", "strike", "strong", "tt", "u",
		}, tag.Name) {
			return nil
		}

		if tag.IsType(tokenizer.Token_StartTag) && tag.Name == "nobr" {
			return nil
		}

		if tag.IsType(tokenizer.Token_EndTag) && slices.Contains([]string{
			"a", "b", "big", "code", "em", "font", "i", "nobr", "s", "small", "strike", "strong", "tt", "u",
		}, tag.Name) {
			return nil
		}

		if tag.IsType(tokenizer.Token_StartTag) && slices.Contains([]string{"applet", "marquee", "object"}, tag.Name) {
			return nil
		}

		if tag.IsType(tokenizer.Token_EndTag) && slices.Contains([]string{"applet", "marquee", "object"}, tag.Name) {
			return nil
		}

		if tag.IsType(tokenizer.Token_StartTag) && tag.Name == "table" {
			return nil
		}

		if tag.IsType(tokenizer.Token_EndTag) && tag.Name == "br" {
			return nil
		}

		if tag.IsType(tokenizer.Token_StartTag) && slices.Contains([]string{
			"area", "br", "embed", "img", "keygen", "wbr",
		}, tag.Name) {
			return nil
		}

		if tag.IsType(tokenizer.Token_StartTag) && tag.Name == "input" {
			return nil
		}

		if tag.IsType(tokenizer.Token_StartTag) && slices.Contains([]string{"param", "source", "track"}, tag.Name) {
			return nil
		}

		if tag.IsType(tokenizer.Token_StartTag) && tag.Name == "hr" {
			return nil
		}

		if tag.IsType(tokenizer.Token_StartTag) && tag.Name == "image" {
			return nil
		}

		if tag.IsType(tokenizer.Token_StartTag) && tag.Name == "textarea" {
			return nil
		}

		if tag.IsType(tokenizer.Token_StartTag) && tag.Name == "xmp" {
			return nil
		}

		if tag.IsType(tokenizer.Token_StartTag) && tag.Name == "iframe" {
			return nil
		}

		if tag.IsType(tokenizer.Token_StartTag) && (tag.Name == "noembed" || (tag.Name == "noscript" && m.scripting)) {
			return nil
		}

		if tag.IsType(tokenizer.Token_StartTag) && tag.Name == "select" {
			return nil
		}

		if tag.IsType(tokenizer.Token_StartTag) && tag.Name == "option" {
			return nil
		}

		if tag.IsType(tokenizer.Token_StartTag) && tag.Name == "optgroup" {
			return nil
		}

		if tag.IsType(tokenizer.Token_EndTag) && tag.Name == "option" {
			return nil
		}

		if tag.IsType(tokenizer.Token_StartTag) && slices.Contains([]string{"rb", "rtc"}, tag.Name) {
			return nil
		}

		if tag.IsType(tokenizer.Token_StartTag) && slices.Contains([]string{"rp", "rt"}, tag.Name) {
			return nil
		}

		if tag.IsType(tokenizer.Token_StartTag) && tag.Name == "math" {
			return nil
		}

		if tag.IsType(tokenizer.Token_StartTag) && tag.Name == "svg" {
			return nil
		}

		if tag.IsType(tokenizer.Token_StartTag) && slices.Contains([]string{
			"caption", "col", "colgroup", "frame", "head", "tbody",
			"td", "tfoot", "th", "thead", "tr"}, tag.Name) {
			return nil
		}

		if tag.IsType(tokenizer.Token_StartTag) {
			return nil
		}

		if tag.IsType(tokenizer.Token_EndTag) {
			return nil
		}
	}

	if tag, ok := token.(*tokenizer.CommentToken); ok {
		_, err := m.insertComment(tag, nil)
		return err
	}

	if token.IsType(tokenizer.Token_EOF) {
		return nil
	}

	return nil
}

func (m *HTMLParser) Mode_Text(token tokenizer.Token) error {

	if tag, ok := token.(*tokenizer.TokenCharacter); ok {
		return nil
	}

	if token.IsType(tokenizer.Token_DOCTYPE) {
		return nil
	}

	if tag, ok := token.(*tokenizer.TagToken); ok {
		if tag.IsType(tokenizer.Token_EndTag) && tag.Name == "script" {
			return nil
		}

		if tag.IsType(tokenizer.Token_EndTag) {

		}
	}

	return nil
}

func (m *HTMLParser) Mode_InTable(token tokenizer.Token) error {
	panic("not implemented")
}

func (m *HTMLParser) Mode_InTableText(token tokenizer.Token) error {
	panic("not implemented")
}

func (m *HTMLParser) Mode_InCaption(token tokenizer.Token) error {
	panic("not implemented")
}

func (m *HTMLParser) Mode_InColumnGroup(token tokenizer.Token) error {
	panic("not implemented")
}

func (m *HTMLParser) Mode_InTableBody(token tokenizer.Token) error {
	panic("not implemented")
}

func (m *HTMLParser) Mode_InRow(token tokenizer.Token) error {
	panic("not implemented")
}

func (m *HTMLParser) Mode_InCell(token tokenizer.Token) error {
	panic("not implemented")
}

func (m *HTMLParser) Mode_InTemplate(token tokenizer.Token) error {
	panic("not implemented")
}

func (m *HTMLParser) Mode_AfterBody(token tokenizer.Token) error {
	panic("not implemented")
}

func (m *HTMLParser) Mode_InFrameset(token tokenizer.Token) error {
	panic("not implemented")
}

func (m *HTMLParser) Mode_AfterFrameset(token tokenizer.Token) error {
	panic("not implemented")
}

func (m *HTMLParser) Mode_AfterAfterBody(token tokenizer.Token) error {
	panic("not implemented")
}

func (m *HTMLParser) Mode_AfterAfterFrameset(token tokenizer.Token) error {
	panic("not implemented")
}

func NewHTMLParser(reader io.RuneReader) *HTMLParser {
	return &HTMLParser{}
}
