package html_parser

import (
	"io"
	"strings"

	"github.com/VisualSource/plex/internal/dom"
	"github.com/VisualSource/plex/internal/html_tokenizer"
	"github.com/VisualSource/plex/internal/utils"
)

// https://html.spec.whatwg.org/multipage/parsing.html#parse-state
type HtmlParser struct {
	tokenizer *html_tokenizer.Tokenizer
	// https://html.spec.whatwg.org/multipage/parsing.html#the-insertion-mode
	insertionMode InsertionMode
	// https://html.spec.whatwg.org/multipage/parsing.html#the-stack-of-open-elements
	openElementsStack []dom.Node
	// https://html.spec.whatwg.org/#the-list-of-active-formatting-elements
	activeFormattingElements []dom.Node
	// https://html.spec.whatwg.org/#other-parsing-state-flags
	scriptingMode ScriptingMode

	framesetOk bool

	fosterParenting bool

	speculativeParser *SpeculativeHTMLParser

	document *dom.Document
}

func NewHtmlParser(stream io.Reader) *HtmlParser {
	return &HtmlParser{
		insertionMode: mode_Initial,
		tokenizer:     html_tokenizer.NewTokenizer(stream),
	}
}

// https://html.spec.whatwg.org/#tree-construction
func (p *HtmlParser) Parse() (*dom.Document, error) {
	defer p.parseEnd()

	for {
		err := p.tokenizer.Next()
		if err != nil && err != io.EOF {
			return nil, err
		}

		for p.tokenizer.HasEmittedTokens() {
			token := p.tokenizer.ConsumeToken()

			aj := p.adjustedCurrentNode()
			isStanderd := false

			if aj.Namespace == NamespaceHTML || len(p.openElementsStack) == 0 {
				isStanderd = true
			} else {
				switch tag := token.(type) {
				case html_tokenizer.TokenTag:
					name := tag.GetName()
					isStanderd = (isMathMLIntegrationPoint(aj) && name != "mglyph" && name != "malignmark") ||
						isHTMLIntergrationPoint(aj) ||
						aj.Namespace == NamespaceMathML && aj.Name == "annotation-xml" && name == "svg"
				case html_tokenizer.TokenCharacter:
					isStanderd = isMathMLIntegrationPoint(aj) || isHTMLIntergrationPoint(aj)
				case html_tokenizer.TokenEOF:
					isStanderd = true
				}
			}

			if !isStanderd {
				if err := p.foreginContent(token); err != nil {
					return nil, err
				}
				continue
			}

			var err error
			switch p.insertionMode {
			case mode_Initial:
				err = p.state_Initial(token)
			case mode_BeforeHtml:
				err = p.state_BeforeHtml(token)
			case mode_BeforeHead:
				err = p.state_BeforeHead(token)
			case mode_InHead:
				err = p.state_InHead(token)
			case mode_InHeadNoScript:
				err = p.state_InHeadNoScript(token)
			case mode_AfterHead:
				err = p.state_AfterHead(token)
			case mode_InBody:
				err = p.state_InBody(token)
			case mode_Text:
				err = p.state_Text(token)
			case mode_InTable:
				err = p.state_InTable(token)
			case mode_InTableText:
				err = p.state_InTableText(token)
			case mode_InCaption:
				err = p.state_InCaption(token)
			case mode_InColumnGroup:
				err = p.state_InColumnGroup(token)
			case mode_InTableBody:
				err = p.state_InTableBody(token)
			case mode_InRow:
				err = p.state_InRow(token)
			case mode_InCell:
				err = p.state_InCell(token)
			case mode_InTemplate:
				err = p.state_InTemplate(token)
			case mode_AfterBody:
				err = p.state_AfterBody(token)
			case mode_InFrameset:
				err = p.state_InFrameset(token)
			case mode_AfterFrameset:
				err = p.state_AfterAfterFrameset(token)
			case mode_AfterAfterBody:
				err = p.state_AfterAfterBody(token)
			case mode_AfterAfterFrameset:
				err = p.state_AfterAfterFrameset(token)
			}

			if err != nil {
				return nil, err
			}

		}
	}

	return nil, nil
}

// https://html.spec.whatwg.org/multipage/parsing.html#adjusted-current-node
func (p *HtmlParser) adjustedCurrentNode() dom.Node {

	return p.currentNode()
}

// https://html.spec.whatwg.org/multipage/parsing.html#current-node
func (p *HtmlParser) currentNode() dom.Node {
	node := p.openElementsStack[len(p.openElementsStack)-1]

	return node
}

// https://html.spec.whatwg.org/multipage/parsing.html#the-end
func (*HtmlParser) parseEnd() {}

//#region Helpers

// https://html.spec.whatwg.org/multipage/parsing.html#appropriate-place-for-inserting-a-node
func (p *HtmlParser) getInsertionPosition() dom.Node {

	if p.fosterParenting {

	} else {

	}

	return nil
}

// https://html.spec.whatwg.org/multipage/parsing.html#create-an-element-for-the-token
func (p *HtmlParser) createElement(token html_tokenizer.Token, namespace Namespace, intendedParent dom.Node) dom.Node {
	if p.speculativeParser != nil {

		return nil
	}

	return nil
}

// https://html.spec.whatwg.org/multipage/parsing.html#insert-an-element-at-the-adjusted-insertion-location
func (p *HtmlParser) insertElement(element dom.Node) {
	adjInsertLocation := p.getInsertionPosition()

	if adjInsertLocation == nil {
		return
	}
}

func (p *HtmlParser) insertForeginElement(token html_tokenizer.Token, namespace Namespace, onlyAddToElementStack bool) dom.Node {
	adjInsertLocation := p.getInsertionPosition()

	el := p.createElement(token, namespace, adjInsertLocation)

	if !onlyAddToElementStack {
		p.insertElement(el)
	}

	p.openElementsStack = append(p.openElementsStack, el)

	return el
}

func (p *HtmlParser) insertHtmlElement(token html_tokenizer.Token) dom.Node {
	return p.insertForeginElement(token, NamespaceHTML, false)
}

func (p *HtmlParser) insertComment(data string, position dom.Node) {

	if position == nil {
		position = p.getInsertionPosition()
	}
}

//#endregion

//#region InsertionModes

// https://html.spec.whatwg.org/multipage/parsing.html#the-initial-insertion-mode
func (p *HtmlParser) state_Initial(token html_tokenizer.Token) error {

	switch tag := token.(type) {
	case html_tokenizer.TokenComment:
		p.insertComment(tag.Value, nil)
		return nil
	case html_tokenizer.TokenDOCTYPE:
		pubIdent := tag.GetPublicIdentifer()
		sysIdent := tag.GetSystemIdentifer()
		name := tag.GetName()

		if !name.Is("html") || pubIdent.IsSome() || sysIdent.IsSome() && !sysIdent.Is("about:legacy-compat") {
			// parse error!
		}

		node := dom.NewDocumentType(
			utils.ValueOf(name.Value, ""),
			utils.ValueOf(pubIdent.Value, ""),
			utils.ValueOf(sysIdent.Value, ""),
		)

		p.document.AppendChild(node)

		quirksMode := 0

		// no iframe srcdoc && parserNoChange=false

		if tag.GetForceQuirks() || !name.Is("html") {
		}

		if pubIdent.IsSome() {
			if strings.EqualFold("-//W3O//DTD W3 HTML Strict 3.0//EN//", *pubIdent.Value) ||
				strings.EqualFold("-/W3C/DTD HTML 4.0 Transitional/EN", *pubIdent.Value) ||
				strings.EqualFold("HTML", *pubIdent.Value) {

			}
		}

		p.insertionMode = mode_BeforeHtml
		return nil
	case html_tokenizer.TokenCharacter:
		switch tag.Value {
		case '\t', '\n', '\f', '\r', ' ':
			return nil
		}
	}

	p.insertionMode = mode_BeforeHtml

	return nil
}

// https://html.spec.whatwg.org/multipage/parsing.html#the-before-html-insertion-mode
func (p *HtmlParser) state_BeforeHtml(token html_tokenizer.Token) error { return nil }

// https://html.spec.whatwg.org/multipage/parsing.html#the-before-head-insertion-mode
func (p *HtmlParser) state_BeforeHead(token html_tokenizer.Token) error { return nil }

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-inhead
func (p *HtmlParser) state_InHead(token html_tokenizer.Token) error { return nil }

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-inheadnoscript
func (p *HtmlParser) state_InHeadNoScript(token html_tokenizer.Token) error { return nil }

// https://html.spec.whatwg.org/multipage/parsing.html#the-after-head-insertion-mode
func (p *HtmlParser) state_AfterHead(token html_tokenizer.Token) error { return nil }

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-inbody
func (p *HtmlParser) state_InBody(token html_tokenizer.Token) error { return nil }

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-incdata
func (p *HtmlParser) state_Text(token html_tokenizer.Token) error { return nil }

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-intable
func (p *HtmlParser) state_InTable(token html_tokenizer.Token) error { return nil }

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-intabletext
func (p *HtmlParser) state_InTableText(token html_tokenizer.Token) error { return nil }

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-incaption
func (p *HtmlParser) state_InCaption(token html_tokenizer.Token) error { return nil }

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-incolgroup
func (p *HtmlParser) state_InColumnGroup(token html_tokenizer.Token) error { return nil }

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-intbody
func (p *HtmlParser) state_InTableBody(token html_tokenizer.Token) error { return nil }

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-intr
func (p *HtmlParser) state_InRow(token html_tokenizer.Token) error { return nil }

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-intd
func (p *HtmlParser) state_InCell(token html_tokenizer.Token) error { return nil }

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-intemplate
func (p *HtmlParser) state_InTemplate(token html_tokenizer.Token) error { return nil }

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-afterbody
func (p *HtmlParser) state_AfterBody(token html_tokenizer.Token) error { return nil }

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-inframeset
func (p *HtmlParser) state_InFrameset(token html_tokenizer.Token) error { return nil }

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-afterframeset
func (p *HtmlParser) state_AfterFrameset(token html_tokenizer.Token) error { return nil }

// https://html.spec.whatwg.org/multipage/parsing.html#the-after-after-body-insertion-mode
func (p *HtmlParser) state_AfterAfterBody(token html_tokenizer.Token) error { return nil }

// https://html.spec.whatwg.org/multipage/parsing.html#the-after-after-frameset-insertion-mode
func (p *HtmlParser) state_AfterAfterFrameset(token html_tokenizer.Token) error { return nil }

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-inforeign
func (p *HtmlParser) foreginContent(token html_tokenizer.Token) error { return nil }

//#endregion
