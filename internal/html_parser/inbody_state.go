package html_parser

import (
	"slices"
	"strings"

	"github.com/VisualSource/plex/internal/dom"
	"github.com/VisualSource/plex/internal/html_tokenizer"
	"github.com/VisualSource/plex/internal/utils"
)

func inBody_HandleTag(p *HtmlParser, tag *html_tokenizer.TokenTag) error {
	name := tag.GetName()
	if tag.GetType() == html_tokenizer.TokenStartTag {
		switch name {
		case "html":
			//TODO: parse error

			if node, _ := p.lastElementOfType("template"); node != nil {
				return nil
			}

			if el, ok := p.openElementsStack[0].(*dom.Element); ok {
				for attrName, attrValue := range tag.Attributes {
					if !el.HasAttribute(attrName) {
						el.SetAttributeNode(attrValue)
					}
				}
			}

			return nil
		case "base", "basefont", "bgsound", "link", "meta", "noframes", "script", "style", "template", "title":
			return p.state_InHead(tag)
		case "body":
			//TODO: parse error

			_, hasTemplate := p.lastElementOfType("template")
			if len(p.openElementsStack) == 1 || p.openElementsStack[1].Tag() != "body" || hasTemplate != -1 {
				return nil
			}
			p.framesetOk = false

			if el, ok := p.openElementsStack[1].(*dom.Element); ok {
				for attrName, attrValue := range tag.Attributes {
					if !el.HasAttribute(attrName) {
						el.SetAttributeNode(attrValue)
					}
				}
			}

			return nil
		case "frameset":
			//TODO: parse error
			if len(p.openElementsStack) == 1 || p.openElementsStack[1].Tag() != "body" {
				return nil
			}

			if !p.framesetOk {
				return nil
			}

			p.openElementsStack[1].Remove()

			for len(p.openElementsStack) > 1 {
				p.popOpenStack()
			}

			p.insertHtmlElement(tag)
			p.insertionMode = mode_InFrameset
			return nil
		case "address", "article", "aside", "blockquote", "center", "details",
			"dialog", "dir", "div", "dl", "fieldset", "figcaption", "figure", "footer",
			"header", "hgroup", "main", "menu", "nav", "ol", "p", "search", "section", "summary", "ul":
			if isInButtonScope(p, "p") {
				inBody_ClosePTag(p)
			}

			p.insertHtmlElement(tag)
			return nil
		case "h1", "h2", "h3", "h4", "h5", "h6":
			if isInButtonScope(p, "p") {
				inBody_ClosePTag(p)
			}

			current := p.currentNode()
			if current.Namespace() == dom.NamespaceHTML {
				switch current.Tag() {
				case "h1", "h2", "h3", "h4", "h5", "h6":
					p.popOpenStack()
				}
			}

			p.insertHtmlElement(tag)
			return nil
		case "pre", "listing":
			if isInButtonScope(p, "p") {
				inBody_ClosePTag(p)
			}

			p.insertHtmlElement(tag)

			p.skipNextLineFeed = true

			p.framesetOk = false

			return nil
		case "form":
			template, _ := p.lastElementOfType("template")
			if p.form != nil && template == nil {
				//TODO: parse error
				return nil
			}

			if isInButtonScope(p, "p") {
				inBody_ClosePTag(p)
			}

			node := p.insertHtmlElement(tag)
			if template, _ = p.lastElementOfType("template"); template == nil {
				p.form = node
			}
			return nil
		case "li":
			p.framesetOk = false

			idx := len(p.openElementsStack) - 1
			for idx >= 0 {
				node := p.openElementsStack[idx]
				if node.Tag() == "li" {
					generateImpliedEndTags(p, "li")
					if p.currentNode().Tag() != "li" {
						//TODO: parse error
					}

					for {
						if popped := p.popOpenStack(); popped == nil || popped.Tag() == "li" {
							break
						}
					}

					break
				} else if isSpecialElement(node.Tag(), utils.Some(node.Namespace())) && !slices.Contains([]string{"address", "div", "p"}, node.Tag()) {
					break
				}

				idx--
			}

			if isInButtonScope(p, "p") {
				inBody_ClosePTag(p)
			}

			p.insertHtmlElement(tag)
			return nil
		case "dd", "dt":
			p.framesetOk = false

			idx := len(p.openElementsStack) - 1
			for idx >= 0 {
				node := p.openElementsStack[idx]
				name := node.Tag()
				if name == "dt" || name == "dd" {
					generateImpliedEndTags(p, name)
					if p.currentNode().Tag() != name {
						//TODO: parse error
					}
					for {
						if el := p.popOpenStack(); el == nil || el.Tag() == name {
							break
						}
					}
					break
				} else if isSpecialElement(name, utils.Some(node.Namespace())) && !slices.Contains([]string{"address", "div", "p"}, name) {
					break
				}

				idx--
			}

			if isInButtonScope(p, "p") {
				inBody_ClosePTag(p)
			}

			p.insertHtmlElement(tag)
			return nil
		case "plaintext":
			if isInButtonScope(p, "p") {
				inBody_ClosePTag(p)
			}

			p.insertHtmlElement(tag)
			p.tokenizer.SetState(html_tokenizer.State_PlainText)
			return nil
		case "button":
			if p.haveAnElementTargetNode("button", func(tag string, namespace dom.Namespace) bool {
				return hasParticularElementInScope(tag, namespace)
			}) {
				//TODO: parse error
				generateImpliedEndTags(p)
				for {
					node := p.popOpenStack()
					if node == nil || node.Tag() == "button" {
						break
					}
				}
			}

			p.reconstructActiveFormattingElements()
			p.insertHtmlElement(tag)
			p.framesetOk = false
			return nil
		case "a":
			// Find the last marker, or use start of list if none exists.
			markerIdx := -1
			for i := len(p.activeFormattingElements) - 1; i >= 0; i-- {
				if p.activeFormattingElements[i].IsMarker {
					markerIdx = i
					break
				}
			}

			// Search for an existing <a> element after the last marker.
			anchorIdx := slices.IndexFunc(p.activeFormattingElements[markerIdx+1:], func(el activeFormattingItem) bool {
				return el.Element != nil && el.Element.Tag() == "a"
			})

			if anchorIdx != -1 {
				// Adjust index relative to the full slice.
				anchorIdx += markerIdx + 1
				anchorNode := p.activeFormattingElements[anchorIdx].Element
				//TODO: parse error
				inBody_adoptionAgency(p, tag)
				// Remove from active formatting elements if adoption agency didn't.
				if i := slices.Index(p.activeFormattingElements, activeFormattingItem{Element: anchorNode}); i != -1 {
					p.activeFormattingElements = slices.Delete(p.activeFormattingElements, i, i+1)
				}
				// Remove from open elements stack if adoption agency didn't.
				if i := slices.Index(p.openElementsStack, anchorNode); i != -1 {
					p.openElementsStack = slices.Delete(p.openElementsStack, i, i+1)
				}
			}

			p.reconstructActiveFormattingElements()
			node := p.insertHtmlElement(tag)
			p.pushActiveFormattingElement(node)
			return nil
		case "b", "big", "code", "em", "font", "i", "s", "small", "strike", "strong", "tt", "u":
			p.reconstructActiveFormattingElements()
			node := p.insertHtmlElement(tag)
			p.pushActiveFormattingElement(node)
			return nil
		case "nobr":
			p.reconstructActiveFormattingElements()

			if hasElementInScope(p, "nobr") {
				//TODO: parse error
				inBody_adoptionAgency(p, tag)
				p.reconstructActiveFormattingElements()
			}

			node := p.insertHtmlElement(tag)
			p.pushActiveFormattingElement(node)
			return nil
		case "applet", "marquee", "object":
			p.reconstructActiveFormattingElements()
			p.insertHtmlElement(tag)
			p.activeFormattingElements = append(p.activeFormattingElements, newActiveFormattingMarker())
			p.framesetOk = false
			return nil
		case "table":
			if p.document.QuirksMode != dom.QuirksMode_Quirks && isInButtonScope(p, "p") {
				inBody_ClosePTag(p)
			}

			p.insertHtmlElement(tag)
			p.framesetOk = false
			p.insertionMode = mode_InTable
			return nil
		case "area", "br", "embed", "img", "keygen", "wbr":
			p.reconstructActiveFormattingElements()
			p.insertHtmlElement(tag)
			p.popOpenStack()
			//TODO: ack self-closing
			p.framesetOk = false
			return nil
		case "input":
			if p.isFragmentParsing && p.context != nil && p.context.Tag() == "select" {
				//TODO: parse error
				return nil
			}

			if hasElementInScope(p, "select") {
				//TODO: parse error

				for {
					if node := p.popOpenStack(); node == nil || node.Tag() == "select" {
						break
					}
				}
			}

			p.reconstructActiveFormattingElements()
			p.insertHtmlElement(tag)
			p.popOpenStack()

			//TODO: ack self closing

			typeAttr := tag.Attributes.Get("type")
			if typeAttr.IsNone() || !strings.EqualFold(*typeAttr.Value, "hidden") {
				p.framesetOk = false
			}
			return nil
		case "param", "source", "track":
			p.insertHtmlElement(tag)
			p.popOpenStack()
			//TOOD: ack self close if set
			return nil
		case "hr":
			if isInButtonScope(p, "p") {
				inBody_ClosePTag(p)
			}

			if hasElementInScope(p, "select") {
				generateImpliedEndTags(p)

				if hasElementInScope(p, "option") || hasElementInScope(p, "optgroup") {
					//TODO: parse error
				}
			}

			p.insertHtmlElement(tag)
			p.popOpenStack()
			// ack self close is set
			p.framesetOk = false
			return nil
		case "image":

			// tag name to "img" and reporecess it. (Don't ask)
			tag.SetName("img")
			p.tokenizer.ReconsumeToken(tag)
			return nil
		case "textarea":
			p.insertHtmlElement(tag)

			p.skipNextLineFeed = true

			p.tokenizer.SetState(html_tokenizer.State_RCData)
			p.originalInsertionMode = p.insertionMode
			p.framesetOk = false
			p.insertionMode = mode_Text
			return nil
		case "xmp":
			if isInButtonScope(p, "p") {
				inBody_ClosePTag(p)
			}
			p.reconstructActiveFormattingElements()
			fallthrough
		case "iframe":
			p.framesetOk = false
			fallthrough
		case "noembed":
			genericElementParse(p, tag, "text")
			return nil
		case "select":
			if p.isFragmentParsing && p.context != nil && p.context.Tag() == "select" {
				//TOOD: parse error
				return nil
			}

			if hasElementInScope(p, "select") {
				//TODO: parse error

				for {
					if node := p.popOpenStack(); node == nil || node.Tag() == "select" {
						break
					}
				}

				return nil
			}

			p.reconstructActiveFormattingElements()
			p.insertHtmlElement(tag)
			p.framesetOk = false
			return nil
		case "option":
			if hasElementInScope(p, "select") {
				generateImpliedEndTags(p, "optgroup")
				if hasElementInScope(p, "option") {
					//TODO: parse error
				}
			} else if node := p.currentNode(); node != nil && node.Tag() == "option" {
				p.popOpenStack()

			}

			p.reconstructActiveFormattingElements()
			p.insertHtmlElement(tag)
			return nil
		case "optgroup":
			if hasElementInScope(p, "select") {
				generateImpliedEndTags(p)

				if hasElementInScope(p, "option") || hasElementInScope(p, "optgroup") {
					//TODO: parse error
				}
			} else if node := p.currentNode(); node != nil && node.Tag() == "option" {
				p.popOpenStack()
			}

			p.reconstructActiveFormattingElements()
			p.insertHtmlElement(tag)
			return nil
		case "rb", "rtc":
			if hasElementInScope(p, "ruby") {
				generateImpliedEndTags(p)
				if node := p.currentNode(); node == nil || node.Tag() != "ruby" {
					//TODO: parse error
				}
			}
			p.insertHtmlElement(tag)
			return nil
		case "rp", "rt":
			if hasElementInScope(p, "ruby") {
				generateImpliedEndTags(p, "rtc")
				if node := p.currentNode(); node == nil || !(node.Tag() == "rtc" || node.Tag() == "ruby") {
					//TODO: parse error
				}
			}
			p.insertHtmlElement(tag)
			return nil
		case "math":
			p.reconstructActiveFormattingElements()

			adjustMathMLAttributes(tag)
			adjustForeignAttributes(tag)
			p.insertForeignElement(tag, dom.NamespaceMathML, false)

			if tag.IsSelfClosingSet() {
				p.popOpenStack()
				//TODO: ack self close
			}

			return nil
		case "svg":
			p.reconstructActiveFormattingElements()
			adjustSvgAttributes(tag)
			adjustForeignAttributes(tag)
			p.insertForeignElement(tag, dom.NamespaceSVG, false)

			if tag.IsSelfClosingSet() {
				p.popOpenStack()
				//TODO: ack self close
			}
			return nil
		case "caption", "col", "colgroup", "frame", "head", "tbody", "td", "tfoot", "th", "thead", "tr":
			return nil
		case "noscript":
			if p.scriptingMode != mode_Disabled {
				genericElementParse(p, tag, "text")
				return nil
			}
			fallthrough
		default:
			p.reconstructActiveFormattingElements()
			p.insertHtmlElement(tag)
			return nil
		}
	}

	switch name {
	case "template":
		return p.state_InHead(tag)
	case "body":
		if !hasElementInScope(p, "body") {
			return nil
		}

		//TODO: if no dd,dt,li,optgroup,option,p,rb,rp,rt,rtc,tbody,td,tfoot,th,thead,tr,body,html -> parse error

		p.insertionMode = mode_AfterBody
		return nil
	case "html":
		if !hasElementInScope(p, "html") {
			return nil
		}

		//TODO: no dd,dt,li,optgroup,option,p,rb,rp,rt,rtc,tbody,td,tfoot,th,thead,tr,body,html -> parse error

		p.insertionMode = mode_AfterBody
		p.tokenizer.ReconsumeToken(tag)
		return nil
	case "address", "article", "aside", "blockquote", "button", "center", "details", "dialog", "dir", "div", "dl", "fieldset",
		"figcaption", "figure", "footer", "header", "hgroup", "listing", "main", "menu", "nav", "ol", "pre", "search",
		"section", "select", "summary", "ul":
		if !hasElementInScope(p, name) {
			//TODO: parse error
			return nil
		}

		generateImpliedEndTags(p)
		if node := p.currentNode(); node.Namespace() != dom.NamespaceHTML || node.Tag() != tag.GetName() {
			//TODO: parse error
		}

		for {
			if node := p.popOpenStack(); node == nil || node.Tag() == name && node.Namespace() == dom.NamespaceHTML {
				break
			}
		}

		return nil
	case "form":
		if template, _ := p.lastElementOfType("template"); template != nil {
			if !p.haveAnElementTargetNode("form", func(tag string, namespace dom.Namespace) bool {
				return hasParticularElementInScope(tag, namespace)
			}) {
				//TODO: parse error
				return nil
			}

			generateImpliedEndTags(p)

			if node := p.currentNode(); node == nil || node.Tag() != "form" {
				//TODO: parse error
			}

			for {
				if node := p.popOpenStack(); node == nil || node.Tag() == name && node.Namespace() == dom.NamespaceHTML {
					break
				}
			}

			return nil
		}

		node := p.form
		p.form = nil

		if node == nil || !p.haveAnElementTargetNode(node.Tag(), func(tag string, namespace dom.Namespace) bool { return hasParticularElementInScope(tag, namespace) }) {
			//TODO: parse error
			return nil
		}

		generateImpliedEndTags(p)

		if node != p.currentNode() {
			//TODO: parse error
		}

		idx := slices.Index(p.openElementsStack, node)
		if idx == -1 {
			return nil
		}

		p.openElementsStack = slices.Delete(p.openElementsStack, idx, idx+1)
		return nil
	case "p":
		if !isInButtonScope(p, "p") {
			//TODO: parse error
			p.insertHtmlElement(html_tokenizer.NewTokenTag("p", html_tokenizer.TokenStartTag, utils.Some(false)))
		}

		inBody_ClosePTag(p)
		return nil
	case "li":
		if !isInListScope(p, "li") {
			//TODO: parse error
			return nil
		}

		generateImpliedEndTags(p, "li")
		if node := p.currentNode(); node == nil || node.Tag() != "li" {
			//TODO: parse error
		}

		for {
			if node := p.popOpenStack(); node == nil || (node.Tag() == name && node.Namespace() == dom.NamespaceHTML) {
				break
			}
		}
		return nil
	case "dd", "dt":
		if !p.haveAnElementTargetNode(name, func(tag string, namespace dom.Namespace) bool {
			return hasParticularElementInScope(tag, namespace)
		}) {
			//TODO: parse error
			return nil
		}

		generateImpliedEndTags(p, name)

		if node := p.currentNode(); node == nil || node.Tag() != name {
			//TODO: parse error
		}

		for {
			if node := p.popOpenStack(); node == nil || (node.Tag() == name && node.Namespace() == dom.NamespaceHTML) {
				break
			}
		}
		return nil
	case "h1", "h2", "h3", "h4", "h5", "h6":
		if !(hasElementInScope(p, "h1") || hasElementInScope(p, "h2") || hasElementInScope(p, "h3") || hasElementInScope(p, "h4") || hasElementInScope(p, "h5") || hasElementInScope(p, "h6")) {
			//TODO: parse error
			return nil
		}

		generateImpliedEndTags(p)

		if node := p.currentNode(); node == nil || node.Tag() != name {
			//TODO: parse error
		}

		for {
			node := p.popOpenStack()
			if node == nil {
				break
			}
			if node.Namespace() == dom.NamespaceHTML {
				t := node.Tag()
				if t == "h1" || t == "h2" || t == "h3" || t == "h4" || t == "h5" || t == "h6" {
					break
				}
			}
		}
		return nil
	case "a", "b", "big", "code", "em", "font", "i", "nobr", "s", "small", "strike", "strong", "tt", "u":
		inBody_adoptionAgency(p, tag)
		return nil
	case "applet", "marquee", "object":
		if !p.haveAnElementTargetNode(name, func(tag string, namespace dom.Namespace) bool {
			return hasParticularElementInScope(tag, namespace)
		}) {
			//TODO: parse error
			return nil
		}

		generateImpliedEndTags(p)
		if node := p.currentNode(); node == nil || node.Tag() != name {
			//TODO: parse error
		}

		for {
			if node := p.popOpenStack(); node == nil || (node.Tag() == name && node.Namespace() == dom.NamespaceHTML) {
				break
			}
		}

		clearFormattingElsToLastMarker(p)
		return nil
	case "br":
		clear(tag.Attributes)

		p.reconstructActiveFormattingElements()
		p.insertHtmlElement(tag)
		p.popOpenStack()
		//TODO: ack self-closing
		p.framesetOk = false
		return nil
	case "sarcasm":
		// deep breath
		fallthrough
	default:
		inBody_anyOtherEndTag(p, tag)
	}

	return nil
}

// https://html.spec.whatwg.org/multipage/parsing.html#close-a-p-element
func inBody_ClosePTag(p *HtmlParser) {
	generateImpliedEndTags(p, "p")
	if p.currentNode().Tag() != "p" {
		//TODO: parse error
	}

	for {
		node := p.currentNode()
		isP := node.Tag() == "p"
		p.popOpenStack()
		if isP || node == nil {
			break
		}
	}

}

func inBody_anyOtherEndTag(p *HtmlParser, t *html_tokenizer.TokenTag) {
	idx := len(p.openElementsStack) - 1
	for {
		node := p.openElementsStack[idx]
		if node.Namespace() == dom.NamespaceHTML && node.Tag() == t.GetName() {
			generateImpliedEndTags(p, t.GetName())
			if node != p.currentNode() {
				//TODO: parse error
			}
			p.openElementsStack = p.openElementsStack[:idx]
			return
		}
		if isSpecialElement(node.Tag(), utils.Some(node.Namespace())) {
			//TODO: parse error
			return
		}
		idx--
	}
}

// https://html.spec.whatwg.org/multipage/parsing.html#adoption-agency-algorithm
func inBody_adoptionAgency(p *HtmlParser, tagToken *html_tokenizer.TokenTag) {
	subject := tagToken.GetName()

	currentNode := p.currentNode()
	if currentNode != nil &&
		currentNode.Namespace() == dom.NamespaceHTML &&
		currentNode.Tag() == subject &&
		!slices.ContainsFunc(p.activeFormattingElements, func(e activeFormattingItem) bool {
			return e.Element == currentNode
		}) {
		p.popOpenStack()
		return
	}

	outerLoopCounter := 0
	for {
		if outerLoopCounter >= 8 {
			return
		}
		outerLoopCounter++

		// Find the last element in the active formatting list between end and last marker with tag subject
		formattingElemIdx := -1
		for i := len(p.activeFormattingElements) - 1; i >= 0; i-- {
			if p.activeFormattingElements[i].IsMarker {
				break
			}
			if p.activeFormattingElements[i].Element.Tag() == subject {
				formattingElemIdx = i
				break
			}
		}
		if formattingElemIdx == -1 {
			inBody_anyOtherEndTag(p, tagToken)
			return
		}

		formattingElement := p.activeFormattingElements[formattingElemIdx].Element

		stackIdx := slices.Index(p.openElementsStack, formattingElement)
		if stackIdx == -1 {
			//TODO: parse error
			p.activeFormattingElements = slices.Delete(p.activeFormattingElements, formattingElemIdx, formattingElemIdx+1)
			return
		}

		if !p.haveAnElementTargetNode(subject, func(t string, ns dom.Namespace) bool {
			return hasParticularElementInScope(t, ns)
		}) {
			//TODO: parse error
			return
		}

		if formattingElement != p.currentNode() {
			//TODO: parse error
		}

		// Find furthestBlock: first (topmost) special element in the stack above formattingElement
		var furthestBlock dom.Node
		furthestBlockIdx := -1
		for i := stackIdx + 1; i < len(p.openElementsStack); i++ {
			if isSpecialElement(p.openElementsStack[i].Tag(), utils.Some(p.openElementsStack[i].Namespace())) {
				furthestBlock = p.openElementsStack[i]
				furthestBlockIdx = i
				break
			}
		}

		if furthestBlock == nil {
			for len(p.openElementsStack) > stackIdx {
				p.popOpenStack()
			}
			p.activeFormattingElements = slices.Delete(p.activeFormattingElements, formattingElemIdx, formattingElemIdx+1)
			return
		}

		commonAncestor := p.openElementsStack[stackIdx-1]
		bookmark := formattingElemIdx

		node := furthestBlock
		nodeStackIdx := furthestBlockIdx
		lastNode := furthestBlock

		innerLoopCounter := 0
		for {
			innerLoopCounter++

			// Step to the element immediately above node in the stack
			currentIdx := slices.Index(p.openElementsStack, node)
			if currentIdx != -1 {
				nodeStackIdx = currentIdx
			}
			nodeStackIdx--
			if nodeStackIdx < 0 { //TODO: see if need, could be affects of invalid/unimplemented step
				break
			}
			node = p.openElementsStack[nodeStackIdx]

			if node == formattingElement {
				break
			}

			nodeFormattingIdx := slices.IndexFunc(p.activeFormattingElements, func(e activeFormattingItem) bool {
				return e.Element == node
			})

			if innerLoopCounter > 3 && nodeFormattingIdx != -1 {
				p.activeFormattingElements = slices.Delete(p.activeFormattingElements, nodeFormattingIdx, nodeFormattingIdx+1)
				if nodeFormattingIdx < bookmark {
					bookmark--
				}
				nodeFormattingIdx = -1
			}

			if nodeFormattingIdx == -1 {
				removeIdx := slices.Index(p.openElementsStack, node)
				if removeIdx != -1 {
					p.openElementsStack = slices.Delete(p.openElementsStack, removeIdx, removeIdx+1)
					if removeIdx <= furthestBlockIdx {
						furthestBlockIdx--
					}
					if removeIdx <= stackIdx {
						stackIdx--
					}
					if removeIdx < nodeStackIdx {
						nodeStackIdx--
					}
				}
				continue
			}

			// Create element for the token, with commonAncestor as intended parent
			nodeEl := node.(dom.ElementNode)
			tok := html_tokenizer.NewTokenTag(nodeEl.Tag(), html_tokenizer.TokenStartTag, utils.None[bool]())
			for _, attr := range nodeEl.Attributes() {
				name := attr.GetName()
				tok.Attributes[name] = &dom.Attribute{
					NamespaceUri: attr.NamespaceUri,
					Prefix:       attr.Prefix,
					LocalName:    attr.LocalName,
					Value:        attr.Value,
				}
			}
			newElement := p.createElement(tok, dom.NamespaceHTML, commonAncestor)

			p.activeFormattingElements[nodeFormattingIdx] = activeFormattingItem{Element: newElement}

			replaceIdx := slices.Index(p.openElementsStack, node)
			if replaceIdx != -1 {
				p.openElementsStack[replaceIdx] = newElement
				if replaceIdx == furthestBlockIdx {
					furthestBlock = newElement
				}
				nodeStackIdx = replaceIdx
			}
			node = newElement

			if lastNode == furthestBlock {
				bookmark = nodeFormattingIdx + 1
			}

			lastNode.Remove()
			node.AppendChild(lastNode)
			lastNode = node
		}

		// Insert lastNode at the appropriate place using commonAncestor as override target
		parent, before := p.appropriatePlaceForInsertingNode(commonAncestor)
		if parent != nil {
			lastNode.Remove()
			insertNode(lastNode, parent, before)
		}

		// Create new element for formattingElement's token with furthestBlock as intended parent
		formattingEl := formattingElement.(dom.ElementNode)
		newTok := html_tokenizer.NewTokenTag(formattingEl.Tag(), html_tokenizer.TokenStartTag, utils.None[bool]())
		for _, attr := range formattingEl.Attributes() {
			name := attr.GetName()
			newTok.Attributes[name] = &dom.Attribute{
				NamespaceUri: attr.NamespaceUri,
				Prefix:       attr.Prefix,
				LocalName:    attr.LocalName,
				Value:        attr.Value,
			}
		}
		newFormattingElement := p.createElement(newTok, dom.NamespaceHTML, furthestBlock)

		// Move furthestBlock's children to the new element
		children := slices.Clone(furthestBlock.Children())
		for _, child := range children {
			furthestBlock.RemoveChild(child)
			newFormattingElement.AppendChild(child)
		}
		furthestBlock.AppendChild(newFormattingElement)

		// Remove formattingElement from active formatting list; insert new element at bookmark
		formattingElemIdx = slices.IndexFunc(p.activeFormattingElements, func(e activeFormattingItem) bool {
			return e.Element == formattingElement
		})
		if formattingElemIdx != -1 {
			p.activeFormattingElements = slices.Delete(p.activeFormattingElements, formattingElemIdx, formattingElemIdx+1)
			if formattingElemIdx < bookmark {
				bookmark--
			}
		}
		if bookmark > len(p.activeFormattingElements) {
			bookmark = len(p.activeFormattingElements)
		}
		p.activeFormattingElements = slices.Insert(p.activeFormattingElements, bookmark, activeFormattingItem{Element: newFormattingElement})

		// Remove formattingElement from stack; insert new element immediately below furthestBlock
		stackIdx = slices.Index(p.openElementsStack, formattingElement)
		if stackIdx != -1 {
			p.openElementsStack = slices.Delete(p.openElementsStack, stackIdx, stackIdx+1)
		}
		furthestBlockIdx = slices.Index(p.openElementsStack, furthestBlock)
		if furthestBlockIdx != -1 {
			p.openElementsStack = slices.Insert(p.openElementsStack, furthestBlockIdx+1, newFormattingElement)
		}
	}
}
