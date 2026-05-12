package html_parser

import (
	"slices"
	"strings"

	"github.com/VisualSource/plex/internal/dom"
	"github.com/VisualSource/plex/internal/html_tokenizer"
	"github.com/VisualSource/plex/internal/utils"
)

// https://html.spec.whatwg.org/multipage/parsing.html#html-integration-point
func isHTMLIntegrationPoint(node dom.Node) bool {
	if tag, ok := node.(*dom.Element); ok {
		switch tag.Tag() {
		case "annotation-xml":
			if tag.Namespace() != dom.NamespaceMathML {
				return false
			}
			attr := tag.GetAttribute("encoding")
			if attr == nil {
				return false
			}

			return strings.EqualFold(attr.Value, "text/html") || strings.EqualFold(attr.Value, "application/xhtml+xml")
		case "foreignObject", "desc", "title":
			return tag.Namespace() == dom.NamespaceSVG
		}
	}

	return false
}

// https://html.spec.whatwg.org/multipage/parsing.html#mathml-text-integration-point
func isMathMLIntegrationPoint(node dom.Node) bool {
	if tag, ok := node.(*dom.Element); ok && tag.Namespace() == dom.NamespaceMathML {
		switch tag.Tag() {
		case "mi", "mo", "mn", "ms", "mtext":
			return true
		}
	}

	return false
}

// The following elements have varying levels of special parsing rules: HTML's
//
// @see https://html.spec.whatwg.org/multipage/parsing.html#special
func isSpecialElement(tag string, namespace dom.NamespaceOption) bool {
	switch tag {
	case "address", "applet", "area", "article", "aside", "base", "basefont", "bgsound", "blockquote", "body", "br",
		"button", "caption", "center", "col", "colgroup", "dd", "details", "dir", "div", "dl", "dt", "embed", "fieldset", "figcaption",
		"figure", "footer", "form", "frame", "frameset", "h1", "h2", "h3", "h4", "h5", "head", "header", "hgroup", "hr", "html", "iframe", "img",
		"input", "keygen", "li", "link", "listing", "main", "marquee", "menu", "meta", "nav", "noembed", "noframes", "noscript", "object",
		"ol", "p", "param", "plaintext", "pre", "script", "search", "section", "select", "source", "style", "summary", "table", "tbody", "td",
		"template", "textarea", "tfoot", "th", "thead", "tr", "track", "ul", "wbr", "xmp":
		return namespace.Is(dom.NamespaceHTML)

	case "mi", "mo", "mn", "ms", "mtext", "annotation-xml":
		if !namespace.Is(dom.NamespaceMathML) {
			return false
		}
		return true
	case "foreignObject", "desc":
		if !namespace.Is(dom.NamespaceSVG) {
			return false
		}

		return true

	case "title":
		return namespace.Is(dom.NamespaceHTML) || namespace.Is(dom.NamespaceSVG)
	}

	return false
}

// @see https://html.spec.whatwg.org/multipage/parsing.html#has-an-element-in-scope
func hasParticularElementInScope(tag string, namespace dom.Namespace) bool {
	switch tag {
	case "applet", "caption", "html", "table", "td", "marquee", "object", "select", "template":
		return true
	case "mi", "mo", "mn", "ms", "mtext", "annotation-xml":
		if namespace != dom.NamespaceMathML {
			return false
		}
		return true
	case "foreignObject", "desc", "title":
		if namespace != dom.NamespaceSVG {
			return false
		}
		return true
	default:
		return false
	}
}

func clearFormattingElsToLastMarker(p *HtmlParser) {
	for {
		lastIdx := len(p.activeFormattingElements) - 1
		item := p.activeFormattingElements[lastIdx]
		p.activeFormattingElements = slices.Delete(p.activeFormattingElements, lastIdx, lastIdx+1)

		if item.IsMarker {
			break
		}
	}
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

func genericElementParse(p *HtmlParser, token *html_tokenizer.TokenTag, alg string) {
	p.insertHtmlElement(token)

	if alg == "text" {
		p.tokenizer.SetState(html_tokenizer.State_RawText)
	} else {
		p.tokenizer.SetState(html_tokenizer.State_RCData)
	}

	p.originalInsertionMode = p.insertionMode
	p.insertionMode = mode_Text
}

// https://html.spec.whatwg.org/multipage/parsing.html#generate-implied-end-tags
func generateImpliedEndTags(p *HtmlParser, ignore ...string) {
	current := p.currentNode()
	for {
		switch current.Tag() {
		case "dd", "dt", "li", "optgroup", "option", "p", "rb", "rp", "rt", "rtc":
			if slices.Contains(ignore, current.Tag()) {
				return
			}
			p.popOpenStack()
			current = p.currentNode()
		default:
			return
		}
	}
}

func generateAllImpliedEndTagsThoroughly(p *HtmlParser) {
	node := p.currentNode()
	for slices.Contains([]string{"caption", "colgroup", "dd", "dt", "li", "optgroup", "option", "p", "rb", "rp", "rt", "rtc", "tbody", "td", "tfoot", "th", "thead", "tr"}, node.Tag()) {
		p.popOpenStack()
		node = p.currentNode()
	}
}

func closeCell(p *HtmlParser) {
	generateImpliedEndTags(p)

	if node := p.currentNode(); node.Tag() != "td" || node.Tag() != "th" {
		//TODO: parse error
	}

	for {
		node := p.popOpenStack()
		if node == nil || node.Tag() == "td" || node.Tag() == "th" {
			break
		}
	}

	clearFormattingElsToLastMarker(p)
	p.insertionMode = mode_InRow
}

//#region attributes

var svgAttributeAdjustments = map[string]string{
	"attributename":       "attributeName",
	"attributetype":       "attributeType",
	"basefrequency":       "baseFrequency",
	"baseprofile":         "baseProfile",
	"calcmode":            "calcMode",
	"clippathunits":       "clipPathUnits",
	"diffuseconstant":     "diffuseConstant",
	"edgemode":            "edgeMode",
	"filterunits":         "filterUnits",
	"glyphref":            "glyphRef",
	"gradienttransform":   "gradientTransform",
	"gradientunits":       "gradientUnits",
	"kernelmatrix":        "kernelMatrix",
	"kernelunitlength":    "kernelUnitLength",
	"keypoints":           "keyPoints",
	"keysplines":          "keySplines",
	"keytimes":            "keyTimes",
	"lengthadjust":        "lengthAdjust",
	"limitingconeangle":   "limitingConeAngle",
	"markerheight":        "markerHeight",
	"markerunits":         "markerUnits",
	"markerwidth":         "markerWidth",
	"maskcontentunits":    "maskContentUnits",
	"maskunits":           "maskUnits",
	"numoctaves":          "numOctaves",
	"pathlength":          "pathLength",
	"patterncontentunits": "patternContentUnits",
	"patterntransform":    "patternTransform",
	"patternunits":        "patternUnits",
	"pointsatx":           "pointsAtX",
	"pointsaty":           "pointsAtY",
	"pointsatz":           "pointsAtZ",
	"preservealpha":       "preserveAlpha",
	"preserveaspectratio": "preserveAspectRatio",
	"primitiveunits":      "primitiveUnits",
	"refx":                "refX",
	"refy":                "refY",
	"repeatcount":         "repeatCount",
	"repeatdur":           "repeatDur",
	"requiredextensions":  "requiredExtensions",
	"requiredfeatures":    "requiredFeatures",
	"specularconstant":    "specularConstant",
	"specularexponent":    "specularExponent",
	"spreadmethod":        "spreadMethod",
	"startoffset":         "startOffset",
	"stddeviation":        "stdDeviation",
	"stitchtiles":         "stitchTiles",
	"surfacescale":        "surfaceScale",
	"systemlanguage":      "systemLanguage",
	"tablevalues":         "tableValues",
	"targetx":             "targetX",
	"targety":             "targetY",
	"textlength":          "textLength",
	"viewbox":             "viewBox",
	"viewtarget":          "viewTarget",
	"xchannelselector":    "xChannelSelector",
	"ychannelselector":    "yChannelSelector",
	"zoomandpan":          "zoomAndPan",
}

func adjustSvgAttributes(tag *html_tokenizer.TokenTag) {
	for key, attr := range tag.Attributes {
		if mapped, ok := svgAttributeAdjustments[key]; ok {
			attr.LocalName = mapped
		}
	}
}

func adjustMathMLAttributes(tag *html_tokenizer.TokenTag) {
	value, ok := tag.Attributes["definitionurl"]
	if !ok {
		return
	}

	value.LocalName = "definitionURL"
	tag.Attributes["definitionURL"] = value
	delete(tag.Attributes, "definitionurl")
}

func adjustForeignAttributes(tag *html_tokenizer.TokenTag) {
	for key, attr := range tag.Attributes {
		switch key {
		case "xlink:actuate", "xlink:arcrole",
			"xlink:href", "xlink:role", "xlink:show", "xlink:title", "xlink:type":
			attr.NamespaceUri = dom.NamespaceXLink
			attr.Prefix = utils.Some("xlink")
			attr.LocalName = key[len("xlink:"):]
		case "xml:lang", "xml:space":
			attr.NamespaceUri = dom.NamespaceXML
			attr.Prefix = utils.Some("xml")
			attr.LocalName = key[len("xml:"):]
		case "xmlns":
			attr.NamespaceUri = dom.NamespaceXMLNS
		case "xmlns:xlink":
			attr.NamespaceUri = dom.NamespaceXMLNS
			attr.Prefix = utils.Some("xmlns")
			attr.LocalName = "xlink"
		}
	}
}

//#endregion

//#region clear stack

// https://html.spec.whatwg.org/multipage/parsing.html#clear-the-stack-back-to-a-table-context
func clearStackBackToTableContext(p *HtmlParser) {
	node := p.currentNode()
	for !(node.Namespace() == dom.NamespaceHTML && slices.Contains([]string{"table", "template", "html"}, node.Tag())) {
		p.popOpenStack()
		node = p.currentNode()
	}
}

// https://html.spec.whatwg.org/multipage/parsing.html#clear-the-stack-back-to-a-table-body-context
func clearStackBackToTableBodyContext(p *HtmlParser) {
	x := p.currentNode()
	for !(x.Namespace() == dom.NamespaceHTML && slices.Contains([]string{"tbody", "tfoot", "thead", "template", "html"}, x.Tag())) {
		p.popOpenStack()
		x = p.currentNode()
	}
}

// https://html.spec.whatwg.org/multipage/parsing.html#clear-the-stack-back-to-a-table-row-context
func clearStackBackToRowContext(p *HtmlParser) {
	x := p.currentNode()
	for !(x.Namespace() == dom.NamespaceHTML && slices.Contains([]string{"tr", "template", "html"}, x.Tag())) {
		p.popOpenStack()
		x = p.currentNode()
	}
}

//#endregion

//#region in scope

// https://html.spec.whatwg.org/multipage/parsing.html#has-an-element-in-list-item-scope
func isInListScope(p *HtmlParser, tag string) bool {
	return p.haveAnElementTargetNode(tag, func(name string, namespace dom.Namespace) bool {
		return hasParticularElementInScope(name, namespace) || (namespace == dom.NamespaceHTML && name == "ol" || name == "ul")
	})
}

// https://html.spec.whatwg.org/multipage/parsing.html#has-an-element-in-table-scope
func isInTableScope(p *HtmlParser, tag string) bool {
	return p.haveAnElementTargetNode(tag, func(name string, namespace dom.Namespace) bool {
		return namespace == dom.NamespaceHTML && (name == "html" || name == "table" || name == "template")
	})
}

// https://html.spec.whatwg.org/multipage/parsing.html#has-an-element-in-button-scope
func isInButtonScope(p *HtmlParser, tag string) bool {
	return p.haveAnElementTargetNode(tag, func(name string, namespace dom.Namespace) bool {
		return hasParticularElementInScope(name, namespace) || (name == "button" && namespace == dom.NamespaceHTML)
	})
}

func hasElementInScope(p *HtmlParser, name string) bool {
	return p.haveAnElementTargetNode(name, func(tag string, namespace dom.Namespace) bool {
		return hasParticularElementInScope(tag, namespace)
	})
}

//#endregion
