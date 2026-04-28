package html_parser

import (
	"io"
	"slices"
	"strings"

	"github.com/VisualSource/plex/internal/dom"
	"github.com/VisualSource/plex/internal/html_tokenizer"
	"github.com/VisualSource/plex/internal/utils"
)

// https://html.spec.whatwg.org/multipage/parsing.html#parse-state
type HtmlParser struct {
	tokenizer *html_tokenizer.Tokenizer
	// https://html.spec.whatwg.org/multipage/parsing.html#the-insertion-mode
	insertionMode         InsertionMode
	originalInsertionMode InsertionMode
	// https://html.spec.whatwg.org/multipage/parsing.html#the-stack-of-open-elements
	openElementsStack []dom.Node
	// https://html.spec.whatwg.org/#the-list-of-active-formatting-elements
	activeFormattingElements []dom.Node
	// https://html.spec.whatwg.org/#other-parsing-state-flags
	scriptingMode ScriptingMode

	framesetOk bool

	fosterParenting bool

	head dom.Node

	speculativeParser *SpeculativeHTMLParser

	document *dom.Document

	isFragmentParsing bool
}

func NewHtmlParser(stream io.Reader) *HtmlParser {
	return &HtmlParser{
		insertionMode: mode_Initial,
		tokenizer:     html_tokenizer.NewTokenizer(stream),
		document:      dom.NewDocument(),
	}
}

// https://html.spec.whatwg.org/#tree-construction
func (p *HtmlParser) Parse() (*dom.Document, error) {
parseLoop:
	for {
		err := p.tokenizer.Next()
		if err != nil && err != io.EOF {
			return nil, err
		}

		for p.tokenizer.HasEmittedTokens() {
			token := p.tokenizer.ConsumeToken()

			aj := p.adjustedCurrentNode()
			isStandard := false

			if aj != nil && aj.Namespace() == dom.NamespaceHTML || len(p.openElementsStack) == 0 {
				isStandard = true
			} else {
				switch tag := token.(type) {
				case html_tokenizer.TokenTag:
					name := tag.GetName()
					isStandard = (isMathMLIntegrationPoint(aj) && name != "mglyph" && name != "malignmark") ||
						isHTMLIntegrationPoint(aj) ||
						aj.Namespace() == dom.NamespaceMathML && aj.Tag() == "annotation-xml" && name == "svg"
				case html_tokenizer.TokenCharacter:
					isStandard = isMathMLIntegrationPoint(aj) || isHTMLIntegrationPoint(aj)
				case html_tokenizer.TokenEOF:
					isStandard = true
				}
			}

			if !isStandard {
				if err := p.foreignContent(token); err != nil {
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
				if err == io.EOF {
					break parseLoop
				}
				return nil, err
			}
		}
	}

	p.parseEnd()

	return p.document, nil
}

func (p *HtmlParser) openStackPop() {
	p.openElementsStack = p.openElementsStack[:len(p.openElementsStack)-1]
}

// https://html.spec.whatwg.org/multipage/parsing.html#adjusted-current-node
func (p *HtmlParser) adjustedCurrentNode() dom.Node {

	return p.currentNode()
}

// https://html.spec.whatwg.org/multipage/parsing.html#current-node
func (p *HtmlParser) currentNode() dom.Node {
	if len(p.openElementsStack) == 0 {
		return nil
	}
	node := p.openElementsStack[len(p.openElementsStack)-1]

	return node
}

// https://html.spec.whatwg.org/multipage/parsing.html#the-end
func (*HtmlParser) parseEnd() {}

//#region Helpers

func (p *HtmlParser) lastElementOfType(nodeType string) (dom.Node, int) {
	for i := len(p.openElementsStack) - 1; i >= 0; i-- {
		if p.openElementsStack[i].Tag() == nodeType {
			return p.openElementsStack[i], i
		}
	}
	return nil, -1
}

// https://html.spec.whatwg.org/multipage/parsing.html#appropriate-place-for-inserting-a-node
func (p *HtmlParser) appropriatePlaceForInsertingNode(overrideTarget dom.Node) (dom.Node, InsertionPosition) {
	var target dom.Node
	if overrideTarget != nil {
		target = overrideTarget
	} else {
		target = p.currentNode()
	}

	var adjusted dom.Node = target
	if p.fosterParenting && slices.Contains([]string{"table", "tbody", "tfoot", "thead", "tr"}, target.Tag()) {
		lastTemplate, tempIdx := p.lastElementOfType("template")
		lastTable, tableIdx := p.lastElementOfType("table")

		if lastTemplate != nil && (tableIdx != -1 || tempIdx > tableIdx) {
			return lastTemplate, Insert_After
		} else if tableIdx == -1 {
			return p.openElementsStack[0], Insert_After
		} else if lastTable != nil && lastTable.Parent() != nil {
			return lastTable.Parent(), Insert_Before
		}

		prev := p.openElementsStack[tableIdx-1]
		return prev, Insert_After
	}

	return adjusted, Insert_After
}

// https://html.spec.whatwg.org/multipage/parsing.html#create-an-element-for-the-token
func (p *HtmlParser) createElement(token html_tokenizer.TokenTag, namespace dom.Namespace, intendedParent dom.Node) dom.Node {
	if p.speculativeParser != nil {
		return nil //TODO: create mock element
	}
	//TODO: create speculative mock element

	document := intendedParent.Document()
	is := token.Attributes.Get("is")

	//TODO: custom element registery lookup
	//TODO: custom element Definition lookup
	registry := utils.None[string]()
	definition := utils.None[string]()

	willExecuteScript := definition.IsNone() && !p.isFragmentParsing

	if willExecuteScript {
		//TODO: document write/open re-entrancy
	}

	element := dom.NewElement(
		document,
		token.GetName(),
		utils.Some(namespace),
		utils.None[string](),
		is,
		willExecuteScript,
		registry,
		intendedParent,
	)

	for key, value := range token.Attributes {
		element.SetAttribute(key, value)
	}

	if willExecuteScript {
		//TODO: custom element flush reactions and tear down guard
	}

	if xmlnsAttr := element.GetAttributeNS(dom.NamespaceXMLNS, "xmlns"); xmlnsAttr != nil {
		if xmlnsAttr.Value != string(element.Namespace()) {
			//TODO: parse error
		}
	}

	if xLinkAttr := element.GetAttributeNS(dom.NamespaceXMLNS, "xlink"); xLinkAttr != nil {
		if xLinkAttr.Value != string(dom.NamespaceXLink) {
			//TODO: parse error
		}
	}

	// TODO: step 14
	// TODO: step 15

	return element
}

// https://html.spec.whatwg.org/multipage/parsing.html#insert-an-element-at-the-adjusted-insertion-location
func (p *HtmlParser) insertElement(element dom.Node) {
	adjInsertLocation, insertPosition := p.appropriatePlaceForInsertingNode(nil)

	if adjInsertLocation == nil {
		return
	}

	switch insertPosition {
	case Insert_Before:
		adjInsertLocation.PrependChild(element)
	case Insert_After:
		adjInsertLocation.AppendChild(element)
	}

}

// https://html.spec.whatwg.org/multipage/parsing.html#insert-a-foreign-element
func (p *HtmlParser) insertForeignElement(token html_tokenizer.TokenTag, namespace dom.Namespace, onlyAddToElementStack bool) dom.Node {
	adjInsertLocation, _ := p.appropriatePlaceForInsertingNode(nil)

	el := p.createElement(token, namespace, adjInsertLocation)

	if !onlyAddToElementStack {
		p.insertElement(el)
	}

	p.openElementsStack = append(p.openElementsStack, el)

	return el
}

// https://html.spec.whatwg.org/multipage/parsing.html#insert-an-html-element
func (p *HtmlParser) insertHtmlElement(token html_tokenizer.TokenTag) dom.Node {
	return p.insertForeignElement(token, dom.NamespaceHTML, false)
}

func (p *HtmlParser) insertComment(data string, position dom.Node) {

	var aj dom.Node = position
	var insertPos InsertionPosition = Insert_After
	if position == nil {
		aj, insertPos = p.appropriatePlaceForInsertingNode(nil)
	}

	document := aj.Document()
	comment := dom.NewComment(document, aj, data)

	switch insertPos {
	case Insert_After:
		aj.AppendChild(comment)
	case Insert_Before:
		aj.PrependChild(comment)
	}
}

// https://html.spec.whatwg.org/multipage/parsing.html#insert-a-character
func (p *HtmlParser) insertCharacter(value rune) {

	aj, pos := p.appropriatePlaceForInsertingNode(nil)
	if aj.IsNode() == 0 {
		return
	}

	prev := aj.PreviousSibling()
	if textNode, ok := prev.(*dom.Text); ok {
		textNode.Data += string(value)
		return
	}

	document := aj.Document()
	text := dom.NewTextNode(document, string(value), aj)

	switch pos {
	case Insert_After:
		aj.AppendChild(text)
	case Insert_Before:
		aj.PrependChild(text)
	}
}

// https://html.spec.whatwg.org/multipage/parsing.html#reconstruct-the-active-formatting-elements
func (p *HtmlParser) reconstructActiveFormattingElements() {}

func (p *HtmlParser) genericElementParse(token html_tokenizer.TokenTag, alg string) {
	p.insertHtmlElement(token)

	if alg == "text" {
		p.tokenizer.SetState(html_tokenizer.State_PlainText)
	} else {
		p.tokenizer.SetState(html_tokenizer.State_RCData)
	}

	p.originalInsertionMode = p.insertionMode
	p.insertionMode = mode_Text
}

//#endregion

//#region InsertionModes

// https://html.spec.whatwg.org/multipage/parsing.html#the-initial-insertion-mode
func (p *HtmlParser) state_Initial(token html_tokenizer.Token) error {

	switch tag := token.(type) {
	case *html_tokenizer.TokenComment:
		p.insertComment(tag.Value, nil)
		return nil
	case *html_tokenizer.TokenDOCTYPE:
		pubIdent := tag.GetPublicIdentifier()
		sysIdent := tag.GetSystemIdentifier()
		name := tag.GetName()

		if !name.Is("html") || pubIdent.IsSome() || sysIdent.IsSome() && !sysIdent.Is("about:legacy-compat") {
			//TODO: parse error
		}

		node := dom.NewDocumentType(
			utils.ValueOf(name.Value, ""),
			utils.ValueOf(pubIdent.Value, ""),
			utils.ValueOf(sysIdent.Value, ""),
		)

		p.document.AppendChild(node)

		if !p.document.IsIframeSrcDoc() && !p.document.ParserNoChangeMode {
			if tag.GetForceQuirks() || !name.Is("html") {
				p.document.QuirksMode = dom.QuirksMode_Quirks
			} else if pubIdent.IsSome() {
				if (sysIdent.IsNone() || sysIdent.Is("")) && (strings.HasPrefix(strings.ToUpper(*pubIdent.Value), "-//W3C//DTD HTML 4.01 Frameset//") ||
					strings.HasPrefix(strings.ToUpper(*pubIdent.Value), "-//W3C//DTD HTML 4.01 Transitional//")) {
					p.document.QuirksMode = dom.QuirksMode_Limited
				} else if slices.ContainsFunc(doctypeDtD, func(dtd string) bool { return strings.HasPrefix(strings.ToUpper(*pubIdent.Value), dtd) }) {
					p.document.QuirksMode = dom.QuirksMode_Quirks
				} else if strings.HasPrefix(strings.ToUpper(*pubIdent.Value), "-//W3C//DTD XHTML 1.0 Frameset//") || strings.HasPrefix(strings.ToUpper(*pubIdent.Value), "-//W3C//DTD XHTML 1.0 Transitional//") {
					p.document.QuirksMode = dom.QuirksMode_Limited
				}
			}
		}

		p.insertionMode = mode_BeforeHtml
		return nil
	case *html_tokenizer.TokenCharacter:
		switch tag.Value {
		case '\t', '\n', '\f', '\r', ' ':
			return nil
		}
	}

	if !p.document.IsIframeSrcDoc() {
		//TODO parser error
	}

	if !p.document.ParserNoChangeMode {
		p.document.QuirksMode = dom.QuirksMode_Quirks
	}

	p.insertionMode = mode_BeforeHtml
	p.tokenizer.ReconsumeToken(token)

	return nil
}

// https://html.spec.whatwg.org/multipage/parsing.html#the-before-html-insertion-mode
func (p *HtmlParser) state_BeforeHtml(token html_tokenizer.Token) error {

	switch tag := token.(type) {
	case *html_tokenizer.TokenDOCTYPE:
		return nil
	case *html_tokenizer.TokenComment:
		p.insertComment(tag.Value, nil)
	case *html_tokenizer.TokenCharacter:
		switch tag.Value {
		case '\t', '\n', '\f', '\r', ' ':
			return nil
		}
	case *html_tokenizer.TokenTag:
		name := tag.GetName()
		tagType := tag.GetType()
		if tagType == html_tokenizer.TokenStartTag && name == "html" {

			node := p.createElement(*tag, dom.NamespaceHTML, p.document)
			p.openElementsStack = append(p.openElementsStack, node)

			p.insertionMode = mode_BeforeHead
			return nil
		} else if tagType == html_tokenizer.TokenEndTag && !slices.Contains([]string{"head", "body", "html", "br"}, name) {
			return nil
		}
	}

	html := html_tokenizer.NewTokenTag("html", html_tokenizer.TokenStartTag, utils.None[bool]())
	node := p.createElement(*html, dom.NamespaceHTML, p.document)
	p.openElementsStack = append(p.openElementsStack, node)

	p.insertionMode = mode_BeforeHead
	p.tokenizer.ReconsumeToken(token)

	return nil
}

// https://html.spec.whatwg.org/multipage/parsing.html#the-before-head-insertion-mode
func (p *HtmlParser) state_BeforeHead(token html_tokenizer.Token) error {

	switch tag := token.(type) {
	case *html_tokenizer.TokenCharacter:
		switch tag.Value {
		case '\t', '\n', '\f', ' ', '\r':
			return nil
		}
	case *html_tokenizer.TokenComment:
		p.insertComment(tag.Value, nil)
		return nil
	case *html_tokenizer.TokenDOCTYPE:
		//TODO: parser error
		return nil
	case *html_tokenizer.TokenTag:
		{
			name := tag.GetName()
			if tag.GetType() == html_tokenizer.TokenStartTag {
				switch name {
				case "html":
					return p.state_InBody(token)
				case "head":
					node := p.insertHtmlElement(*tag)

					p.head = node
					p.insertionMode = mode_InHead
					return nil
				}
			} else if !slices.Contains([]string{"head", "body", "html", "br"}, name) {
				//TODO: parser error
				return nil
			}
		}
	}
	head := html_tokenizer.NewTokenTag("head", html_tokenizer.TokenStartTag, utils.None[bool]())
	node := p.insertHtmlElement(*head)
	p.head = node
	p.insertionMode = mode_InHead

	p.tokenizer.ReconsumeToken(token)
	return nil
}

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-inhead
func (p *HtmlParser) state_InHead(token html_tokenizer.Token) error {

	switch tag := token.(type) {
	case *html_tokenizer.TokenCharacter:
		switch tag.Value {
		case '\t', '\n', '\f', '\r', ' ':
			p.insertCharacter(tag.Value)
			return nil
		}
	case *html_tokenizer.TokenComment:
		p.insertComment(tag.Value, nil)
		return nil
	case *html_tokenizer.TokenDOCTYPE:
		//TODO: parser error
		return nil
	case *html_tokenizer.TokenTag:
		tagType := tag.GetType()
		name := tag.GetName()

		if tagType == html_tokenizer.TokenStartTag {
			switch name {
			case "html":
				return p.state_InBody(token)
			case "base", "basefont", "bgsound", "link":
				p.insertHtmlElement(*tag)
				p.openStackPop()

				if !tag.IsSelfClosingSet() {
					//TODO: parse error
				}
				return nil
			case "meta":
				p.insertHtmlElement(*tag)
				p.openStackPop()

				if !tag.IsSelfClosingSet() {
					//TODO: parse error
				}

				if p.speculativeParser == nil {
					/*
						TODO:
						If the element has a charset attribute,
						and getting an encoding from its value results in an encoding,
						and the confidence is currently tentative,
						then change the encoding to the resulting encoding.

						ELSE

						if the element has an http-equiv attribute
						whose value is an ASCII case-insensitive match for "Content-Type",
						and the element has a content attribute, and applying the algorithm
						for extracting a character encoding from a meta element to that attribute's
						value returns an encoding, and the confidence is currently tentative,
						then change the encoding to the extracted encoding.

					*/
				}
				return nil
			case "title":
				p.genericElementParse(*tag, "rcdata")
				return nil
			case "noscript":
				if p.scriptingMode != mode_Disabled {
					p.genericElementParse(*tag, "text")
				} else {
					p.insertHtmlElement(*tag)
					p.insertionMode = mode_InHeadNoScript
				}
				return nil
			case "noframes", "style":
				p.genericElementParse(*tag, "text")
				return nil
			case "script":

				p.originalInsertionMode = p.insertionMode
				p.insertionMode = mode_Text
				return nil
			case "template":
				return nil
			case "head":
				//TOOD : parse error
				return nil
			}
		} else {
			switch name {
			case "head":
				return nil
			case "body", "html", "br":
			case "template":

				p.framesetOk = false
				p.insertionMode = mode_InTemplate

				//TODO
				return nil
			default:
				//TODO: parse error
				return nil
			}
		}
	}

	p.openStackPop()
	p.insertionMode = mode_AfterHead
	p.tokenizer.ReconsumeToken(token)

	return nil
}

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-inheadnoscript
func (p *HtmlParser) state_InHeadNoScript(token html_tokenizer.Token) error {

	switch tag := token.(type) {
	case html_tokenizer.TokenDOCTYPE:
		//TODO: parse error
		return nil
	case html_tokenizer.TokenTag:
		name := tag.GetName()
		if tag.GetType() == html_tokenizer.TokenStartTag {
			switch name {
			case "html":
				return p.state_InBody(token)
			case "basefont", "bssound", "link", "meta", "noframes", "style":
				return p.state_InHead(token)
			case "head", "noscript":
				// TODO: parse error
				return nil
			}
		} else {
			switch name {
			case "noscript":
				p.openStackPop()
				p.insertionMode = mode_InHead
				return nil
			default:
				if name != "br" {
					return nil
				}

				//TODO: parse error
			}
		}
	case html_tokenizer.TokenCharacter:
		switch tag.Value {
		case '\t', '\f', '\n', '\r', ' ':
			return p.state_InHead(token)
		}
	case html_tokenizer.TokenComment:
		return p.state_InHead(token)
	}

	//TODO: parse error

	p.openStackPop()
	p.insertionMode = mode_InHead
	p.tokenizer.ReconsumeToken(token)

	return nil
}

// https://html.spec.whatwg.org/multipage/parsing.html#the-after-head-insertion-mode
func (p *HtmlParser) state_AfterHead(token html_tokenizer.Token) error {
	switch tag := token.(type) {
	case *html_tokenizer.TokenCharacter:
		switch tag.Value {
		case '\t', '\n', '\f', '\r', ' ':
			p.insertCharacter(tag.Value)
			return nil
		}
	case *html_tokenizer.TokenComment:
		p.insertComment(tag.Value, nil)
		return nil
	case *html_tokenizer.TokenDOCTYPE:
		//TODO: parse error
		return nil
	case *html_tokenizer.TokenTag:
		name := tag.GetName()

		if tag.GetType() == html_tokenizer.TokenStartTag {
			switch name {
			case "html":
				return p.state_InBody(token)
			case "body":
				p.insertHtmlElement(*tag)

				p.framesetOk = true
				p.insertionMode = mode_InBody
				return nil
			case "frameset":
				p.insertHtmlElement(*tag)
				p.insertionMode = mode_InFrameset
				return nil
			case "base", "basefont", "bgsound", "link", "meta", "noframes", "script", "style", "template", "title":
				//TODO: parse error
				p.openElementsStack = append(p.openElementsStack, p.head)

				err := p.state_InHead(token)
				if err != nil {
					return err
				}

				//TODO: remove p.head from open element Stack
				return nil
			case "head":
				//TODO: parse error
				return nil
			}

		} else {
			switch name {
			case "body", "html", "br":
			default:
				//TODO: parse error
				return nil
			}
		}
	}
	body := html_tokenizer.NewTokenTag("body", html_tokenizer.TokenStartTag, utils.None[bool]())
	p.insertHtmlElement(*body)

	p.insertionMode = mode_InBody
	p.tokenizer.ReconsumeToken(token)
	return nil
}

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-inbody
func (p *HtmlParser) state_InBody(token html_tokenizer.Token) error {

	switch tag := token.(type) {
	case *html_tokenizer.TokenCharacter:
		switch tag.Value {
		case '\u0000':
			//TODO: parse error
			return nil
		case '\t', '\n', '\f', '\r', ' ':
			p.reconstructActiveFormattingElements()
			p.insertCharacter(tag.Value)
			return nil
		default:
			p.reconstructActiveFormattingElements()
			p.insertCharacter(tag.Value)

			p.framesetOk = false
			return nil
		}
	case *html_tokenizer.TokenComment:
		p.insertComment(tag.Value, nil)
		return nil
	case *html_tokenizer.TokenDOCTYPE:
		//TODO: parse error
		return nil
	case *html_tokenizer.TokenTag:
		name := tag.GetName()
		if tag.GetType() == html_tokenizer.TokenStartTag {
			switch name {
			case "html":
				//TODO: parse error

				node, _ := p.lastElementOfType("template")
				if node != nil {
					return nil
				}

				if el, ok := p.openElementsStack[0].(*dom.Element); ok {
					for attrName, attrValue := range tag.Attributes {
						if !el.HasAttribute(attrName) {
							el.SetAttribute(attrName, attrValue)
						}
					}
				}

				return nil
			case "base", "basefont", "bgsound", "link", "meta", "noframes", "script", "style", "template", "title":
				return p.state_InHead(token)
			case "body":
				//TODO: parse error
				return nil
			case "frameset":

				if !p.framesetOk {
					return nil
				}

				p.insertionMode = mode_InFrameset
				return nil
			case "address", "article", "aside", "blockquote", "center", "details",
				"dialog", "dir", "div", "dl", "fieldset", "figcaption", "figure", "footer",
				"header", "hgroup", "main", "menu", "nav", "ol", "p", "search", "section", "summary", "ul":

				p.insertHtmlElement(*tag)
				return nil
			case "h1", "h2", "h3", "h4", "h5", "h6":
				//TODO

				p.insertHtmlElement(*tag)
				return nil
			case "pre", "listing":

				//TODO

				p.insertHtmlElement(*tag)
				p.framesetOk = false

				return nil
			case "form":
				//TODO
				return nil
			case "li":
				p.framesetOk = false

				//TODO

				return nil
			case "dd", "dt":
				p.framesetOk = false

				//TODO

				return nil
			case "plaintext":
				//TODO

				p.insertHtmlElement(*tag)
				p.tokenizer.SetState(html_tokenizer.State_PlainText)
				return nil
			case "button":

				//TODO

				p.insertHtmlElement(*tag)
				p.framesetOk = false
				return nil
			case "a":

				p.insertHtmlElement(*tag)
				return nil
			case "b", "big", "code", "em", "font", "i", "s", "small", "strike", "strong", "tt", "u":
				p.insertHtmlElement(*tag)
				return nil
			case "nobr":
				p.insertHtmlElement(*tag)
				return nil
			case "applet", "marquee", "object":
				p.insertHtmlElement(*tag)
				return nil
			case "table":
				p.insertHtmlElement(*tag)
				p.framesetOk = false
				p.insertionMode = mode_InTable
				return nil
			case "area", "br", "embed", "img", "keygen", "wbr":
				p.insertHtmlElement(*tag)
				p.framesetOk = false
				return nil
			case "input":

				return nil
			case "param", "source", "track":
				p.insertHtmlElement(*tag)
				p.openStackPop()
				return nil
			case "hr":
				p.insertHtmlElement(*tag)
				p.framesetOk = false
				return nil
			case "image":

				// tag name to "img" and reporecess it. (Don't ask)
				tag.SetName("img")
				p.tokenizer.ReconsumeToken(token)
				return nil
			case "textarea":
				p.insertHtmlElement(*tag)

				p.tokenizer.SetState(html_tokenizer.State_RCData)
				p.originalInsertionMode = p.insertionMode
				p.framesetOk = false
				p.insertionMode = mode_Text
				return nil
			case "xmp":

				p.framesetOk = false
				p.genericElementParse(*tag, "text")
				return nil
			case "iframe":
				p.framesetOk = false
				p.genericElementParse(*tag, "text")
				return nil
			case "noembed":
				p.genericElementParse(*tag, "text")
				return nil
			case "noscript":
				if p.scriptingMode != mode_Disabled {
					p.genericElementParse(*tag, "text")
				}
				return nil
			case "select":
				p.insertHtmlElement(*tag)
				p.framesetOk = false
				return nil
			case "option":
				p.insertHtmlElement(*tag)
				return nil
			case "optgroup":
				p.insertHtmlElement(*tag)
				return nil
			case "rb", "rtc":
				p.insertHtmlElement(*tag)
				return nil
			case "rp", "rt":
				p.insertHtmlElement(*tag)
				return nil
			case "math":
				return nil
			case "svg":
				return nil
			case "caption", "col", "colgroup", "frame", "head", "tbody", "td", "tfoot", "th", "thead", "tr":
				return nil
			default:
				p.insertHtmlElement(*tag)
				return nil
			}
		} else {
			switch name {
			case "template":
				return p.state_InHead(token)
			case "body":

				p.insertionMode = mode_AfterBody
				return nil
			case "html":

				p.insertionMode = mode_AfterBody
				p.tokenizer.ReconsumeToken(token)
				return nil
			case "address", "article", "aside", "blockquote", "button", "center", "details", "dialong", "dir", "div", "dl", "fieldset",
				"figcaption", "figure", "footer", "header", "hgroup", "listing", "main", "menu", "nav", "ol", "pre", "search",
				"section", "select", "summary", "ul":
				return nil
			case "form":
				return nil
			case "p":
				return nil
			case "li":
				return nil
			case "dd", "dt":
				return nil
			case "h1", "h2", "h3", "h4", "h5", "h6":
				return nil
			case "sarcasm":
				return nil
			case "a", "b", "big", "code", "em", "font", "i", "nobr", "s", "small", "strike", "strong", "tt", "u":
				return nil
			case "applet", "marquee", "object":
				return nil
			case "br":
				return nil
			default:

				return nil

			}
		}
	case *html_tokenizer.TokenEOF:
		//TODO:
		return io.EOF
	}

	return nil
}

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-incdata
func (p *HtmlParser) state_Text(token html_tokenizer.Token) error {
	switch tag := token.(type) {
	case html_tokenizer.TokenCharacter:
		p.insertCharacter(tag.Value)
	case html_tokenizer.TokenEOF:
		p.openStackPop()
		p.insertionMode = p.originalInsertionMode
		p.tokenizer.ReconsumeToken(token)
		return nil
	case html_tokenizer.TokenTag:
		if tag.GetType() != html_tokenizer.TokenEndTag {
			return nil
		}
		if tag.GetName() == "script" {
			return nil
		}

		p.insertionMode = p.originalInsertionMode
	}
	return nil
}

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-intable
func (p *HtmlParser) state_InTable(token html_tokenizer.Token) error {

	switch tag := token.(type) {
	case html_tokenizer.TokenCharacter:
		node := p.currentNode()
		if slices.Contains([]string{"table", "tbody", "template", "tfoot", "tr"}, node.Tag()) {
			p.originalInsertionMode = p.insertionMode
			p.insertionMode = mode_InTableText
			return nil
		}
	case html_tokenizer.TokenComment:
		p.insertComment(tag.Value, nil)
		return nil
	case html_tokenizer.TokenDOCTYPE:
		//TODO: parse error
		return nil
	case html_tokenizer.TokenTag:
		name := tag.GetName()
		if tag.GetType() == html_tokenizer.TokenStartTag {
			switch name {
			case "caption":
				//TODO
				p.insertHtmlElement(tag)
				p.insertionMode = mode_InCaption
				return nil
			case "colgroup":
				//TODO
				p.insertHtmlElement(tag)
				p.insertionMode = mode_InColumnGroup
				return nil
			case "col":
				//TODO
				colgroup := html_tokenizer.NewTokenTag("colgroup", html_tokenizer.TokenStartTag, utils.None[bool]())
				p.insertHtmlElement(*colgroup)
				p.insertionMode = mode_InColumnGroup
				p.tokenizer.ReconsumeToken(token)
				return nil
			case "tbody", "tfoot", "thead":
				//TODO
				p.insertHtmlElement(*html_tokenizer.NewTokenTag("tbody", html_tokenizer.TokenStartTag, utils.None[bool]()))
				p.insertionMode = mode_InTableBody
				return nil
			case "td", "th", "tr":
				//TODO
				p.insertHtmlElement(*html_tokenizer.NewTokenTag("tbody", html_tokenizer.TokenStartTag, utils.None[bool]()))
				p.insertionMode = mode_InTableBody
				p.tokenizer.ReconsumeToken(token)
				return nil
			case "table":
				//TODO: parser error
				//TODO
				return nil
			case "style", "script", "template":
				return p.state_InHead(token)
			case "input":
				value := tag.Attributes.Get("type")
				if !(value.IsNone() || !strings.EqualFold(*value.Value, "hidden")) {
					//TODO
					p.insertHtmlElement(tag)
					return nil
				}
			case "form":
				//TODO
				return nil
			}
		} else {
			switch name {
			case "table":
				//TODO
				return nil
			case "body", "caption", "col", "colgroup", "html", "tbody", "td", "tfoot", "th", "thead", "tr":
				//TODO: parse error
				return nil
			case "template":
				return p.state_InHead(token)
			}
		}
	case html_tokenizer.TokenEOF:
		return p.state_InBody(token)
	}

	//TODO: parse error
	p.fosterParenting = true
	if err := p.state_InBody(token); err != nil {
		return err
	}
	p.fosterParenting = false

	return nil
}

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-intabletext
func (p *HtmlParser) state_InTableText(token html_tokenizer.Token) error {

	switch tag := token.(type) {
	case html_tokenizer.TokenCharacter:
		switch tag.Value {
		case '\u0000':
			//TODO: parse error
		default:
		}
	default:
		p.insertionMode = p.originalInsertionMode
		p.tokenizer.ReconsumeToken(token)
	}

	return nil
}

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-incaption
func (p *HtmlParser) state_InCaption(token html_tokenizer.Token) error {
	switch tag := token.(type) {
	case html_tokenizer.TokenCharacter:
	case html_tokenizer.TokenTag:
		name := tag.GetName()
		if tag.GetType() == html_tokenizer.TokenStartTag {
			switch name {
			case "caption", "col", "colgroup", "tbody", "td", "tfoot", "th", "thead", "tr":
				//TODO

				p.insertionMode = mode_InTable
				p.tokenizer.ReconsumeToken(token)
				return nil
			}
		} else {
			switch name {
			case "caption":
				//TODO
				p.insertionMode = mode_InTable
				return nil
			case "body", "col", "colgroup", "html", "tbody", "td", "tfoot", "th", "thead", "tr":
				//TODO: parser error
				return nil
			}
		}
	}

	return p.state_InBody(token)
}

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-incolgroup
func (p *HtmlParser) state_InColumnGroup(token html_tokenizer.Token) error {

	switch tag := token.(type) {
	case html_tokenizer.TokenCharacter:
		switch tag.Value {
		case '\t', '\n', '\f', '\r', ' ':
			p.insertCharacter(tag.Value)
			return nil
		}
	case html_tokenizer.TokenComment:
		p.insertComment(tag.Value, nil)
		return nil
	case html_tokenizer.TokenDOCTYPE:
		//TODO: parse error
		return nil
	case html_tokenizer.TokenTag:
		name := tag.GetName()
		if tag.GetType() == html_tokenizer.TokenStartTag {
			switch name {
			case "html":
				return p.state_InBody(token)
			case "col":
				p.insertHtmlElement(tag)
				p.openStackPop()
				//TODO
				return nil
			case "template":
				return p.state_InHead(token)
			}
		} else {
			switch name {
			case "colgroup":
				//TODO
				return nil
			case "col":
				//TODO: parse error
				return nil
			case "template":
				return p.state_InHead(token)
			}
		}
	case html_tokenizer.TokenEOF:
		return p.state_InBody(token)
	}

	//TODO
	p.insertionMode = mode_InTable
	p.tokenizer.ReconsumeToken(token)
	return nil
}

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-intbody
func (p *HtmlParser) state_InTableBody(token html_tokenizer.Token) error {
	if tag, ok := token.(*html_tokenizer.TokenTag); ok {
		name := tag.GetName()

		if tag.GetType() == html_tokenizer.TokenStartTag {
			switch name {
			case "tr":
				//TODO
				p.insertHtmlElement(*tag)
				p.insertionMode = mode_InRow
				return nil
			case "th", "td":
				//TODO
				p.insertHtmlElement(*html_tokenizer.NewTokenTag("tr", html_tokenizer.TokenStartTag, utils.None[bool]()))
				p.insertionMode = mode_InRow
				p.tokenizer.ReconsumeToken(token)
				return nil
			case "caption", "col", "colgroup", "tbody", "tfoot":

				p.tokenizer.ReconsumeToken(token)
				return nil
			}
		} else {
			switch name {
			case "tbody", "tfoot", "thead":

				return nil
			case "table":

				p.tokenizer.ReconsumeToken(token)
				return nil
			case "body", "caption", "col", "colgroup", "html", "td", "th", "tr":
				//TODO: parse error
				return nil
			}
		}
	}

	return p.state_InTable(token)
}

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-intr
func (p *HtmlParser) state_InRow(token html_tokenizer.Token) error {
	if tag, ok := token.(*html_tokenizer.TokenTag); ok {
		name := tag.GetName()

		if tag.GetType() == html_tokenizer.TokenStartTag {
			switch name {
			case "th", "td":
				//TODO
				p.insertHtmlElement(*tag)
				p.insertionMode = mode_InCell
				return nil
			case "caption", "col", "colgroup", "tbody", "tfoot", "thead", "tr":
				//TODO
				return nil
			}
		} else {
			switch name {
			case "tr":
				//TODO
				return nil
			case "table":
				//TODO
				return nil
			case "tbody", "tfoot", "thead":
				//TODO
				return nil
			case "body", "caption", "col", "colgroup", "html", "td", "th":
				//TODO: parse error
				return nil
			}
		}
	}

	return p.state_InTable(token)
}

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-intd
func (p *HtmlParser) state_InCell(token html_tokenizer.Token) error {
	if tag, ok := token.(*html_tokenizer.TokenTag); ok {
		name := tag.GetName()

		if tag.GetType() == html_tokenizer.TokenEndTag {
			switch name {
			case "td", "th":

				//TODO

				p.insertionMode = mode_InRow
				return nil
			case "body", "caption", "col", "colgroup", "html":
				//TODO: parse error
				return nil
			case "table", "tbody", "tfoot", "thead", "tr":

				//TODO

				p.tokenizer.ReconsumeToken(token)
				return nil
			}
		} else {
			switch name {
			case "caption", "col", "colgroup", "tbody", "td", "tfoot", "thead", "tr":

				//TODO

				p.tokenizer.ReconsumeToken(token)
				return nil
			}

		}
	}

	return p.state_InBody(token)
}

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-intemplate
func (p *HtmlParser) state_InTemplate(token html_tokenizer.Token) error {

	switch tag := token.(type) {
	case html_tokenizer.TokenCharacter,
		html_tokenizer.TokenComment,
		html_tokenizer.TokenDOCTYPE:
		return p.state_InBody(token)
	case html_tokenizer.TokenTag:
		name := tag.GetName()

		if tag.GetType() == html_tokenizer.TokenStartTag {
			switch name {
			case "base", "basefont", "bgsound", "link", "meta", "noframes", "script", "style", "template", "title":
				return p.state_InHead(token)
			case "caption", "colgroup", "tbody", "tfoot", "thead":
				//TODO

				p.insertionMode = mode_InTable
				p.tokenizer.ReconsumeToken(token)
				return nil
			case "col":
				//TODO
				p.insertionMode = mode_InColumnGroup
				return nil
			case "tr":
				//TODO
				p.insertionMode = mode_InTableBody
				return nil
			case "td", "th":
				//TODO
				p.insertionMode = mode_InRow
				p.tokenizer.ReconsumeToken(token)
				return nil
			default:
				//TODO
				p.insertionMode = mode_InBody
				p.tokenizer.ReconsumeToken(token)
				return nil
			}
		} else {
			switch name {
			case "template":
				return p.state_InHead(token)
			default:
				//TODO: parse error
				return nil
			}
		}
	case html_tokenizer.TokenEOF:
		//TODO
		p.tokenizer.ReconsumeToken(token)
		return nil
	}

	return nil
}

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-afterbody
func (p *HtmlParser) state_AfterBody(token html_tokenizer.Token) error {

	switch tag := token.(type) {
	case html_tokenizer.TokenCharacter:
		switch tag.Value {
		case '\t', '\n', '\f', '\r', ' ':
			return p.state_InBody(token)
		}
	case html_tokenizer.TokenComment:
		node := p.openElementsStack[0]

		p.insertComment(tag.Value, node)

		return nil
	case html_tokenizer.TokenDOCTYPE:
		//TODO: parse error
		return nil
	case html_tokenizer.TokenTag:
		if tag.GetName() == "html" {
			if tag.GetType() == html_tokenizer.TokenStartTag {
				return p.state_InBody(token)
			}

			//TODO
			p.insertionMode = mode_AfterAfterBody
			return nil
		}
	case html_tokenizer.TokenEOF:
		return io.EOF
	}

	//TODO: parse error
	p.insertionMode = mode_InBody
	p.tokenizer.ReconsumeToken(token)
	return nil
}

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-inframeset
func (p *HtmlParser) state_InFrameset(token html_tokenizer.Token) error {

	switch tag := token.(type) {
	case html_tokenizer.TokenCharacter:
		switch tag.Value {
		case '\t', '\n', '\f', '\r', ' ':
			p.insertCharacter(tag.Value)
			return nil
		}
	case html_tokenizer.TokenComment:
		p.insertComment(tag.Value, nil)
		return nil
	case html_tokenizer.TokenDOCTYPE:
		//TODO: parse error
		return nil
	case html_tokenizer.TokenTag:
		name := tag.GetName()

		if tag.GetType() == html_tokenizer.TokenStartTag {
			switch name {
			case "html":
				return p.state_InBody(token)
			case "frameset":
				p.insertHtmlElement(tag)
				return nil
			case "frame":
				p.insertHtmlElement(tag)
				p.openStackPop()
				//TODO
				return nil
			case "noframes":
				return p.state_InHead(token)
			}
		} else {
			switch name {
			case "frameset":
				//TODO

				p.insertionMode = mode_AfterFrameset
				return nil
			}
		}

	case html_tokenizer.TokenEOF:
		//TODO
		return io.EOF
	}

	//TODO: parse error
	return nil
}

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-afterframeset
func (p *HtmlParser) state_AfterFrameset(token html_tokenizer.Token) error {

	switch tag := token.(type) {
	case html_tokenizer.TokenCharacter:
		switch tag.Value {
		case '\t', '\n', '\f', '\r', ' ':
			p.insertCharacter(tag.Value)
			return nil
		}
	case html_tokenizer.TokenComment:
		p.insertComment(tag.Value, nil)
		return nil
	case html_tokenizer.TokenDOCTYPE:
		//TODO: parse error
		return nil
	case html_tokenizer.TokenTag:
		name := tag.GetName()

		if tag.GetType() == html_tokenizer.TokenStartTag {
			switch name {
			case "html":
				return p.state_InBody(token)
			case "noframes":
				p.insertionMode = mode_AfterAfterFrameset
				return nil
			}
		} else if name == "html" {
			return p.state_AfterAfterFrameset(token)
		}
	case html_tokenizer.TokenEOF:
		return io.EOF
	}

	//TODO: parse erro
	return nil
}

// https://html.spec.whatwg.org/multipage/parsing.html#the-after-after-body-insertion-mode
func (p *HtmlParser) state_AfterAfterBody(token html_tokenizer.Token) error {
	switch tag := token.(type) {
	case html_tokenizer.TokenComment:
		//TODO
		p.insertComment(tag.Value, nil)
		return nil
	case html_tokenizer.TokenDOCTYPE:
		return p.state_InBody(token)
	case html_tokenizer.TokenCharacter:
		switch tag.Value {
		case '\t', '\n', '\f', '\r', ' ':
			return p.state_InBody(token)
		}
	case html_tokenizer.TokenTag:
		if tag.GetName() == "html" && tag.GetType() == html_tokenizer.TokenStartTag {
			return p.state_InBody(token)
		}
	case html_tokenizer.TokenEOF:
		return io.EOF
	}

	//TODO: parse error
	return p.state_InBody(token)
}

// https://html.spec.whatwg.org/multipage/parsing.html#the-after-after-frameset-insertion-mode
func (p *HtmlParser) state_AfterAfterFrameset(token html_tokenizer.Token) error {

	switch tag := token.(type) {
	case html_tokenizer.TokenComment:
		//TODO
		p.insertComment(tag.Value, nil)
		return nil
	case html_tokenizer.TokenDOCTYPE:
		return p.state_InBody(token)
	case html_tokenizer.TokenTag:
		if tag.GetType() == html_tokenizer.TokenStartTag {
			switch tag.GetName() {
			case "html":
				return p.state_InBody(token)
			case "noframes":
				return p.state_InHead(token)
			}
		}
	case html_tokenizer.TokenEOF:
		return io.EOF
	}

	//TODO: parse error
	return nil
}

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-inforeign
func (p *HtmlParser) foreignContent(token html_tokenizer.Token) error {

	switch tag := token.(type) {
	case html_tokenizer.TokenCharacter:
		switch tag.Value {
		case '\t', '\n', '\f', '\r', ' ':
			p.insertCharacter(tag.Value)
			return nil
		default:
			p.insertCharacter(tag.Value)
			p.framesetOk = false
			return nil
		}
	case html_tokenizer.TokenComment:
		p.insertComment(tag.Value, nil)
		return nil
	case html_tokenizer.TokenDOCTYPE:
		//TODO: parse error
		return nil
	case html_tokenizer.TokenTag:
		name := tag.GetName()

		if tag.GetType() == html_tokenizer.TokenStartTag {
			switch name {
			case "font":
				hasColor := tag.Attributes.Get("color").IsSome()
				hasFace := tag.Attributes.Get("face").IsSome()
				hasSize := tag.Attributes.Get("size").IsSome()
				if !(hasColor || hasFace || hasSize) {
					break
				}
				fallthrough
			case "b", "big", "blockquote", "body", "br", "center", "code", "dd", "div",
				"dl", "dt", "em", "embed", "h1", "h2", "h3", "h4", "h5", "h6", "head", "hr", "i", "img", "li",
				"listing", "menu", "meta", "nobr", "ol", "p", "pre", "ruby", "s", "small", "span", "strong", "strike",
				"sub", "sup", "table", "tt", "u", "ul", "var":
				//TODO parse error

				//TODO
				return nil
			default:
				//TODO

				if tag.IsSelfClosingSet() {
					if name == "script" && p.currentNode().Namespace() == dom.NamespaceSVG {

					} else {
						p.openStackPop()
					}
				}
			}
		} else {
			switch name {
			case "script":
				if p.currentNode().Namespace() == dom.NamespaceSVG {

					return nil
				}
				fallthrough
			default:
				//TODO
			}
		}

	}

	return nil
}

//#endregion
