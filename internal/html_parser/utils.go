package html_parser

import (
	"strings"

	"github.com/VisualSource/plex/internal/dom"
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
