package html_parser

import (
	"strings"

	"github.com/VisualSource/plex/internal/dom"
	"github.com/VisualSource/plex/internal/dom/elements"
)

// https://html.spec.whatwg.org/multipage/parsing.html#html-integration-point
func isHTMLIntegrationPoint(node dom.Node) bool {
	if tag, ok := node.(*elements.Element); ok {
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
	if tag, ok := node.(*elements.Element); ok && tag.Namespace() == dom.NamespaceMathML {
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
