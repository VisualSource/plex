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

			if token.IsType(tokenizer.Token_EOF) {
				break outer
			}

		}
	}

	return nil
}

func (m *HTMLParser) InsertHtmlElement(token tokenizer.Token, namespace string, onlyAddToElementStack bool) (*dom.HTMLElement, error) {

	tag, ok := token.(*tokenizer.TagToken)
	if !ok {
		return nil, errors.New("Invalid token type")
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
func (m *HTMLParser) Mode_Initial(token tokenizer.Token) error {

	if tag, ok := token.(*tokenizer.TokenCharacter); ok {
		if slices.Contains([]rune{'\t', '\n', '\f', '\r', ' '}, tag.Data) {
			return nil
		}
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

func (m *HTMLParser) Mode_BeforeHtml(token tokenizer.Token) error {

	if _, ok := token.(*tokenizer.DoctypeToken); ok {
		return nil
	}

	if tag, ok := token.(*tokenizer.CommentToken); ok {
		m.document.Append(dom.NewCommentNode(tag.Data))
		return nil
	}

	if tag, ok := token.(*tokenizer.TokenCharacter); ok {
		if slices.Contains([]rune{'\t', '\n', '\f', '\r', ' '}, tag.Data) {
			return nil
		}
	}

	if tag, ok := token.(*tokenizer.TagToken); ok {
		if tag.IsType(tokenizer.Token_StartTag) && tag.Name == "html" {
			el := dom.NewHTMLElement("html")

			m.openEls = append(m.openEls, el)
			m.document.Append(el)

			m.insertionMode = mode_BeforeHead
			return nil
		}
		if tag.IsType(tokenizer.Token_EndTag) && !slices.Contains([]string{}, tag.Name) {
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

func (m *HTMLParser) Mode_BeforeHead(token tokenizer.Token) error {

	if tag, ok := token.(*tokenizer.TokenCharacter); ok &&
		slices.Contains([]rune{'\t', '\n', '\f', '\f', '\r', ' '}, tag.Data) {
		return nil
	}

	if tag, ok := token.(*tokenizer.CommentToken); ok {
		if node, ok := m.openEls[len(m.openEls)-1].(*dom.HTMLElement); ok {
			node.Append(dom.NewCommentNode(tag.Data))
		}
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
func (m *HTMLParser) Mode_InHead(token tokenizer.Token) error {

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

func (m *HTMLParser) Mode_InHeaderNoScript(token tokenizer.Token) error {
	return nil
}

func (m *HTMLParser) Mode_AfterHead(token tokenizer.Token) error {
	return nil
}

func (m *HTMLParser) Mode_InBody(token tokenizer.Token) error {
	return nil
}

func (m *HTMLParser) Mode_Text(token tokenizer.Token) error {
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
