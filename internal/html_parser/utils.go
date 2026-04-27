package html_parser

import "github.com/VisualSource/plex/internal/dom"

// https://html.spec.whatwg.org/multipage/parsing.html#html-integration-point
func isHTMLIntegrationPoint(node dom.Node) bool {
	return false
}

// https://html.spec.whatwg.org/multipage/parsing.html#mathml-text-integration-point
func isMathMLIntegrationPoint(node dom.Node) bool {

	return false
}
