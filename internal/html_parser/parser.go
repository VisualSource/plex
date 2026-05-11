package html_parser

import (
	"errors"
	"io"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/VisualSource/plex/internal/dom"
	"github.com/VisualSource/plex/internal/html_tokenizer"
	"github.com/VisualSource/plex/internal/utils"
)

type activeFormattingItem struct {
	IsMarker bool
	Element  dom.Node
}

func newActiveFormatingMarker() activeFormattingItem {
	return activeFormattingItem{IsMarker: true}
}

// https://html.spec.whatwg.org/multipage/parsing.html#parse-state
type HtmlParser struct {
	tokenizer *html_tokenizer.Tokenizer
	// https://html.spec.whatwg.org/multipage/parsing.html#the-insertion-mode
	insertionMode         InsertionMode
	originalInsertionMode InsertionMode
	// https://html.spec.whatwg.org/multipage/parsing.html#the-stack-of-open-elements
	openElementsStack []dom.Node
	// https://html.spec.whatwg.org/#the-list-of-active-formatting-elements
	activeFormattingElements []activeFormattingItem
	// https://html.spec.whatwg.org/#other-parsing-state-flags
	scriptingMode ScriptingMode

	templateInsertionModesStack []InsertionMode
	//https://html.spec.whatwg.org/multipage/parsing.html#frameset-ok-flag
	framesetOk bool

	skipNextLineFeed bool

	fosterParenting bool

	context dom.Node

	head dom.Node
	form dom.Node

	speculativeParser *SpeculativeHTMLParser

	document *dom.Document

	isFragmentParsing bool

	pendingTableCharacters []html_tokenizer.TokenCharacter
}

func NewHtmlParser(stream io.Reader) *HtmlParser {
	return &HtmlParser{
		insertionMode: mode_Initial,
		tokenizer:     html_tokenizer.NewTokenizer(stream),
		document:      dom.NewDocument(),
		framesetOk:    true,
	}
}

// https://html.spec.whatwg.org/#tree-construction
func (p *HtmlParser) Parse() (*dom.Document, error) {

parseLoop:
	for {
		aj := p.adjustedCurrentNode()
		p.tokenizer.SetAdjustedNodeIsNotHTML(aj != nil && aj.Namespace() != dom.NamespaceHTML)

		err := p.tokenizer.Next()
		if err != nil && err != io.EOF {
			return nil, err
		}

		for p.tokenizer.HasEmittedTokens() {
			token := p.tokenizer.ConsumeToken()

			aj := p.adjustedCurrentNode()
			isHtmlContext := false

			switch {
			case len(p.openElementsStack) == 0:
				fallthrough
			case aj != nil && aj.Namespace() == dom.NamespaceHTML:
				isHtmlContext = true
			default:
				switch tag := token.(type) {
				case *html_tokenizer.TokenCharacter:
					isHtmlContext = isHTMLIntegrationPoint(aj) || isMathMLIntegrationPoint(aj)
				case *html_tokenizer.TokenTag:
					name := tag.GetName()
					if tag.GetType() != html_tokenizer.TokenStartTag {
						break
					}
					isHtmlContext = isHTMLIntegrationPoint(aj) ||
						(isMathMLIntegrationPoint(aj) && !(name == "mglyph" || name == "malignmark")) ||
						(aj.Namespace() == dom.NamespaceMathML && aj.Tag() == "annotation-xml" && name == "svg")
				case *html_tokenizer.TokenEOF:
					isHtmlContext = true
				}
			}

			if !isHtmlContext {
				if err := p.foreignContent(token); err != nil {
					return nil, err
				}
				continue
			}

			if err := p.processHTMLContent(token); err != nil {
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

func (p *HtmlParser) processHTMLContent(token html_tokenizer.Token) error {
	switch p.insertionMode {
	case mode_Initial:
		return p.state_Initial(token)
	case mode_BeforeHtml:
		return p.state_BeforeHtml(token)
	case mode_BeforeHead:
		return p.state_BeforeHead(token)
	case mode_InHead:
		return p.state_InHead(token)
	case mode_InHeadNoScript:
		return p.state_InHeadNoScript(token)
	case mode_AfterHead:
		return p.state_AfterHead(token)
	case mode_InBody:
		return p.state_InBody(token)
	case mode_Text:
		return p.state_Text(token)
	case mode_InTable:
		return p.state_InTable(token)
	case mode_InTableText:
		return p.state_InTableText(token)
	case mode_InCaption:
		return p.state_InCaption(token)
	case mode_InColumnGroup:
		return p.state_InColumnGroup(token)
	case mode_InTableBody:
		return p.state_InTableBody(token)
	case mode_InRow:
		return p.state_InRow(token)
	case mode_InCell:
		return p.state_InCell(token)
	case mode_InTemplate:
		return p.state_InTemplate(token)
	case mode_AfterBody:
		return p.state_AfterBody(token)
	case mode_InFrameset:
		return p.state_InFrameset(token)
	case mode_AfterFrameset:
		return p.state_AfterFrameset(token)
	case mode_AfterAfterBody:
		return p.state_AfterAfterBody(token)
	case mode_AfterAfterFrameset:
		return p.state_AfterAfterFrameset(token)
	default:
		return errors.New("unknown state")
	}
}

func (p *HtmlParser) openStackPop() dom.Node {
	l := len(p.openElementsStack) - 1
	if l <= 0 {
		return nil
	}

	removed := p.openElementsStack[l]
	p.openElementsStack = slices.Delete(p.openElementsStack, l, l+1)

	return removed
}

func (p *HtmlParser) popTemplateInsertionMode() {
	idx := len(p.templateInsertionModesStack) - 1
	p.templateInsertionModesStack = slices.Delete(p.templateInsertionModesStack, idx, idx+1)
}

// https://html.spec.whatwg.org/multipage/parsing.html#adjusted-current-node
func (p *HtmlParser) adjustedCurrentNode() dom.Node {
	if p.isFragmentParsing && len(p.openElementsStack) == 1 {
		return p.context
	}

	return p.currentNode()
}

// https://html.spec.whatwg.org/multipage/parsing.html#current-template-insertion-mode
func (p *HtmlParser) currentTemplateInsertionMode() InsertionMode {
	if len(p.templateInsertionModesStack) == 0 {
		return mode_Unset
	}
	return p.templateInsertionModesStack[len(p.templateInsertionModesStack)-1]
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
func (*HtmlParser) parseEnd() {

}

//#region Helpers

func (p *HtmlParser) lastElementOfType(nodeType string) (dom.Node, int) {
	for i := len(p.openElementsStack) - 1; i >= 0; i-- {
		if p.openElementsStack[i].Tag() == nodeType {
			return p.openElementsStack[i], i
		}
	}
	return nil, -1
}

// InsertionLocation carries the parent node and, for foster-parenting, the
// sibling to insert before (nil means append as the last child).

func insertNode(node dom.Node, parent dom.Node, before dom.Node) {
	if before == nil {
		parent.AppendChild(node)
	} else {
		parent.InsertBefore(node, before)
	}
}

// https://html.spec.whatwg.org/multipage/parsing.html#appropriate-place-for-inserting-a-node
func (p *HtmlParser) appropriatePlaceForInsertingNode(overrideTarget dom.Node) (parent dom.Node, before dom.Node) {
	var target dom.Node
	if overrideTarget != nil {
		target = overrideTarget
	} else {
		target = p.currentNode()
	}

	adjusted := target
	if p.fosterParenting && slices.Contains([]string{"table", "tbody", "tfoot", "thead", "tr"}, target.Tag()) {
		lastTemplate, tempIdx := p.lastElementOfType("template")
		lastTable, tableIdx := p.lastElementOfType("table")

		if lastTemplate != nil && (tableIdx != -1 || tempIdx > tableIdx) {
			return lastTemplate, nil
		} else if tableIdx == -1 {
			return p.openElementsStack[0], nil
		} else if lastTable != nil && lastTable.Parent() != nil {
			return lastTable.Parent(), lastTable
		}

		return p.openElementsStack[tableIdx-1], nil
	}

	return adjusted, nil
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

	for _, value := range token.Attributes {
		element.SetAttributeNode(value)
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
	parent, before := p.appropriatePlaceForInsertingNode(nil)
	if parent == nil {
		return
	}

	insertNode(element, parent, before)
}

// https://html.spec.whatwg.org/multipage/parsing.html#insert-a-foreign-element
func (p *HtmlParser) insertForeignElement(token html_tokenizer.TokenTag, namespace dom.Namespace, onlyAddToElementStack bool) dom.Node {
	parent, _ := p.appropriatePlaceForInsertingNode(nil)

	el := p.createElement(token, namespace, parent)

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
	var loc dom.Node
	var before dom.Node = nil
	if position == nil {
		loc, before = p.appropriatePlaceForInsertingNode(nil)
	} else {
		loc = position
	}

	comment := dom.NewComment(loc.Document(), loc, data)
	insertNode(comment, loc, before)
}

// https://html.spec.whatwg.org/multipage/parsing.html#insert-a-character
func (p *HtmlParser) insertCharacter(value rune) {
	parent, before := p.appropriatePlaceForInsertingNode(nil)
	if _, ok := parent.(*dom.Document); ok {
		return
	}

	// Find the node immediately before the insertion point and coalesce into
	// it if it is already a Text node, avoiding redundant sibling Text nodes.
	children := parent.Children()
	if before == nil {
		// Normal append: check the current last child.
		if len(children) > 0 {
			if textNode, ok := children[len(children)-1].(*dom.Text); ok {
				textNode.Data += string(value)
				return
			}
		}
	} else {
		// Foster-parented insert-before: check the child just before loc.Before.
		idx := slices.Index(children, before)
		if idx > 0 {
			if textNode, ok := children[idx-1].(*dom.Text); ok {
				textNode.Data += string(value)
				return
			}
		}
	}

	text := dom.NewTextNode(parent.Document(), string(value), parent)
	insertNode(text, parent, before)
}

// https://html.spec.whatwg.org/multipage/parsing.html#push-onto-the-list-of-active-formatting-elements
func (p *HtmlParser) pushActiveFormattingElement(element dom.Node) {
	markerIdx := -1
	for i := len(p.activeFormattingElements) - 1; i >= 0; i-- {
		if p.activeFormattingElements[i].IsMarker {
			markerIdx = i
			break
		}
	}

	newEl, _ := element.(dom.ElementNode)
	count := 0
	firstMatchIdx := -1
	for i := markerIdx + 1; i < len(p.activeFormattingElements); i++ {
		entry := p.activeFormattingElements[i]
		if entry.IsMarker || entry.Element == nil {
			continue
		}
		if entry.Element.Tag() != element.Tag() || entry.Element.Namespace() != element.Namespace() {
			continue
		}
		entryEl, _ := entry.Element.(dom.ElementNode)
		if !formattingAttrsMatch(newEl, entryEl) {
			continue
		}
		count++
		if firstMatchIdx == -1 {
			firstMatchIdx = i
		}
	}

	if count >= 3 && firstMatchIdx != -1 {
		p.activeFormattingElements = slices.Delete(p.activeFormattingElements, firstMatchIdx, firstMatchIdx+1)
	}

	p.activeFormattingElements = append(p.activeFormattingElements, activeFormattingItem{Element: element})
}

func formattingAttrsMatch(a, b dom.ElementNode) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	aAttrs := a.Attributes()
	bAttrs := b.Attributes()
	if len(aAttrs) != len(bAttrs) {
		return false
	}
	for _, attr := range aAttrs {
		found := false
		for _, bAttr := range bAttrs {
			if attr.LocalName == bAttr.LocalName && attr.NamespaceUri == bAttr.NamespaceUri && attr.Value == bAttr.Value {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// https://html.spec.whatwg.org/multipage/parsing.html#reconstruct-the-active-formatting-elements
func (p *HtmlParser) reconstructActiveFormattingElements() {
	if len(p.activeFormattingElements) == 0 {
		return
	}

	last := p.activeFormattingElements[len(p.activeFormattingElements)-1]
	if last.IsMarker || slices.Index(p.openElementsStack, last.Element) != -1 {
		return
	}

	idx := len(p.activeFormattingElements) - 1
	creating := false
	for {
		if !creating {
			if idx > 0 {
				idx--
				entry := p.activeFormattingElements[idx]
				if entry.IsMarker || slices.Index(p.openElementsStack, entry.Element) != -1 {
					idx++ // advance past the boundary
					creating = true
				}
				continue
			}
			creating = true
		}

		entry := p.activeFormattingElements[idx]
		el := entry.Element.(dom.ElementNode)
		tok := html_tokenizer.NewTokenTag(el.Tag(), html_tokenizer.TokenStartTag, utils.None[bool]())
		for _, attr := range el.Attributes() {
			name := attr.GetName()
			tok.Attributes[name] = attr
		}
		newElement := p.insertHtmlElement(*tok)
		p.activeFormattingElements[idx] = activeFormattingItem{Element: newElement}

		if idx == len(p.activeFormattingElements)-1 {
			break
		}
		idx++
	}
}

func (p *HtmlParser) genericElementParse(token html_tokenizer.TokenTag, alg string) {
	p.insertHtmlElement(token)

	if alg == "text" {
		p.tokenizer.SetState(html_tokenizer.State_RawText)
	} else {
		p.tokenizer.SetState(html_tokenizer.State_RCData)
	}

	p.originalInsertionMode = p.insertionMode
	p.insertionMode = mode_Text
}

// https://html.spec.whatwg.org/multipage/parsing.html#reset-the-insertion-mode-appropriately
func (p *HtmlParser) resetInsertionModeAppropriately() {
	last := false

	idx := len(p.openElementsStack) - 1
	node := p.openElementsStack[idx]

	for {
		if node == p.openElementsStack[0] {
			last = true

			if p.isFragmentParsing {
				//TODO: set node to context element
			}
		}

		tag := node.Tag()

		switch tag {
		case "td", "th":
			if !last {
				p.insertionMode = mode_InCell
				return
			}
		case "tr":
			p.insertionMode = mode_InRow
			return
		case "tbody", "thead", "tfoot":
			p.insertionMode = mode_InTableBody
			return
		case "caption":
			p.insertionMode = mode_InCaption
			return
		case "colgroup":
			p.insertionMode = mode_InColumnGroup
			return
		case "table":
			p.insertionMode = mode_InTable
			return
		case "template":
			p.insertionMode = p.currentTemplateInsertionMode()
			return
		case "head":
			if !last {
				p.insertionMode = mode_InHead
				return
			}
		case "body":
			p.insertionMode = mode_InBody
			return
		case "frameset":
			p.insertionMode = mode_InFrameset
			return
		case "html":
			if p.head == nil {
				p.insertionMode = mode_BeforeHead
				return
			}
			p.insertionMode = mode_AfterHead
			return
		}

		if last {
			p.insertionMode = mode_InBody
			return
		}

		idx--
		node = p.openElementsStack[idx]
	}
}

// https://html.spec.whatwg.org/multipage/parsing.html#clear-the-stack-back-to-a-table-context
func (p *HtmlParser) clearStackBackToTableContext() {
	node := p.currentNode()
	for !slices.Contains([]string{"table", "template", "html"}, node.Tag()) {
		p.openStackPop()
		node = p.currentNode()
	}
}

// https://html.spec.whatwg.org/multipage/parsing.html#clear-the-stack-back-to-a-table-body-context
func (p *HtmlParser) clearStackBackToTableBodyContext() {
	x := p.currentNode()
	for !slices.Contains([]string{"tbody", "tfoot", "thead", "template", "html"}, x.Tag()) {
		p.openStackPop()
		x = p.currentNode()
	}
}

// https://html.spec.whatwg.org/multipage/parsing.html#clear-the-stack-back-to-a-table-row-context
func (p *HtmlParser) clearStackBackToRowContext() {
	x := p.currentNode()
	for !slices.Contains([]string{"tr", "template", "html"}, x.Tag()) {
		p.openStackPop()
		x = p.currentNode()
	}
}

// https://html.spec.whatwg.org/multipage/parsing.html#generate-implied-end-tags
func (p *HtmlParser) generateImpliedEndTags(ignore ...string) {
	current := p.currentNode()
	for {
		switch current.Tag() {
		case "dd", "dt", "li", "optgroup", "option", "p", "rb", "rp", "rt", "rtc":
			if slices.Contains(ignore, current.Tag()) {
				return
			}
			p.openStackPop()
			current = p.currentNode()
		default:
			return
		}
	}
}

// https://html.spec.whatwg.org/multipage/parsing.html#has-an-element-in-the-specific-scope
func (p *HtmlParser) haveAnElementTargetNode(target string, checkElementType func(tag string, namespace dom.Namespace) bool) bool {
	for idx := len(p.openElementsStack) - 1; idx >= 0; idx-- {
		node := p.openElementsStack[idx]
		if node.Tag() == target {
			return true
		} else if checkElementType(node.Tag(), node.Namespace()) {
			return false
		}
	}
	return false
}

// https://html.spec.whatwg.org/multipage/parsing.html#has-an-element-in-list-item-scope
func (p HtmlParser) isInListScope(tag string) bool {
	return p.haveAnElementTargetNode(tag, func(name string, namespace dom.Namespace) bool {
		return hasParticularElementInScope(name, namespace) || (namespace == dom.NamespaceHTML && name == "ol" || name == "ul")
	})
}

// https://html.spec.whatwg.org/multipage/parsing.html#has-an-element-in-table-scope
func (p HtmlParser) isInTableScope(tag string) bool {
	return p.haveAnElementTargetNode(tag, func(name string, namespace dom.Namespace) bool {
		return namespace == dom.NamespaceHTML && (name == "html" || name == "table" || name == "template")
	})
}

// https://html.spec.whatwg.org/multipage/parsing.html#has-an-element-in-button-scope
func (p *HtmlParser) isInButtonScope(tag string) bool {
	return p.haveAnElementTargetNode(tag, func(name string, namespace dom.Namespace) bool {
		return hasParticularElementInScope(name, namespace) || (name == "button" && namespace == dom.NamespaceHTML)
	})
}

func (p *HtmlParser) hasElementInScope(name string) bool {
	return p.haveAnElementTargetNode(name, func(tag string, namespace dom.Namespace) bool {
		return hasParticularElementInScope(tag, namespace)
	})
}

//#endregion

//#region InsertionModes

// https://html.spec.whatwg.org/multipage/parsing.html#the-initial-insertion-mode
func (p *HtmlParser) state_Initial(token html_tokenizer.Token) error {

	switch tag := token.(type) {
	case *html_tokenizer.TokenComment:
		p.insertComment(tag.Value, p.document)
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
		p.insertComment(tag.Value, p.document)
		return nil
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
			p.document.AppendChild(node)
			p.openElementsStack = append(p.openElementsStack, node)

			p.insertionMode = mode_BeforeHead
			return nil
		} else if tagType == html_tokenizer.TokenEndTag && !slices.Contains([]string{"head", "body", "html", "br"}, name) {
			return nil
		}
	}

	html := html_tokenizer.NewTokenTag("html", html_tokenizer.TokenStartTag, utils.None[bool]())
	node := p.createElement(*html, dom.NamespaceHTML, p.document)
	p.document.AppendChild(node)
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

func generateAllImpliedEndTagsThoroughly(p *HtmlParser) {
	node := p.currentNode()
	for !slices.Contains([]string{"caption", "colgroup", "dd", "dt", "li", "optgroup", "option", "p", "rb", "rp", "rt", "rtc", "tbody", "td", "tfoot", "th", "thead", "tr"}, node.Tag()) {
		p.openStackPop()
	}
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
				parent, before := p.appropriatePlaceForInsertingNode(nil)

				el := p.createElement(*tag, dom.NamespaceHTML, parent)

				if p.scriptingMode != mode_Fragment {
					//TODO: set parse as document
				}

				//TODO: set force async fase

				if p.scriptingMode == mode_Inert {
					//TODO: set already started = true
				}

				insertNode(el, parent, before)

				p.openElementsStack = append(p.openElementsStack, el)

				p.tokenizer.SetState(html_tokenizer.State_ScriptData)
				p.originalInsertionMode = p.insertionMode
				p.insertionMode = mode_Text
				return nil
			case "template":
				p.activeFormattingElements = append(p.activeFormattingElements, newActiveFormatingMarker())
				p.framesetOk = false
				p.insertionMode = mode_InTemplate
				p.templateInsertionModesStack = append(p.templateInsertionModesStack, mode_InTemplate)

				intendedParent, _ := p.appropriatePlaceForInsertingNode(nil)
				document := intendedParent.Document()

				shadowrootmodeOpt := tag.Attributes.Get("shadowrootmode")
				hasShadowMode := shadowrootmodeOpt.IsSome() &&
					*shadowrootmodeOpt.Value != "" && *shadowrootmodeOpt.Value != "none"
				adjustedCurrent := p.adjustedCurrentNode()
				isNotTopmost := len(p.openElementsStack) > 0 && adjustedCurrent != p.openElementsStack[0]

				if !hasShadowMode || !document.AllowDeclarativeShadowRoots || !isNotTopmost {
					p.insertHtmlElement(*tag)
					return nil
				}

				// Declarative shadow DOM path
				declarativeShadowHost := adjustedCurrent
				template := p.insertForeignElement(*tag, dom.NamespaceHTML, true)

				mode := *shadowrootmodeOpt.Value

				slotAssignment := "named"
				if slotOpt := tag.Attributes.Get("shadowrootslotassignment"); slotOpt.IsSome() && *slotOpt.Value == "manual" {
					slotAssignment = "manual"
				}

				clonable := tag.Attributes["shadowrootclonable"] != nil
				serializable := tag.Attributes["shadowrootserializable"] != nil
				delegatesFocus := tag.Attributes["shadowrootdelegatesfocus"] != nil
				keepRegistryNull := tag.Attributes["shadowrootcustomelementregistry"] != nil

				hostEl, ok := declarativeShadowHost.(*dom.Element)
				if !ok {
					p.insertElement(template)
					return nil
				}

				if hostEl.IsShadowHost() {
					p.insertElement(template)
					return nil
				}

				shadow, err := hostEl.AttachShadow(document, mode, slotAssignment,
					clonable, serializable, delegatesFocus, keepRegistryNull)
				if err != nil {
					p.insertElement(template)
					return nil
				}

				shadow.Declarative = true
				if tmpl, ok := template.(*dom.TemplateElement); ok {
					tmpl.SetTemplateContents(shadow)
				}
				shadow.AvailableToElementInternals = true
				if keepRegistryNull {
					shadow.KeepCustomElementRegistryNull = true
				}

				return nil
			case "head":
				//TOOD : parse error
				return nil
			}
		} else {
			switch name {
			case "head":
				p.openStackPop()
				p.insertionMode = mode_AfterHead
				return nil
			case "body", "html", "br":
			case "template":
				if temp, _ := p.lastElementOfType("template"); temp == nil {
					//TODO: parse error
					return nil
				}

				generateAllImpliedEndTagsThoroughly(p)
				if node := p.currentNode(); node == nil || node.Tag() != "template" {
					//TODO: parse error
				}

				for {
					if el := p.openStackPop(); el == nil || el.Tag() == "template" {
						break
					}
				}

				clearFormattingElsTolastMarker(p)
				p.popTemplateInsertionMode()
				p.resetInsertionModeAppropriately()
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
	case *html_tokenizer.TokenDOCTYPE:
		//TODO: parse error
		return nil
	case *html_tokenizer.TokenTag:
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
	case *html_tokenizer.TokenCharacter:
		switch tag.Value {
		case '\t', '\f', '\n', '\r', ' ':
			return p.state_InHead(token)
		}
	case *html_tokenizer.TokenComment:
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

				p.framesetOk = false
				p.insertionMode = mode_InBody
				return nil
			case "frameset":
				p.insertHtmlElement(*tag)
				p.insertionMode = mode_InFrameset
				return nil
			case "base", "basefont", "bgsound", "link", "meta", "noframes", "script", "style", "template", "title":
				//TODO: parse error
				p.openElementsStack = append(p.openElementsStack, p.head)

				if err := p.state_InHead(token); err != nil {
					return err
				}

				if idx := slices.Index(p.openElementsStack, p.head); idx != -1 {
					p.openElementsStack = slices.Delete(p.openElementsStack, idx, idx+1)
				}
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
		if p.skipNextLineFeed { // handle pre,listing newlines
			p.skipNextLineFeed = false
			if tag.Value == '\n' {
				return nil
			}
		}
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
		return inBody_HandleTag(p, tag)
	case *html_tokenizer.TokenEOF:
		if len(p.templateInsertionModesStack) != 0 {
			return p.state_InTemplate(token)
		}

		// TODO: if no dd,dt,li,optgroup,option,p,rb,rp,rt,rtc,tbody,td,tfoot,th,thead,tr,body,html -> parse error
		return io.EOF
	}

	return nil
}

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-incdata
func (p *HtmlParser) state_Text(token html_tokenizer.Token) error {
	switch tag := token.(type) {
	case *html_tokenizer.TokenCharacter:
		if p.skipNextLineFeed { // handle textarea newlines
			p.skipNextLineFeed = false
			if tag.Value == '\n' {
				return nil
			}
		}

		p.insertCharacter(tag.Value)
	case *html_tokenizer.TokenEOF:
		if node := p.currentNode(); node.Tag() == "script" {
			//TODO: set started flag
		}
		p.openStackPop()
		p.insertionMode = p.originalInsertionMode
		p.tokenizer.ReconsumeToken(token)
		return nil
	case *html_tokenizer.TokenTag:
		if tag.GetType() != html_tokenizer.TokenEndTag {
			return nil
		}
		if tag.GetName() == "script" {

			p.openStackPop()
			p.insertionMode = p.originalInsertionMode

			//TODO: finish

			return nil
		}

		p.openStackPop()
		p.insertionMode = p.originalInsertionMode
	}
	return nil
}

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-intable
func (p *HtmlParser) state_InTable(token html_tokenizer.Token) error {

	switch tag := token.(type) {
	case *html_tokenizer.TokenCharacter:
		node := p.currentNode()
		if slices.Contains([]string{"table", "tbody", "template", "tfoot", "tr"}, node.Tag()) {
			p.pendingTableCharacters = nil
			p.pendingTableCharacters = make([]html_tokenizer.TokenCharacter, 0)

			p.originalInsertionMode = p.insertionMode
			p.insertionMode = mode_InTableText
			p.tokenizer.ReconsumeToken(token)
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
			case "caption":
				p.clearStackBackToTableContext()
				p.activeFormattingElements = append(p.activeFormattingElements, newActiveFormatingMarker())
				p.insertHtmlElement(*tag)
				p.insertionMode = mode_InCaption
				return nil
			case "colgroup":
				p.clearStackBackToTableContext()
				p.insertHtmlElement(*tag)
				p.insertionMode = mode_InColumnGroup
				return nil
			case "col":
				p.clearStackBackToTableContext()
				colgroup := html_tokenizer.NewTokenTag("colgroup", html_tokenizer.TokenStartTag, utils.None[bool]())
				p.insertHtmlElement(*colgroup)
				p.insertionMode = mode_InColumnGroup
				p.tokenizer.ReconsumeToken(token)
				return nil
			case "tbody", "tfoot", "thead":
				p.clearStackBackToTableContext()
				p.insertHtmlElement(*tag)
				p.insertionMode = mode_InTableBody
				return nil
			case "td", "th", "tr":
				p.clearStackBackToTableContext()

				p.insertHtmlElement(*html_tokenizer.NewTokenTag("tbody", html_tokenizer.TokenStartTag, utils.None[bool]()))
				p.insertionMode = mode_InTableBody
				p.tokenizer.ReconsumeToken(token)
				return nil
			case "table":
				//TODO: parse error
				if !p.isInTableScope("table") {
					return nil
				}

				for {
					if node := p.openStackPop(); node == nil || node.Tag() == "table" {
						break
					}
				}
				p.resetInsertionModeAppropriately()
				p.tokenizer.ReconsumeToken(token)
				return nil
			case "style", "script", "template":
				return p.state_InHead(token)
			case "input":
				value := tag.Attributes.Get("type")
				if value.IsSome() && strings.EqualFold(*value.Value, "hidden") {
					//TODO: parse error
					p.insertHtmlElement(*tag)
					p.openStackPop()
					//TODO: ack self closing
					return nil
				}
			case "form":
				temp, _ := p.lastElementOfType("template")
				if temp != nil || p.form != nil {
					return nil
				}

				node := p.insertHtmlElement(*tag)
				p.form = node
				p.openStackPop()
				return nil
			}
		} else {
			switch name {
			case "table":
				//TODO: parse error
				if !p.isInTableScope("table") {
					return nil
				}

				for {
					if node := p.openStackPop(); node == nil || node.Tag() == "table" {
						break
					}
				}

				p.resetInsertionModeAppropriately()
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
	case *html_tokenizer.TokenCharacter:
		switch tag.Value {
		case '\u0000':
			//TODO: parse error
			return nil
		default:
			p.pendingTableCharacters = append(p.pendingTableCharacters, *tag)
		}
	default:
		if slices.ContainsFunc(p.pendingTableCharacters, func(value html_tokenizer.TokenCharacter) bool {
			return !html_tokenizer.IsWhitespace(value.Value)
		}) {
			//TODO: parse error

			for _, el := range p.pendingTableCharacters {
				//TODO: parse error
				p.fosterParenting = true
				if err := p.state_InBody(&el); err != nil {
					return err
				}
				p.fosterParenting = false
			}
		} else {
			for _, el := range p.pendingTableCharacters {
				p.insertCharacter(el.Value)
			}
		}

		p.insertionMode = p.originalInsertionMode
		p.tokenizer.ReconsumeToken(token)
	}

	return nil
}

func clearFormattingElsTolastMarker(p *HtmlParser) {
	for {
		lastIdx := len(p.activeFormattingElements) - 1
		item := p.activeFormattingElements[lastIdx]
		p.activeFormattingElements = slices.Delete(p.activeFormattingElements, lastIdx, lastIdx+1)

		if item.IsMarker {
			break
		}
	}
}

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-incaption
func (p *HtmlParser) state_InCaption(token html_tokenizer.Token) error {
	switch tag := token.(type) {
	case *html_tokenizer.TokenCharacter:
	case *html_tokenizer.TokenTag:
		name := tag.GetName()
		if tag.GetType() == html_tokenizer.TokenStartTag {
			switch name {
			case "caption", "col", "colgroup", "tbody", "td", "tfoot", "th", "thead", "tr":
				if !p.isInTableScope("caption") {
					return nil
				}

				p.generateImpliedEndTags()
				if node := p.currentNode(); node.Tag() != "caption" {
					//TODO: parse error
				}
				for {
					if node := p.openStackPop(); node == nil || node.Tag() == "caption" {
						break
					}
				}

				clearFormattingElsTolastMarker(p)

				p.insertionMode = mode_InTable
				p.tokenizer.ReconsumeToken(token)
				return nil
			}
		} else {
			switch name {
			case "caption":
				if !p.isInTableScope("caption") {
					return nil
				}
				p.generateImpliedEndTags()
				if node := p.currentNode(); node.Tag() != "caption" {
					//TODO: parse error
				}
				for {
					if node := p.openStackPop(); node == nil || node.Tag() == "caption" {
						break
					}
				}
				clearFormattingElsTolastMarker(p)
				p.insertionMode = mode_InTable
				return nil
			case "table":
				if !p.isInTableScope("caption") {
					return nil
				}

				p.generateImpliedEndTags()
				if node := p.currentNode(); node.Tag() != "caption" {
					//TODO: parse error
				}
				for {
					if node := p.openStackPop(); node == nil || node.Tag() == "caption" {
						break
					}
				}

				clearFormattingElsTolastMarker(p)

				p.insertionMode = mode_InTable
				p.tokenizer.ReconsumeToken(token)

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
			case "col":
				p.insertHtmlElement(*tag)
				p.openStackPop()
				//TODO: ack self closing
				return nil
			case "template":
				return p.state_InHead(token)
			}
		} else {
			switch name {
			case "colgroup":
				if node := p.currentNode(); node.Tag() != "colgroup" {
					//TODO: parse error
					return nil
				}

				p.openStackPop()
				p.insertionMode = mode_InTable
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

	if node := p.currentNode(); node == nil || node.Tag() != "colgroup" {
		//TODO: parse error
		return nil
	}

	p.openStackPop()
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
				p.clearStackBackToTableBodyContext()
				p.insertHtmlElement(*tag)
				p.insertionMode = mode_InRow
				return nil
			case "th", "td":
				//TODO: parse error
				p.clearStackBackToTableBodyContext()
				p.insertHtmlElement(*html_tokenizer.NewTokenTag("tr", html_tokenizer.TokenStartTag, utils.None[bool]()))
				p.insertionMode = mode_InRow
				p.tokenizer.ReconsumeToken(token)
				return nil
			case "caption", "col", "colgroup", "tbody", "tfoot":
				if !(p.isInTableScope("tbody") || p.isInTableScope("thead") || p.isInTableScope("tfoot")) {
					//TODO: parse error
					return nil
				}

				p.clearStackBackToTableBodyContext()
				p.openStackPop()
				p.insertionMode = mode_InTable
				p.tokenizer.ReconsumeToken(token)
				return nil
			}
		} else {
			switch name {
			case "tbody", "tfoot", "thead":
				if !p.isInTableScope(name) {
					//TODO: parse error
					return nil
				}

				p.clearStackBackToTableBodyContext()
				p.openStackPop()
				p.insertionMode = mode_InTable
				return nil
			case "table":
				if !(p.isInTableScope("tbody") || p.isInTableScope("thead") || p.isInTableScope("tfoot")) {
					//TODO: parse error
					return nil
				}

				p.clearStackBackToTableBodyContext()
				p.openStackPop()
				p.insertionMode = mode_InTable
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
				p.clearStackBackToRowContext()
				p.insertHtmlElement(*tag)
				p.insertionMode = mode_InCell

				p.activeFormattingElements = append(p.activeFormattingElements, newActiveFormatingMarker())
				return nil
			case "caption", "col", "colgroup", "tbody", "tfoot", "thead", "tr":
				if !p.isInTableScope("tr") {
					//TODO: parse error
					return nil
				}
				p.clearStackBackToRowContext()
				p.openStackPop()
				p.insertionMode = mode_InTableBody
				p.tokenizer.ReconsumeToken(token)
				return nil
			}
		} else {
			switch name {
			case "tr":
				if !p.isInTableScope("tr") {
					//TODO: parse error
					return nil
				}

				p.clearStackBackToRowContext()
				p.openStackPop()
				p.insertionMode = mode_InTableBody
				return nil
			case "table":
				if !p.isInTableScope("tr") {
					//TODO: parse error
					return nil
				}

				p.clearStackBackToRowContext()
				p.openStackPop()
				p.insertionMode = mode_InTableBody
				p.tokenizer.ReconsumeToken(token)
				return nil
			case "tbody", "tfoot", "thead":
				if !p.isInTableScope(name) {
					//TODO: parse error
					return nil
				}
				if !p.isInTableScope("tr") {
					return nil
				}

				p.clearStackBackToRowContext()
				p.openStackPop()
				p.insertionMode = mode_InTableBody
				p.tokenizer.ReconsumeToken(token)
				return nil
			case "body", "caption", "col", "colgroup", "html", "td", "th":
				//TODO: parse error
				return nil
			}
		}
	}

	return p.state_InTable(token)
}

func closeCell(p *HtmlParser) {
	p.generateImpliedEndTags()

	if node := p.currentNode(); node.Tag() != "td" || node.Tag() != "th" {
		//TODO: parse error
	}

	for {
		node := p.openStackPop()
		if node == nil || node.Tag() == "td" || node.Tag() == "th" {
			break
		}
	}

	clearFormattingElsTolastMarker(p)
	p.insertionMode = mode_InRow
}

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-intd
func (p *HtmlParser) state_InCell(token html_tokenizer.Token) error {
	if tag, ok := token.(*html_tokenizer.TokenTag); ok {
		name := tag.GetName()

		if tag.GetType() == html_tokenizer.TokenEndTag {
			switch name {
			case "td", "th":
				if !p.isInTableScope(name) {
					//TODO: parse error
					return nil
				}

				p.generateImpliedEndTags()
				if node := p.currentNode(); !(node.Tag() == name && node.Namespace() == dom.NamespaceHTML) {
					//TODO: parse error
				}

				for {
					if node := p.openStackPop(); node == nil || node.Tag() == name {
						break
					}
				}

				clearFormattingElsTolastMarker(p)
				p.insertionMode = mode_InRow
				return nil
			case "body", "caption", "col", "colgroup", "html":
				//TODO: parse error
				return nil
			case "table", "tbody", "tfoot", "thead", "tr":
				if !p.isInTableScope(name) {
					//TODO: parse error
					return nil
				}

				closeCell(p)
				p.tokenizer.ReconsumeToken(token)
				return nil
			}
		}

		switch name {
		case "caption", "col", "colgroup", "tbody", "td", "tfoot", "th", "thead", "tr":
			if !(p.isInTableScope("td") || p.isInTableScope("th")) {
				panic("Should have a td or th table element!")
			}
			closeCell(p)
			p.tokenizer.ReconsumeToken(token)
			return nil
		}
	}

	return p.state_InBody(token)
}

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-intemplate
func (p *HtmlParser) state_InTemplate(token html_tokenizer.Token) error {

	switch tag := token.(type) {
	case *html_tokenizer.TokenCharacter,
		*html_tokenizer.TokenComment,
		*html_tokenizer.TokenDOCTYPE:
		return p.state_InBody(token)
	case *html_tokenizer.TokenTag:
		name := tag.GetName()

		if tag.GetType() == html_tokenizer.TokenStartTag {
			switch name {
			case "base", "basefont", "bgsound", "link", "meta", "noframes", "script", "style", "template", "title":
				return p.state_InHead(token)
			case "caption", "colgroup", "tbody", "tfoot", "thead":
				p.popTemplateInsertionMode()
				p.templateInsertionModesStack = append(p.templateInsertionModesStack, mode_InTable)
				p.insertionMode = mode_InTable
				p.tokenizer.ReconsumeToken(token)
				return nil
			case "col":
				p.popTemplateInsertionMode()
				p.templateInsertionModesStack = append(p.templateInsertionModesStack, mode_InColumnGroup)
				p.insertionMode = mode_InColumnGroup
				p.tokenizer.ReconsumeToken(token)
				return nil
			case "tr":
				p.popTemplateInsertionMode()
				p.templateInsertionModesStack = append(p.templateInsertionModesStack, mode_InTableBody)
				p.insertionMode = mode_InTableBody
				p.tokenizer.ReconsumeToken(token)
				return nil
			case "td", "th":
				p.popTemplateInsertionMode()
				p.templateInsertionModesStack = append(p.templateInsertionModesStack, mode_InRow)
				p.insertionMode = mode_InRow
				p.tokenizer.ReconsumeToken(token)
				return nil
			default:
				p.popTemplateInsertionMode()
				p.templateInsertionModesStack = append(p.templateInsertionModesStack, mode_InBody)
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
	case *html_tokenizer.TokenEOF:
		if last, _ := p.lastElementOfType("template"); last == nil {
			return io.EOF
		}
		//TODO: parse error

		for {
			if node := p.openStackPop(); node == nil || node.Tag() == "template" {
				break
			}
		}

		clearFormattingElsTolastMarker(p)
		p.popTemplateInsertionMode()
		p.resetInsertionModeAppropriately()
		p.tokenizer.ReconsumeToken(token)
		return nil
	}

	return nil
}

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-afterbody
func (p *HtmlParser) state_AfterBody(token html_tokenizer.Token) error {

	switch tag := token.(type) {
	case *html_tokenizer.TokenCharacter:
		switch tag.Value {
		case '\t', '\n', '\f', '\r', ' ':
			return p.state_InBody(token)
		}
	case *html_tokenizer.TokenComment:
		node := p.openElementsStack[0]

		p.insertComment(tag.Value, node)

		return nil
	case *html_tokenizer.TokenDOCTYPE:
		//TODO: parse error
		return nil
	case *html_tokenizer.TokenTag:
		if tag.GetName() == "html" {
			if tag.GetType() == html_tokenizer.TokenStartTag {
				return p.state_InBody(token)
			}

			if p.isFragmentParsing {
				//TODO: parse error
				return nil
			}

			p.insertionMode = mode_AfterAfterBody
			return nil
		}
	case *html_tokenizer.TokenEOF:
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
			case "frameset":
				p.insertHtmlElement(*tag)
				return nil
			case "frame":
				p.insertHtmlElement(*tag)
				p.openStackPop()
				//TODO: ack self closing
				return nil
			case "noframes":
				return p.state_InHead(token)
			}
		} else {
			switch name {
			case "frameset":
				if node := p.currentNode(); node.Tag() == "html" {
					//TODO: parse error
					return nil
				}

				p.openStackPop()

				if !p.isFragmentParsing && p.currentNode().Tag() != "frameset" {
					p.insertionMode = mode_AfterFrameset
				}
				return nil
			}
		}

	case *html_tokenizer.TokenEOF:
		if node := p.currentNode(); node == nil || node.Tag() != "html" {
			//TODO: parse error
		}
		return io.EOF
	}

	//TODO: parse error
	return nil
}

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-afterframeset
func (p *HtmlParser) state_AfterFrameset(token html_tokenizer.Token) error {

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
			case "noframes":
				return p.state_InHead(token)
			}
		} else if name == "html" {
			p.insertionMode = mode_AfterAfterFrameset
			return nil
		}
	case *html_tokenizer.TokenEOF:
		return io.EOF
	}

	//TODO: parse error
	return nil
}

// https://html.spec.whatwg.org/multipage/parsing.html#the-after-after-body-insertion-mode
func (p *HtmlParser) state_AfterAfterBody(token html_tokenizer.Token) error {
	switch tag := token.(type) {
	case *html_tokenizer.TokenComment:
		p.insertComment(tag.Value, p.document)
		return nil
	case *html_tokenizer.TokenDOCTYPE:
		return p.state_InBody(token)
	case *html_tokenizer.TokenCharacter:
		switch tag.Value {
		case '\t', '\n', '\f', '\r', ' ':
			return p.state_InBody(token)
		}
	case *html_tokenizer.TokenTag:
		if tag.GetName() == "html" && tag.GetType() == html_tokenizer.TokenStartTag {
			return p.state_InBody(token)
		}
	case *html_tokenizer.TokenEOF:
		return io.EOF
	}

	//TODO: parse error
	p.insertionMode = mode_InBody
	p.tokenizer.ReconsumeToken(token)
	return nil
}

// https://html.spec.whatwg.org/multipage/parsing.html#the-after-after-frameset-insertion-mode
func (p *HtmlParser) state_AfterAfterFrameset(token html_tokenizer.Token) error {

	switch tag := token.(type) {
	case *html_tokenizer.TokenComment:
		//TODO
		p.insertComment(tag.Value, p.document)
		return nil
	case *html_tokenizer.TokenCharacter:
		switch tag.Value {
		case '\t', '\n', '\f', '\r', ' ':
			return p.state_InBody(token)
		}
	case *html_tokenizer.TokenDOCTYPE:
		return p.state_InBody(token)
	case *html_tokenizer.TokenTag:
		if tag.GetType() == html_tokenizer.TokenStartTag {
			switch tag.GetName() {
			case "html":
				return p.state_InBody(token)
			case "noframes":
				return p.state_InHead(token)
			}
		}
	case *html_tokenizer.TokenEOF:
		return io.EOF
	}

	//TODO: parse error
	return nil
}

// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-inforeign
func (p *HtmlParser) foreignContent(token html_tokenizer.Token) error {

	switch tag := token.(type) {
	case *html_tokenizer.TokenCharacter:
		switch tag.Value {
		case '\u0000':
			p.insertCharacter(utf8.RuneError)
		case '\t', '\n', '\f', '\r', ' ':
			p.insertCharacter(tag.Value)
			return nil
		default:
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
			case "font":
				hasColor := tag.Attributes.Get("color").IsSome()
				hasFace := tag.Attributes.Get("face").IsSome()
				hasSize := tag.Attributes.Get("size").IsSome()
				if !(hasColor || hasFace || hasSize) {
					break
				}
				fallthrough
			case "b", "big", "blockquote", "body", "br", "center", "code", "dd", "div",
				"dl", "dt", "em", "embed", "h1", "h2", "h3", "h4", "h5", "h6", "head", "hr",
				"i", "img", "li", "listing", "menu", "meta", "nobr", "ol", "p", "pre", "ruby",
				"s", "small", "span", "strong", "strike", "sub", "sup", "table", "tt", "u", "ul", "var":
				//TODO: parse error

				node := p.currentNode()
				for !(isMathMLIntegrationPoint(node) || isHTMLIntegrationPoint(node) || node.Namespace() == dom.NamespaceHTML) {
					p.openStackPop()
					node = p.currentNode()
				}

				return p.processHTMLContent(token)
			default:
				aj := p.adjustedCurrentNode()
				if aj.Namespace() == dom.NamespaceMathML {
					adjustMathMLAttributes(tag)
				} else if aj.Namespace() == dom.NamespaceSVG {
					switch name {
					case "altglyph":
						tag.SetName("altGlyph")
					case "altglyphdef":
						tag.SetName("altGlyphDef")
					case "altglyphitem":
						tag.SetName("altGlyphItem")
					case "animatecolor":
						tag.SetName("animateColor")
					case "animatemotion":
						tag.SetName("animateMotion")
					case "animatetransform":
						tag.SetName("animateTransform")
					case "clippath":
						tag.SetName("clipPath")
					case "feblend":
						tag.SetName("feBlend")
					case "fecolormatrix":
						tag.SetName("feColorMatrix")
					case "fecomponenttransfer":
						tag.SetName("feComponentTransfer")
					case "fecomposite":
						tag.SetName("feComposite")
					case "feconvolvematrix":
						tag.SetName("feConvolveMatrix")
					case "fediffuselighting":
						tag.SetName("feDiffuseLighting")
					case "fedisplacementmap":
						tag.SetName("feDisplacementMap")
					case "fedistantlight":
						tag.SetName("feDistantLight")
					case "fedropshadow":
						tag.SetName("feDropShadown")
					case "feflood":
						tag.SetName("feFlood")
					case "fefunca":
						tag.SetName("feFuncA")
					case "fefuncb":
						tag.SetName("feFuncB")
					case "fefuncg":
						tag.SetName("feFuncG")
					case "fefuncr":
						tag.SetName("feFuncR")
					case "fegaussianblur":
						tag.SetName("feGaussianBlur")
					case "feimage":
						tag.SetName("feImage")
					case "femerge":
						tag.SetName("feMerge")
					case "femergenode":
						tag.SetName("feMergeNode")
					case "femorphology":
						tag.SetName("feMorphology")
					case "feoffset":
						tag.SetName("feOffset")
					case "fepointlight":
						tag.SetName("fePointLight")
					case "fespecularlighting":
						tag.SetName("feSpecularLighting")
					case "fespotlight":
						tag.SetName("feSpotLight")
					case "fetile":
						tag.SetName("feTile")
					case "feturbulence":
						tag.SetName("feTurbulence")
					case "foreignobject":
						tag.SetName("foreignObject")
					case "glyphref":
						tag.SetName("glyphRef")
					case "lineargradient":
						tag.SetName("linearGradient")
					case "radialgradient":
						tag.SetName("radialGradient")
					case "textpath":
						tag.SetName("textPath")
					}

					adjustSvgAttributes(tag)
				}
				adjustForeignAttributes(tag)

				p.insertForeignElement(*tag, aj.Namespace(), false)

				if tag.IsSelfClosingSet() {
					if name == "script" && p.currentNode().Namespace() == dom.NamespaceSVG {
						p.openStackPop()

						//TODO: ack self closing
					} else {
						p.openStackPop()
						//TODO: ack self closing
					}
				}
			}
		} else {
			switch name {

			case "br", "p":
				//TOOD: parse error
				node := p.currentNode()
				for !(isMathMLIntegrationPoint(node) || isHTMLIntegrationPoint(node) || node.Namespace() == dom.NamespaceHTML) {
					p.openStackPop()
					node = p.currentNode()
				}

				return p.processHTMLContent(token)
			case "script":
				if p.currentNode().Namespace() == dom.NamespaceSVG {
					p.openStackPop()

					//TODO:

					return nil
				}
				fallthrough
			default:
				node := p.currentNode()
				nodeIdx := len(p.openElementsStack) - 1

				if strings.ToLower(node.Tag()) != name {
					// parse error
				}

				for {
					if nodeIdx == 0 {
						return nil
					}

					if strings.ToLower(node.Tag()) == name {
						for p.currentNode() != node {
							p.openStackPop()
						}
						p.openStackPop()
						return nil
					}

					nodeIdx--
					node = p.openElementsStack[nodeIdx]

					if node.Namespace() != dom.NamespaceHTML {
						continue
					}

					return p.processHTMLContent(token)
				}
			}
		}
	}

	return nil
}

//#endregion
